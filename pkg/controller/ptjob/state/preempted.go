package state

import "github.com/qed-usc/pinta-scheduler/pkg/controller/api"

type preemptedState struct {
	job *api.JobInfo
}

func (ps *preemptedState) Execute() error {
	panic("implement me")
}
