package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type psWorker struct {
	job *pintav1.PintaJob
}

func (pw *psWorker) BuildVCJob() *volcanov1alpha1.Job {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pw.job.Status) > 0 {
		lastPintaJobStatus = pw.job.Status[0]
	}

	return &volcanov1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pw.job.Name,
			Namespace: pw.job.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pw.job, pintav1.SchemeGroupVersion.WithKind("PintaJob")),
			},
		},
		Spec: volcanov1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  lastPintaJobStatus.NumMasters + lastPintaJobStatus.NumReplicas,
			Volumes:       pw.job.Spec.Volumes,
			Tasks: []volcanov1alpha1.TaskSpec{
				{
					Name:     "ps",
					Replicas: lastPintaJobStatus.NumMasters,
					Template: corev1.PodTemplateSpec{
						Spec: pw.job.Spec.Master.Spec,
					},
					Policies: nil,
				},
				{
					Name:     "worker",
					Replicas: lastPintaJobStatus.NumReplicas,
					Template: corev1.PodTemplateSpec{
						Spec: pw.job.Spec.Replica.Spec,
					},
					Policies: []volcanov1alpha1.LifecyclePolicy{
						{
							Event:  "TaskCompleted",
							Action: "CompleteJob",
						},
					},
				},
			},
			Plugins: map[string][]string{
				"env": {},
				"svc": {},
			},
		},
	}
}

func (pw *psWorker) ReconcileVCJob(vcJob *volcanov1alpha1.Job) (bool, error) {
	if !(len(vcJob.Spec.Tasks) == 2 && vcJob.Spec.Tasks[0].Name == "ps" && vcJob.Spec.Tasks[1].Name == "worker") {
		return false, fmt.Errorf("unexpected Volcano Job tasks during reconciliation, job was created incorrectly")
	}

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pw.job.Status) > 0 {
		lastPintaJobStatus = pw.job.Status[0]
	}

	if vcJob.Spec.Tasks[0].Replicas == lastPintaJobStatus.NumMasters && vcJob.Spec.Tasks[1].Replicas == lastPintaJobStatus.NumReplicas {
		return false, nil
	}

	vcJob.Spec.Tasks[0].Replicas = lastPintaJobStatus.NumMasters
	vcJob.Spec.Tasks[1].Replicas = lastPintaJobStatus.NumReplicas
	return true, nil
}
