package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	controllercache "github.com/qed-usc/pinta-scheduler/pkg/controller/cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type symmetric struct {
	cache controllercache.Cache
	job   *pintav1.PintaJob
}

func (s *symmetric) BuildVCJob() (*volcanov1alpha1.Job, error) {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(s.job.Status) > 0 {
		lastPintaJobStatus = s.job.Status[0]
	}

	replicaSpec := volcanov1alpha1.TaskSpec{
		Name:     "replica",
		Replicas: lastPintaJobStatus.NumReplicas,
		Template: corev1.PodTemplateSpec{
			Spec: *s.job.Spec.Replica.Spec.DeepCopy(), // we are patching this below
		},
		Policies: []volcanov1alpha1.LifecyclePolicy{
			{
				Event:  "TaskCompleted",
				Action: "CompleteJob",
			},
		},
	}
	err := patchPodSpecWithRoleSpec(&replicaSpec.Template.Spec, &s.job.Spec.Replica, s.cache.TranslateResources)
	if err != nil {
		return nil, err
	}

	return &volcanov1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.job.Name,
			Namespace: s.job.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(s.job, pintav1.SchemeGroupVersion.WithKind("PintaJob")),
			},
		},
		Spec: volcanov1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  lastPintaJobStatus.NumReplicas,
			Volumes:       s.job.Spec.Volumes,
			Tasks:         []volcanov1alpha1.TaskSpec{replicaSpec},
			Plugins: map[string][]string{
				"env": {},
				"svc": {},
			},
		},
	}, nil
}

func (s *symmetric) ReconcileVCJob(vcJob *volcanov1alpha1.Job) (bool, error) {
	if !(len(vcJob.Spec.Tasks) == 1 && vcJob.Spec.Tasks[0].Name == "replica") {
		return false, fmt.Errorf("unexpected Volcano Job tasks during reconciliation, job was created incorrectly")
	}

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(s.job.Status) > 0 {
		lastPintaJobStatus = s.job.Status[0]
	}

	if vcJob.Spec.Tasks[0].Replicas == lastPintaJobStatus.NumReplicas {
		return false, nil
	}

	vcJob.Spec.Tasks[0].Replicas = lastPintaJobStatus.NumReplicas
	return true, nil
}
