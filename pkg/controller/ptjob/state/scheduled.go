package state

import "github.com/qed-usc/pinta-scheduler/pkg/controller/api"

type scheduledState struct {
	job *api.JobInfo
}

func (ss *scheduledState) Execute() error {
	panic("implement me")
}
