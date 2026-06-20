package proc

import "github.com/pranshuparmar/witr/pkg/model"

func init() { registerRuntime(incusRuntime{}) }

type incusRuntime struct{}

func (incusRuntime) Name() string                       { return "incus" }
func (incusRuntime) Available() bool                    { return binAvailable("incus") }
func (incusRuntime) List() []*model.ContainerMatch      { return lxdLikeList("incus", "incus") }
func (incusRuntime) HostPID(id string) int              { return lxdLikeHostPID("incus", id) }
func (incusRuntime) Enrich(match *model.ContainerMatch) { lxdLikeEnrich("incus", match) }
