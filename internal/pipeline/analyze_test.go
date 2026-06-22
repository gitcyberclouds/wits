package pipeline

import (
	"os"
	"testing"
)

// TestAnalyzePID_Self drives the full pipeline (ancestry walk → source
// detection → result assembly) against the running test process, which is
// guaranteed to exist on every platform.
func TestAnalyzePID_Self(t *testing.T) {
	self := os.Getpid()

	res, err := AnalyzePID(AnalyzeConfig{PID: self})
	if err != nil {
		t.Fatalf("AnalyzePID(self=%d): %v", self, err)
	}

	if res.Process.PID != self {
		t.Errorf("Process.PID = %d, want %d", res.Process.PID, self)
	}
	if len(res.Ancestry) == 0 {
		t.Fatal("expected a non-empty ancestry chain")
	}
	if last := res.Ancestry[len(res.Ancestry)-1]; last.PID != self {
		t.Errorf("ancestry should end at the target PID; got %d, want %d", last.PID, self)
	}
	if res.ResolvedTarget == "" {
		t.Error("ResolvedTarget should be set to the process command")
	}
	// Detect always classifies a source (at minimum SourceUnknown), never blank.
	if res.Source.Type == "" {
		t.Error("Source.Type should be classified")
	}
}

// TestAnalyzePID_Verbose ensures the verbose/tree path (extended info + child
// collection) runs without error and still returns the target.
func TestAnalyzePID_Verbose(t *testing.T) {
	self := os.Getpid()

	res, err := AnalyzePID(AnalyzeConfig{PID: self, Verbose: true, Tree: true})
	if err != nil {
		t.Fatalf("verbose AnalyzePID(self=%d): %v", self, err)
	}
	if res.Process.PID != self {
		t.Errorf("Process.PID = %d, want %d", res.Process.PID, self)
	}
}

// TestAnalyzePID_Nonexistent confirms the pipeline surfaces an error rather than
// returning a zero-value result for a PID that doesn't exist.
func TestAnalyzePID_Nonexistent(t *testing.T) {
	if _, err := AnalyzePID(AnalyzeConfig{PID: 2147483646}); err == nil {
		t.Error("expected an error for a nonexistent PID")
	}
}
