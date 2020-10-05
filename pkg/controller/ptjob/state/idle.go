package state

import "github.com/qed-usc/pinta-scheduler/pkg/controller/api"

type idleState struct {
	job *api.JobInfo
}

func (is *idleState) Execute() error {
	panic("implement me")
}
