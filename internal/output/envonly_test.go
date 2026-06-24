package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRenderEnvOnly(t *testing.T) {
	r := model.Result{
		Process:  model.Process{PID: 1234, Command: "nginx", Cmdline: "/usr/sbin/nginx", Env: []string{"PATH=/usr/bin", "HOME=/root"}},
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, {PID: 1234, Command: "nginx"}},
	}

	t.Run("with env (plain and colored)", func(t *testing.T) {
		var plain bytes.Buffer
		RenderEnvOnly(&plain, r, false)
		if !strings.Contains(plain.String(), "Environment") || !strings.Contains(plain.String(), "PATH=/usr/bin") {
			t.Errorf("plain env render wrong:\n%s", plain.String())
		}
		var colored bytes.Buffer
		RenderEnvOnly(&colored, r, true)
		if !strings.Contains(colored.String(), "HOME=/root") {
			t.Errorf("colored env render missing a variable:\n%s", colored.String())
		}
	})

	t.Run("no env", func(t *testing.T) {
		r2 := r
		r2.Process.Env = nil
		var buf bytes.Buffer
		RenderEnvOnly(&buf, r2, false)
		if !strings.Contains(buf.String(), "No environment variables found.") {
			t.Errorf("expected empty-env message; got:\n%s", buf.String())
		}
	})
}
