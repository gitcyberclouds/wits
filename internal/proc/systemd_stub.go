//go:build !linux

package proc

import "fmt"

func ResolveSystemdService(port int) (string, error) {
	return "", fmt.Errorf("systemd is only supported on Linux")
}
