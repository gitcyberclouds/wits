// tutorial.js — per-scenario incidents.
//
// Each scenario is a single short incident: a cold open that plays itself, then
// a box with real problems the visitor investigates with witr and *fixes*
// (processes they kill, a lock that clears). A health tracker counts down to
// zero; hitting zero is the finale with the install command. Feature coverage
// (port/file/tree/multi-match/kill) falls out of the investigation; the rest
// live as optional side quests that tick off as they're tried.

const gone = (pid) => (w) => !w.processes.some((p) => p.pid === pid);
const targetsPid = (pid) => (ts) => ts.some((t) => t.type === 'pid' && t.value === String(pid));
const targetsName = (name) => (ts) => ts.some((t) => t.type === 'name' && name.includes(t.value.toLowerCase()));

export const INCIDENTS = {
  webbox: {
    coldOpen: [
      { type: 'line', html: '<span class="co-prompt">deploy@webbox</span><span class="co-sep">:</span><span class="co-dir">~</span><span class="co-sep">$</span> ./deploy.sh', delay: 500 },
      { type: 'line', html: '<span class="a-dim">▸ building expense-manager …</span> <span class="a-green">done</span>', delay: 650 },
      { type: 'line', html: '<span class="a-dim">▸ health-checking :5000 …</span> <span class="a-green">ok</span>', delay: 650 },
      { type: 'line', html: '<span class="a-dim">▸ starting metrics endpoint on :8000 …</span>', delay: 800 },
      { type: 'line', html: '<span class="a-red">✗ Error: listen EADDRINUSE: address already in use 0.0.0.0:8000</span>', delay: 500 },
      { type: 'line', html: '<span class="a-dim">  deploy aborted. something is already on that port.</span>', delay: 1100 },
      { type: 'note', html: 'Every deploy hits this eventually. <b>witr</b> answers it in one command — <i>what</i> is on the port, and <i>why</i>:', delay: 900 },
      { type: 'run', cmd: 'witr --port 8000', delay: 400 },
    ],
    briefing: 'That was one problem. A quick sweep flags <b>three</b> on <b>webbox</b>. Investigate each with witr, clean it up, and get the box back to <span class="a-green">green</span> — the tracker on the left counts down.',
    issues: [
      {
        id: 'squatter', severity: 'high', title: 'Public dev server squatting on :8000',
        blurb: "A forgotten <code>python3 -m http.server</code> (pid 8123), backgrounded from an SSH session and bound to <b>0.0.0.0</b> — it's blocking the deploy <i>and</i> exposed to the whole network.",
        find: 'witr --port 8000', fixHint: 'kill 8123',
        touched: (ts) => ts.some((t) => (t.type === 'port' && t.value === '8000')) || targetsPid(8123)(ts),
        resolved: gone(8123), done: "Port freed. That's the deploy unblocked and an accidental exposure closed.",
      },
      {
        id: 'tunnel', severity: 'high', title: 'Public ngrok tunnel to the app',
        blurb: "An <code>ngrok</code> tunnel (pid 14290) is publishing the private app on :5000 straight to the internet. Find it — hint: <code>witr ng</code> matches more than one thing — then shut it down.",
        find: 'witr --pid 14290', fixHint: 'kill 14290',
        touched: (ts) => targetsPid(14290)(ts) || targetsName('ngrok')(ts),
        resolved: gone(14290), done: 'Tunnel closed. The app is private again.',
      },
      {
        id: 'lock', severity: 'warn', title: 'apt is blocked — dpkg lock held',
        blurb: "Someone reported <code>apt</code> won't run. Find who holds <code>/var/lib/dpkg/lock</code> with <code>--file</code>. This one you <b>don't</b> kill — see what it is first.",
        find: 'witr --file /var/lib/dpkg/lock',
        touched: (ts) => ts.some((t) => (t.type === 'file' && t.value.includes('dpkg')) || (t.type === 'pid' && t.value === '33871')),
        resolved: gone(33871),
        autoResolve: {
          delayMs: 3500, remove: [33871],
          waiting: 'The dpkg lock is held by a scheduled <b>unattended-upgrade</b> — you don’t kill that. Give it a moment; it should finish on its own.',
          done: 'The unattended-upgrade finished and released the dpkg lock — nothing to kill. Sometimes the answer is just knowing <i>why</i>.',
        },
      },
    ],
    sideQuests: [
      { id: 'verbose', cmd: 'witr node --verbose', label: 'the full deep-dive (memory, threads, sockets)', test: (c) => c.flags.verbose },
      { id: 'json', cmd: 'witr node --json', label: 'machine-readable output for scripts', test: (c) => c.flags.json },
      { id: 'container', cmd: 'witr --container redis', label: 'the Redis container with no host process', test: (c) => c.targets.some((t) => t.type === 'container') },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui' },
    ],
  },

  devbox: {
    coldOpen: [
      { type: 'line', html: '<span class="co-prompt">maya@devbox</span><span class="co-sep">:</span><span class="co-dir">~/projects/shop</span><span class="co-sep">$</span> git commit -m "wip"', delay: 500 },
      { type: 'line', html: '<span class="a-red">fatal: Unable to create \'.git/index.lock\': File exists.</span>', delay: 550 },
      { type: 'line', html: '<span class="a-dim">  Another git process seems to be running in this repository.</span>', delay: 1100 },
      { type: 'note', html: 'A stale lock — but which process? <b>witr</b> resolves the file to its owner:', delay: 900 },
      { type: 'run', cmd: 'witr --file /home/maya/projects/shop/.git/index.lock', delay: 400 },
    ],
    briefing: "This laptop is a mess. Three things need cleaning up on <b>devbox</b> — a stuck git lock, a zombie, and something eating the CPU. Trace each with witr and sort it out.",
    issues: [
      {
        id: 'gitlock', severity: 'high', title: 'git index.lock blocking every commit',
        blurb: "A crashed <code>git commit</code> (pid 7300) is still holding <code>.git/index.lock</code>, so every new git command fails with “File exists”. Release it.",
        find: 'witr --file /home/maya/projects/shop/.git/index.lock', fixHint: 'kill 7300',
        touched: (ts) => targetsPid(7300)(ts) || ts.some((t) => t.type === 'file' && t.value.includes('index.lock')),
        resolved: gone(7300), done: 'Lock released — git works again.',
      },
      {
        id: 'zombie', severity: 'warn', title: 'Zombie process nobody reaped',
        blurb: "A defunct <code>python3</code> (pid 6120) is stuck as a <b>zombie</b>. You don't kill a zombie — you get its parent to reap it. <code>witr --pid 6120</code> shows whose child it is: <code>build.sh</code>, pid 6100.",
        find: 'witr --pid 6120', fixHint: 'kill 6100',
        touched: (ts) => targetsPid(6120)(ts) || targetsPid(6100)(ts),
        resolved: gone(6120), done: 'Parent gone, zombie reaped. A defunct child only clears when its parent waits on it (or dies).',
      },
      {
        id: 'ffmpeg', severity: 'high', title: 'Runaway ffmpeg pinning the CPU',
        blurb: "An <code>ffmpeg</code> encode (pid 6001) has been stuck near <b>98% CPU</b> since it started — the fans are screaming. Find it (<code>witr ffmpeg</code>) and stop it.",
        find: 'witr ffmpeg', fixHint: 'kill 6001',
        touched: (ts) => targetsPid(6001)(ts) || targetsName('ffmpeg')(ts),
        resolved: gone(6001), done: "CPU's back to idle. The fans can rest.",
      },
    ],
    sideQuests: [
      { id: 'tree', cmd: 'witr code --tree', label: "VS Code's whole process family", test: (c) => c.flags.tree },
      { id: 'port', cmd: 'witr --port 5173', label: 'what the Vite dev server is', test: (c) => c.targets.some((t) => t.type === 'port') },
      { id: 'containers', cmd: 'witr --container shop', label: 'the docker-compose stack (multi-match)', test: (c) => c.targets.some((t) => t.type === 'container') },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui' },
    ],
  },
};

