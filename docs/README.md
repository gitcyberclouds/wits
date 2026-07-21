# witr playground

An interactive, zero-install, in-browser playground for [witr](../README.md). It
runs a **simulated Linux box** and lets visitors investigate it with real witr
commands — as a guided tutorial and as a free-form sandbox.

Nothing here touches the visitor's machine. Every process, port, container, and
lock is authored data; the terminal simulates witr, not a real shell.

**Live:** this folder *is* the site — GitHub Pages serves it directly (see
[Deployment](#deployment)) at `https://<owner>.github.io/witr/`.

> This directory also holds `cli/` — the generated `witr` man page and markdown
> reference (produced by `make docs`). That's unrelated to the playground; it
> just shares the Pages folder.

---

## What it does

- **Terminal-first.** A dependency-free terminal widget runs `witr …` against
  the simulated world and renders witr's real ANSI output. A handful of flavour
  commands (`ls`, `cat`, `ps`, `neofetch`, …) make the box feel real to poke at.
- **The incident (tutorial).** A cold open plays itself — a deploy fails with
  `EADDRINUSE`, witr traces the cause in one command — then hands the visitor a
  box with three real problems (a public dev server squatting on a port, a rogue
  ngrok tunnel, a stuck `dpkg` lock). They investigate with witr and **fix** them
  (`kill` actually removes processes; the lock clears on its own). A health
  tracker counts down to zero; hitting zero is the finale with the install
  command. Feature coverage (`--port`, `--file`, multi-match, the chain) falls
  out of the investigation; `--json`, `--verbose`, `--container`, and the TUI are
  optional side quests.
- **Reactive world.** The loaded world is a mutable clone: `kill`/`pkill` remove
  processes (and their subtrees), which the engine, the constellation, the TUI,
  and the incident tracker all reflect live. **Reset** restores the pristine box.
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
cd docs
python3 -m http.server 8099
# open http://localhost:8099/
```

## Project layout

```
docs/
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
    check-fixtures.mjs  engine ⇄ golden fixture diff
  vendor/
    three.module.min.js three.js r160 (vendored, MIT)
  cli/                  generated witr man page + markdown (from `make docs`)
  .nojekyll             serve files as-is (no Jekyll processing)
```

## Regenerating fixtures

Regenerate after changing a world file or witr's output format. The generator
is build-tagged, so it never affects the normal `go build ./...`:

```bash
# from the repo root
go run -tags fixtures ./docs/fixtures/gen
node docs/scripts/check-fixtures.mjs
```

Fixtures embed absolute timestamps and a pinned clock (`_meta.json`), so every
regeneration changes the timestamps — that's expected. The check uses the pinned
clock, so it stays deterministic.

## Adding a scenario

1. Add `worlds/<id>.json` (see the schema the existing worlds follow).
2. Add the id to `WORLD_IDS` in `js/app.js` and a card in `index.html`.
3. Optionally add fixtures for it in `fixtures/gen/main.go`.

## Deployment

The playground is served straight from this folder by GitHub Pages. Enable it
once:

**Settings → Pages → Build and deployment → Source: Deploy from a branch →
Branch: `main` / `/docs`.**

Every push to `main` then publishes automatically — no build step, no workflow.
`.nojekyll` is present so files (including `fixtures/_meta.json`) are served
verbatim. CI (`.github/workflows/playground.yml`) only runs the fidelity check.

## Credits

[three.js](https://threejs.org/) (r160, MIT) is vendored under `vendor/`. All
other code is part of witr and shares its license.
