// tutorial.js — the guided mission track.
//
// Each mission frames one witr feature as a small mystery. The tutorial watches
// executed commands and advances when the current mission's concept is
// demonstrated (matchers are deliberately lenient — exploring counts).

export const MISSIONS = [
  {
    id: 'first-question',
    title: 'Ask the first question',
    story: "Something called <b>node</b> is running on this box. You didn't start it. Ask witr <i>why</i> it's running.",
    hint: 'witr node',
    check: (c) => c.hasName && c.plain && c.exit !== 2,
    learned: "witr resolves a name to a PID, then walks up the process tree to show the <b>causal chain</b> — the “Why It Exists” line. Here: <code>systemd → PM2 → node</code>. PM2 keeps it alive.",
  },
  {
    id: 'mystery-port',
    title: 'The mystery port',
    story: 'Port <b>5000</b> is taken and a deploy is failing because of it. Find out which process is sitting on it — and add <code>--short</code> to get just the chain.',
    hint: 'witr --port 5000 --short',
    check: (c) => c.hasPort,
    learned: 'Ports map to PIDs. <code>--port 5000</code> resolves the listener and explains it like any process. <code>--short</code> collapses the answer to a single causal line — handy in scripts and chat.',
  },
  {
    id: 'family-tree',
    title: 'How deep does it go?',
    story: 'That SSH session spawned a shell, which spawned… things. Render the ancestry of <b>pid 40141</b> as a tree with <code>--tree</code> to see the whole lineage and its children.',
    hint: 'witr --pid 40141 --tree',
    check: (c) => c.flags.tree,
    learned: 'The <code>--tree</code> view draws the full ancestry top-down and lists the target’s children (up to 10). Great for “what did this shell leave running?”',
  },
  {
    id: 'too-many',
    title: 'Too many matches',
    story: 'Type <code>witr ng</code>. Substring matching finds more than one thing. Then narrow it: use <code>-x</code> for exact names, or <code>--pid</code> to pick one.',
    hint: 'witr ng',
    check: (c) => c.multi || c.flags.exact,
    learned: 'By default names match as substrings, so <code>ng</code> hits nginx <i>and</i> ngrok. witr lists every match with its PID and command so you can re-run with <code>--pid</code>, or use <code>-x</code> for exact matching.',
  },
  {
    id: 'stuck-lock',
    title: 'The stuck lock',
    story: "<code>apt</code> won't run: “could not get lock /var/lib/dpkg/lock”. Something is holding it. Ask witr who, with <code>--file</code>.",
    hint: 'witr --file /var/lib/dpkg/lock',
    check: (c) => c.hasFile,
    learned: '<code>--file</code> resolves the process holding a file open or locked. Here it’s an <b>unattended-upgrade</b> run that grabbed the dpkg lock — wait it out or stop that unit.',
  },
  {
    id: 'inside-box',
    title: 'Inside the box',
    story: 'The app talks to Redis, but <code>ps</code> shows no redis process — it lives in a container. Look it up with <code>--container redis</code>.',
    hint: 'witr --container redis',
    check: (c) => c.hasContainer,
    learned: '<code>--container</code> searches Docker, Podman, containerd, and more — by name, image, or compose service. It reconstructs the <code>runtime → compose project → container</code> chain even when the workload process isn’t visible from the host.',
  },
  {
    id: 'machine-readable',
    title: 'Machine-readable',
    story: 'Now pipe witr into a script. Add <code>--json</code> to any query for structured output, or <code>--warnings</code> to surface only problems.',
    hint: 'witr node --json',
    check: (c) => c.flags.json || c.flags.warnings,
    learned: 'Every mode has a <code>--json</code> form, and witr returns meaningful <b>exit codes</b>: <code>0</code> clean, <code>1</code> warnings, <code>2</code> not found. That’s what makes it usable in CI and health checks.',
  },
  {
    id: 'audit',
    title: 'Audit the whole box',
    story: 'Finish with a real audit. Query several things at once — mix names, <code>--pid</code>, <code>--port</code> — and add <code>--verbose</code> for the deep dive (memory, threads, sockets, children).',
    hint: 'witr node --port 5432 --verbose',
    check: (c) => c.targets.length > 1 || c.flags.verbose || c.flags.env,
    learned: 'All target flags are repeatable and mixable; results print in the order you typed them, with labeled dividers. <code>--verbose</code> adds the full context, <code>--env</code> dumps environment variables. You’ve now seen every mode.',
  },
  {
    id: 'dashboard',
    title: 'Do it live',
    story: 'One last thing: run <code>witr</code> with no arguments to open the live <b>TUI dashboard</b> — processes, ports, containers, and locks, refreshing in real time.',
    hint: 'witr',
    check: (c) => c.action === 'tui',
    learned: "That's the whole tool. On your own machine the TUI refreshes live and lets you send signals and renice from the keyboard. You're ready — install it and run it for real.",
  },
];

export class Tutorial {
  constructor() {
    this.index = 0;
    this.active = false;
    this.completed = new Set();
    this.onChange = null;
    this.onComplete = null;
    this.onFinish = null;
  }

  start() { this.active = true; this.index = 0; this.completed.clear(); this._emit(); }
  stop() { this.active = false; this._emit(); }
  current() { return MISSIONS[this.index]; }
  isDone() { return this.index >= MISSIONS.length; }

  jumpTo(i) { this.index = Math.max(0, Math.min(i, MISSIONS.length - 1)); this._emit(); }

  // ctx: { line, targets, flags, exit, action, multi, hasName, hasPort, hasFile, hasContainer, plain }
  observe(ctx) {
    if (!this.active || this.isDone()) return false;
    const m = MISSIONS[this.index];
    let ok = false;
    try { ok = m.check(ctx); } catch (_) { ok = false; }
    if (!ok) return false;
    this.completed.add(m.id);
    if (this.onComplete) this.onComplete(m);
    this.index++;
    if (this.isDone()) { if (this.onFinish) this.onFinish(); }
    this._emit();
    return true;
  }

  _emit() { if (this.onChange) this.onChange(); }
}
