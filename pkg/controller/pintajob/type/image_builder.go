package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type imageBuilder struct {
	job *pintav1.PintaJob
}

func (ib *imageBuilder) BuildVCJob() *volcanov1alpha1.Job {
	return &volcanov1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ib.job.Name,
			Namespace: ib.job.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ib.job, pintav1.SchemeGroupVersion.WithKind("PintaJob")),
			},
		},
		Spec: volcanov1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  1,
			Volumes:       ib.job.Spec.Volumes,
			Tasks: []volcanov1alpha1.TaskSpec{
				{
					Name:     "image-builder",
					Replicas: 1,
					Template: corev1.PodTemplateSpec{
						Spec: ib.job.Spec.Replica.Spec,
					},
				},
			},
		},
	}
}

func (ib *imageBuilder) ReconcileVCJob(vcJob *volcanov1alpha1.Job) (bool, error) {
	if !(len(vcJob.Spec.Tasks) == 1 && vcJob.Spec.Tasks[0].Name == "image-builder") {
		return false, fmt.Errorf("unexpected Volcano Job tasks during reconciliation, job was created incorrectly")
	}

	if vcJob.Spec.Tasks[0].Replicas == 1 {
		return false, nil
	}

	vcJob.Spec.Tasks[0].Replicas = 1
	return true, nil
}
