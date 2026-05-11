package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestIsPublicBind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sockets []model.Socket
		want    bool
	}{
		{
			name: "listening on IPv4 any-address is public",
			sockets: []model.Socket{
				{Address: "0.0.0.0", Port: 443, State: "LISTEN"},
			},
			want: true,
		},
		{
			name: "listening on IPv6 any-address is public",
			sockets: []model.Socket{
				{Address: "::", Port: 443, State: "LISTEN"},
			},
			want: true,
		},
		{
			name: "loopback listener is not public",
			sockets: []model.Socket{
				{Address: "127.0.0.1", Port: 443, State: "LISTEN"},
			},
			want: false,
		},
		{
			name: "specific private address is not public",
			sockets: []model.Socket{
				{Address: "192.168.1.5", Port: 443, State: "LISTEN"},
			},
			want: false,
		},
		{
			name: "ESTABLISHED to public address is NOT a public bind",
			sockets: []model.Socket{
				{Address: "0.0.0.0", Port: 443, State: "ESTABLISHED"},
			},
			want: false,
		},
		{
			name: "outbound connection to 0.0.0.0 should not flag",
			sockets: []model.Socket{
				{Address: "0.0.0.0", Port: 12345, State: "CLOSE_WAIT"},
				{Address: "0.0.0.0", Port: 23456, State: "TIME_WAIT"},
			},
			want: false,
		},
		{
			name: "mix: one public listener wins",
			sockets: []model.Socket{
				{Address: "127.0.0.1", Port: 8080, State: "LISTEN"},
				{Address: "0.0.0.0", Port: 443, State: "LISTEN"},
			},
			want: true,
		},
		{
			name:    "empty",
			sockets: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsPublicBind(tt.sockets); got != tt.want {
				t.Errorf("IsPublicBind(%+v) = %v, want %v", tt.sockets, got, tt.want)
			}
		})
	}
}
