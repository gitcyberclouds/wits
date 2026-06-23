package app

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

func TestHandleResolveError(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		var errBuf bytes.Buffer
		cmd.SetErr(&errBuf)
		cmd.SetOut(&errBuf)
		return cmd
	}

	t.Run("generic not-found maps to ExitNotFound", func(t *testing.T) {
		var outw bytes.Buffer
		var jsonResults []string
		code := handleResolveError(newCmd(), &outw, output.NewPrinter(&outw),
			model.Target{Type: model.TargetName, Value: "ghost"},
			errors.New("no matching process found"),
			appFlags{}, false, &jsonResults)
		if code != ExitNotFound {
			t.Errorf("code = %d, want %d (ExitNotFound)", code, ExitNotFound)
		}
	})

	t.Run("unsupported target maps to ExitInvalidInput", func(t *testing.T) {
		var outw bytes.Buffer
		var jsonResults []string
		code := handleResolveError(newCmd(), &outw, output.NewPrinter(&outw),
			model.Target{Type: model.TargetFile, Value: "/x"},
			target.ErrUnsupported,
			appFlags{}, false, &jsonResults)
		if code != ExitInvalidInput {
			t.Errorf("code = %d, want %d (ExitInvalidInput)", code, ExitInvalidInput)
		}
	})

	t.Run("multi-mode JSON appends an error entry", func(t *testing.T) {
		var outw bytes.Buffer
		var jsonResults []string
		handleResolveError(newCmd(), &outw, output.NewPrinter(&outw),
			model.Target{Type: model.TargetName, Value: "ghost"},
			errors.New("no matching process found"),
			appFlags{json: true}, true, &jsonResults)
		if len(jsonResults) != 1 {
			t.Errorf("expected 1 JSON error entry, got %d", len(jsonResults))
		}
	})
}
