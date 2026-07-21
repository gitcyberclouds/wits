// map.js — the "process constellation": a three.js view of the fake machine.
//
// Processes are nodes, ppid links are edges. When a witr query resolves, the
// causal chain (systemd → ... → target) lights up while everything else dims —
// the text output says the chain, the map shows it. Nodes are clickable, so the
// map doubles as a launcher.

import * as THREE from '../vendor/three.module.min.js';

const COLORS = {
  base: 0x2b3a4a,
  edge: 0x243445,
  proc: 0x5fb0d8,
  listener: 0x6cc58f,
  container: 0xc58fd8,
  root: 0xd8d2c0,
  warn: 0xd8a24a,
  highlight: 0x8ce0a2,
  highlightEdge: 0x6cc58f,
};

export class SystemMap {
  constructor(canvas, labelLayer) {
    this.canvas = canvas;
    this.labelLayer = labelLayer;
    this.nodes = [];
    this.nodeByPid = new Map();
    this.onSelect = null;
    this.hovered = null;
    this.highlightSet = new Set();
    this.reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    this.TARGET = 130; // constellation is normalised to this radius

    this.scene = new THREE.Scene();
    this.camera = new THREE.PerspectiveCamera(45, 1, 1, 4000);
    this.camera.position.set(this.TARGET * 0.28, this.TARGET * 0.32, this.TARGET * 2.15);
    this.camera.lookAt(0, 0, 0);
    // Camera easing: the view glides to frame the highlighted chain and back.
    this._defaultCamPos = this.camera.position.clone();
    this._camPosTarget = this.camera.position.clone();
    this._camLook = new THREE.Vector3(0, 0, 0);
    this._camLookTarget = new THREE.Vector3(0, 0, 0);
    this._frozen = false;
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));

    this.group = new THREE.Group();
    this.scene.add(this.group);
    this._addStars();

    this.raycaster = new THREE.Raycaster();
    this.pointer = new THREE.Vector2(-2, -2);
    this._haloTexture = makeHaloTexture();

    canvas.addEventListener('pointermove', (e) => this._onPointerMove(e));
    canvas.addEventListener('pointerleave', () => { this.pointer.set(-2, -2); });
    canvas.addEventListener('click', (e) => this._onClick(e));

    this._animate = this._animate.bind(this);
  }

  start() {
    this.resize();
    requestAnimationFrame(this._animate);
  }

  resize() {
    const r = this.canvas.getBoundingClientRect();
    const w = Math.max(1, r.width), h = Math.max(1, r.height);
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  _addStars() {
    const N = 260;
    const pos = new Float32Array(N * 3);
    // Deterministic pseudo-random scatter on a shell (no Math.random needed).
    for (let i = 0; i < N; i++) {
      const a = i * 2.399963; // golden angle
      const r = 420 + ((i * 137) % 380);
      const y = ((i * 71) % 500) - 250;
      pos[i * 3] = Math.cos(a) * r;
      pos[i * 3 + 1] = y;
      pos[i * 3 + 2] = Math.sin(a) * r - 120;
    }
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
    const stars = new THREE.Points(geo, new THREE.PointsMaterial({ color: 0x3a4c60, size: 1.7, transparent: true, opacity: 0.7 }));
    this.scene.add(stars);
    this._stars = stars;
  }

  // ---- build the graph from a world -------------------------------------

  setWorld(world) {
    this.world = world;
    // Clear previous.
    while (this.group.children.length) this.group.remove(this.group.children[0]);
    this.nodes = [];
    this.nodeByPid.clear();
    this.labelLayer.innerHTML = '';
    this.highlightSet.clear();

    const procs = world.processes;
    const byPid = new Map(procs.map((p) => [p.pid, p]));
    const childrenOf = (pid) => procs.filter((p) => p.ppid === pid && p.pid !== pid);

    // Depth from root.
    const depth = new Map();
    const computeDepth = (p) => {
      if (depth.has(p.pid)) return depth.get(p.pid);
      const parent = byPid.get(p.ppid);
      const d = parent ? computeDepth(parent) + 1 : 0;
      depth.set(p.pid, d);
      return d;
    };
    procs.forEach(computeDepth);

    // Leaf-ordered y layout (DFS).
    const roots = procs.filter((p) => !byPid.get(p.ppid)).sort((a, b) => a.pid - b.pid);
    let leafCursor = 0;
    const yOf = new Map();
    const layout = (p) => {
      const kids = childrenOf(p.pid).sort((a, b) => a.pid - b.pid);
      if (kids.length === 0) { yOf.set(p.pid, leafCursor++); return yOf.get(p.pid); }
      const ys = kids.map(layout);
      const y = ys.reduce((a, b) => a + b, 0) / ys.length;
      yOf.set(p.pid, y);
      return y;
    };
    roots.forEach(layout);
    const maxLeaf = Math.max(1, leafCursor - 1);

    // Raw positions (depth → x, leaf order → y, deterministic jitter → z),
    // then normalise to a fixed radius so the framing is stable for any world.
    const X = 60, Y = 20;
    const raw = new Map();
    for (const p of procs) {
      const d = depth.get(p.pid);
      const yy = (yOf.get(p.pid) - maxLeaf / 2) * Y;
      const zz = ((p.pid * 53) % 60) - 30;
      raw.set(p.pid, new THREE.Vector3(d * X, -yy, zz));
    }
    const center = new THREE.Vector3();
    for (const v of raw.values()) center.add(v);
    center.multiplyScalar(1 / Math.max(1, raw.size));
    let maxExt = 1;
    for (const v of raw.values()) maxExt = Math.max(maxExt, v.distanceTo(center));
    const scale = this.TARGET / maxExt;
    for (const v of raw.values()) v.sub(center).multiplyScalar(scale);

    const nodeGeo = new THREE.SphereGeometry(5.4, 22, 22);

    for (const p of procs) {
      const d = depth.get(p.pid);
      const pos = raw.get(p.pid);

      const isRoot = d === 0;
      const isListener = (p.sockets || []).some((s) => s.state === 'LISTEN');
      const hasWarn = (p.warnings || []).length > 0 || (p.health && p.health !== 'healthy');
      let color = COLORS.proc;
      if (isRoot) color = COLORS.root;
      else if (hasWarn) color = COLORS.warn;
      else if (isListener) color = COLORS.listener;

      const mat = new THREE.MeshBasicMaterial({ color });
      const mesh = new THREE.Mesh(nodeGeo, mat);
      mesh.position.copy(pos);
      this.group.add(mesh);

      const halo = new THREE.Sprite(new THREE.SpriteMaterial({
        map: this._haloTexture, color, transparent: true, opacity: 0.5,
        blending: THREE.AdditiveBlending, depthWrite: false,
      }));
      halo.scale.set(30, 30, 1);
      halo.position.copy(pos);
      this.group.add(halo);

      const label = document.createElement('div');
      label.className = 'map-label';
      label.textContent = p.command;
      this.labelLayer.appendChild(label);

      const node = { pid: p.pid, proc: p, pos, mesh, halo, mat, color, label, isListener, isRoot, hasWarn };
      this.nodes.push(node);
      this.nodeByPid.set(p.pid, node);
    }

    // Edges (ppid links).
    const edgePts = [];
    for (const p of procs) {
      const parent = this.nodeByPid.get(p.ppid);
      const self = this.nodeByPid.get(p.pid);
      if (parent && self) { edgePts.push(parent.pos.clone(), self.pos.clone()); }
    }
    const edgeGeo = new THREE.BufferGeometry().setFromPoints(edgePts);
    this.edges = new THREE.LineSegments(edgeGeo, new THREE.LineBasicMaterial({ color: COLORS.edge, transparent: true, opacity: 0.55 }));
    this.group.add(this.edges);

    // Highlighted-chain edge overlay (empty initially).
    this.hlEdges = new THREE.LineSegments(new THREE.BufferGeometry(), new THREE.LineBasicMaterial({ color: COLORS.highlightEdge, transparent: true, opacity: 0.95 }));
    this.group.add(this.hlEdges);

    this.group.position.set(0, 0, 0);
    this.clearHighlight();
  }

  // ---- highlight --------------------------------------------------------

  highlightPids(pids) {
    this.highlightSet = new Set(pids);
    // Build chain edge segments in order.
    const pts = [];
    for (let i = 0; i + 1 < pids.length; i++) {
      const a = this.nodeByPid.get(pids[i]);
      const b = this.nodeByPid.get(pids[i + 1]);
      if (a && b) pts.push(a.pos.clone(), b.pos.clone());
    }
    this.hlEdges.geometry.dispose();
    this.hlEdges.geometry = new THREE.BufferGeometry().setFromPoints(pts);
    this._applyStyles();
    this._pulse = 0;
    this._frameHighlight(pids);
  }

  // Freeze the rotation and glide the camera to frame the highlighted chain.
  _frameHighlight(pids) {
    this.group.updateMatrixWorld(true);
    const worlds = [];
    const centroid = new THREE.Vector3();
    for (const pid of pids) {
      const n = this.nodeByPid.get(pid);
      if (!n) continue;
      const wp = n.pos.clone().applyMatrix4(this.group.matrixWorld);
      worlds.push(wp);
      centroid.add(wp);
    }
    if (!worlds.length) return;
    centroid.multiplyScalar(1 / worlds.length);
    let radius = 0;
    for (const wp of worlds) radius = Math.max(radius, wp.distanceTo(centroid));
    // Pad for the enlarged/pulsing highlighted node spheres and their glow.
    radius = Math.max(radius, 24) + 16;

    this._frozen = true;
    const fov = (this.camera.fov * Math.PI) / 180;
    // Fit against the tighter (vertical) axis with generous margin, so even a
    // long chain (systemd → sshd → sshd → sshd → bash → …) sits fully in frame.
    let dist = (radius * 2.1) / Math.tan(fov / 2);
    dist = Math.min(Math.max(dist, this.TARGET * 0.95), this.TARGET * 3.6);
    const dir = this.camera.position.clone().sub(centroid);
    if (dir.lengthSq() < 1) dir.set(0.25, 0.3, 1);
    dir.normalize();
    this._camPosTarget = centroid.clone().add(dir.multiplyScalar(dist));
    this._camLookTarget = centroid.clone();
  }

  clearHighlight() {
    this.highlightSet = new Set();
    if (this.hlEdges) { this.hlEdges.geometry.dispose(); this.hlEdges.geometry = new THREE.BufferGeometry(); }
    this._applyStyles();
    this._frozen = false;
    if (this._defaultCamPos) {
      this._camPosTarget = this._defaultCamPos.clone();
      this._camLookTarget = new THREE.Vector3(0, 0, 0);
    }
  }

  // Remove a node when its process is killed, then rebuild the edge lines.
  removeProcess(pid) {
    const node = this.nodeByPid.get(pid);
    if (!node) return;
    this.group.remove(node.mesh);
    this.group.remove(node.halo);
    node.mat.dispose?.();
    node.halo.material.dispose?.();
    if (node.label && node.label.remove) node.label.remove();
    this.nodes = this.nodes.filter((n) => n.pid !== pid);
    this.nodeByPid.delete(pid);

    const pts = [];
    for (const n of this.nodes) {
      const parent = this.nodeByPid.get(n.proc.ppid);
      if (parent) pts.push(parent.pos.clone(), n.pos.clone());
    }
    this.edges.geometry.dispose();
    this.edges.geometry = new THREE.BufferGeometry().setFromPoints(pts);
    if (this.highlightSet.has(pid)) this.clearHighlight();
  }

  _applyStyles() {
    const has = this.highlightSet.size > 0;
    for (const n of this.nodes) {
      const on = this.highlightSet.has(n.pid);
      if (!has) {
        n.mat.color.setHex(n.color);
        n.mesh.scale.setScalar(1);
        n.halo.material.opacity = 0.45;
        n.halo.material.color.setHex(n.color);
      } else if (on) {
        n.mat.color.setHex(COLORS.highlight);
        n.mesh.scale.setScalar(1.5);
        n.halo.material.opacity = 0.9;
        n.halo.material.color.setHex(COLORS.highlight);
      } else {
        n.mat.color.setHex(COLORS.base);
        n.mesh.scale.setScalar(0.8);
        n.halo.material.opacity = 0.08;
        n.halo.material.color.setHex(COLORS.base);
      }
    }
    if (this.edges) this.edges.material.opacity = has ? 0.18 : 0.55;
  }

  // ---- interaction ------------------------------------------------------

  _pointerToNode(e) {
    const rect = this.canvas.getBoundingClientRect();
    this.pointer.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
    this.pointer.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;
    this.raycaster.setFromCamera(this.pointer, this.camera);
    const meshes = this.nodes.map((n) => n.mesh);
    const hit = this.raycaster.intersectObjects(meshes, false)[0];
    if (!hit) return null;
    return this.nodes.find((n) => n.mesh === hit.object) || null;
  }

  _onPointerMove(e) {
    const node = this._pointerToNode(e);
    this.hovered = node;
    this.canvas.style.cursor = node ? 'pointer' : 'grab';
  }

  _onClick(e) {
    const node = this._pointerToNode(e);
    if (node && this.onSelect) this.onSelect(node.proc);
  }

  // ---- render loop ------------------------------------------------------

  _animate() {
    requestAnimationFrame(this._animate);
    if (!this.reduced && !this._frozen) this.group.rotation.y += 0.0011;
    if (this._stars) this._stars.rotation.y += 0.0003;
    if (this._pulse !== undefined) this._pulse += 0.06;

    // Ease the camera toward its target (default orbit, or a framed chain).
    if (this._camPosTarget) {
      this.camera.position.lerp(this._camPosTarget, 0.055);
      this._camLook.lerp(this._camLookTarget, 0.055);
      this.camera.lookAt(this._camLook);
    }

    // Pulse highlighted nodes.
    if (this.highlightSet.size > 0) {
      const s = 1.5 + Math.sin(this._pulse) * 0.18;
      for (const n of this.nodes) if (this.highlightSet.has(n.pid)) n.mesh.scale.setScalar(s);
    }

    this.renderer.render(this.scene, this.camera);
    this._updateLabels();
  }

  _updateLabels() {
    const rect = this.canvas.getBoundingClientRect();
    const v = new THREE.Vector3();
    const hasHl = this.highlightSet.size > 0;
    for (const n of this.nodes) {
      const show = hasHl ? this.highlightSet.has(n.pid) : (n.isListener || n.isRoot || n.hasWarn || n === this.hovered) || n === this.hovered;
      if (!show) { n.label.style.display = 'none'; continue; }
      v.copy(n.pos).applyMatrix4(this.group.matrixWorld).project(this.camera);
      if (v.z > 1) { n.label.style.display = 'none'; continue; }
      const x = (v.x * 0.5 + 0.5) * rect.width;
      const y = (-v.y * 0.5 + 0.5) * rect.height;
      n.label.style.display = 'block';
      n.label.style.transform = `translate(-50%, -140%) translate(${x}px, ${y}px)`;
      n.label.classList.toggle('map-label-hl', hasHl && this.highlightSet.has(n.pid));
      n.label.classList.toggle('map-label-warn', !hasHl && n.hasWarn);
    }
  }
}

function makeHaloTexture() {
  const size = 64;
  const c = document.createElement('canvas');
  c.width = c.height = size;
  const ctx = c.getContext('2d');
  const g = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  g.addColorStop(0, 'rgba(255,255,255,1)');
  g.addColorStop(0.25, 'rgba(255,255,255,0.5)');
  g.addColorStop(1, 'rgba(255,255,255,0)');
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, size, size);
  const tex = new THREE.CanvasTexture(c);
  return tex;
}
