// shell.js — a tiny fake shell around the witr engine.
//
// It is deliberately NOT a general shell: it simulates witr faithfully and
// offers just enough coreutils flavour (ls, cat, ps, ...) to make the box feel
// real to poke at. Everything routes through the world data.

import { Engine, EXIT } from './engine.js';
import { ESC } from './ansi.js';
import { tokenize, parse } from './parser.js';

const WITR_VERSION = 'v1.4.0';

const HELP = `Available commands in this playground:

  ${ESC.green}witr${ESC.reset} [name|flags]     the tool itself — try: ${ESC.cyan}witr node${ESC.reset}
  ${ESC.green}witr${ESC.reset} (no args)         launch the interactive TUI dashboard
  ${ESC.green}witr --help${ESC.reset}            full witr flag reference

  ${ESC.dim}ls${ESC.reset} [dir]               list a directory
  ${ESC.dim}cat${ESC.reset} <file>             print a file
  ${ESC.dim}ps${ESC.reset}                     list running processes
  ${ESC.dim}pwd / cd / whoami${ESC.reset}      the usual
  ${ESC.dim}uname${ESC.reset} [-a] / ${ESC.dim}hostname${ESC.reset}  system info
  ${ESC.dim}neofetch${ESC.reset}               about this machine
  ${ESC.dim}clear${ESC.reset}                  clear the screen
  ${ESC.dim}scenario${ESC.reset}               show / switch the machine you're on

Everything here runs against a ${ESC.dimYellow}simulated${ESC.reset} machine — nothing touches your computer.
Type ${ESC.cyan}witr node${ESC.reset} to start, or open the tutorial with the button on the left.`;

const WITR_HELP = `witr — Why Is This Running?

Explains where a running thing came from, how it started, and what chain of
systems is responsible for it existing right now.

Usage:
  witr [flags] [name...]

Flags:
  -c, --container strings  container(s) to look up (repeatable)
      --env                show environment variables for the process
  -x, --exact              use exact name matching (no substring search)
  -f, --file strings       file(s) held open by a process (repeatable)
  -h, --help               help for witr
  -i, --interactive        interactive mode (TUI)
      --json               show result as JSON
      --no-color           disable colorized output
  -p, --pid strings        pid(s) to look up (repeatable)
  -o, --port strings       port(s) to look up (repeatable)
  -s, --short              show only ancestry
  -t, --tree               show only ancestry as a tree
      --verbose            show extended process information
  -v, --version            version for witr
      --warnings           show only warnings

Positional arguments are treated as process or service names (substring match).`;

export class Shell {
  constructor(world) {
    this.setWorld(world);
    this.cwd = `/home/${world.promptUser}`;
  }

  setWorld(world) {
    this.world = world;
    this.engine = new Engine(world);
    this.cwd = `/home/${world.promptUser}`;
  }

  prompt() {
    const dir = this.cwd === `/home/${this.world.promptUser}` ? '~' : this.cwd;
    return { user: this.world.promptUser, host: this.world.promptHost, dir };
  }

  // Returns { output, exit, action }.
  exec(line) {
    const trimmed = line.trim();
    if (trimmed === '') return { output: '', exit: 0 };
    const tokens = tokenize(trimmed);
    const cmd = tokens[0];
    const args = tokens.slice(1);

    switch (cmd) {
      case 'witr': return this.witr(args);
      case 'help': case '?': return { output: HELP + '\n', exit: 0 };
      case 'clear': return { output: '', exit: 0, action: 'clear' };
      case 'ls': return this.ls(args);
      case 'cat': return this.cat(args);
      case 'pwd': return { output: this.cwd + '\n', exit: 0 };
      case 'cd': return this.cd(args);
      case 'whoami': return { output: this.world.promptUser + '\n', exit: 0 };
      case 'hostname': return { output: this.world.hostname + '\n', exit: 0 };
      case 'uname': return this.uname(args);
      case 'ps': return this.ps(args);
      case 'neofetch': case 'witr-info': return { output: this.neofetch(), exit: 0 };
      case 'echo': return { output: args.join(' ') + '\n', exit: 0 };
      case 'scenario': case 'scenarios': return { output: '', exit: 0, action: 'scenario' };
      case 'man': return this.man(args);
      case 'exit': case 'logout': return { output: 'There is no escape from a simulation.\n', exit: 0 };
      case 'sudo': return { output: `${this.world.promptUser} is not in the sudoers file. This incident will (not) be reported.\n`, exit: 1 };
      default:
        return { output: `${cmd}: command not found. Type ${ESC.cyan}help${ESC.reset}.\n`, exit: 127 };
    }
  }

