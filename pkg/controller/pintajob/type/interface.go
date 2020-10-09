package _type

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type Type interface {
	// BuildVCJob creates Volcano Job spec from scratch to match PintaJob spec.
	BuildVCJob() *volcanov1alpha1.Job
	// ReconcileVCJob modifies Volcano Job spec to match PintaJob spec.
	// It modifies the vcJob in place, and returns whether vcJob has been changed.
	ReconcileVCJob(vcJob *volcanov1alpha1.Job) (bool, error)
}

func NewType(job *pintav1.PintaJob) Type {
	switch job.Spec.Type {
	case pintav1.Symmetric:
		return &symmetric{job: job}
	case pintav1.MPI:
		return &mpi{job: job}
	case pintav1.PSWorker:
		return &psWorker{job: job}
	case pintav1.ImageBuilder:
		return &imageBuilder{job: job}
	}

	return nil
}
