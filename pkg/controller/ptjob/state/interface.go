package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
)

type State interface {
	// Execute executes the actions based on current state.
	Execute() error
}

// NewState gets the state from the volcano job Phase.
func NewState(jobInfo *api.JobInfo) State {
	job := jobInfo.Job

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(job.Status) > 0 {
		lastPintaJobStatus = job.Status[0]
	}

	switch lastPintaJobStatus.State {
	case pintav1.Idle:
		return &idleState{job: jobInfo}
	case pintav1.Scheduled:
		return &scheduledState{job: jobInfo}
	case pintav1.Running:
		return &runningState{job: jobInfo}
	case pintav1.Preempted:
		return &preemptedState{job: jobInfo}
	case pintav1.Completed, pintav1.Failed:
		return &finishedState{job: jobInfo}
	}

	// Idle state by default
	return &idleState{job: jobInfo}
}
