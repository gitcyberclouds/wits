// tui.js — an interactive, DOM-based rendition of witr's TUI dashboard.
//
// Four tabs (Processes / Ports / Containers / Locks) over the same world data,
// with a live ancestry side panel and an auto-refreshing clock — the same shape
// as the real bubbletea TUI, close enough to teach the workflow in a browser.

import { formatStartedAt } from './engine.js';

const TABS = ['Processes', 'Ports', 'Containers', 'Locks'];

export class TUI {
  constructor(rootEl) {
    this.root = rootEl;
    this.open = false;
    this.tab = 0;
    this.sel = 0;
    this.filter = '';
    this.filtering = false;
    this.onClose = null;
    this._tick = null;
    this._keyHandler = (e) => this._onKey(e);
  }

  show(world, engine) {
    this.world = world;
    this.engine = engine;
    this.open = true;
    this.tab = 0;
    this.sel = 0;
    this.filter = '';
    this.filtering = false;
    this.root.classList.add('tui-open');
    this.root.setAttribute('aria-hidden', 'false');
    document.addEventListener('keydown', this._keyHandler, true);
    this.render();
    this._tick = setInterval(() => this.render(), 1000);
  }

  close() {
    this.open = false;
    this.root.classList.remove('tui-open');
    this.root.setAttribute('aria-hidden', 'true');
    document.removeEventListener('keydown', this._keyHandler, true);
    if (this._tick) clearInterval(this._tick);
    if (this.onClose) this.onClose();
  }

  rows() {
    const w = this.world;
    if (this.tab === 0) {
      let list = [...w.processes].sort((a, b) => a.pid - b.pid);
      if (this.filter) {
        const f = this.filter.toLowerCase();
        list = list.filter((p) => (p.command + ' ' + (p.cmdline || '') + ' ' + p.pid).toLowerCase().includes(f));
      }
      return list;
    }
    if (this.tab === 1) {
      const ports = [];
      for (const p of w.processes) for (const s of p.sockets || []) if (s.state === 'LISTEN') ports.push({ p, s });
      return ports.sort((a, b) => a.s.port - b.s.port);
    }
    if (this.tab === 2) return w.containers || [];
    return w.locks || [];
  }

  _onKey(e) {
    if (!this.open) return;
    if (this.filtering) {
      if (e.key === 'Enter' || e.key === 'Escape') { this.filtering = false; e.preventDefault(); this.render(); return; }
      if (e.key === 'Backspace') { this.filter = this.filter.slice(0, -1); e.preventDefault(); this.sel = 0; this.render(); return; }
      if (e.key.length === 1) { this.filter += e.key; e.preventDefault(); this.sel = 0; this.render(); return; }
      return;
    }
    const rows = this.rows();
    switch (e.key) {
      case 'Escape': case 'q': this.close(); break;
      case 'Tab':
        e.preventDefault();
        this.tab = (this.tab + (e.shiftKey ? TABS.length - 1 : 1)) % TABS.length;
        this.sel = 0; this.filter = ''; this.render();
        break;
      case '1': case '2': case '3': case '4':
        this.tab = Math.min(TABS.length - 1, parseInt(e.key, 10) - 1); this.sel = 0; this.filter = ''; this.render();
        break;
      case 'ArrowDown': case 'j':
        e.preventDefault(); this.sel = Math.min(rows.length - 1, this.sel + 1); this.render(); break;
      case 'ArrowUp': case 'k':
        e.preventDefault(); this.sel = Math.max(0, this.sel - 1); this.render(); break;
      case '/':
        if (this.tab === 0) { e.preventDefault(); this.filtering = true; this.render(); }
        break;
      default: break;
    }
    e.stopPropagation();
  }

  render() {
    if (!this.open) return;
    const w = this.world;
    const rows = this.rows();
    if (this.sel >= rows.length) this.sel = Math.max(0, rows.length - 1);

    const tabsHtml = TABS.map((t, i) =>
      `<button class="tui-tab${i === this.tab ? ' active' : ''}" data-tab="${i}">${i + 1} ${t}</button>`).join('');

    let body = '';
    if (this.tab === 0) body = this._procs(rows);
    else if (this.tab === 1) body = this._ports(rows);
    else if (this.tab === 2) body = this._containers(rows);
    else body = this._locks(rows);

    const footer = this.tab === 0
      ? '↑/↓ move · Tab switch · / filter · q quit · <span class="tui-live">● live</span>'
      : '↑/↓ move · Tab switch · q quit · <span class="tui-live">● live</span>';

    this.root.innerHTML = `
      <div class="tui-window" role="dialog" aria-label="witr interactive dashboard">
        <div class="tui-titlebar">
          <span class="tui-title">witr — ${escapeHtml(w.promptUser)}@${escapeHtml(w.hostname)}</span>
          <span class="tui-sim">simulated</span>
          <button class="tui-x" data-close>✕</button>
        </div>
        <div class="tui-tabs">${tabsHtml}</div>
        <div class="tui-body">${body}</div>
        <div class="tui-footer">${footer}</div>
      </div>`;

    this.root.querySelectorAll('.tui-tab').forEach((b) =>
      b.addEventListener('click', () => { this.tab = +b.dataset.tab; this.sel = 0; this.filter = ''; this.render(); }));
    this.root.querySelector('[data-close]').addEventListener('click', () => this.close());
    this.root.querySelectorAll('.tui-row').forEach((r) =>
      r.addEventListener('click', () => { this.sel = +r.dataset.i; this.render(); }));
  }

