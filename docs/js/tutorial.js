// tutorial.js — the webbox incident.
//
// Instead of a linear feature tour, the tutorial is a single ~5-minute incident:
// a cold open that plays itself, then a box with three real problems the visitor
// investigates with witr and *fixes* (processes they kill, a lock that clears).
// A health tracker counts down to zero; hitting zero is the finale with the
// install command. Feature coverage (port/file/tree/multi-match/kill) falls out
// of the investigation naturally; the rest live as optional side quests.

// The self-playing cold open. Each step is either printed output (with a delay)
// or a witr command that actually runs through the engine.
export const COLD_OPEN = [
  { type: 'line', html: '<span class="co-prompt">deploy@webbox</span><span class="co-sep">:</span><span class="co-dir">~</span><span class="co-sep">$</span> ./deploy.sh', delay: 500 },
  { type: 'line', html: '<span class="a-dim">▸ building expense-manager …</span> <span class="a-green">done</span>', delay: 650 },
  { type: 'line', html: '<span class="a-dim">▸ health-checking :5000 …</span> <span class="a-green">ok</span>', delay: 650 },
  { type: 'line', html: '<span class="a-dim">▸ starting metrics endpoint on :8000 …</span>', delay: 800 },
  { type: 'line', html: '<span class="a-red">✗ Error: listen EADDRINUSE: address already in use 0.0.0.0:8000</span>', delay: 500 },
  { type: 'line', html: '<span class="a-dim">  deploy aborted. something is already on that port.</span>', delay: 1100 },
  { type: 'note', html: 'Every deploy hits this eventually. <b>witr</b> answers it in one command — <i>what</i> is on the port, and <i>why</i>:', delay: 900 },
  { type: 'run', cmd: 'witr --port 8000', delay: 400 },
];

// The three issues. `resolved(world)` is re-checked after every command.
export const ISSUES = [
  {
    id: 'squatter',
    severity: 'high',
    title: 'Public dev server squatting on :8000',
    blurb: "A forgotten <code>python3 -m http.server</code> (pid 8123), backgrounded from an SSH session and bound to <b>0.0.0.0</b> — it's blocking the deploy <i>and</i> exposed to the whole network.",
    find: 'witr --port 8000',
    fixHint: 'kill 8123',
    resolved: (w) => !w.processes.some((p) => p.pid === 8123),
    done: "Port freed. That's the deploy unblocked and an accidental exposure closed.",
  },
  {
    id: 'tunnel',
    severity: 'high',
    title: 'Public ngrok tunnel to the app',
    blurb: "An <code>ngrok</code> tunnel (pid 14290) is publishing the private app on :5000 straight to the internet. Find it — hint: <code>witr ng</code> matches more than one thing — then shut it down.",
    find: 'witr --pid 14290',
    fixHint: 'kill 14290',
    resolved: (w) => !w.processes.some((p) => p.pid === 14290),
    done: 'Tunnel closed. The app is private again.',
  },
  {
    id: 'lock',
    severity: 'warn',
    title: 'apt is blocked — dpkg lock held',
    blurb: "Someone reported <code>apt</code> won't run. Find who holds <code>/var/lib/dpkg/lock</code> with <code>--file</code>. This one you <b>don't</b> kill — see what it is first.",
    find: 'witr --file /var/lib/dpkg/lock',
    // Resolves on its own shortly after you investigate it (the upgrade finishes).
    resolved: (w) => w._lockReleased === true,
    autoResolve: {
      afterFind: true, delayMs: 3500, pid: 33871,
      done: 'The unattended-upgrade finished and released the dpkg lock — nothing to kill. Sometimes the answer is just knowing <i>why</i>.',
    },
  },
];

// Optional things worth trying once the box is green (free-play nudges).
export const SIDE_QUESTS = [
  { cmd: 'witr node --verbose', label: 'the full deep-dive (memory, threads, sockets)' },
  { cmd: 'witr node --json', label: 'machine-readable output for scripts' },
  { cmd: 'witr --container redis', label: 'the Redis container with no host process' },
  { cmd: 'witr', label: 'the live TUI dashboard' },
];

export class Incident {
  constructor() {
    this.active = false;
    this.phase = 'idle'; // idle | coldopen | investigating | done
    this.found = new Set();
    this.resolved = new Set();
    this.onChange = null;
    this.onResolve = null;
    this.onComplete = null;
  }

  start() {
    this.active = true;
    this.phase = 'coldopen';
    this.found.clear();
    this.resolved.clear();
    this._emit();
  }

  stop() { this.active = false; this.phase = 'idle'; this._emit(); }
  beginInvestigation() { if (this.active) { this.phase = 'investigating'; this._emit(); } }

  total() { return ISSUES.length; }
  remaining() { return ISSUES.length - this.resolved.size; }

  status(issue) {
    if (this.resolved.has(issue.id)) return 'resolved';
    if (this.found.has(issue.id)) return 'found';
    return 'open';
  }

  // Called after each executed command. `ctx` carries the parsed command and the
  // (possibly mutated) world; returns the list of issues newly resolved.
  observe(ctx) {
    if (!this.active || this.phase === 'done') return [];

    // Mark an issue "found" when the player queries its target.
    for (const issue of ISSUES) {
      if (this.found.has(issue.id)) continue;
      if (this._touches(issue, ctx)) {
        this.found.add(issue.id);
        this._emit();
      }
    }

    // Re-check resolution against current world state.
    const newlyResolved = [];
    for (const issue of ISSUES) {
      if (this.resolved.has(issue.id)) continue;
      if (issue.resolved(ctx.world)) {
        this.resolved.add(issue.id);
        newlyResolved.push(issue);
      }
    }
    for (const issue of newlyResolved) {
      if (this.onResolve) this.onResolve(issue);
    }
    if (newlyResolved.length) this._emit();
    if (this.remaining() === 0 && this.phase !== 'done') {
      this.phase = 'done';
      if (this.onComplete) this.onComplete();
      this._emit();
    }
    return newlyResolved;
  }

  // Did this command investigate the issue's target?
  _touches(issue, ctx) {
    const { targets } = ctx;
    if (issue.id === 'squatter') {
      return targets.some((t) => (t.type === 'port' && t.value === '8000') || (t.type === 'pid' && t.value === '8123') || (t.type === 'name' && '8000'.includes(t.value)));
    }
    if (issue.id === 'tunnel') {
      return targets.some((t) => (t.type === 'pid' && t.value === '14290') || (t.type === 'name' && 'ngrok'.includes(t.value.toLowerCase())));
    }
    if (issue.id === 'lock') {
      return targets.some((t) => (t.type === 'file' && t.value.includes('dpkg')) || (t.type === 'pid' && t.value === '33871'));
    }
    return false;
  }

  _emit() { if (this.onChange) this.onChange(); }
}
