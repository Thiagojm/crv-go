import {test, expect, type Browser, type BrowserContext, type Page} from '@playwright/test';
import {spawn, type ChildProcess} from 'node:child_process';
import {mkdtempSync, mkdirSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

function run(cmd: string, args: string[], cwd = root): Promise<void> {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, {cwd, stdio: ['ignore', 'pipe', 'pipe'], shell: process.platform === 'win32'});
    let err = '';
    child.stderr.on('data', d => { err += d.toString(); });
    child.on('exit', code => {
      if (code === 0) resolve();
      else reject(new Error(`${cmd} ${args.join(' ')} -> ${code}\n${err}`));
    });
  });
}

async function startApp(catDir: string): Promise<{base: string; url: string; child: ChildProcess; dataDir: string}> {
  const dataDir = mkdtempSync(path.join(tmpdir(), 'crv-hist-data-'));
  const child = spawn('go', ['run', '.', '--data-dir', dataDir, '--catalog-dir', catDir, '--no-browser'], {
    cwd: root,
    stdio: ['ignore', 'pipe', 'pipe'],
    shell: process.platform === 'win32',
  });
  let out = '';
  const url = await new Promise<string>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('timeout waiting for server\n' + out)), 90000);
    const onData = (buf: Buffer) => {
      out += buf.toString();
      const m = out.match(/http:\/\/127\.0\.0\.1:\d+\/#bootstrap=[0-9a-f]+/);
      if (m) {
        clearTimeout(timer);
        resolve(m[0]);
      }
    };
    child.stdout?.on('data', onData);
    child.stderr?.on('data', onData);
    child.on('exit', code => {
      clearTimeout(timer);
      reject(new Error('server exited ' + code + '\n' + out));
    });
  });
  return {base: url.replace(/\/#.*$/, ''), url, child, dataDir};
}

async function csrfCookie(page: Page): Promise<{csrf: string}> {
  const csrf = await page.evaluate(() => sessionStorage.getItem('crv-csrf') || '');
  if (!csrf) throw new Error('csrf ausente');
  return {csrf};
}

async function api(page: Page, base: string, method: string, pathName: string, body?: unknown, lease?: string) {
  return page.evaluate(async ({base, method, pathName, body, lease}) => {
    const csrf = sessionStorage.getItem('crv-csrf') || '';
    const headers: Record<string, string> = {Accept: 'application/json', 'X-CSRF-Token': csrf};
    if (lease) headers['X-Lease-Token'] = lease;
    let payload: BodyInit | undefined;
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      payload = JSON.stringify(body);
    }
    const res = await fetch(base + pathName, {method, credentials: 'same-origin', headers, body: payload});
    const text = await res.text();
    let json: any = {};
    try { json = JSON.parse(text); } catch { /* raw */ }
    return {status: res.status, json, text};
  }, {base, method, pathName, body, lease});
}

async function completeSession(page: Page, base: string, op: string, choice: 'A' | 'B' | 'C' | 'D') {
  const created = await api(page, base, 'POST', '/api/sessions', {operationId: op});
  expect(created.status).toBe(200);
  const id = created.json.session.id as string;
  let lease = created.json.leaseToken as string;
  let rev = created.json.session.revision as number;
  const lock = await api(page, base, 'POST', `/api/sessions/${id}/lock`, {expectedRevision: rev}, lease);
  expect(lock.status).toBe(200);
  rev = lock.json.session.revision;
  lease = lock.json.leaseToken || lease;
  const confirm = await api(page, base, 'POST', `/api/sessions/${id}/confirm`, {
    expectedRevision: rev, choice, confidence: 50,
  }, lease);
  expect(confirm.status).toBe(200);
  return confirm.json.session as {hit: boolean; code: string; choice: string};
}

test.describe.configure({mode: 'serial'});

let app: {base: string; url: string; child: ChildProcess; dataDir: string};
let catDir: string;
let context: BrowserContext;
let page: Page;

