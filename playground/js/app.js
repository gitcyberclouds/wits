// app.js — wires the playground together.

import { Shell } from './shell.js';
import { Terminal } from './terminal.js';
import { SystemMap } from './map.js';
import { Tutorial, MISSIONS } from './tutorial.js';
import { TUI } from './tui.js';
import { parse, tokenize } from './parser.js';

const WORLD_IDS = ['webbox', 'devbox'];
const COMPLETIONS = ['witr', 'ls', 'cat', 'ps', 'pwd', 'cd', 'whoami', 'hostname', 'uname', 'neofetch', 'clear', 'help', 'scenario'];
const WITR_FLAGS = ['--pid', '--port', '--file', '--container', '--short', '--tree', '--json', '--env', '--warnings', '--verbose', '--exact', '--no-color', '--interactive', '--help', '--version'];

class App {
  constructor() {
    this.worlds = {};
    this.worldId = 'webbox';
  }

  async boot() {
    for (const id of WORLD_IDS) {
      const res = await fetch(`./worlds/${id}.json`);
      this.worlds[id] = await res.json();
    }

    this.shell = new Shell(this.worlds[this.worldId]);
    this.term = new Terminal(document.getElementById('terminal'));
    this.map = new SystemMap(document.getElementById('map-canvas'), document.getElementById('map-labels'));
    this.tutorial = new Tutorial();
    this.tui = new TUI(document.getElementById('tui'));

    this.term.onSubmit = (line) => this.handle(line);
    this.term.completer = (v) => this.complete(v);
    this.map.onSelect = (proc) => this.launchFromMap(proc);
    this.tui.onClose = () => this.term.focus();

    this.tutorial.onChange = () => this.renderTutorial();
    this.tutorial.onComplete = (m) => this.celebrate(m);
    this.tutorial.onFinish = () => this.finishTutorial();

    this.map.setWorld(this.worlds[this.worldId]);
    this.map.start();
    window.addEventListener('resize', () => this.map.resize());

    this.wireChrome();
    this.applyWorld();
    this.tutorial.start();
    this.welcome();
    this.term.focus();
  }

  // ---- command handling -------------------------------------------------

  handle(line) {
    const res = this.shell.exec(line);
    if (res.action === 'clear') { this.term.clear(); return; }
    if (res.output) this.term.print(res.output);

    if (res.action === 'tui') {
      this.term.print(dimNote('opening interactive dashboard… (press q or Esc to return)'));
      setTimeout(() => this.tui.show(this.currentWorld(), this.shell.engine), 260);
    }
    if (res.action === 'scenario') this.openScenario();

    // Update the map + tutorial based on what was queried.
    const ctx = this.analyze(line, res);
    this.updateMap(ctx);
    this.tutorial.observe(ctx);
    this.term.setPrompt(this.shell.prompt());
  }

  analyze(line, res) {
    const tokens = tokenize(line.trim());
    const isWitr = tokens[0] === 'witr';
    const { targets, flags } = isWitr ? parse(tokens.slice(1)) : { targets: [], flags: {} };
    const eng = this.shell.engine;

    let multi = false;
    for (const t of targets) {
      if (t.type === 'name' && eng.resolveName(t.value, flags.exact).length > 1) multi = true;
      if (t.type === 'container' && eng.resolveContainer(t.value, flags.exact).length > 1) multi = true;
    }
    const plain = isWitr && !flags.tree && !flags.short && !flags.json && !flags.env && !flags.warnings;
    return {
      line, isWitr, targets, flags,
      exit: res.exit, action: res.action,
      hasName: targets.some((t) => t.type === 'name'),
      hasPort: targets.some((t) => t.type === 'port'),
      hasFile: targets.some((t) => t.type === 'file'),
      hasContainer: targets.some((t) => t.type === 'container'),
      multi, plain,
    };
  }

