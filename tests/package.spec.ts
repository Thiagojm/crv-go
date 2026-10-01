import {test, expect} from '@playwright/test';
import {spawn, type ChildProcess} from 'node:child_process';
import {mkdtempSync, mkdirSync, readFileSync, writeFileSync, copyFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';

// Opt in with an extracted distribution; never builds or runs Go/Node children.
test('extracted distribution: session, restart, exports, restore and missing bank', async ({browser}) => {
  test.skip(!process.env.CRV_PACKAGE_DIR, 'Set CRV_PACKAGE_DIR to an extracted platform package');
  test.setTimeout(240000);
  const dir = path.resolve(process.env.CRV_PACKAGE_DIR!);
  const exe = path.join(dir, process.platform === 'win32' ? 'crv.exe' : 'crv');
  const work = mkdtempSync(path.join(tmpdir(), 'crv package verification '));
  console.info(`Package evidence: ${work}; Chromium ${browser.version()}`);
  const data = path.join(work, 'isolated data');
  const cwd = path.join(work, 'unrelated cwd');
  mkdirSync(cwd);
  let child: ChildProcess | undefined;
  const context = await browser.newContext({viewport: {width: 1280, height: 720}});
  const external: string[] = [];
  await context.route('**/*', route => {
    const url = new URL(route.request().url());
    if (url.hostname === '127.0.0.1' || url.protocol === 'data:') return route.continue();
    external.push(url.origin);
    return route.abort();
  });
  async function start(missing = false, catalogDir?: string) {
    child = spawn(exe, ['--data-dir', missing ? path.join(work, 'missing data') : data, '--no-browser',
      ...(missing ? ['--catalog-dir', catalogDir || path.join(work, 'absent bank')] : [])], {
      cwd, env: {...process.env, PATH: ''}, stdio: ['ignore', 'pipe', 'pipe'],
    });
    return await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => { child?.kill(); reject(new Error('Packaged startup timeout')); }, 90000);
      let output = '';
      const collect = (buf: Buffer) => {
        output += buf.toString();
        const match = output.match(/http:\/\/127\.0\.0\.1:\d+\/#bootstrap=[0-9a-f]+/);
        if (match) { clearTimeout(timer); resolve(match[0]); }
      };
      child!.stdout!.on('data', collect);
      child!.stderr!.on('data', collect);
      child!.on('error', err => { clearTimeout(timer); reject(err); });
      child!.on('exit', code => { clearTimeout(timer); reject(new Error(`Packaged startup exited ${code}`)); });
    });
  }
  async function stop(page: import('@playwright/test').Page) {
    const exited = new Promise<number | null>(resolve => child!.once('exit', resolve));
    await page.getByRole('button', {name: 'Salvar e encerrar'}).click();
    await expect(page.getByText('Aplicativo encerrado.')).toBeVisible();
    expect(await exited).toBe(0);
  }
  try {
    const page = await context.newPage();
    page.setDefaultTimeout(15000);
    await page.goto(await start());
    await expect(page.getByText('Catálogo pronto')).toBeVisible();
    await page.screenshot({path: path.join(work, 'light-1280.png'), fullPage: true});
    await page.getByRole('button', {name: 'Alternar tema'}).click();
    await page.screenshot({path: path.join(work, 'dark-1280.png'), fullPage: true});
    await page.setViewportSize({width: 1000, height: 720});
    await page.screenshot({path: path.join(work, 'dark-1000.png'), fullPage: true});
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.setViewportSize({width: 1280, height: 720});
    await page.getByRole('button', {name: 'Alternar tema'}).click();
    await page.getByRole('button', {name: 'Nova sessão'}).click();
    const created = page.waitForResponse(r => r.url().endsWith('/api/sessions') && r.request().method() === 'POST');
    await page.getByRole('button', {name: 'Iniciar'}).click();
    const id = (await (await created).json()).session.id as string;
    const code = await page.locator('.code').innerText();
    const box = (await page.locator('canvas').boundingBox())!;
    await page.mouse.move(box.x + 40, box.y + 40);
    await page.mouse.down();
    await page.mouse.move(box.x + 140, box.y + 90);
    await page.mouse.up();
    await stop(page);
    await page.goto(await start());
    await expect(page.getByText('Sessão pausada')).toBeVisible();
    await expect(page.locator('.code')).toHaveText(code);
    const saved = await context.request.get(new URL(`/api/sessions/${id}`, page.url()).href);
    expect((await saved.json()).session.record.drawings.ideogram.length).toBeGreaterThan(0);
    await page.getByRole('button', {name: 'Assumir nesta aba'}).click();
    await page.getByRole('button', {name: 'Assumir', exact: true}).click();
    await page.getByRole('button', {name: 'Retomar', exact: true}).click();
    for (let step = 0; step < 3; step++) await page.getByRole('button', {name: 'Continuar'}).click();
    await page.getByRole('button', {name: 'Finalizar registro'}).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).not.toBeVisible();
    await page.getByRole('button', {name: 'Finalizar registro'}).click();
    await page.getByRole('button', {name: 'Bloquear e ver imagens'}).click();
    await expect(page.getByRole('img', {name: 'Alternativa A'})).toBeVisible();
    await page.getByRole('button', {name: 'Selecionar alternativa A'}).click();
    await page.getByRole('button', {name: 'Confirmar escolha'}).click();
    await page.getByRole('button', {name: 'Confirmar e revelar'}).click();
    await expect(page.getByText(/alvo sorteado/)).toBeVisible();
    const base = new URL(page.url()).origin;
    const csv = await context.request.get(base + '/api/exports/csv');
    expect(csv.ok()).toBe(true);
    expect(await csv.text()).toContain(code);
    const print = await context.newPage();
    await print.goto(`${base}/api/exports/sessions/${id}/print`);
    await expect.poll(() => print.locator('img.target-img').evaluate(el => (el as HTMLImageElement).naturalWidth)).toBeGreaterThan(0);
    const pdf = await print.pdf({format: 'A4'});
    expect(pdf.subarray(0, 5).toString()).toBe('%PDF-');
    await print.close();
    const backup = await context.request.get(base + '/api/backup');
    expect(backup.ok()).toBe(true);
    const archive = await backup.body();
    const csrf = await page.evaluate(() => sessionStorage.getItem('crv-csrf')!);
    const restore = await context.request.post(base + '/api/backup/restore', {
      headers: {'X-CSRF-Token': csrf, Origin: base},
      multipart: {confirm: 'true', archive: {name: 'backup.zip', mimeType: 'application/zip', buffer: archive}},
    });
    expect(restore.ok(), await restore.text()).toBe(true);
    const token = (await restore.json()).bootstrapToken;
    await page.goto(`${base}/#bootstrap=${token}`);
    await page.reload();
    await expect(page.getByText('Catálogo pronto')).toBeVisible();
    const restored = await context.request.get(`${base}/api/sessions/${id}`);
    expect((await restored.json()).session.state).toBe('completed');
    await stop(page);
    await page.goto(await start(true));
    await expect(page.getByRole('button', {name: 'Nova sessão'})).toBeDisabled();
    await stop(page);
    const inventory = JSON.parse(readFileSync(path.join(dir, 'farsight', 'catalog-unified.json'), 'utf8'));
    const images = inventory.images.filter((image: {sources: {images: unknown[]}[]}) => image.sources.every(source => source.images.length === 1)).slice(0, 3);
    expect(images).toHaveLength(3);
    const smallBank = path.join(work, 'three image bank');
    mkdirSync(smallBank);
    for (const image of images) for (const relative of image.paths) {
      const destination = path.join(smallBank, relative);
      mkdirSync(path.dirname(destination), {recursive: true});
      copyFileSync(path.join(dir, 'farsight', relative), destination);
    }
    writeFileSync(path.join(smallBank, 'catalog-unified.json'), JSON.stringify({format: inventory.format, images}));
    await page.goto(await start(true, smallBank));
    await expect(page.getByText(/pelo menos 4 imagens elegíveis/)).toBeVisible();
    await expect(page.getByRole('button', {name: 'Nova sessão'})).toBeDisabled();
    await stop(page);
    expect(external).toEqual([]);
  } finally {
    child?.kill();
    await context.close();
  }
});
