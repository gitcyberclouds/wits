package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func tgt(tp model.TargetType, v string) model.Target { return model.Target{Type: tp, Value: v} }

// collectTargetsInOrder is the order-preserving CLI target parser. It duplicates
// some of cobra's flag semantics (it reads raw argv to keep the user's typed
// order), so these cases pin that behavior down before any future refactor.
func TestCollectTargetsInOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rawArgs    []string
		positional []string
		valueFlags []string // non-target flags that consume the next token
		want       []model.Target
	}{
		{
			name:       "single name",
			rawArgs:    []string{"nginx"},
			positional: []string{"nginx"},
			want:       []model.Target{tgt(model.TargetName, "nginx")},
		},
		{
			name:    "pid long flag, space form",
			rawArgs: []string{"--pid", "1234"},
			want:    []model.Target{tgt(model.TargetPID, "1234")},
		},
		{
			name:    "pid short flag",
			rawArgs: []string{"-p", "1234"},
			want:    []model.Target{tgt(model.TargetPID, "1234")},
		},
		{
			name:    "equals form",
			rawArgs: []string{"--port=8080"},
			want:    []model.Target{tgt(model.TargetPort, "8080")},
		},
		{
			name:    "comma split, space form",
			rawArgs: []string{"--port", "80,443"},
			want:    []model.Target{tgt(model.TargetPort, "80"), tgt(model.TargetPort, "443")},
		},
		{
			name:    "comma split, equals form, blanks trimmed",
			rawArgs: []string{"--pid=1, ,2"},
			want:    []model.Target{tgt(model.TargetPID, "1"), tgt(model.TargetPID, "2")},
		},
		{
			name:       "interleaved order preserved",
			rawArgs:    []string{"nginx", "--pid", "1234", "node"},
			positional: []string{"nginx", "node"},
			want: []model.Target{
				tgt(model.TargetName, "nginx"),
				tgt(model.TargetPID, "1234"),
				tgt(model.TargetName, "node"),
			},
		},
		{
			name:       "boolean flags skipped",
			rawArgs:    []string{"--json", "nginx", "--verbose", "-x"},
			positional: []string{"nginx"},
			want:       []model.Target{tgt(model.TargetName, "nginx")},
		},
		{
			name:       "mixed flag types",
			rawArgs:    []string{"--port", "8080", "redis", "-f", "/tmp/x", "-c", "web"},
			positional: []string{"redis"},
			want: []model.Target{
				tgt(model.TargetPort, "8080"),
				tgt(model.TargetName, "redis"),
				tgt(model.TargetFile, "/tmp/x"),
				tgt(model.TargetContainer, "web"),
			},
		},
		{
			name:       "remaining positionals appended",
			rawArgs:    []string{},
			positional: []string{"a", "b"},
			want:       []model.Target{tgt(model.TargetName, "a"), tgt(model.TargetName, "b")},
		},
		{
			// Regression for the latent argv bug: a space-form string-valued
			// flag must consume its value, or that value steals a positional
			// slot and corrupts the interleaved target order. Fails without the
			// flag-arity fix (would yield alpha, beta, pid:5).
			name:       "value-flag value not mistaken for a target",
			rawArgs:    []string{"alpha", "--config", "nginx", "--pid", "5", "beta"},
			positional: []string{"alpha", "beta"},
			valueFlags: []string{"--config"},
			want: []model.Target{
				tgt(model.TargetName, "alpha"),
				tgt(model.TargetPID, "5"),
				tgt(model.TargetName, "beta"),
			},
		},
		{
			name:       "value-flag in equals form needs no lookahead",
			rawArgs:    []string{"--config=app.yml", "nginx"},
			positional: []string{"nginx"},
			valueFlags: []string{"--config"},
			want:       []model.Target{tgt(model.TargetName, "nginx")},
		},
		{
			name:       "short value-flag, space form",
			rawArgs:    []string{"-C", "app.yml", "redis"},
			positional: []string{"redis"},
			valueFlags: []string{"-C"},
			want:       []model.Target{tgt(model.TargetName, "redis")},
		},
		{
			name:    "target flag without a value yields no target",
			rawArgs: []string{"--pid"},
			want:    nil,
		},
		{
			name:    "empty value via equals yields no target",
			rawArgs: []string{"--port="},
			want:    nil,
		},
		{
			// "--" ends option parsing: following tokens are positional names even
			// when they look like flags.
			name:       "double dash ends option parsing",
			rawArgs:    []string{"--", "nginx"},
			positional: []string{"nginx"},
			want:       []model.Target{tgt(model.TargetName, "nginx")},
		},
		{
			name:       "double dash forces flaglike tokens to names",
			rawArgs:    []string{"--", "--pid", "5"},
			positional: []string{"--pid", "5"},
			want:       []model.Target{tgt(model.TargetName, "--pid"), tgt(model.TargetName, "5")},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			takesValue := func(a string) bool {
				for _, f := range tc.valueFlags {
					if a == f {
						return true
					}
				}
				return false
			}
			got := collectTargetsInOrder(tc.rawArgs, tc.positional, takesValue)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("collectTargetsInOrder(%v, %v)\n got: %v\nwant: %v", tc.rawArgs, tc.positional, got, tc.want)
			}
		})
	}
}

func TestTargetLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   model.Target
		want string
	}{
		{tgt(model.TargetPID, "1234"), "pid: 1234"},
		{tgt(model.TargetPort, "80"), "port: 80"},
		{tgt(model.TargetFile, "/x"), "file: /x"},
		{tgt(model.TargetContainer, "c"), "container: c"},
		{tgt(model.TargetName, "n"), "name: n"},
	}
	for _, c := range cases {
		if got := targetLabel(c.in); got != c.want {
			t.Errorf("targetLabel(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestJSONErrorEntry(t *testing.T) {
	t.Parallel()

	s := jsonErrorEntry(tgt(model.TargetPort, "8080"), "boom")
	for _, want := range []string{`"Error"`, "boom", "8080", "port"} {
		if !strings.Contains(s, want) {
			t.Errorf("jsonErrorEntry missing %q in:\n%s", want, s)
		}
	}
}
