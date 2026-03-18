//go:build linux || darwin || freebsd || windows

//go:generate go run ../../internal/tools/docgen -format man -out ../../docs/cli
//go:generate go run ../../internal/tools/docgen -format markdown -out ../../docs/cli

package main

import (
	"github.com/pranshuparmar/witr/internal/app"
	"github.com/pranshuparmar/witr/internal/version"
)

// Override version at build time with ldflags:
//   go build -ldflags "-X github.com/pranshuparmar/witr/internal/version.Version=v0.3.0 -X github.com/pranshuparmar/witr/internal/version.Commit=$(git rev-parse --short HEAD) -X 'github.com/pranshuparmar/witr/internal/version.BuildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr

func main() {
	app.SetVersion(version.Version, version.Commit, version.BuildDate)
	app.Execute()
}