  updateMap(ctx) {
    if (!ctx.isWitr || ctx.targets.length === 0) return;
    const eng = this.shell.engine;
    // Highlight the chain of the first resolvable process target.
    for (const t of ctx.targets) {
      let pid = null;
      if (t.type === 'pid') pid = eng.procByPid.has(+t.value) ? +t.value : null;
      else if (t.type === 'port') pid = eng.resolvePort(+t.value);
      else if (t.type === 'file') pid = eng.resolveFile(t.value);
      else if (t.type === 'name') { const m = eng.resolveName(t.value, ctx.flags.exact); if (m.length === 1) pid = m[0]; }
      else if (t.type === 'container') {
        const runtime = this.currentWorld().processes.find((p) => /docker|containerd/.test(p.command));
        if (runtime) pid = runtime.pid;
      }
      if (pid) {
        const proc = eng.procByPid.get(pid);
        this.map.highlightPids(eng.ancestryOf(proc).map((p) => p.pid));
        return;
      }
    }
    this.map.clearHighlight();
  }

  launchFromMap(proc) {
    if (this.tui.open) return;
    this.term.focus();
    this.term.typeAndRun(`witr --pid ${proc.pid}`);
  }

  // ---- completion -------------------------------------------------------

  complete(value) {
    const tokens = value.split(' ');
    const last = tokens[tokens.length - 1];
    if (tokens.length <= 1) {
      const hits = COMPLETIONS.filter((c) => c.startsWith(last));
      if (hits.length === 1) return hits[0] + ' ';
      return { value, hints: hits };
    }
    if (tokens[0] === 'witr') {
      let pool;
      if (last.startsWith('-')) pool = WITR_FLAGS.filter((f) => f.startsWith(last));
      else pool = this.currentWorld().processes.map((p) => p.command).filter((c, i, a) => a.indexOf(c) === i).filter((c) => c.startsWith(last));
      if (pool.length === 1) { tokens[tokens.length - 1] = pool[0]; return tokens.join(' ') + ' '; }
      if (pool.length > 1) {
        const pre = commonPrefix(pool);
        if (pre.length > last.length) { tokens[tokens.length - 1] = pre; return { value: tokens.join(' '), hints: pool }; }
        return { value, hints: pool };
      }
    }
    return null;
  }

  // ---- tutorial UI ------------------------------------------------------

  renderTutorial() {
    const panel = document.getElementById('tutorial');
    if (!this.tutorial.active) { panel.classList.add('hidden'); return; }
    panel.classList.remove('hidden');

    const dots = MISSIONS.map((m, i) => {
      const done = i < this.tutorial.index;
      const cur = i === this.tutorial.index;
      return `<button class="mdot${done ? ' done' : ''}${cur ? ' cur' : ''}" data-i="${i}" title="${escapeAttr(m.title)}"></button>`;
    }).join('');

    if (this.tutorial.isDone()) {
      panel.innerHTML = `
        <div class="tut-head"><span class="tut-kicker">Tutorial complete</span></div>
        <div class="tut-dots">${dots}</div>
        <h2 class="tut-title">You’ve seen every mode 🎉</h2>
        <p class="tut-story">Run witr on a real machine and it does all of this against live processes.</p>
        <pre class="tut-install">curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash</pre>
        <div class="tut-actions">
          <button class="btn btn-primary" data-freeplay>Keep exploring →</button>
          <button class="btn" data-restart>Restart tutorial</button>
        </div>`;
    } else {
      const m = this.tutorial.current();
      const n = this.tutorial.index + 1;
      panel.innerHTML = `
        <div class="tut-head">
          <span class="tut-kicker">Mission ${n} / ${MISSIONS.length}</span>
          <button class="tut-skip" data-freeplay>Free play →</button>
        </div>
        <div class="tut-dots">${dots}</div>
        <h2 class="tut-title">${m.title}</h2>
        <p class="tut-story">${m.story}</p>
        <div class="tut-actions">
          <button class="btn btn-primary" data-hint>Run <code>${escapeHtml(m.hint)}</code></button>
        </div>
        <p class="tut-tiny">Type it yourself, or press the button. Explore freely — anything that shows the idea counts.</p>`;
    }

    panel.querySelectorAll('.mdot').forEach((d) => d.addEventListener('click', () => { this.tutorial.jumpTo(+d.dataset.i); }));
    const hint = panel.querySelector('[data-hint]');
    if (hint) hint.addEventListener('click', () => { if (!this.term.locked) this.term.typeAndRun(this.tutorial.current().hint); });
    const fp = panel.querySelector('[data-freeplay]');
    if (fp) fp.addEventListener('click', () => this.tutorial.stop());
    const rs = panel.querySelector('[data-restart]');
    if (rs) rs.addEventListener('click', () => this.tutorial.start());
  }

