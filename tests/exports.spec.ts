import {test, expect, type Browser, type BrowserContext, type Page} from '@playwright/test';
import {spawn, type ChildProcess} from 'node:child_process';
import {mkdtempSync, mkdirSync, readFileSync, existsSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const pdfArtifactDir = path.join(root, 'tmp-data', 'phase4-pdf-evidence');

/** Count /Type /Page objects (excludes /Pages trees) in a Chromium-generated PDF. */
function countPdfPages(buf: Buffer): number {
  const raw = buf.toString('latin1');
  const matches = raw.match(/\/Type\s*\/Page(?=[^s])/g);
  return matches ? matches.length : 0;
}

async function inspectPdfWithPypdf(
  pdfPath: string,
  minPages: number,
  needles: string[],
  jsonOut: string,
  opts?: {minImages?: number; samePage?: [string, string][]; headingHasDrawings?: string[]},
): Promise<void> {
  const args = [path.join(root, 'tests', 'inspect_pdf.py'), pdfPath, '--min-pages', String(minPages), '--json-out', jsonOut];
  for (const n of needles) {
    args.push('--contains', n);
  }
  if (opts?.minImages != null) {
    args.push('--min-images', String(opts.minImages));
  }
  for (const pair of opts?.samePage || []) {
    args.push('--same-page', pair[0], pair[1]);
  }
  for (const heading of opts?.headingHasDrawings || []) {
    args.push('--heading-has-drawings', heading);
  }
  // Avoid shell:true so needles with spaces survive on Windows.
  await new Promise<void>((resolve, reject) => {
    const child = spawn(process.platform === 'win32' ? 'python' : 'python3', args, {
      cwd: root,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: false,
    });
    let err = '';
    child.stderr.on('data', d => { err += d.toString(); });
    child.on('exit', code => {
      if (code === 0) resolve();
      else reject(new Error(`inspect_pdf.py -> ${code}\n${err}`));
    });
  });
}

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
  const dataDir = mkdtempSync(path.join(tmpdir(), 'crv-exp-data-'));
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

async function completeSession(
  page: Page,
  base: string,
  op: string,
  choice: 'A' | 'B' | 'C' | 'D',
  record?: Record<string, unknown>,
) {
  const created = await api(page, base, 'POST', '/api/sessions', {operationId: op});
  expect(created.status).toBe(200);
  const id = created.json.session.id as string;
  let lease = created.json.leaseToken as string;
  let rev = created.json.session.revision as number;
  if (record) {
    const saved = await api(page, base, 'PUT', `/api/sessions/${id}/record`, {
      expectedRevision: rev,
      record,
    }, lease);
    expect(saved.status, JSON.stringify(saved.json || saved.text)).toBe(200);
    rev = saved.json.session.revision;
    lease = saved.json.leaseToken || lease;
  }
  const lock = await api(page, base, 'POST', `/api/sessions/${id}/lock`, {expectedRevision: rev}, lease);
  expect(lock.status).toBe(200);
  rev = lock.json.session.revision;
  lease = lock.json.leaseToken || lease;
  const confirm = await api(page, base, 'POST', `/api/sessions/${id}/confirm`, {
    expectedRevision: rev, choice, confidence: 50,
  }, lease);
  expect(confirm.status).toBe(200);
  return confirm.json.session as {code: string; id: string};
}

function richPrintRecord(longMarker: string) {
  const longBody = `${longMarker}\n` + ('trecho longo para paginação natural. '.repeat(120));
  return {
    version: 'crv-record-v1',
    disposition: 'calmo',
    concentration: 'Alta',
    attributesOpen: false,
    groups: {},
    aol1: 'aol1-pdf',
    sensory: longBody,
    aol2: 'aol2-pdf',
    forms: 'curvo',
    dimensions: 'largo',
    positions: 'acima',
    spatial: 'aberto',
    aol3: 'aol3-pdf',
    drawings: {
      ideogram: [{points: [{x: 10, y: 10}, {x: 80, y: 40}, {x: 120, y: 20}], width: 3}],
      sketch: [{points: [{x: 20, y: 80}, {x: 60, y: 30}, {x: 140, y: 90}], width: 2}],
    },
    summary: ['destaque-pdf', '', '', '', ''],
    confidence: 70,
    step: 4,
  };
}

test.describe.configure({mode: 'serial'});

let app: {base: string; url: string; child: ChildProcess; dataDir: string};
let context: BrowserContext;
let page: Page;

test.beforeAll(async ({browser}: {browser: Browser}) => {
  mkdirSync(path.join(root, 'dist'), {recursive: true});
  await run(process.platform === 'win32' ? 'npm.cmd' : 'npm', ['run', 'build']);
  const catDir = mkdtempSync(path.join(tmpdir(), 'crv-exp-cat-'));
  await run('go', ['run', path.join('tests', 'synth.go'), catDir]);
  app = await startApp(catDir);
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

test('CSV, vista de impressão e backup ZIP', async () => {
  const sess = await completeSession(page, app.base, 'e2e-export-1', 'A');

  await page.getByRole('button', {name: 'Histórico'}).click();
  await expect(page.getByRole('heading', {name: 'Histórico'})).toBeVisible();
  await expect(page.getByText(sess.code)).toBeVisible();

  const csvWait = page.waitForEvent('download');
  await page.getByRole('button', {name: 'Exportar CSV'}).click();
  const csvDl = await csvWait;
  const csvPath = await csvDl.path();
  expect(csvPath).toBeTruthy();
  const csvText = readFileSync(csvPath!, 'utf8');
  expect(csvText).toContain('code');
  expect(csvText).toContain('completed');
  expect(csvText.toLowerCase()).not.toContain('sha256');
  expect(csvText.toLowerCase()).not.toContain('farsight');

  await page.getByRole('button', {name: sess.code}).first().click();
  await expect(page.getByRole('button', {name: 'Exportar PDF'})).toBeVisible();
  const popupPromise = page.waitForEvent('popup');
  await page.getByRole('button', {name: 'Exportar PDF'}).click();
  const popup = await popupPromise;
  await popup.waitForLoadState('domcontentloaded');
  const html = await popup.content();
  expect(html).toMatch(/Imprimir|Salvar como PDF/i);
  expect(html).toContain(sess.code);
  await popup.close();

  await page.getByRole('button', {name: 'Configurações'}).click();
  await expect(page.getByRole('heading', {name: 'Cópia de segurança'})).toBeVisible();
  const zipWait = page.waitForEvent('download');
  await page.getByRole('button', {name: 'Baixar backup ZIP'}).click();
  const zipDl = await zipWait;
  const zipPath = path.join(tmpdir(), `crv-e2e-backup-${Date.now()}.zip`);
  await zipDl.saveAs(zipPath);
  expect(existsSync(zipPath)).toBeTruthy();
  expect(readFileSync(zipPath).length).toBeGreaterThan(100);
});

test('PDF headless: gera páginas e inspeciona desenhos, texto longo, créditos e quebras', async () => {
  const longMarker = 'MARCADOR-TEXTO-LONGO-PDF-E2E';
  const sess = await completeSession(page, app.base, 'e2e-export-pdf-rich', 'B', richPrintRecord(longMarker));

  mkdirSync(pdfArtifactDir, {recursive: true});
  const printPage = await context.newPage();
  await printPage.goto(`${app.base}/api/exports/sessions/${encodeURIComponent(sess.id)}/print`);
  await printPage.waitForLoadState('domcontentloaded');

  await expect(printPage.getByRole('heading', {name: `Sessão ${sess.code}`})).toBeVisible();
  await expect(printPage.getByText(/Imprimir|Salvar como PDF/i)).toBeVisible();
  await expect(printPage.getByText(longMarker)).toBeVisible();
  await expect(printPage.getByRole('heading', {name: 'Desenhos'})).toBeVisible();
  await expect(printPage.locator('.pagebreak')).toHaveCount(1);
  await expect(printPage.locator('.drawing svg polyline')).toHaveCount(2);
  await expect(printPage.getByText('Crédito')).toBeVisible();
  await expect(printPage.getByText('fixture')).toBeVisible();
  await expect(printPage.getByText(/Alvo sintético/)).toBeVisible();
  const targetImg = printPage.locator('img.target-img');
  await expect(targetImg).toBeVisible();
  await expect.poll(async () => targetImg.evaluate(el => (el as HTMLImageElement).naturalWidth)).toBeGreaterThan(0);
  const targetSrc = await targetImg.getAttribute('src');
  expect(targetSrc || '').toMatch(/^data:image\/png;base64,/);
  expect(targetSrc || '').not.toContain('ZgotmplZ');
  await expect(printPage.locator('.drawing-block')).toHaveCount(2);

  await printPage.emulateMedia({media: 'print'});
  await printPage.screenshot({
    path: path.join(pdfArtifactDir, 'print-page1-meta.png'),
    fullPage: false,
  });
  await printPage.locator('.pagebreak').scrollIntoViewIfNeeded();
  await printPage.screenshot({
    path: path.join(pdfArtifactDir, 'print-page2-drawings.png'),
    fullPage: false,
  });
  await printPage.screenshot({
    path: path.join(pdfArtifactDir, 'print-full.png'),
    fullPage: true,
  });

  const pdfBuf = Buffer.from(await printPage.pdf({
    format: 'A4',
    printBackground: true,
    preferCSSPageSize: false,
    margin: {top: '12mm', bottom: '12mm', left: '12mm', right: '12mm'},
  }));
  const pdfPath = path.join(pdfArtifactDir, `session-${sess.id}.pdf`);
  writeFileSync(pdfPath, pdfBuf);
  expect(pdfBuf.subarray(0, 5).toString('latin1')).toBe('%PDF-');
  expect(pdfBuf.length).toBeGreaterThan(2_000);

  const pages = countPdfPages(pdfBuf);
  expect(pages).toBeGreaterThanOrEqual(2);

  const inspectJson = path.join(pdfArtifactDir, 'pdf-inspect.json');
  await inspectPdfWithPypdf(pdfPath, 2, [
    sess.code,
    longMarker,
    'fixture',
    'Alvo sintético',
    'Desenhos',
    'Esboço',
    'Ideograma',
  ], inspectJson, {
    minImages: 1,
    headingHasDrawings: ['Ideograma', 'Esboço'],
  });
  const inspected = JSON.parse(readFileSync(inspectJson, 'utf8')) as {
    pages: number;
    pageChars: number[];
    images: number;
    headingHasDrawings?: Record<string, boolean>;
    ok: boolean;
  };
  expect(inspected.ok).toBe(true);
  expect(inspected.pages).toBeGreaterThanOrEqual(2);
  // Explicit CSS page-break before drawings plus long sensory text should span multiple pages.
  expect(inspected.pages).toBeGreaterThanOrEqual(pages);
  expect(inspected.pageChars.some(n => n > 0)).toBe(true);
  expect(inspected.images).toBeGreaterThanOrEqual(1);
  expect(inspected.headingHasDrawings?.Ideograma).toBe(true);
  expect(inspected.headingHasDrawings?.Esboço).toBe(true);

  writeFileSync(
    path.join(pdfArtifactDir, 'summary.json'),
    JSON.stringify({
      sessionId: sess.id,
      code: sess.code,
      pdfBytes: pdfBuf.length,
      pagesChromiumObjects: pages,
      pagesPypdf: inspected.pages,
      pageChars: inspected.pageChars,
      imagesPypdf: inspected.images,
      targetNaturalWidthOk: true,
      longMarkerPresentInHtml: true,
      drawingsPolylines: 2,
      credit: 'fixture',
      artifactPdf: pdfPath,
      screenshots: [
        'print-page1-meta.png',
        'print-page2-drawings.png',
        'print-full.png',
      ],
    }, null, 2),
  );

  await printPage.close();
});
