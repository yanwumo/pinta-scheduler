package hell

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
)

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (hell *Policy) Name() string {
	return "hell"
}

func (hell *Policy) Initialize() {}

func (hell *Policy) Execute(snapshot *api.ClusterInfo) {
	// Do nothing
}

func (hell *Policy) UnInitialize() {}
