// app.js — wires the playground together.

import { Shell } from './shell.js';
import { Terminal } from './terminal.js';
import { SystemMap } from './map.js';
import { Incident, ISSUES, SIDE_QUESTS, COLD_OPEN } from './tutorial.js';
import { TUI } from './tui.js';
import { parse, tokenize } from './parser.js';

const WORLD_IDS = ['webbox', 'devbox'];
const COMPLETIONS = ['witr', 'ls', 'cat', 'ps', 'kill', 'pwd', 'cd', 'whoami', 'hostname', 'uname', 'neofetch', 'clear', 'help', 'scenario'];
const WITR_FLAGS = ['--pid', '--port', '--file', '--container', '--short', '--tree', '--json', '--env', '--warnings', '--verbose', '--exact', '--no-color', '--interactive', '--help', '--version'];
const INSTALL_CMD = 'curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash';

class App {
  constructor() {
    this.pristine = {};   // worlds as loaded (never mutated)
    this.worldId = 'webbox';
    this.live = null;     // the mutable working copy
    this._skipCold = false;
    this._lockTimer = null;
  }

  async boot() {
    for (const id of WORLD_IDS) {
      const res = await fetch(`./worlds/${id}.json`);
      this.pristine[id] = await res.json();
    }
    this.live = cloneWorld(this.pristine[this.worldId]);

    this.shell = new Shell(this.live);
    this.term = new Terminal(document.getElementById('terminal'));
    this.map = new SystemMap(document.getElementById('map-canvas'), document.getElementById('map-labels'));
    this.incident = new Incident();
    this.tui = new TUI(document.getElementById('tui'));

    this.term.onSubmit = (line) => this.handle(line);
    this.term.completer = (v) => this.complete(v);
    this.map.onSelect = (proc) => this.launchFromMap(proc);
    this.tui.onClose = () => this.term.focus();

    this.incident.onChange = () => this.renderIncident();
    this.incident.onResolve = (issue) => this.onIssueResolved(issue);
    this.incident.onComplete = () => this.onIncidentComplete();

    this.map.setWorld(this.live);
    this.map.start();
    window.addEventListener('resize', () => this.map.resize());

    this.wireChrome();
    this.applyWorld();
    this.enterScenario(true);
    this.term.focus();
  }

  // ---- scenario entry ---------------------------------------------------

  enterScenario(initial) {
    if (this.worldId === 'webbox') {
      this.incident.start();      // phase = coldopen
      this.playColdOpen();
    } else {
      this.incident.stop();
      this.welcome();
    }
  }

  // ---- cold open (plays itself) -----------------------------------------

  async playColdOpen() {
    this._skipCold = false;
    this.term.locked = true;
    this.renderIncident();
    for (const step of COLD_OPEN) {
      await this.sleep(step.delay);
      if (step.type === 'line') this.term.printHtml(`<div class="co-line">${step.html}</div>`);
      else if (step.type === 'note') this.term.printHtml(`<div class="co-note">${step.html}</div>`);
      else if (step.type === 'run') {
        this.term.locked = false;
        await this.term.typeAndRun(step.cmd, { speed: this._skipCold ? 6 : 34 });
      }
    }
    this.term.locked = false;
    this.incident.beginInvestigation();
    this.term.printHtml(`<div class="co-brief"><span class="co-brief-tag">🚨 incident</span> That was one problem. A quick sweep flags <b>three</b> on <b>webbox</b>. Investigate each with witr, clean it up, and get the box back to <span class="a-green">green</span> — the tracker on the left counts down.</div>`);
    this.renderIncident();
    this.term.focus();
  }

  skipColdOpen() { this._skipCold = true; }

  sleep(ms) {
    return new Promise((resolve) => {
      if (this._skipCold) return resolve();
      setTimeout(resolve, ms);
    });
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
    if (res.action === 'killed' && res.killed) {
      for (const pid of res.killed) this.map.removeProcess(pid);
      this.refreshHostChip();
    }

    const ctx = this.analyze(line, res);
    this.updateMap(ctx);
    this.incident.observe(ctx);
    this.maybeScheduleLockRelease();
    this.term.setPrompt(this.shell.prompt());
  }

  analyze(line, res) {
    const tokens = tokenize(line.trim());
    const isWitr = tokens[0] === 'witr';
    const { targets, flags } = isWitr ? parse(tokens.slice(1)) : { targets: [], flags: {} };
    return { line, isWitr, targets, flags, exit: res.exit, action: res.action, world: this.currentWorld() };
  }

