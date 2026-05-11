package output

import (
	"reflect"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDisplayState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"LISTEN", "LISTENING"},
		{"ESTABLISHED", "ESTABLISHED"},
		{"CLOSE_WAIT", "CLOSE_WAIT"},
		{"OPEN", "OPEN"},
		{"", "?"},
	}

	for _, tt := range tests {
		if got := displayState(tt.in); got != tt.want {
			t.Errorf("displayState(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSocketSortRank(t *testing.T) {
	t.Parallel()

	// LISTEN must always rank above ESTABLISHED, and both above unknown.
	// OPEN sits between LISTEN and ESTABLISHED.
	if socketSortRank("LISTEN") >= socketSortRank("ESTABLISHED") {
		t.Errorf("LISTEN should rank below ESTABLISHED")
	}
	if socketSortRank("OPEN") <= socketSortRank("LISTEN") {
		t.Errorf("OPEN should rank above LISTEN")
	}
	if socketSortRank("ESTABLISHED") >= socketSortRank("TIME_WAIT") {
		t.Errorf("ESTABLISHED should rank above unknown states")
	}
	if socketSortRank("LISTEN") != 0 {
		t.Errorf("LISTEN should be rank 0 so it appears first")
	}
}

func TestFormatSocket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    model.Socket
		want string
	}{
		{
			name: "tcp listener",
			s:    model.Socket{Address: "0.0.0.0", Port: 443, Protocol: "TCP", State: "LISTEN"},
			want: "0.0.0.0:443 (TCP | LISTENING)",
		},
		{
			name: "established",
			s:    model.Socket{Address: "127.0.0.1", Port: 43525, Protocol: "TCP", State: "ESTABLISHED"},
			want: "127.0.0.1:43525 (TCP | ESTABLISHED)",
		},
		{
			name: "ipv6 wraps host:port via JoinHostPort",
			s:    model.Socket{Address: "::1", Port: 8080, Protocol: "TCP6", State: "LISTEN"},
			want: "[::1]:8080 (TCP6 | LISTENING)",
		},
		{
			name: "udp open",
			s:    model.Socket{Address: "0.0.0.0", Port: 53, Protocol: "UDP", State: "OPEN"},
			want: "0.0.0.0:53 (UDP | OPEN)",
		},
		{
			name: "missing protocol falls back to ?",
			s:    model.Socket{Address: "127.0.0.1", Port: 9999, Protocol: "", State: "LISTEN"},
			want: "127.0.0.1:9999 (? | LISTENING)",
		},
		{
			name: "blank state renders as ?",
			s:    model.Socket{Address: "127.0.0.1", Port: 9999, Protocol: "TCP", State: ""},
			want: "127.0.0.1:9999 (TCP | ?)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatSocket(tt.s); got != tt.want {
				t.Errorf("formatSocket(%+v) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestVisibleSocketsDropsIncomplete(t *testing.T) {
	t.Parallel()

	in := []model.Socket{
		{Address: "127.0.0.1", Port: 8080, State: "LISTEN"},
		{Address: "", Port: 80, State: "LISTEN"},         // missing address
		{Address: "127.0.0.1", Port: 0, State: "LISTEN"}, // missing port
		{Address: "0.0.0.0", Port: 443, State: "LISTEN"},
	}
	got := visibleSockets(in)
	if len(got) != 2 {
		t.Fatalf("visibleSockets dropped wrong number: got %d, want 2", len(got))
	}
	if got[0].Port != 8080 || got[1].Port != 443 {
		t.Errorf("visibleSockets unexpected order/content: %+v", got)
	}
}

func TestSortSockets(t *testing.T) {
	t.Parallel()

	// Scenario lifted from the real bug report: address grouping with a
	// LISTEN that must sit above its ESTABLISHED sibling on the same
	// address:port, and ports ascending within each address.
	in := []model.Socket{
		{Address: "192.168.176.122", Port: 58236, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36146, State: "ESTABLISHED"},
		{Address: "127.0.0.1", Port: 43525, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36170, State: "ESTABLISHED"},
		{Address: "127.0.0.1", Port: 43525, State: "LISTEN"},
		{Address: "192.168.176.122", Port: 36162, State: "ESTABLISHED"},
	}
	want := []model.Socket{
		{Address: "127.0.0.1", Port: 43525, State: "LISTEN"},
		{Address: "127.0.0.1", Port: 43525, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36146, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36162, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36170, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 58236, State: "ESTABLISHED"},
	}

	sortSockets(in)
	if !reflect.DeepEqual(in, want) {
		t.Errorf("sortSockets order mismatch.\n got: %+v\nwant: %+v", in, want)
	}
}

func TestSortSocketsListenBeforeEstablishedSamePort(t *testing.T) {
	t.Parallel()

	// Insertion order is ESTABLISHED first, but the sort must surface
	// LISTEN above it.
	in := []model.Socket{
		{Address: "10.0.0.1", Port: 80, State: "ESTABLISHED"},
		{Address: "10.0.0.1", Port: 80, State: "LISTEN"},
	}
	sortSockets(in)
	if in[0].State != "LISTEN" {
		t.Errorf("LISTEN must sort above ESTABLISHED on same address:port; got %+v", in)
	}
}

func TestSortSocketsStableWithinSameRank(t *testing.T) {
	t.Parallel()

	// Two ESTABLISHED sockets on different ports of the same address should
	// be ordered by port. Stability is asserted via the deterministic input
	// ordering: lower port first regardless of input position.
	in := []model.Socket{
		{Address: "10.0.0.1", Port: 9000, State: "ESTABLISHED"},
		{Address: "10.0.0.1", Port: 8000, State: "ESTABLISHED"},
	}
	sortSockets(in)
	if in[0].Port != 8000 {
		t.Errorf("expected lower port first within same address; got %+v", in)
	}
}
