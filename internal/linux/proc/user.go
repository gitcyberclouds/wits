package proc

import (
	"os"
	"strconv"
	"syscall"
)

func readUser(pid int) string {
	path := "/proc/" + strconv.Itoa(pid)

	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "unknown"
	}

	return strconv.Itoa(int(stat.Uid))
}