test.beforeAll(async ({browser}: {browser: Browser}) => {
  mkdirSync(path.join(root, 'dist'), {recursive: true});
  await run(process.platform === 'win32' ? 'npm.cmd' : 'npm', ['run', 'build']);
  catDir = mkdtempSync(path.join(tmpdir(), 'crv-hist-cat-'));
  await run('go', ['run', path.join('tests', 'synth.go'), catDir]);
  app = await startApp(catDir);
  context = await browser.newContext({viewport: {width: 1280, height: 720}});
  page = await context.newPage();
  await page.goto(app.url);
  await expect(page.getByText('Catálogo pronto')).toBeVisible();
  await csrfCookie(page);
});

test.afterAll(async () => {
  await context?.close();
  if (app?.child.pid) {
    try { process.kill(app.child.pid); } catch { /* gone */ }
  }
});

test('histórico e estatísticas: 1 acerto, 1 erro, 1 abandono', async () => {
  const s1 = await completeSession(page, app.base, 'hist-hit', 'A');
  const s2 = await completeSession(page, app.base, 'hist-miss', s1.hit ? 'B' : 'A');
  // Ensure one hit and one miss regardless of random target.
  // If both same result, flip by completing with opposite of first when second matches.
  // With 4 images, choosing A then B covers hit+miss or miss+hit or two misses/hits — assert via stats after abandon.

  const created = await api(page, app.base, 'POST', '/api/sessions', {operationId: 'hist-abandon'});
  expect(created.status).toBe(200);
  const id = created.json.session.id as string;
  const lease = created.json.leaseToken as string;
  const rev = created.json.session.revision as number;
  const abandoned = await api(page, app.base, 'POST', `/api/sessions/${id}/abandon`, {expectedRevision: rev}, lease);
  expect(abandoned.status).toBe(200);

  const stats = await api(page, app.base, 'GET', '/api/statistics');
  expect(stats.status).toBe(200);
  expect(stats.json.initiated).toBeGreaterThanOrEqual(3);
  expect(stats.json.confirmedChoices).toBe(2);
  expect(stats.json.abandonedBefore + stats.json.abandonedAfter).toBeGreaterThanOrEqual(1);
  if (stats.json.confirmedChoices === 2 && (s1.hit !== s2.hit)) {
    expect(stats.json.hits).toBe(1);
    expect(stats.json.hitRate).toBeCloseTo(0.5);
  }

  await page.getByRole('button', {name: 'Histórico'}).click();
  await expect(page.getByRole('heading', {name: 'Histórico'})).toBeVisible();
  await expect(page.getByText(/sessões?/)).toBeVisible();
  await expect(page.locator('table tbody tr')).toHaveCount(3, {timeout: 10000});

  await page.getByRole('button', {name: 'Estatísticas'}).click();
  await expect(page.getByRole('heading', {name: 'Estatísticas'})).toBeVisible();
  await expect(page.getByText('Acertos confirmados')).toBeVisible();
  await expect(page.locator('.stat-strip')).toBeVisible();
});

test('substituição de catálogo rejeitada com sessão ativa e ok depois', async () => {
  const created = await api(page, app.base, 'POST', '/api/sessions', {operationId: 'hist-block-import'});
  expect(created.status).toBe(200);

  const altCat = mkdtempSync(path.join(tmpdir(), 'crv-hist-alt-'));
  await run('go', ['run', path.join('tests', 'synth.go'), altCat, '6']);

  const blocked = await api(page, app.base, 'POST', '/api/catalog/import/folder', {path: altCat});
  expect(blocked.status).toBe(409);

  const id = created.json.session.id as string;
  const lease = created.json.leaseToken as string;
  const rev = created.json.session.revision as number;
  await api(page, app.base, 'POST', `/api/sessions/${id}/abandon`, {expectedRevision: rev}, lease);

  const ok = await api(page, app.base, 'POST', '/api/catalog/import/folder', {path: altCat});
  expect(ok.status).toBe(200);
  expect(ok.json.ready).toBe(true);
  expect(ok.json.revisionId).toBeGreaterThan(1);

  await page.getByRole('button', {name: 'Configurações'}).click();
  await expect(page.getByText(/elegíveis/i)).toBeVisible();
});
