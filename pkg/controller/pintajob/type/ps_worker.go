package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	controllercache "github.com/qed-usc/pinta-scheduler/pkg/controller/cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type psWorker struct {
	cache controllercache.Cache
	job   *pintav1.PintaJob
}

func (pw *psWorker) BuildVCJob() (*volcanov1alpha1.Job, error) {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pw.job.Status) > 0 {
		lastPintaJobStatus = pw.job.Status[0]
	}

	masterSpec := volcanov1alpha1.TaskSpec{
		Name:     "ps",
		Replicas: lastPintaJobStatus.NumMasters,
		Template: corev1.PodTemplateSpec{
			Spec: *pw.job.Spec.Master.Spec.DeepCopy(), // we are patching this below
		},
		Policies: nil,
	}
	err := patchPodSpecWithRoleSpec(&masterSpec.Template.Spec, &pw.job.Spec.Master, pw.cache.TranslateResources)
	if err != nil {
		return nil, err
	}

	replicaSpec := volcanov1alpha1.TaskSpec{
		Name:     "worker",
		Replicas: lastPintaJobStatus.NumReplicas,
		Template: corev1.PodTemplateSpec{
			Spec: *pw.job.Spec.Replica.Spec.DeepCopy(), // we are patching this below
		},
		Policies: []volcanov1alpha1.LifecyclePolicy{
			{
				Event:  "TaskCompleted",
				Action: "CompleteJob",
			},
		},
	}
	err = patchPodSpecWithRoleSpec(&replicaSpec.Template.Spec, &pw.job.Spec.Replica, pw.cache.TranslateResources)
	if err != nil {
		return nil, err
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
			Tasks:         []volcanov1alpha1.TaskSpec{masterSpec, replicaSpec},
			Plugins: map[string][]string{
				"env": {},
				"svc": {},
			},
		},
	}, nil
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
