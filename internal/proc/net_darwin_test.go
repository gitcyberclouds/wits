//go:build darwin

package proc

import "testing"

func TestParseNetstatAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		wantAddr string
		wantPort int
	}{
		// macOS lsof / netstat use dot-separated format by default.
		{name: "ipv4 dot-separated", raw: "127.0.0.1.8080", wantAddr: "127.0.0.1", wantPort: 8080},
		{name: "ipv4 colon-separated", raw: "127.0.0.1:8080", wantAddr: "127.0.0.1", wantPort: 8080},

		// Wildcard means "all interfaces" → 0.0.0.0.
		{name: "wildcard dot", raw: "*.443", wantAddr: "0.0.0.0", wantPort: 443},
		{name: "wildcard colon", raw: "*:443", wantAddr: "0.0.0.0", wantPort: 443},

		// IPv6 forms.
		{name: "ipv6 any-address bracketed dot", raw: "[::].443", wantAddr: "::", wantPort: 443},
		{name: "ipv6 any-address bracketed colon", raw: "[::]:443", wantAddr: "::", wantPort: 443},
		{name: "ipv6 loopback bracketed", raw: "[::1].8080", wantAddr: "::1", wantPort: 8080},
		{name: "ipv6 specific bracketed", raw: "[fe80::1].22", wantAddr: "fe80::1", wantPort: 22},

		// Garbage / malformed.
		{name: "empty", raw: "", wantAddr: "", wantPort: 0},
		{name: "bare star", raw: "*", wantAddr: "", wantPort: 0},
		{name: "unterminated bracket", raw: "[::1.8080", wantAddr: "", wantPort: 0},
		{name: "non-numeric port", raw: "127.0.0.1.abc", wantAddr: "", wantPort: 0},
		{name: "ipv6 with non-numeric port", raw: "[::1].abc", wantAddr: "", wantPort: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotAddr, gotPort := parseNetstatAddr(tt.raw)
			if gotAddr != tt.wantAddr || gotPort != tt.wantPort {
				t.Errorf("parseNetstatAddr(%q) = (%q, %d), want (%q, %d)",
					tt.raw, gotAddr, gotPort, tt.wantAddr, tt.wantPort)
			}
		})
	}
}
