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

async function startApp(): Promise<{base: string; url: string; child: ChildProcess}> {
  const dataDir = mkdtempSync(path.join(tmpdir(), 'crv-e2e-data-'));
  const catDir = mkdtempSync(path.join(tmpdir(), 'crv-e2e-cat-'));
  await run('go', ['run', path.join('tests', 'synth.go'), catDir]);
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
  const base = url.replace(/\/#.*$/, '');
  return {base, url, child};
}

test.describe.configure({mode: 'serial'});

let app: {base: string; url: string; child: ChildProcess};
let context: BrowserContext;
let page: Page;

test.beforeAll(async ({browser}: {browser: Browser}) => {
  mkdirSync(path.join(root, 'dist'), {recursive: true});
  await run(process.platform === 'win32' ? 'npm.cmd' : 'npm', ['run', 'build']);
  app = await startApp();
  context = await browser.newContext({viewport: {width: 1280, height: 720}});
  page = await context.newPage();
  await page.goto(app.url);
  await expect(page.getByText('Catálogo pronto')).toBeVisible();
});

test.afterAll(async () => {
  await context?.close();
  if (app?.child.pid) {
    try { process.kill(app.child.pid); } catch { /* gone */ }
  }
});

test('fluxo cego: desenho, campos, bloqueio, escolha e feedback', async () => {
  const bodies: string[] = [];
  page.on('response', async res => {
    const u = res.url();
    if (!u.includes('/api/sessions') || u.includes('/images/')) return;
    if (!(res.headers()['content-type'] || '').includes('json')) return;
    bodies.push(await res.text().catch(() => ''));
  });

  await page.getByRole('button', {name: 'Nova sessão'}).click();
  await page.getByRole('button', {name: 'Iniciar'}).click();
  await expect(page.locator('.code')).toBeVisible();

  const canvas = page.locator('canvas');
  const box = await canvas.boundingBox();
  expect(box).toBeTruthy();
  await page.mouse.move(box!.x + 40, box!.y + 40);
  await page.mouse.down();
  await page.mouse.move(box!.x + 140, box!.y + 90);
  await page.mouse.up();

  await page.getByRole('button', {name: 'Registrar impressões'}).click();
  await page.getByText('Curvo', {exact: true}).click();
  await page.getByLabel('Movimento / forma — escrita livre').fill('linha livre');
  await page.getByRole('button', {name: 'Como preencher'}).click();
  await page.getByText('Ver exemplo fictício').click();
  await page.getByRole('button', {name: /Entendi/}).click();

  expect(bodies.join('\n').toLowerCase()).not.toMatch(/sha256|farsight|targetsha/);

  await page.getByRole('button', {name: 'Continuar'}).click();
  await page.getByRole('button', {name: 'Continuar'}).click();
  await page.getByRole('button', {name: 'Continuar'}).click();
  await expect(page.getByRole('heading', {name: 'Revisão'})).toBeVisible();
  await page.getByRole('button', {name: 'Finalizar registro'}).click();
  await page.getByRole('button', {name: 'Bloquear e ver imagens'}).click();
  await expect(page.getByRole('img', {name: 'Alternativa A'})).toBeVisible();
  expect(bodies.filter(b => b.includes('"state":"locked"')).join('\n').toLowerCase()).not.toMatch(/sha256|targetsha/);

  await page.getByRole('button', {name: 'Selecionar alternativa A'}).click();
  await page.getByRole('button', {name: 'Confirmar escolha'}).click();
  await page.getByRole('button', {name: 'Confirmar e revelar'}).click();
  await expect(page.getByText(/alvo sorteado/)).toBeVisible();
  await page.getByPlaceholder('Esta reflexão fica separada do registro original.').fill('comentário posterior');
});

test('recarga pausa e preserva o registro; segunda aba é somente leitura', async () => {
  await page.getByRole('button', {name: 'CRV — início'}).click();
  await page.getByRole('button', {name: 'Nova sessão'}).click();
  await page.getByRole('button', {name: 'Iniciar'}).click();
  await page.getByRole('button', {name: 'Registrar impressões'}).click();
  await page.getByText('Sólido', {exact: true}).click();
  await page.reload();
  await expect(page.getByText('Sessão pausada')).toBeVisible();
  await expect(page.getByText('Sólido')).toBeVisible();

  const page2 = await context.newPage();
  await page2.goto(app.base + '/');
  const cont = page2.getByRole('button', {name: 'Continuar sessão'});
  if (await cont.isVisible()) await cont.click();
  await expect(page2.getByText(/Outra aba está editando|Assumir nesta aba/)).toBeVisible();
  await page2.close();
});

test('abandonar e encerrar o servidor', async () => {
  if (await page.getByText('Sessão pausada').isVisible()) {
    await page.getByRole('button', {name: 'Retomar', exact: true}).click();
  }
  await page.getByRole('button', {name: 'Abandonar'}).click();
  await page.getByRole('button', {name: 'Abandonar', exact: true}).last().click();
  await expect(page.getByRole('button', {name: 'Nova sessão'})).toBeVisible();
  await page.getByRole('button', {name: 'Salvar e encerrar'}).click();
  await expect(page.getByText('Aplicativo encerrado.')).toBeVisible();
});
