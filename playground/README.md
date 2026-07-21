# witr playground

An interactive, zero-install, in-browser playground for [witr](../README.md). It
runs a **simulated Linux box** and lets visitors investigate it with real witr
commands — as a guided tutorial and as a free-form sandbox.

Nothing here touches the visitor's machine. Every process, port, container, and
lock is authored data; the terminal simulates witr, not a real shell.

**Live:** enable Pages (see [Deployment](#deployment)) and it publishes to
`https://<owner>.github.io/witr/`.

---

## What it does

- **Terminal-first.** A dependency-free terminal widget runs `witr …` against
  the simulated world and renders witr's real ANSI output. A handful of flavour
  commands (`ls`, `cat`, `ps`, `neofetch`, …) make the box feel real to poke at.
- **Tutorial mode.** Nine missions frame each witr feature as a small mystery
  (a mystery port, a stuck `dpkg` lock, a Redis container with no host process,
  a zombie …). Completing them walks through names, `--port`, `--tree`, `--exact`,
  `--file`, `--container`, `--json`, `--verbose`, and the TUI.
- **Playground mode.** Free rein to type any witr command against the box, or
  switch scenarios (a production web box, a messy dev laptop).
- **Process constellation.** A three.js view of the machine. When a query
  resolves, the causal chain (`systemd → … → target`) lights up while everything
  else dims — the text says the chain, the map shows it. Nodes are clickable.
- **Interactive TUI.** `witr` with no arguments opens a live dashboard
  (Processes / Ports / Containers / Locks) with an ancestry side-panel — the
  same shape as witr's real bubbletea TUI.

## Fidelity

The whole point is that the playground never lies about what witr prints.

- `js/engine.js` is a faithful port of witr's output layer
  (`internal/output/*.go`) and app routing (`internal/app/app.go`).
- `fixtures/gen/` is a small Go program that renders **golden fixtures using
  witr's actual output package**. `scripts/check-fixtures.mjs` replays the JS
  engine over the same world (with a pinned clock) and asserts byte-for-byte
  equality. CI runs this on every change — if the engine drifts from witr, the
  build fails.

## Run it locally

Any static file server works (ES modules need `http://`, not `file://`):

```bash
cd playground
python3 -m http.server 8099
# open http://localhost:8099/
```

## Project layout

```
playground/
  index.html            page shell
  css/styles.css        terminal-first theme (dark + light)
  js/
    ansi.js             ANSI escape → HTML
    engine.js           faithful witr output engine  ← fidelity-critical
    parser.js           witr command-line parser
    shell.js            command routing + flavour commands
    terminal.js         dependency-free terminal widget
    map.js              three.js process constellation
    tui.js              interactive TUI dashboard
    tutorial.js         mission definitions + progression
    app.js              wires it all together
  worlds/               the simulated machines (single source of truth)
    webbox.json         production box (tutorial)
    devbox.json         dev laptop (sandbox)
  fixtures/             golden output from the real witr binary
    gen/main.go         generator (build-tagged: `-tags fixtures`)
  scripts/
    check-fixtures.mjs  JS engine ⇄ golden fixture diff
  vendor/
    three.module.min.js three.js r160 (vendored, MIT)
```

## Regenerating fixtures

Regenerate after changing a world file or witr's output format. The generator
is build-tagged, so it never affects the normal `go build ./...`:

```bash
# from the repo root
go run -tags fixtures ./playground/fixtures/gen
node playground/scripts/check-fixtures.mjs
```

Fixtures embed absolute timestamps and a pinned clock (`_meta.json`), so every
regeneration changes the timestamps — that's expected. The check uses the pinned
clock, so it stays deterministic.

## Adding a scenario

1. Add `worlds/<id>.json` (see the schema the existing worlds follow).
2. Add the id to `WORLD_IDS` in `js/app.js` and a card in `index.html`.
3. Optionally add fixtures for it in `fixtures/gen/main.go`.

## Deployment

`.github/workflows/playground.yml` publishes to GitHub Pages on every push to
`main` that touches `playground/**`. Enable it once:

**Settings → Pages → Build and deployment → Source: GitHub Actions.**

The same workflow runs the fidelity check on pull requests.

## Credits

[three.js](https://threejs.org/) (r160, MIT) is vendored under `vendor/`. All
other code is part of witr and shares its license.
