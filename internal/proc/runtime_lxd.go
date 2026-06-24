package proc

import "github.com/pranshuparmar/witr/pkg/model"

func init() { registerRuntime(lxdRuntime{}) }

type lxdRuntime struct{}

func (lxdRuntime) Name() string { return "lxd" }

// LXD's client binary is `lxc`, which collides with classic LXC's CLI prefix.
// We require both `lxc` and the `lxd` daemon binary to be present so we don't
// accidentally call into classic LXC's tooling (which doesn't take a `list`
// subcommand) on a system that has lxc-* installed but no LXD.
func (lxdRuntime) Available() bool                    { return binAvailable("lxc") && binAvailable("lxd") }
func (lxdRuntime) List() []*model.ContainerMatch      { return lxdLikeList("lxc", "lxd") }
func (lxdRuntime) HostPID(id string) int              { return lxdLikeHostPID("lxc", id) }
func (lxdRuntime) Enrich(match *model.ContainerMatch) { lxdLikeEnrich("lxc", match) }
