// check-wasm.mjs — smoke-test the real (WASM) engine.
//
// Loads playground/wasm/witr.wasm (built by build-wasm.sh) with the vendored
// wasm_exec.js and asserts that witr's real engine, running in WebAssembly,
// produces the expected structure over the webbox world. Guards the WASM path
// in CI. Requires the wasm to be built first.
//
//   node playground/scripts/check-wasm.mjs

import { readFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const wasmPath = join(here, '..', 'wasm', 'witr.wasm');
const execPath = join(here, '..', 'vendor', 'wasm_exec.js');
const worldPath = join(here, '..', 'worlds', 'webbox.json');

if (!existsSync(wasmPath)) {
  console.error(`✗ ${wasmPath} not found — run playground/scripts/build-wasm.sh first`);
  process.exit(1);
}

await import(execPath);
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(readFileSync(wasmPath), go.importObject);
go.run(instance);

const world = readFileSync(worldPath, 'utf8');
const NOW = Date.UTC(2026, 6, 21, 12, 0, 0);
const strip = (s) => s.replace(/\x1b\[[0-9;]*m/g, '');
const run = (argv) => globalThis.witrRun(world, NOW, argv);

const checks = [
  { argv: ['node'], exit: 0, includes: ['Process     : node (pid 14233)', 'systemd (pid 1) → PM2 v5.3.1: God (pid 5034) → node (pid 14233)', 'pm2-deploy.service (systemd)', 'Restarts    : 3'] },
  { argv: ['--pid', '1201'], exit: 1, includes: ['nginx.service (systemd)', 'Process is running as root', 'Process is listening on a public interface'] },
  { argv: ['--pid', '40141', '--tree'], exit: 0, includes: ['systemd (pid 1)', '└─ bash (pid 40141)', '├─ python3 (pid 8123)'] },
  { argv: ['--port', '5000', '--short'], exit: 0, includes: ['systemd (pid 1) → PM2 v5.3.1: God (pid 5034) → node (pid 14233)'] },
  { argv: ['ng'], exit: 4, includes: ['Multiple matching processes found', '[4] ngrok (pid 14290)'] },
  { argv: ['--file', '/var/lib/dpkg/lock'], exit: 1, includes: ['Process     : unattended-upgr (pid 33871)', 'apt-daily-upgrade.service (systemd)'] },
  { argv: ['--container', 'redis'], exit: 0, includes: ['Container   : expense-manager-cache-1', 'docker → expense-manager (docker-compose) → expense-manager-cache-1', 'The owning process is not visible'] },
];

let failed = 0;
for (const c of checks) {
  const r = run(c.argv);
  const out = strip(r.output);
  const label = 'witr ' + c.argv.join(' ');
  if (r.exit !== c.exit) {
    console.error(`✗ ${label}: exit ${r.exit}, expected ${c.exit}`);
    failed++;
  }
  for (const needle of c.includes) {
    if (!out.includes(needle)) {
      console.error(`✗ ${label}: missing expected line:\n    ${JSON.stringify(needle)}`);
      failed++;
    }
  }
}

console.log(`${failed === 0 ? '✓' : '✗'} wasm engine: ${checks.length} commands checked, ${failed} assertion(s) failed`);
process.exit(failed === 0 ? 0 : 1);
