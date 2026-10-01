import { spawnSync } from 'node:child_process';
import {
  cpSync, existsSync, lstatSync, mkdirSync, readFileSync,
  readdirSync, rmSync, writeFileSync, chmodSync, utimesSync,
} from 'node:fs';
import { createHash } from 'node:crypto';
import { gunzipSync, gzipSync } from 'node:zlib';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.join(root, 'packages');
const epoch = new Date('2026-01-01T00:00:00Z');
const isWindows = process.platform === 'win32';
const npm = isWindows ? 'npm.cmd' : 'npm';
const targets = [
  { os: 'windows', goos: 'windows', ext: '.exe' },
  { os: 'linux', goos: 'linux', ext: '' },
];
const generatedNames = new Set(targets.flatMap(({ os }) => {
  const name = `crv-go-${os}-amd64`;
  return [name, `${name}.tar.gz`];
}));

function run(command, args, options = {}) {
  const cmd = isWindows && command === npm ? 'cmd.exe' : command;
  const cmdArgs = isWindows && command === npm ? ['/d', '/s', '/c', `${command} ${args.join(' ')}`] : args;
  const result = spawnSync(cmd, cmdArgs, { cwd: root, encoding: 'utf8', ...options });
  if (result.error || result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed\n${result.stderr || result.error || result.stdout}`);
  }
  return result.stdout.trim();
}

function makeTarExecutable(archive, relativePath) {
  const tar = gunzipSync(readFileSync(archive));
  let executableOffset = -1;
  for (let offset = 0; offset + 512 <= tar.length;) {
    const header = tar.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) break;
    const name = header.subarray(0, 100).toString().replace(/\0.*$/, '');
    const prefix = header.subarray(345, 500).toString().replace(/\0.*$/, '');
    const entry = prefix ? `${prefix}/${name}` : name;
    const size = Number.parseInt(header.subarray(124, 136).toString().replace(/\0.*$/, '').trim() || '0', 8);
    if (entry === relativePath) {
      if (executableOffset !== -1) throw new Error(`Duplicate archive entry: ${relativePath}`);
      executableOffset = offset;
    }
    if (!Number.isFinite(size) || size < 0) throw new Error(`Invalid tar size for ${entry}`);
    offset += 512 + Math.ceil(size / 512) * 512;
  }
  if (executableOffset === -1) throw new Error(`Archive entry not found for executable mode: ${relativePath}`);
  tar.write(`${(0o755).toString(8).padStart(7, '0')}\0`, executableOffset + 100, 8, 'ascii');
  tar.fill(0x20, executableOffset + 148, executableOffset + 156);
  const checksum = tar.subarray(executableOffset, executableOffset + 512).reduce((sum, byte) => sum + byte, 0);
  tar.write(`${checksum.toString(8).padStart(6, '0')}\0 `, executableOffset + 148, 8, 'ascii');
  writeFileSync(archive, gzipSync(tar, { mtime: 0 }));
}

function sha256(file) {
  return createHash('sha256').update(readFileSync(file)).digest('hex');
}

function copyTree(source, destination) {
  mkdirSync(destination, { recursive: true });
  for (const entry of readdirSync(source, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const from = path.join(source, entry.name);
    const to = path.join(destination, entry.name);
    if (entry.isSymbolicLink()) throw new Error(`Symbolic links are not allowed in distribution input: ${from}`);
    if (entry.isDirectory()) copyTree(from, to);
    else if (entry.isFile()) cpSync(from, to);
    else throw new Error(`Unsupported distribution input: ${from}`);
  }
}

function normalizeTimes(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const item = path.join(directory, entry.name);
    if (entry.isDirectory()) normalizeTimes(item);
    utimesSync(item, epoch, epoch);
  }
  utimesSync(directory, epoch, epoch);
}

function copyLicense(source, destination) {
  mkdirSync(path.dirname(destination), { recursive: true });
  cpSync(source, destination);
}

function bundledLicenses(stage) {
  const licenseDir = path.join(stage, 'LICENSES');
  mkdirSync(licenseDir, { recursive: true });
  copyLicense(path.join(root, 'LICENSE'), path.join(stage, 'LICENSE.txt'));
  copyLicense(path.join(root, 'LICENSE'), path.join(licenseDir, 'CRV-Go-MIT.txt'));
  copyLicense(path.join(root, 'docs', 'INTER-OFL.txt'), path.join(licenseDir, 'Inter-OFL-1.1.txt'));

  const gomodcache = run('go', ['env', 'GOMODCACHE']);
  copyLicense(path.join(run('go', ['env', 'GOROOT']), 'LICENSE'), path.join(licenseDir, 'go', 'Go-Standard-Library-BSD-3-Clause.txt'));
  const goMod = readFileSync(path.join(root, 'go.mod'), 'utf8');
  const modules = [...goMod.matchAll(/^\s+([^\s()]+)\s+(v[^\s]+)(?:\s+\/\/ indirect)?$/gm)]
    .map(([, module, version]) => [module, version]);
  let copied = 0;
  for (const [module, version] of modules) {
    const escaped = module.replace(/[A-Z]/g, (letter) => `!${letter.toLowerCase()}`);
    const moduleDir = path.join(gomodcache, `${escaped}@${version}`);
    const candidates = readdirSync(moduleDir, { withFileTypes: true })
      .filter((entry) => entry.isFile() && /^(LICENSE|COPYING|NOTICE)([-_.].*)?$/i.test(entry.name))
      .sort((a, b) => a.name.localeCompare(b.name));
    if (!candidates.length) throw new Error(`No license file found for Go module ${module}@${version}`);
    for (const license of candidates) {
      copyLicense(path.join(moduleDir, license.name), path.join(licenseDir, 'go', `${module.replaceAll('/', '_')}@${version}-${license.name}`));
      copied++;
    }
  }

  // npm packages are build inputs; their exact licenses and integrity hashes are recorded below.
  const lock = JSON.parse(readFileSync(path.join(root, 'package-lock.json'), 'utf8'));
  const npmLicenses = [];
  for (const [packagePath, data] of Object.entries(lock.packages)) {
    if (!packagePath.startsWith('node_modules/')) continue;
    const installed = path.join(root, packagePath);
    if (!existsSync(installed)) continue;
    const name = packagePath.slice('node_modules/'.length).replaceAll('/', '_');
    const licenses = readdirSync(installed, { withFileTypes: true })
      .filter((entry) => entry.isFile() && /^(LICENSE|COPYING|NOTICE)([-_.].*)?$/i.test(entry.name))
      .sort((a, b) => a.name.localeCompare(b.name));
    for (const license of licenses) {
      const filename = `${name}@${data.version}-${license.name}`;
      copyLicense(path.join(installed, license.name), path.join(licenseDir, 'npm', filename));
      npmLicenses.push(filename);
    }
  }
  if (!copied) throw new Error('No Go dependency license files were bundled');
  return { modules, npmLicenses, npmPackages: Object.entries(lock.packages).filter(([key]) => key.startsWith('node_modules/')) };
}

function notices(stage, target, inventory, versions, deps) {
  const lines = [
    readFileSync(path.join(root, 'docs', 'THIRD_PARTY_NOTICES.md'), 'utf8').trim(),
    '',
    '## Build evidence for this archive',
    '',
    `- Target: ${target.goos}/amd64`,
    `- Go: ${versions.go}`,
    `- Node.js: ${versions.node}; npm: ${versions.npm}`,
    `- SHA-256 go.mod: ${sha256(path.join(root, 'go.mod'))}`,
    `- SHA-256 go.sum (all Go module checksums): ${sha256(path.join(root, 'go.sum'))}`,
    `- SHA-256 package-lock.json (npm dependency integrity records): ${sha256(path.join(root, 'package-lock.json'))}`,
    `- Catalog format: ${inventory.format}`,
    `- Catalog inventory SHA-256: ${sha256(path.join(root, 'farsight', 'catalog-unified.json'))}`,
    `- Catalog inventory candidates: ${inventory.images.length}; source files: ${inventory.sourceManifest?.files?.length ?? 'not declared'}`,
    `- Catalog source capture: ${inventory.sourceManifest?.captured_at ?? inventory.provenance?.[0]?.captured_at ?? 'date not declared'}`,
    '',
    `### Go modules (${deps.modules.length})`,
    '',
    ...deps.modules.map(([name, version]) => `- ${name}@${version}`),
    '',
    '### Go dependency checksums (go.sum)',
    '',
    readFileSync(path.join(root, 'go.sum'), 'utf8').trim(),
    '',
    `### npm lock packages (${deps.npmPackages.length})`,
    '',
    ...deps.npmPackages.map(([name, data]) => `- ${name.slice('node_modules/'.length)}@${data.version}: ${data.license ?? 'license not declared'}; integrity ${data.integrity ?? 'not recorded in lockfile'}`),
    '',
    `Bundled Go license files: ${deps.modules.length} module groups plus the Go standard library; bundled npm license files found: ${deps.npmLicenses.length}.`,
  ];
  writeFileSync(path.join(stage, 'THIRD_PARTY_NOTICES.txt'), `${lines.join('\n')}\n`, 'utf8');
}

