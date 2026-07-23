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
      { type: 'line', html: '<span class="co-prompt">deploy@webbox</span><span class="co-sep">:</span><span class="co-dir">~</span><span class="co-sep">$</span> ./deploy.sh', delay: 333 },
      { type: 'line', html: '<span class="a-dim">▸ building expense-manager …</span> <span class="a-green">done</span>', delay: 433 },
      { type: 'line', html: '<span class="a-dim">▸ health-checking :5000 …</span> <span class="a-green">ok</span>', delay: 433 },
      { type: 'line', html: '<span class="a-dim">▸ starting metrics endpoint on :8000 …</span>', delay: 533 },
      { type: 'line', html: '<span class="a-red">✗ Error: listen EADDRINUSE: address already in use 0.0.0.0:8000</span>', delay: 333 },
      { type: 'line', html: '<span class="a-dim">  deploy aborted. something is already on that port.</span>', delay: 733 },
      { type: 'note', html: 'Every deploy hits this eventually — and <b>witr</b> answers it in one command. Your turn: investigate what’s on the port.', delay: 900 },
    ],
    briefing: 'That <span class="a-red">EADDRINUSE</span> is one of <b>three</b> questions a quick sweep raised on <b>webbox</b>. None of these need forcing — trace each and let witr tell you the <i>why</i>. Start by investigating what’s on :8000; the tracker on the left counts down.',
    issues: [
      {
        id: 'squatter', severity: 'high', title: 'What is holding :8000?',
        blurb: "witr already traced :8000 to a stray <code>python3 -m http.server</code> (pid 8123) a teammate backgrounded over SSH. That's the whole answer — now it's your call: free the port with <code>kill 8123</code>, or just point the deploy at another port. No forced action; witr told you the cause.",
        find: 'witr --port 8000',
        touched: (c) => c.targets.some((t) => (t.type === 'port' && t.value === '8000')) || targetsPid(8123)(c.targets),
        resolveOnFind: true,
        done: "witr pinned :8000 to a forgotten <code>http.server</code> (pid 8123). Free the port or use another — either way you now know the cause. No guessing.",
      },
      {
        id: 'lock', severity: 'warn', title: 'apt won’t run — who holds the dpkg lock?',
        blurb: "Someone reported <code>apt</code> is stuck. Ask witr who holds <code>/var/lib/dpkg/lock</code> with <code>--file</code> — no need to kill anything, just find out what it is.",
        find: 'witr --file /var/lib/dpkg/lock',
        touched: (c) => c.targets.some((t) => (t.type === 'file' && t.value.includes('dpkg')) || (t.type === 'pid' && t.value === '33871')),
        resolveOnFind: true,
        done: "It's the scheduled <b>unattended-upgrade</b> (pid 33871) — completely expected, you'd just forgotten it runs on a timer. Nothing is stuck; it releases the lock the moment it finishes. Mystery solved, no action needed.",
      },
      {
        id: 'verbose', severity: 'warn', title: 'How heavy is the app, really?',
        blurb: "The Node app has been up for weeks — before you guess at its footprint, look. <code>witr node --verbose</code> adds memory, threads, open files, sockets and I/O to the answer: the full picture in one command.",
        find: 'witr node --verbose',
        touched: (c) => !!c.flags.verbose && c.targets.some((t) => t.type === 'name' && 'node'.includes((t.value || '').toLowerCase())),
        resolveOnFind: true,
        done: "That's the deep dive — memory, threads, open files, sockets and I/O, everything witr knows about the process in one view. Add <code>--verbose</code> to any query when you need the full footprint.",
      },
    ],
    sideQuests: [
      { id: 'tree', cmd: 'witr node --tree', label: 'the full family tree', test: (c) => c.flags.tree, note: '<b>--tree</b> draws the whole ancestry top-down and lists the target’s children.' },
      { id: 'short', cmd: 'witr node --short', label: 'the one-line causal chain', test: (c) => c.flags.short, note: '<b>--short</b> collapses the answer to a single causal line — great for scripts and chat.' },
      { id: 'json', cmd: 'witr node --json', label: 'machine-readable output', test: (c) => c.flags.json, note: 'Any query takes <b>--json</b>; witr also returns exit codes — <code>0</code> clean, <code>1</code> warnings, <code>2</code> not-found.' },
      { id: 'verbose', cmd: 'witr node --verbose', label: 'the deep dive', test: (c) => c.flags.verbose, note: '<b>--verbose</b> adds memory, threads, open files and sockets — the full picture.' },
      { id: 'env', cmd: 'witr node --env', label: 'environment variables', test: (c) => c.flags.env, note: '<b>--env</b> dumps the process’s environment variables.' },
      { id: 'container', cmd: 'witr --container redis', label: 'a container with no host process', test: (c) => c.targets.some((t) => t.type === 'container'), note: '<b>--container</b> finds Docker/Podman/compose workloads by name, image or service — even with no visible host process.' },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui', note: '<b>witr</b> with no arguments opens the live TUI — Processes / Ports / Containers / Locks.' },
    ],
  },

  devbox: {
    coldOpen: [
      { type: 'line', html: '<span class="co-prompt">pranshu@devbox</span><span class="co-sep">:</span><span class="co-dir">~/projects/shop</span><span class="co-sep">$</span> git commit -m "wip"', delay: 500 },
      { type: 'line', html: '<span class="a-red">fatal: Unable to create \'.git/index.lock\': File exists.</span>', delay: 550 },
      { type: 'line', html: '<span class="a-dim">  Another git process seems to be running in this repository.</span>', delay: 1100 },
      { type: 'note', html: 'A stale lock — but which process? Your turn: ask <b>witr</b> to resolve the file to its owner.', delay: 900 },
    ],
    briefing: "That stale <code>.git/index.lock</code> is one of three things gumming up <b>devbox</b> — plus a <code>python3</code> zombie nobody reaped and something pinning the CPU. Trace each with witr and sort it out.",
    issues: [
      {
        id: 'gitlock', severity: 'high', title: 'git index.lock blocking every commit',
        blurb: "witr traces <code>.git/index.lock</code> to a <code>git commit</code> (pid 7300) that hung and never finished — so every new git command fails with “File exists.” It isn't doing any work; the stale lock just needs releasing.",
        find: 'witr --file /home/pranshu/projects/shop/.git/index.lock', fixHint: 'kill 7300', fixLabel: 'Clear the stale lock',
        touched: (c) => targetsPid(7300)(c.targets) || c.targets.some((t) => t.type === 'file' && t.value.includes('index.lock')),
        resolved: gone(7300), done: 'Lock released — git works again.',
      },
      {
        id: 'zombie', severity: 'warn', title: 'Zombie process nobody reaped',
        blurb: "A defunct <code>python3</code> (pid 6120) is stuck as a <b>zombie</b>. You don't kill a zombie — you get its parent to reap it. <code>witr --pid 6120</code> shows whose child it is: <code>build.sh</code>, pid 6100.",
        find: 'witr --pid 6120', fixHint: 'kill 6100',
        touched: (c) => targetsPid(6120)(c.targets) || targetsPid(6100)(c.targets),
        resolved: gone(6120), done: 'Parent gone, zombie reaped. A defunct child only clears when its parent waits on it (or dies).',
      },
      {
        id: 'ffmpeg', severity: 'high', title: 'Runaway ffmpeg pinning the CPU',
        blurb: "An <code>ffmpeg</code> encode (pid 6001) has been stuck near <b>98% CPU</b> since it started — the fans are screaming. Find it (<code>witr ffmpeg</code>) and stop it.",
        find: 'witr ffmpeg', fixHint: 'kill 6001',
        touched: (c) => targetsPid(6001)(c.targets) || targetsName('ffmpeg')(c.targets),
        resolved: gone(6001), done: "CPU's back to idle. The fans can rest.",
      },
    ],
    sideQuests: [
      { id: 'tree', cmd: 'witr code --tree', label: "VS Code's whole process family", test: (c) => c.flags.tree, note: '<b>--tree</b> draws the whole ancestry top-down and lists the target’s children.' },
      { id: 'short', cmd: 'witr --port 5173 --short', label: 'the Vite server in one line', test: (c) => c.flags.short, note: '<b>--short</b> collapses the answer to a single causal line — great for scripts and chat.' },
      { id: 'json', cmd: 'witr code --json', label: 'machine-readable output', test: (c) => c.flags.json, note: 'Any query takes <b>--json</b>; witr also returns exit codes — <code>0</code> clean, <code>1</code> warnings, <code>2</code> not-found.' },
      { id: 'verbose', cmd: 'witr --port 5173 --verbose', label: 'the deep dive', test: (c) => c.flags.verbose, note: '<b>--verbose</b> adds memory, threads, open files and sockets — the full picture.' },
      { id: 'env', cmd: 'witr --port 5173 --env', label: 'environment variables', test: (c) => c.flags.env, note: '<b>--env</b> dumps the process’s environment variables.' },
      { id: 'containers', cmd: 'witr --container shop', label: 'the docker-compose stack', test: (c) => c.targets.some((t) => t.type === 'container'), note: '<b>--container</b> matches every container in a compose project — pass the exact name to pick one.' },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui', note: '<b>witr</b> with no arguments opens the live TUI — Processes / Ports / Containers / Locks.' },
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
    this.onQuestTried = null;
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
    ctx = { ...ctx, targets: ctx.targets || [], flags: ctx.flags || {} };

    // Side quests keep tracking even after the incident is resolved, so the
    // finale checklist ticks off as they're tried.
    for (const q of this.sideQuests()) {
      if (!this.tried.has(q.id)) {
        try {
          if (q.test(ctx)) {
            this.tried.add(q.id);
            if (this.onQuestTried) this.onQuestTried(q);
            this._emit();
          }
        } catch (_) {}
      }
    }
    if (this.phase === 'done') return [];

    const newlyResolved = [];
    for (const issue of this.issues()) {
      if (!this.found.has(issue.id) && issue.touched(ctx)) {
        this.found.add(issue.id);
        // Informational issues are answered the moment they're investigated —
        // witr tells you the why, and that's the whole task.
        if (issue.resolveOnFind && !this.resolved.has(issue.id)) {
          this.resolved.add(issue.id);
          newlyResolved.push(issue);
        }
        this._emit();
      }
    }

    for (const issue of this.issues()) {
      if (this.resolved.has(issue.id)) continue;
      if (typeof issue.resolved === 'function' && issue.resolved(ctx.world)) {
        this.resolved.add(issue.id); newlyResolved.push(issue);
      }
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