  _procs(rows) {
    const filterBar = this.filtering || this.filter
      ? `<div class="tui-filter">/${escapeHtml(this.filter)}${this.filtering ? '<span class="tui-caret">▏</span>' : ''}</div>` : '';
    let table = `<div class="tui-table"><div class="tui-head"><span class="c-pid">PID</span><span class="c-user">USER</span><span class="c-start">STARTED</span><span class="c-cmd">COMMAND</span></div>`;
    rows.forEach((p, i) => {
      const [rel] = formatStartedAt(this.engine.now() - (p.startedAgo || 0) * 1000, this.engine.now());
      const tag = p.health && p.health !== 'healthy' ? ` <span class="tui-tag">[${escapeHtml(p.health)}]</span>` : '';
      table += `<div class="tui-row${i === this.sel ? ' sel' : ''}" data-i="${i}">` +
        `<span class="c-pid">${p.pid}</span>` +
        `<span class="c-user">${escapeHtml(p.user || '')}</span>` +
        `<span class="c-start">${escapeHtml(rel)}</span>` +
        `<span class="c-cmd">${escapeHtml(p.cmdline || p.command)}${tag}</span></div>`;
    });
    table += '</div>';

    // Side panel: ancestry of selected.
    const sel = rows[this.sel];
    let side = '<div class="tui-side"><div class="tui-side-empty">no process</div></div>';
    if (sel) {
      const chain = this.engine.ancestryOf(sel);
      const kids = this.engine.childrenOf(sel.pid);
      let s = `<div class="tui-side"><div class="tui-side-h">Why is <b>${escapeHtml(sel.command)}</b> running?</div><div class="tui-chain">`;
      chain.forEach((p, i) => {
        const last = i === chain.length - 1;
        s += `<div class="tui-chain-node${last ? ' target' : ''}">${'  '.repeat(i)}${i > 0 ? '└─ ' : ''}${escapeHtml(p.command)} <span class="tui-dim">(pid ${p.pid})</span></div>`;
      });
      s += '</div>';
      const src = sel.source || this.engine.resolveSource(sel, chain);
      s += `<div class="tui-side-row"><span class="tui-k">Source</span>${escapeHtml(src.name ? src.name + ' (' + src.type + ')' : src.type)}</div>`;
      if (sel.workingDir) s += `<div class="tui-side-row"><span class="tui-k">Cwd</span>${escapeHtml(sel.workingDir)}</div>`;
      const socks = (sel.sockets || []).filter((x) => x.address && x.port);
      if (socks.length) s += `<div class="tui-side-row"><span class="tui-k">Sockets</span>${socks.map((x) => escapeHtml(x.address + ':' + x.port)).join(', ')}</div>`;
      if (kids.length) s += `<div class="tui-side-row"><span class="tui-k">Children</span>${kids.length}</div>`;
      if ((sel.warnings || []).length) s += `<div class="tui-side-warn">⚠ ${escapeHtml(sel.warnings[0])}</div>`;
      s += '</div>';
      side = s;
    }
    return `<div class="tui-split">${table}${side}${filterBar}</div>`;
  }

  _ports(rows) {
    let t = `<div class="tui-table wide"><div class="tui-head"><span>ADDRESS:PORT</span><span>PROTO</span><span>STATE</span><span>PID</span><span>PROCESS</span></div>`;
    rows.forEach(({ p, s }, i) => {
      const addr = s.address.includes(':') ? `[${s.address}]:${s.port}` : `${s.address}:${s.port}`;
      t += `<div class="tui-row${i === this.sel ? ' sel' : ''}" data-i="${i}"><span>${escapeHtml(addr)}</span><span>${escapeHtml(s.protocol)}</span><span>LISTENING</span><span>${p.pid}</span><span>${escapeHtml(p.command)}</span></div>`;
    });
    return t + '</div>';
  }

  _containers(rows) {
    if (rows.length === 0) return '<div class="tui-side-empty">no containers</div>';
    let t = `<div class="tui-table wide"><div class="tui-head"><span>NAME</span><span>IMAGE</span><span>STATE</span><span>PORTS</span><span>COMPOSE</span></div>`;
    rows.forEach((c, i) => {
      const compose = c.composeProject ? `${c.composeProject}/${c.composeService}` : '';
      const stateCls = c.state === 'running' ? 'ok' : (c.state === 'restarting' ? 'warn' : 'dim');
      t += `<div class="tui-row${i === this.sel ? ' sel' : ''}" data-i="${i}"><span>${escapeHtml(c.name)}</span><span>${escapeHtml(c.image)}</span><span class="st-${stateCls}">${escapeHtml(c.status || c.state)}</span><span>${escapeHtml(c.ports || '—')}</span><span>${escapeHtml(compose)}</span></div>`;
    });
    return t + '</div>';
  }

  _locks(rows) {
    if (rows.length === 0) return '<div class="tui-side-empty">no file locks</div>';
    let t = `<div class="tui-table wide"><div class="tui-head"><span>PID</span><span>PROCESS</span><span>TYPE</span><span>MODE</span><span>PATH</span></div>`;
    rows.forEach((l, i) => {
      t += `<div class="tui-row${i === this.sel ? ' sel' : ''}" data-i="${i}"><span>${l.pid}</span><span>${escapeHtml(l.process)}</span><span>${escapeHtml(l.type)}</span><span>${escapeHtml(l.mode)}</span><span>${escapeHtml(l.path)}</span></div>`;
    });
    return t + '</div>';
  }
}

function escapeHtml(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
