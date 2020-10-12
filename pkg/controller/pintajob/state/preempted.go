package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type preemptedState struct {
	updater *updater.Updater
}

func (ps *preemptedState) Name() string {
	return "preempted"
}

func (ps *preemptedState) Execute() error {
	vcJobStatus := ps.updater.GetVCJobStatus()

	// Check if the job is completed
	if vcJobStatus == volcanov1alpha1.Completed {
		// No more Volcano Job reconciliation
		// Preempted -> Completed
		return ps.updater.UpdatePintaJobStatusState(pintav1.Completed)
	}

	// Check if the job is resumed by scheduler
	// If not, stay at preempted state
	pintaJobStatus := ps.updater.GetLastPintaJobStatus()
	if pintaJobStatus.NumMasters == 0 && pintaJobStatus.NumReplicas == 0 {
		return nil
	}

	err := ps.updater.Reconcile()
	if err != nil {
		return err
	}

	return ps.updater.UpdatePintaJobStatusState(pintav1.Running)
}