  celebrate(m) {
    this.term.print('');
    this.term.printHtml(`<div class="learned"><span class="learned-badge">✓ ${escapeHtml(m.title)}</span> ${m.learned}</div>`);
  }

  finishTutorial() {
    this.term.printHtml('<div class="learned finale">That’s the whole tool. Switch to the <b>devbox</b> scenario for a messier machine, or install witr for real.</div>');
  }

  // ---- chrome: scenario switch, buttons, welcome ------------------------

  wireChrome() {
    document.getElementById('btn-tutorial').addEventListener('click', () => {
      this.tutorial.active ? this.tutorial.stop() : this.tutorial.start();
      this.term.focus();
    });
    document.getElementById('btn-scenario').addEventListener('click', () => this.openScenario());
    document.getElementById('btn-reset').addEventListener('click', () => {
      this.term.clear(); this.welcome(); this.term.focus();
    });
    const modal = document.getElementById('scenario-modal');
    modal.addEventListener('click', (e) => { if (e.target === modal) modal.classList.remove('open'); });
    document.querySelectorAll('[data-scenario]').forEach((b) =>
      b.addEventListener('click', () => this.switchWorld(b.dataset.scenario)));

    // Mobile suggested-command chips.
    document.getElementById('chips').addEventListener('click', (e) => {
      const chip = e.target.closest('[data-cmd]');
      if (chip && !this.term.locked) this.term.typeAndRun(chip.dataset.cmd);
    });
  }

  openScenario() { document.getElementById('scenario-modal').classList.add('open'); }

  switchWorld(id) {
    if (!this.worlds[id]) return;
    this.worldId = id;
    this.shell.setWorld(this.worlds[id]);
    this.map.setWorld(this.worlds[id]);
    this.map.resize();
    document.getElementById('scenario-modal').classList.remove('open');
    this.term.clear();
    if (id === 'webbox') this.tutorial.start(); else this.tutorial.stop();
    this.applyWorld();
    this.welcome();
    this.term.setPrompt(this.shell.prompt());
    this.term.focus();
  }

  currentWorld() { return this.worlds[this.worldId]; }

  applyWorld() {
    const w = this.currentWorld();
    document.getElementById('host-name').textContent = `${w.promptUser}@${w.hostname}`;
    document.getElementById('host-distro').textContent = `${w.distro} · ${w.processes.length} procs`;
    document.getElementById('term-title').textContent = `${w.promptUser}@${w.hostname}: ~`;
    this.term.setPrompt(this.shell.prompt());
    this.renderTutorial();
    // Update chips to the current scenario's greatest hits.
    const chips = this.worldId === 'webbox'
      ? ['witr node', 'witr --port 5000', 'witr ng', 'witr --file /var/lib/dpkg/lock', 'witr']
      : ['witr code', 'witr --pid 6120', 'witr --container shop', 'witr ffmpeg', 'witr'];
    document.getElementById('chips').innerHTML = chips.map((c) => `<button class="chip" data-cmd="${escapeAttr(c)}">${escapeHtml(c)}</button>`).join('');
  }

  welcome() {
    const w = this.currentWorld();
    this.term.printHtml(`<div class="welcome">
      <div class="welcome-logo">witr <span>· why is this running?</span></div>
      <div class="welcome-sub">You're on <b>${escapeHtml(w.promptUser)}@${escapeHtml(w.hostname)}</b> — a <span class="sim-badge">simulated</span> ${escapeHtml(w.distro)} box. Nothing here touches your real computer.</div>
      <div class="welcome-hint">Try <code>witr node</code>, explore with <code>ls</code> / <code>ps</code>, or follow the tutorial on the left. Type <code>help</code> anytime.</div>
    </div>`);
  }
}

function dimNote(s) { return `\x1b[90m${s}\x1b[0m\n`; }
function commonPrefix(arr) {
  if (arr.length === 0) return '';
  let p = arr[0];
  for (const s of arr) { while (!s.startsWith(p)) p = p.slice(0, -1); }
  return p;
}
function escapeHtml(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
function escapeAttr(s) { return escapeHtml(s).replace(/"/g, '&quot;'); }

new App().boot().catch((e) => {
  document.getElementById('terminal').textContent = 'Failed to load playground: ' + e.message;
  // eslint-disable-next-line no-console
  console.error(e);
});
