//go:build linux

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectLaunchd(_ []model.Process) *model.Source {
  return nil
}