  updateMap(ctx) {
    if (!ctx.isWitr || ctx.targets.length === 0) return;
    const eng = this.shell.engine;
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
        if (proc) { this.map.highlightPids(eng.ancestryOf(proc).map((p) => p.pid)); return; }
      }
    }
    this.map.clearHighlight();
  }

  launchFromMap(proc) {
    if (this.tui.open || this.term.locked) return;
    this.term.focus();
    this.term.typeAndRun(`witr --pid ${proc.pid}`);
  }

  // ---- lock auto-resolve ------------------------------------------------

  maybeScheduleLockRelease() {
    if (!this.incident.active) return;
    if (this._lockTimer || this.incident.resolved.has('lock')) return;
    if (!this.incident.found.has('lock')) return;
    const cfg = ISSUES.find((i) => i.id === 'lock').autoResolve;
    this.term.printHtml(`<div class="learned"><span class="learned-badge a-dimyellow">…</span> The dpkg lock is held by a scheduled <b>unattended-upgrade</b> — you don’t kill that. Give it a moment; it should finish on its own.</div>`);
    this._lockTimer = setTimeout(() => {
      const w = this.currentWorld();
      w.processes = w.processes.filter((p) => p.pid !== cfg.pid);
      w.locks = (w.locks || []).filter((l) => l.pid !== cfg.pid);
      w._lockReleased = true;
      this.shell.engine.reindex();
      this.map.removeProcess(cfg.pid);
      this.refreshHostChip();
      this.incident.observe({ targets: [], world: w });
    }, cfg.delayMs);
  }

  // ---- incident outcomes ------------------------------------------------

  onIssueResolved(issue) {
    this.term.printHtml(`<div class="learned"><span class="learned-badge">✓ resolved</span> ${issue.done || (issue.autoResolve && issue.autoResolve.done) || ''}</div>`);
  }

  onIncidentComplete() {
    const quests = SIDE_QUESTS.map((q) => `<button class="sq" data-cmd="${escapeAttr(q.cmd)}"><code>${escapeHtml(q.cmd)}</code> — ${q.label}</button>`).join('');
    this.term.printHtml(`<div class="finale-card">
      <div class="finale-badge">✓ webbox is green</div>
      <div class="finale-title">You just ran an incident with witr — port, lock, tunnel, all traced to <i>why</i> in one command each.</div>
      <div class="finale-sub">It does exactly this on a real machine, against live processes:</div>
      <pre class="tut-install">${INSTALL_CMD}</pre>
      <div class="finale-quests"><span class="fq-h">Keep poking:</span>${quests}</div>
    </div>`);
    this.term.scroll();
  }

  // ---- incident / free-play panel ---------------------------------------

  renderIncident() {
    const panel = document.getElementById('tutorial');
    if (!this.incident.active) { panel.classList.add('hidden'); return; }
    panel.classList.remove('hidden');

    if (this.incident.phase === 'coldopen') {
      panel.innerHTML = `
        <div class="tut-head"><span class="tut-kicker alert">● incident detected</span>
          <button class="tut-skip" data-skip>Skip intro ⏭</button></div>
        <h2 class="tut-title">webbox</h2>
        <p class="tut-story">A deploy just failed. Watching witr trace the cause…</p>`;
      const sk = panel.querySelector('[data-skip]');
      if (sk) sk.addEventListener('click', () => this.skipColdOpen());
      return;
    }

    const done = this.incident.remaining() === 0;
    const total = this.incident.total();
    const resolved = total - this.incident.remaining();
    const rows = ISSUES.map((issue) => {
      const st = this.incident.status(issue);
      const icon = st === 'resolved' ? '✓' : (st === 'found' ? '◔' : '○');
      let action = '';
      if (st === 'open') {
        action = `<button class="btn btn-sm" data-cmd="${escapeAttr(issue.find)}">Investigate</button>`;
      } else if (st === 'found' && issue.fixHint) {
        action = `<button class="btn btn-sm btn-primary" data-cmd="${escapeAttr(issue.fixHint)}">Fix · <code>${escapeHtml(issue.fixHint)}</code></button>`;
      } else if (st === 'found') {
        action = `<span class="issue-wait">clearing on its own…</span>`;
      }
      return `<div class="issue ${st} sev-${issue.severity}">
        <div class="issue-top"><span class="issue-ic">${icon}</span><span class="issue-title">${issue.title}</span></div>
        ${st !== 'resolved' ? `<div class="issue-blurb">${issue.blurb}</div>` : `<div class="issue-blurb done">${issue.done || (issue.autoResolve && issue.autoResolve.done) || ''}</div>`}
        ${action ? `<div class="issue-act">${action}</div>` : ''}
      </div>`;
    }).join('');

    panel.innerHTML = `
      <div class="tut-head">
        <span class="tut-kicker ${done ? 'ok' : 'alert'}">${done ? '● all clear' : '● incident · webbox'}</span>
        <button class="tut-skip" data-freeplay>Free play →</button>
      </div>
      <div class="health"><div class="health-bar"><span style="width:${(resolved / total) * 100}%"></span></div>
        <span class="health-n">${resolved} / ${total} resolved</span></div>
      <div class="issues">${rows}</div>
      ${done ? `<div class="tut-actions"><button class="btn btn-primary" data-freeplay>Explore freely →</button><button class="btn" data-replay>Replay incident</button></div>` : ''}`;

    panel.querySelectorAll('[data-cmd]').forEach((b) =>
      b.addEventListener('click', () => { if (!this.term.locked) this.term.typeAndRun(b.dataset.cmd); }));
    const fp = panel.querySelector('[data-freeplay]');
    if (fp) fp.addEventListener('click', () => { this.incident.stop(); this.term.focus(); });
    const rp = panel.querySelector('[data-replay]');
    if (rp) rp.addEventListener('click', () => this.resetScenario());
  }

  welcome() {
    const w = this.currentWorld();
    this.term.printHtml(`<div class="welcome">
      <div class="welcome-logo">witr <span>· why is this running?</span></div>
      <div class="welcome-sub">Free play on <b>${escapeHtml(w.promptUser)}@${escapeHtml(w.hostname)}</b> — a <span class="sim-badge">simulated</span> ${escapeHtml(w.distro)}. Nothing here touches your real computer.</div>
      <div class="welcome-hint">Try <code>witr code</code>, hunt the <code>witr --pid 6120</code> zombie, explore with <code>ls</code> / <code>ps</code>, or open the <code>witr</code> dashboard. Type <code>help</code> anytime.</div>
    </div>`);
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

  // ---- chrome -----------------------------------------------------------

  wireChrome() {
    document.getElementById('btn-tutorial').addEventListener('click', () => {
      if (this.incident.active) this.incident.stop();
      else if (this.worldId === 'webbox') { this.term.clear(); this.resetScenario(); }
      this.term.focus();
    });
    document.getElementById('btn-scenario').addEventListener('click', () => this.openScenario());
    document.getElementById('btn-reset').addEventListener('click', () => this.resetScenario());
    const modal = document.getElementById('scenario-modal');
    modal.addEventListener('click', (e) => { if (e.target === modal) modal.classList.remove('open'); });
    document.querySelectorAll('[data-scenario]').forEach((b) =>
      b.addEventListener('click', () => this.switchWorld(b.dataset.scenario)));

    document.getElementById('chips').addEventListener('click', (e) => {
      const chip = e.target.closest('[data-cmd]');
      if (chip && !this.term.locked) this.term.typeAndRun(chip.dataset.cmd);
    });
  }

  openScenario() { document.getElementById('scenario-modal').classList.add('open'); }

  // Reset the current scenario to its pristine state (restores killed procs).
  resetScenario() {
    if (this._lockTimer) { clearTimeout(this._lockTimer); this._lockTimer = null; }
    this.live = cloneWorld(this.pristine[this.worldId]);
    this.shell.setWorld(this.live);
    this.map.setWorld(this.live);
    this.map.resize();
    this.term.clear();
    this.applyWorld();
    this.enterScenario(false);
    this.term.setPrompt(this.shell.prompt());
    this.term.focus();
  }

  switchWorld(id) {
    if (!this.pristine[id]) return;
    this.worldId = id;
    document.getElementById('scenario-modal').classList.remove('open');
    this.resetScenario();
  }

  currentWorld() { return this.live; }

  refreshHostChip() {
    const w = this.currentWorld();
    document.getElementById('host-distro').textContent = `${w.distro} · ${w.processes.length} procs`;
  }

  applyWorld() {
    const w = this.currentWorld();
    document.getElementById('host-name').textContent = `${w.promptUser}@${w.hostname}`;
    document.getElementById('term-title').textContent = `${w.promptUser}@${w.hostname}: ~`;
    this.refreshHostChip();
    this.term.setPrompt(this.shell.prompt());
    this.renderIncident();
    const chips = this.worldId === 'webbox'
      ? ['witr --port 8000', 'kill 8123', 'witr ng', 'witr --file /var/lib/dpkg/lock', 'witr']
      : ['witr code', 'witr --pid 6120', 'witr --container shop', 'witr ffmpeg', 'witr'];
    document.getElementById('chips').innerHTML = chips.map((c) => `<button class="chip" data-cmd="${escapeAttr(c)}">${escapeHtml(c)}</button>`).join('');
  }
}

function cloneWorld(w) {
  return typeof structuredClone === 'function' ? structuredClone(w) : JSON.parse(JSON.stringify(w));
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
