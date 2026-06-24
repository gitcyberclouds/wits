//go:build linux

package proc

import "testing"

func TestParseStatSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		pid      int
		stat     string
		wantPPID int
		wantCmd  string
		wantErr  bool
	}{
		{"simple", 42, "42 (bash) S 1 42 42 0 -1 4194560", 1, "bash", false},
		// comm can contain spaces and nested parens; the parser must split on the
		// LAST ')', not the first, to recover the real command and PPID.
		{"comm with spaces and parens", 100, "100 (my (weird) proc) S 7 1 1 0", 7, "my (weird) proc", false},
		{"no parens is an error", 1, "garbage without parens", 0, "", true},
		{"truncated after comm is an error", 5, "5 (x) S", 0, "", true},
		// A stat ending exactly at the comm's ')' must error, not panic on the
		// raw[close+2:] slice.
		{"ends at comm close paren is an error", 5, "5 (x)", 0, "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseStatSnapshot(tt.pid, []byte(tt.stat))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseStatSnapshot err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if p.PID != tt.pid || p.PPID != tt.wantPPID || p.Command != tt.wantCmd {
				t.Errorf("got {PID:%d PPID:%d Cmd:%q}, want {PID:%d PPID:%d Cmd:%q}",
					p.PID, p.PPID, p.Command, tt.pid, tt.wantPPID, tt.wantCmd)
			}
		})
	}
}
