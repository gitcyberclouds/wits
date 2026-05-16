//go:build freebsd

package proc

import (
	"strconv"
	"sync"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// Cached results of ListOpenPorts to avoid re-invoking sockstat for every
// process visited during an ancestry walk.
var (
	openPortsCache     []model.OpenPort
	openPortsCacheTime time.Time
	openPortsCacheMu   sync.Mutex
	openPortsCacheTTL  = 2 * time.Second
)

func listOpenPortsCached() []model.OpenPort {
	openPortsCacheMu.Lock()
	defer openPortsCacheMu.Unlock()

	if openPortsCache != nil && time.Since(openPortsCacheTime) < openPortsCacheTTL {
		return openPortsCache
	}
	ports, err := ListOpenPorts()
	if err != nil {
		return nil
	}
	openPortsCache = ports
	openPortsCacheTime = time.Now()
	return ports
}

// socketsForPID returns every IP socket owned by a PID, including non-listening
// sockets. Backed by `sockstat` via ListOpenPorts.
func socketsForPID(pid int) []model.Socket {
	all := listOpenPortsCached()
	var sockets []model.Socket
	seen := make(map[string]bool)
	for _, p := range all {
		if p.PID != pid {
			continue
		}
		key := p.Protocol + "|" + p.Address + "|" + strconv.Itoa(p.Port) + "|" + p.State
		if seen[key] {
			continue
		}
		seen[key] = true
		sockets = append(sockets, model.Socket{
			Port:     p.Port,
			Address:  p.Address,
			Protocol: p.Protocol,
			State:    p.State,
		})
	}
	return sockets
}
