package state

import "github.com/qed-usc/pinta-scheduler/pkg/controller/api"

type runningState struct {
	job *api.JobInfo
}

func (rs *runningState) Execute() error {
	panic("implement me")
}
