package source

import "github.com/pranshuparmar/witr/pkg/model"

// IsPublicBind reports whether any listening socket is bound to a public
// (any-address) interface. Non-listening sockets are ignored — an established
// outbound connection to a public address is not the same as exposing one.
func IsPublicBind(sockets []model.Socket) bool {
	for _, s := range sockets {
		if s.State != "LISTEN" {
			continue
		}
		if s.Address == "0.0.0.0" || s.Address == "::" {
			return true
		}
	}
	return false
}
