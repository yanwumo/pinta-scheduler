package fcfs

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
)

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (fcfs *Policy) Name() string {
	return "fcfs"
}

func (fcfs *Policy) Initialize() {}

func (fcfs *Policy) Execute(snapshot *api.ClusterInfo) {
	// Do nothing
}

func (fcfs *Policy) UnInitialize() {}
