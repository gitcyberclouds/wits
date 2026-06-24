package app

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestExitCodeError(t *testing.T) {
	base := errors.New("boom")
	err := withExitCode(ExitPermission, base)

	if err.Error() != "boom" {
		t.Errorf("Error() = %q, want %q", err.Error(), "boom")
	}
	if !errors.Is(err, base) {
		t.Error("Unwrap should expose the wrapped error")
	}
	var ece *exitCodeError
	if !errors.As(err, &ece) || ece.code != ExitPermission {
		t.Errorf("errors.As should recover the exit code; got %+v", ece)
	}
}

func TestBoolFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("verbose", false, "")

	if boolFlag(cmd, "verbose") {
		t.Error("verbose should default to false")
	}
	_ = cmd.Flags().Set("verbose", "true")
	if !boolFlag(cmd, "verbose") {
		t.Error("verbose should be true after Set")
	}
	if boolFlag(cmd, "does-not-exist") {
		t.Error("an unknown flag should read as false")
	}
}

func TestFlagTakesValue(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSliceP("pid", "p", nil, "")
	cmd.Flags().BoolP("verbose", "v", false, "")
	takes := flagTakesValue(cmd)

	cases := []struct {
		arg  string
		want bool
	}{
		{"--pid", true},      // value-taking flag
		{"--verbose", false}, // boolean flag
		{"--pid=5", false},   // value is attached
		{"-p", true},         // value-taking shorthand
		{"-v", false},        // boolean shorthand
		{"notaflag", false},  // not a flag token at all
	}
	for _, tc := range cases {
		if got := takes(tc.arg); got != tc.want {
			t.Errorf("flagTakesValue()(%q) = %v, want %v", tc.arg, got, tc.want)
		}
	}
}

func TestRootCommand(t *testing.T) {
	root := Root()
	if root == nil {
		t.Fatal("Root() returned nil")
	}
	for _, name := range []string{"pid", "port", "file", "container", "json", "verbose", "interactive"} {
		if root.Flags().Lookup(name) == nil {
			t.Errorf("root command is missing the --%s flag", name)
		}
	}
}

func TestSetVersion(t *testing.T) {
	oldV, oldC, oldB := version, commit, buildDate
	t.Cleanup(func() { SetVersion(oldV, oldC, oldB) })

	SetVersion("v9.9.9", "abcdef1", "2026-01-02")
	if version != "v9.9.9" || commit != "abcdef1" || buildDate != "2026-01-02" {
		t.Errorf("package vars not updated: %q %q %q", version, commit, buildDate)
	}
	if Root().Version != "v9.9.9" {
		t.Errorf("rootCmd.Version = %q, want v9.9.9", Root().Version)
	}
}
