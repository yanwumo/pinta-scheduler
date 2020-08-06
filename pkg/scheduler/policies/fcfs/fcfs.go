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
	for _, job := range snapshot.Jobs {
		job.NumMasters = 1
		job.NumReplicas = 2
	}
}

func (fcfs *Policy) UnInitialize() {}
