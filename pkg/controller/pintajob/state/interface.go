package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
)

type State interface {
	// Name returns the state name
	Name() string
	// Execute executes the actions based on current state.
	Execute() error
}

// NewState gets the state from the volcano job Phase.
func NewState(updater *updater.Updater) State {
	status := updater.GetLastPintaJobStatus()

	switch status.State {
	case pintav1.Idle:
		return &idleState{updater: updater}
	case pintav1.Scheduled:
		return &scheduledState{updater: updater}
	case pintav1.Running:
		return &runningState{updater: updater}
	case pintav1.Preempted:
		return &preemptedState{updater: updater}
	case pintav1.Completed, pintav1.Failed:
		return &finishedState{updater: updater}
	}

	return &emptyState{updater: updater}
}
