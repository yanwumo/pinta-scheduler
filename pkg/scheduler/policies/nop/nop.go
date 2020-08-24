package nop

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
)

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (nop *Policy) Name() string {
	return "nop"
}

func (nop *Policy) Initialize() {}

func (nop *Policy) Execute(snapshot *api.ClusterInfo) {
	// Job spec fallthrough
	for _, job := range snapshot.Jobs {
		job.NumMasters = job.PresetNumMasters
		job.NumReplicas = job.PresetNumReplicas
	}
}

func (nop *Policy) UnInitialize() {}
