//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

// readListeningSockets returns a map of pseudo-inodes to sockets
// On FreeBSD, we use sockstat to get listening sockets
// We use a combination of PID:port as the "inode" since FreeBSD doesn't expose inodes like Linux
func readListeningSockets() (map[string]Socket, error) {
	sockets := make(map[string]Socket)

	// Use sockstat to get listening TCP sockets
	// -4 = IPv4, -6 = IPv6, -l = listening, -P tcp = TCP protocol
	// Run for both IPv4 and IPv6
	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag, "-l", "-P", "tcp").Output()
		if err != nil {
			continue
		}

		parseSockstatOutput(string(out), sockets)
	}

	if len(sockets) == 0 {
		// Try netstat as fallback
		return readListeningSocketsNetstat()
	}

	return sockets, nil
}

func parseSockstatOutput(output string, sockets map[string]Socket) {
	// sockstat output format:
	// USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS
	// root     nginx      1234  6  tcp4   *:80                  *:*

	for line := range strings.Lines(output) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Skip header
		if fields[0] == "USER" {
			continue
		}

		pid := fields[2]
		localAddr := fields[5]

		// Parse local address
		address, port := parseSockstatAddr(localAddr)
		if port > 0 {
			// Use PID:port as pseudo-inode
			inode := pid + ":" + strconv.Itoa(port)
			sockets[inode] = Socket{
				Inode:   inode,
				Port:    port,
				Address: address,
			}
		}
	}
}

func readListeningSocketsNetstat() (map[string]Socket, error) {
	sockets := make(map[string]Socket)

	// Use netstat as fallback
	// netstat -an -p tcp shows all TCP connections
	out, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return sockets, nil
	}

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local address is typically field 3 (0-indexed)
		localAddr := fields[3]
		address, port := parseSockstatAddr(localAddr)
		if port > 0 {
			// Generate a unique key
			inode := "netstat:" + localAddr
			sockets[inode] = Socket{
				Inode:   inode,
				Port:    port,
				Address: address,
			}
		}
	}

	return sockets, nil
}

// parseSockstatAddr parses addresses like "*:80", "127.0.0.1:8080", "[::1]:8080"
func parseSockstatAddr(addr string) (string, int) {
	// Handle IPv6 format [::]:port or [::1]:port
	if strings.HasPrefix(addr, "[") {
		bracketEnd := strings.LastIndex(addr, "]")
		if bracketEnd == -1 {
			return "", 0
		}
		ip := addr[1:bracketEnd]
		rest := addr[bracketEnd+1:]
		// rest should be ":port"
		if len(rest) > 1 && rest[0] == ':' {
			port, err := strconv.Atoi(rest[1:])
			if err == nil {
				if ip == "::" || ip == "" {
					return "::", port
				}
				return ip, port
			}
		}
		return "", 0
	}

	// Handle wildcard format "*:port"
	if strings.HasPrefix(addr, "*:") {
		port, err := strconv.Atoi(addr[2:])
		if err == nil {
			return "0.0.0.0", port
		}
		return "", 0
	}

	// Handle IPv4 format "127.0.0.1:8080"
	// FreeBSD sockstat uses colon as separator
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ip := addr[:idx]
		portStr := addr[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			if ip == "*" {
				return "0.0.0.0", port
			}
			return ip, port
		}
	}

	// Handle dot-separated format (some FreeBSD versions)
	// "127.0.0.1.8080"
	if idx := strings.LastIndex(addr, "."); idx != -1 {
		portStr := addr[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			ip := addr[:idx]
			return ip, port
		}
	}

	return "", 0
}
