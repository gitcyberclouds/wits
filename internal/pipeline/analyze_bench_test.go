package pipeline

import (
	"os"
	"testing"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

// Performance budget
// ------------------
// These benchmarks pin down where `witr <target>` spends its time and give the
// later caching and per-PID memoization work a fixed yardstick. They drive the
// real pipeline against the running test process, whose ancestry exists on
// every platform.
//
//   go test ./internal/pipeline/ -run '^$' -bench . -benchmem
//
// Baseline (WSL/Ubuntu dev host, 2026-06, ~6-deep ancestry under `go test`):
//   BenchmarkAnalyzePID         ~83 ms/op   1.7k allocs   standard report
//   BenchmarkAnalyzePIDVerbose  ~332 ms/op  3.6k allocs   --verbose + tree
//   BenchmarkResolveAncestry    ~77 ms/op   1.7k allocs   the ancestry walk
//   BenchmarkReadProcess        ~12 ms/op   290 allocs    one process read
//
// The walk is ~93% of AnalyzePID: ResolveAncestry calls ReadProcess per
// ancestor, and on Linux each ReadProcess shells out to `systemctl` and probes
// for a git repo — one systemctl per hop drives the ~12 ms ReadProcess cost,
// multiplied by ancestry depth.
//
// TARGETS (warm cache, workstation-class host):
//   - Subprocess budget (host-independent — the real lever): <= 2 `systemctl`
//     spawns per ancestry walk regardless of depth; <= 2 per-PID tool
//     invocations per hop on macOS/FreeBSD. Today: ~1 systemctl per hop.
//   - Wall-clock: standard AnalyzePID < 40 ms, ResolveAncestry < 30 ms,
//     ReadProcess < 5 ms; --verbose < 200 ms. Removing the per-hop systemctl
//     fan-out should roughly halve today's numbers.
//
// Re-run before/after the caching work and record the deltas in that PR.

func BenchmarkAnalyzePID(b *testing.B) {
	self := os.Getpid()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := AnalyzePID(AnalyzeConfig{PID: self}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnalyzePIDVerbose(b *testing.B) {
	self := os.Getpid()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := AnalyzePID(AnalyzeConfig{PID: self, Verbose: true, Tree: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveAncestry(b *testing.B) {
	self := os.Getpid()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := procpkg.ResolveAncestry(self); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadProcess(b *testing.B) {
	self := os.Getpid()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := procpkg.ReadProcess(self); err != nil {
			b.Fatal(err)
		}
	}
}
