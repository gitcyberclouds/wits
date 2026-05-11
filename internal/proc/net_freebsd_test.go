//go:build freebsd

package proc

import "testing"

func TestParseSockstatAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		proto    string
		wantAddr string
		wantPort int
	}{
		// IPv4 happy paths.
		{name: "ipv4 specific", raw: "127.0.0.1:8080", proto: "tcp4", wantAddr: "127.0.0.1", wantPort: 8080},
		{name: "ipv4 wildcard colon", raw: "*:443", proto: "tcp4", wantAddr: "0.0.0.0", wantPort: 443},
		{name: "ipv4 wildcard via '*' ip", raw: "*:53", proto: "udp4", wantAddr: "0.0.0.0", wantPort: 53},

		// IPv6 happy paths — protocol disambiguates wildcards.
		{name: "ipv6 wildcard via tcp6 proto", raw: "*:443", proto: "tcp6", wantAddr: "::", wantPort: 443},
		{name: "ipv6 bracketed loopback", raw: "[::1]:8080", proto: "tcp6", wantAddr: "::1", wantPort: 8080},
		{name: "ipv6 bracketed link-local", raw: "[fe80::1]:22", proto: "tcp6", wantAddr: "fe80::1", wantPort: 22},

		// Dot-separated fallback (older FreeBSD output).
		{name: "ipv4 dot-separated", raw: "127.0.0.1.8080", proto: "tcp4", wantAddr: "127.0.0.1", wantPort: 8080},

		// Garbage / malformed.
		{name: "empty", raw: "", proto: "tcp4", wantAddr: "", wantPort: 0},
		{name: "unterminated bracket", raw: "[::1:8080", proto: "tcp6", wantAddr: "", wantPort: 0},
		{name: "ipv6 missing port separator", raw: "[::1]8080", proto: "tcp6", wantAddr: "", wantPort: 0},
		{name: "non-numeric port", raw: "127.0.0.1:abc", proto: "tcp4", wantAddr: "", wantPort: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotAddr, gotPort := parseSockstatAddr(tt.raw, tt.proto)
			if gotAddr != tt.wantAddr || gotPort != tt.wantPort {
				t.Errorf("parseSockstatAddr(%q, %q) = (%q, %d), want (%q, %d)",
					tt.raw, tt.proto, gotAddr, gotPort, tt.wantAddr, tt.wantPort)
			}
		})
	}
}
