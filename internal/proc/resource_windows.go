//go:build windows

package proc

import "github.com/pranshuparmar/witr/pkg/model"

func GetResourceContext(pid int) *model.ResourceContext {
	return nil
}