function packagedInstructions(target) {
  const binary = `crv${target.ext}`;
  return `CRV Go — distribuição local\n\n` +
    `Execute ${binary} (Windows: duplo clique ou PowerShell; Linux: ./crv). ` +
    `A interface abre no navegador padrão e funciona sem internet, Node ou Go instalados.\n\n` +
    `Mantenha a pasta farsight ao lado do executável. Na primeira execução, o catálogo é validado e copiado ` +
    `para os dados locais do usuário. Históricos e preferências pertencem a esta instalação e não são salvos ` +
    `na pasta do pacote. Para mover ou preservar dados, use Configurações > backup.\n\n` +
    `No Linux, conceda permissão de execução com chmod +x crv antes de iniciar. Para encerrar, use “Salvar e encerrar” ` +
    `na aplicação. Consulte THIRD_PARTY_NOTICES.txt para licenças, créditos e hashes das dependências deste build.\n`;
}

function safeRemove(pathname) {
  const resolved = path.resolve(pathname);
  if (path.dirname(resolved) !== output || !generatedNames.has(path.basename(resolved))) {
    throw new Error(`Refusing to remove outside generated package paths: ${resolved}`);
  }
  rmSync(resolved, { recursive: true, force: true });
}

try {
  run(npm, ['run', 'build']);
  const catalogPath = path.join(root, 'farsight');
  if (!existsSync(path.join(catalogPath, 'catalog-unified.json')) || !existsSync(path.join(catalogPath, 'images'))) {
    throw new Error('Bundled farsight catalog is incomplete');
  }
  const inventory = JSON.parse(readFileSync(path.join(catalogPath, 'catalog-unified.json'), 'utf8'));
  if (inventory.format !== 'crv-local-image-inventory-v1' || !Array.isArray(inventory.images)) {
    throw new Error('Bundled catalog has an unsupported format');
  }
  const versions = {
    go: run('go', ['version']),
    node: process.version,
    npm: run(npm, ['--version']),
  };
  const packages = JSON.parse(readFileSync(path.join(root, 'package.json'), 'utf8'));
  const frontendFiles = readdirSync(path.join(root, 'dist'));
  if (!frontendFiles.includes('index.html') || frontendFiles.some((name) => /catalog-unified|farsight|targets/i.test(name))) {
    throw new Error('Frontend output is missing index.html or contains catalog files');
  }

  mkdirSync(output, { recursive: true });
  const oldOutputs = readdirSync(output).filter((name) => generatedNames.has(name));
  for (const name of oldOutputs) safeRemove(path.join(output, name));
  const sums = [];
  for (const target of targets) {
    const name = `crv-go-${target.os}-amd64`;
    const stage = path.join(output, name);
    mkdirSync(stage, { recursive: true });
    const binaryPath = path.join(stage, `crv${target.ext}`);
    run('go', ['build', '-trimpath', '-buildvcs=false', '-o', binaryPath, '.'], {
      env: { ...process.env, CGO_ENABLED: '0', GOOS: target.goos, GOARCH: 'amd64' },
    });
    if (target.goos === 'linux') chmodSync(binaryPath, 0o755);
    copyTree(catalogPath, path.join(stage, 'farsight'));
    writeFileSync(path.join(stage, 'README.txt'), packagedInstructions(target), 'utf8');
    const deps = bundledLicenses(stage);
    notices(stage, target, inventory, versions, deps);
    normalizeTimes(stage);
    const archive = `${stage}.tar.gz`;
    run('tar', ['-czf', archive, '-C', output, name]);
    if (target.goos === 'linux') makeTarExecutable(archive, `${name}/crv`);
    else writeFileSync(archive, gzipSync(gunzipSync(readFileSync(archive)), { mtime: 0 }));
    sums.push(`${sha256(archive)}  ${path.basename(archive)}`);
    console.log(`${archive} (${(lstatSync(archive).size / 1024 / 1024).toFixed(1)} MiB)`);
  }
  writeFileSync(path.join(output, 'SHA256SUMS.txt'), `${sums.join('\n')}\n`, 'utf8');
  console.log(`Packaged ${targets.length} offline archives using ${versions.go}, ${versions.node}, npm ${versions.npm}, app ${packages.version}.`);
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
