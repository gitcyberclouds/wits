//go:build windows

package proc

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestParseEnvBlock(t *testing.T) {
	var block []uint16
	for _, e := range []string{"A=1", `PATH=C:\Windows`} {
		block = append(block, utf16.Encode([]rune(e))...)
		block = append(block, 0)
	}
	block = append(block, 0) // terminating empty entry

	if envBlockEnd(block) < 0 {
		t.Error("envBlockEnd should find the terminator")
	}
	got := parseEnvBlock(block)
	want := []string{"A=1", `PATH=C:\Windows`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseEnvBlock = %v, want %v", got, want)
	}
}

// TestReadProcessEnvSelf confirms the PEB environment read returns this
// process's environment (PATH is always set).
func TestReadProcessEnvSelf(t *testing.T) {
	p, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatalf("ReadProcess(self): %v", err)
	}
	if len(p.Env) == 0 {
		t.Fatal("expected a non-empty environment for self")
	}
	for _, e := range p.Env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			return
		}
	}
	t.Errorf("PATH not found among %d env entries", len(p.Env))
}
