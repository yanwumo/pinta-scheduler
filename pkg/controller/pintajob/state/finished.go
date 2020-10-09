package state

import (
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
)

type finishedState struct {
	updater *updater.Updater
}

func (fs *finishedState) Name() string {
	return "finished"
}

func (fs *finishedState) Execute() error {
	// Do nothing
	return nil
}