export class Incident {
  constructor() {
    this.active = false;
    this.phase = 'idle'; // idle | coldopen | investigating | done
    this.def = null;
    this.found = new Set();
    this.resolved = new Set();
    this.tried = new Set();
    this.onChange = null;
    this.onResolve = null;
    this.onComplete = null;
  }

  load(def) { this.def = def; }
  issues() { return this.def ? this.def.issues : []; }
  sideQuests() { return this.def ? this.def.sideQuests || [] : []; }

  start() {
    this.active = true;
    this.phase = 'coldopen';
    this.found.clear();
    this.resolved.clear();
    this.tried.clear();
    this._emit();
  }

  stop() { this.active = false; this.phase = 'idle'; this._emit(); }
  beginInvestigation() { if (this.active) { this.phase = 'investigating'; this._emit(); } }

  total() { return this.issues().length; }
  remaining() { return this.total() - this.resolved.size; }

  status(issue) {
    if (this.resolved.has(issue.id)) return 'resolved';
    if (this.found.has(issue.id)) return 'found';
    return 'open';
  }

  // Called after each executed command. ctx = { targets, flags, action, world }.
  observe(ctx) {
    if (!this.active) return [];

    // Side quests keep tracking even after the incident is resolved, so the
    // finale checklist ticks off as they're tried.
    for (const q of this.sideQuests()) {
      if (!this.tried.has(q.id)) { try { if (q.test(ctx)) { this.tried.add(q.id); this._emit(); } } catch (_) {} }
    }
    if (this.phase === 'done') return [];

    for (const issue of this.issues()) {
      if (!this.found.has(issue.id) && issue.touched(ctx.targets || [])) {
        this.found.add(issue.id);
        this._emit();
      }
    }

    const newlyResolved = [];
    for (const issue of this.issues()) {
      if (this.resolved.has(issue.id)) continue;
      if (issue.resolved(ctx.world)) { this.resolved.add(issue.id); newlyResolved.push(issue); }
    }
    for (const issue of newlyResolved) if (this.onResolve) this.onResolve(issue);
    if (newlyResolved.length) this._emit();
    if (this.remaining() === 0 && this.total() > 0 && this.phase !== 'done') {
      this.phase = 'done';
      if (this.onComplete) this.onComplete();
      this._emit();
    }
    return newlyResolved;
  }

  _emit() { if (this.onChange) this.onChange(); }
}
