package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
	"k8s.io/klog"
)

type idleState struct {
	updater *updater.Updater
}

func (is *idleState) Name() string {
	return "idle"
}

func (is *idleState) Execute() error {
	// Check if the job is scheduled by scheduler
	// If not, stay at idle state
	pintaJobStatus := is.updater.GetLastPintaJobStatus()
	if pintaJobStatus.NumMasters == 0 && pintaJobStatus.NumReplicas == 0 {
		return nil
	}

	err := is.updater.Reconcile()
	if err != nil {
		return err
	}

	return is.updater.UpdatePintaJobStatusState(pintav1.Scheduled)
}
