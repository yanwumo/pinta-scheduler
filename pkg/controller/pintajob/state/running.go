package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type runningState struct {
	updater *updater.Updater
}

func (rs *runningState) Name() string {
	return "running"
}

func (rs *runningState) Execute() error {
	vcJobStatus := rs.updater.GetVCJobStatus()

	// Check if the job is completed
	if vcJobStatus == volcanov1alpha1.Completed {
		// No more Volcano Job reconciliation
		// Running -> Completed
		return rs.updater.UpdatePintaJobStatusState(pintav1.Completed)
	}

	err := rs.updater.Reconcile()
	if err != nil {
		return err
	}

	// Check if the job is preempted by scheduler
	pintaJobStatus := rs.updater.GetLastPintaJobStatus()
	if pintaJobStatus.NumMasters == 0 && pintaJobStatus.NumReplicas == 0 {
		// Running -> Preempted
		return rs.updater.UpdatePintaJobStatusState(pintav1.Running)
	}

	return nil
}
