package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	stat, err := os.ReadFile(statPath)
	if err != nil {
		return model.Process{}, err
	}

	// stat format is evil, command is inside ()
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])

	ppid, _ := strconv.Atoi(fields[1])
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)

	startedAt := bootTime().Add(time.Duration(startTicks) * time.Second / ticksPerSecond())

	user := readUser(pid)

	// Working directory
	cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		cwd = "unknown"
	}

	sockets, _ := readListeningSockets()
	inodes := socketsForPID(pid)

	var ports []int
	var addrs []string

	for _, inode := range inodes {
		if s, ok := sockets[inode]; ok {
			ports = append(ports, s.Port)
			addrs = append(addrs, s.Address)
		}
	}

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        comm,
		StartedAt:      startedAt,
		User:           user,
		WorkingDir:     cwd,
		ListeningPorts: ports,
		BindAddresses:  addrs,
	}, nil
}
