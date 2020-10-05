package state

import "github.com/qed-usc/pinta-scheduler/pkg/controller/api"

type finishedState struct {
	job *api.JobInfo
}

func (fs *finishedState) Execute() error {
	panic("implement me")
}
