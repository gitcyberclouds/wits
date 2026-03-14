package target

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func Resolve(t model.Target, exact bool) ([]int, error) {
	val := strings.TrimSpace(t.Value)

	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid pid")
		}
		if pid <= 0 {
			return nil, fmt.Errorf("invalid pid: must be a positive integer")
		}
		return []int{pid}, nil

	case model.TargetPort:
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid port")
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid port: must be between 1 and 65535")
		}
		return ResolvePort(port)

	case model.TargetName:
		return ResolveName(val, exact)

	case model.TargetFile:
		return ResolveFile(val)

	default:
		return nil, fmt.Errorf("unknown target")
	}
}
