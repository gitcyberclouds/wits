#!/usr/bin/env bash
# Build the witr engine to WebAssembly for the playground's "real engine" mode.
#
#   playground/scripts/build-wasm.sh
#
# Produces playground/wasm/witr.wasm and vendors the matching wasm_exec.js glue.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
out_dir="$repo_root/playground/wasm"
vendor_dir="$repo_root/playground/vendor"
mkdir -p "$out_dir"

echo "building witr.wasm (GOOS=js GOARCH=wasm)…"
cd "$repo_root"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$out_dir/witr.wasm" ./cmd/witr-wasm

# wasm_exec.js must match the toolchain that built the binary.
goroot="$(go env GOROOT)"
if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then
  cp "$goroot/lib/wasm/wasm_exec.js" "$vendor_dir/wasm_exec.js"
elif [ -f "$goroot/misc/wasm/wasm_exec.js" ]; then
  cp "$goroot/misc/wasm/wasm_exec.js" "$vendor_dir/wasm_exec.js"
else
  echo "warning: wasm_exec.js not found under $goroot" >&2
fi

size="$(du -h "$out_dir/witr.wasm" | cut -f1)"
echo "wrote $out_dir/witr.wasm ($size) and vendored wasm_exec.js"
