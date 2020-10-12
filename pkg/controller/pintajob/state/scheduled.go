package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type scheduledState struct {
	updater *updater.Updater
}

func (ss *scheduledState) Name() string {
	return "scheduled"
}

func (ss *scheduledState) Execute() error {
	status := ss.updater.GetVCJobStatus()

	// Check if the job is completed
	if status == volcanov1alpha1.Completed {
		// No more Volcano Job reconciliation
		// Scheduled -> Completed
		return ss.updater.UpdatePintaJobStatusState(pintav1.Completed)
	}

	err := ss.updater.Reconcile()
	if err != nil {
		return err
	}

	// Check if the job starts to run
	// If not, stay at scheduled state
	if status != volcanov1alpha1.Running {
		return nil
	}

	// Scheduled -> Running
	return ss.updater.UpdatePintaJobStatusState(pintav1.Running)
}
