// wasm-engine.js — loads witr's real engine, compiled to WebAssembly.
//
// The 4 MB module is fetched lazily, only when the visitor opts into the "real
// engine" mode, so it never slows the initial page load. Once running, it
// exposes globalThis.witrRun (installed by cmd/witr-wasm) which executes the
// actual witr Go pipeline over the injected world.

let readyPromise = null;

function loadScript(src) {
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = src;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error(`failed to load ${src}`));
    document.head.appendChild(s);
  });
}

// Returns a runner: (worldJSON, nowMs, argv[]) -> { output, exit }.
export function loadWasmEngine() {
  if (readyPromise) return readyPromise;
  readyPromise = (async () => {
    await loadScript('./vendor/wasm_exec.js');
    if (typeof globalThis.Go !== 'function') throw new Error('wasm_exec.js did not initialise');
    const go = new globalThis.Go();

    let instance;
    try {
      const res = await WebAssembly.instantiateStreaming(fetch('./wasm/witr.wasm'), go.importObject);
      instance = res.instance;
    } catch (_) {
      // Fallback when the server doesn't send application/wasm.
      const buf = await (await fetch('./wasm/witr.wasm')).arrayBuffer();
      const res = await WebAssembly.instantiate(buf, go.importObject);
      instance = res.instance;
    }

    // main() installs witrRun synchronously, then blocks on select{} keeping the
    // Go runtime alive. Do NOT await go.run — it only resolves when Go exits.
    go.run(instance);
    if (typeof globalThis.witrRun !== 'function') throw new Error('witr wasm did not initialise');

    return (worldJSON, nowMs, argv) => globalThis.witrRun(worldJSON, nowMs, argv);
  })();
  return readyPromise;
}
