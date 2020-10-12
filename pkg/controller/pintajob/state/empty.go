package state

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
)

type emptyState struct {
	updater *updater.Updater
}

func (es *emptyState) Name() string {
	return "empty"
}

func (es *emptyState) Execute() error {
	return es.updater.UpdatePintaJobStatusState(pintav1.Idle)
}