  witr(args) {
    const { targets, flags, errors } = parse(args);
    if (flags.help) return { output: WITR_HELP + '\n', exit: 0 };
    if (flags.version) return { output: `witr ${WITR_VERSION}\n`, exit: 0 };
    if (errors.length > 0) {
      return { output: errors.map((e) => `Error: ${e}`).join('\n') + '\n', exit: EXIT.INVALID_INPUT };
    }
    // No targets, or explicit -i → launch the TUI.
    if (targets.length === 0 || flags.interactive) {
      return { output: '', exit: 0, action: 'tui' };
    }
    const { text, exit } = this.engine.run({ targets, flags });
    return { output: text, exit };
  }

  // ---- filesystem flavour ----------------------------------------------

  resolvePath(p) {
    if (!p) return this.cwd;
    if (p === '~') return `/home/${this.world.promptUser}`;
    if (p.startsWith('~/')) return `/home/${this.world.promptUser}/` + p.slice(2);
    if (p.startsWith('/')) return normalizePath(p);
    return normalizePath(this.cwd + '/' + p);
  }

  ls(args) {
    const target = args.find((a) => !a.startsWith('-'));
    const path = this.resolvePath(target);
    const fs = this.world.fs || {};
    const entries = fs[path];
    if (!entries) {
      // A file rather than a dir?
      if ((this.world.files || {})[path]) return { output: path + '\n', exit: 0 };
      return { output: `ls: cannot access '${target || path}': No such file or directory\n`, exit: 2 };
    }
    const colored = entries.map((e) => (e.endsWith('/') ? `${ESC.blue}${e}${ESC.reset}` : e));
    return { output: colored.join('  ') + '\n', exit: 0 };
  }

  cat(args) {
    if (args.length === 0) return { output: 'cat: missing file operand\n', exit: 1 };
    const path = this.resolvePath(args[0]);
    const content = (this.world.files || {})[path];
    if (content == null) {
      if ((this.world.fs || {})[path]) return { output: `cat: ${args[0]}: Is a directory\n`, exit: 1 };
      return { output: `cat: ${args[0]}: No such file or directory\n`, exit: 1 };
    }
    return { output: content.endsWith('\n') ? content : content + '\n', exit: 0 };
  }

  cd(args) {
    if (args.length === 0 || args[0] === '~') { this.cwd = `/home/${this.world.promptUser}`; return { output: '', exit: 0 }; }
    const path = this.resolvePath(args[0]);
    const fs = this.world.fs || {};
    if (fs[path] || fs[path + '/']) { this.cwd = path; return { output: '', exit: 0 }; }
    return { output: `cd: ${args[0]}: No such file or directory\n`, exit: 1 };
  }

  uname(args) {
    if (args.includes('-a')) {
      const w = this.world;
      return { output: `Linux ${w.hostname} ${w.kernel} #1 SMP ${w.arch} GNU/Linux\n`, exit: 0 };
    }
    return { output: 'Linux\n', exit: 0 };
  }

  ps() {
    let o = `${ESC.dim}  PID USER      COMMAND${ESC.reset}\n`;
    const procs = [...this.world.processes].sort((a, b) => a.pid - b.pid);
    for (const p of procs) {
      const pid = String(p.pid).padStart(5);
      const user = (p.user || '').padEnd(9).slice(0, 9);
      o += `${pid} ${user} ${p.cmdline || p.command}\n`;
    }
    return { output: o, exit: 0 };
  }

  neofetch() {
    const w = this.world;
    const art = [
      `${ESC.cyan}   __      _ _        ${ESC.reset}`,
      `${ESC.cyan}  / /_ __ (_) |_ _ __ ${ESC.reset}`,
      `${ESC.cyan} / / '_ \\| | __| '__|${ESC.reset}`,
      `${ESC.cyan}/ /| | | | | |_| |   ${ESC.reset}`,
      `${ESC.cyan}\\_/|_| |_|_|\\__|_|   ${ESC.reset}`,
    ];
    const info = [
      `${ESC.green}${w.promptUser}@${w.hostname}${ESC.reset}`,
      `${ESC.dim}─────────────────${ESC.reset}`,
      `${ESC.blue}OS${ESC.reset}:     ${w.distro} ${w.arch}`,
      `${ESC.blue}Kernel${ESC.reset}: ${w.kernel}`,
      `${ESC.blue}Procs${ESC.reset}:  ${w.processes.length}`,
      `${ESC.blue}Shell${ESC.reset}:  witr-playground`,
      `${ESC.dimYellow}note${ESC.reset}:   this machine is simulated`,
    ];
    let o = '';
    const rows = Math.max(art.length, info.length);
    for (let i = 0; i < rows; i++) {
      const left = art[i] || '                     ';
      o += `${left}   ${info[i] || ''}\n`;
    }
    return o;
  }

  man(args) {
    if (args[0] === 'witr') return { output: WITR_HELP + '\n', exit: 0 };
    return { output: `No manual entry for ${args[0] || ''}\n`, exit: 1 };
  }
}

function normalizePath(p) {
  const parts = p.split('/');
  const out = [];
  for (const part of parts) {
    if (part === '' || part === '.') continue;
    if (part === '..') out.pop();
    else out.push(part);
  }
  return '/' + out.join('/');
}
