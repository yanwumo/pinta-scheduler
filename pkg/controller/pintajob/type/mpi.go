package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	controllercache "github.com/qed-usc/pinta-scheduler/pkg/controller/cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type mpi struct {
	cache controllercache.Cache
	job   *pintav1.PintaJob
}

func (m *mpi) BuildVCJob() (*volcanov1alpha1.Job, error) {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(m.job.Status) > 0 {
		lastPintaJobStatus = m.job.Status[0]
	}

	masterSpec := volcanov1alpha1.TaskSpec{
		Name:     "master",
		Replicas: lastPintaJobStatus.NumMasters,
		Template: corev1.PodTemplateSpec{
			Spec: *m.job.Spec.Master.Spec.DeepCopy(), // we are patching this below
		},
		Policies: []volcanov1alpha1.LifecyclePolicy{
			{
				Event:  "TaskCompleted",
				Action: "CompleteJob",
			},
		},
	}
	err := patchPodSpecWithRoleSpec(&masterSpec.Template.Spec, &m.job.Spec.Master, m.cache.TranslateResources)
	if err != nil {
		return nil, err
	}

	replicaSpec := volcanov1alpha1.TaskSpec{
		Name:     "replica",
		Replicas: lastPintaJobStatus.NumReplicas,
		Template: corev1.PodTemplateSpec{
			Spec: *m.job.Spec.Replica.Spec.DeepCopy(), // we are patching this below
		},
	}
	err = patchPodSpecWithRoleSpec(&replicaSpec.Template.Spec, &m.job.Spec.Replica, m.cache.TranslateResources)
	if err != nil {
		return nil, err
	}

	return &volcanov1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.job.Name,
			Namespace: m.job.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(m.job, pintav1.SchemeGroupVersion.WithKind("PintaJob")),
			},
		},
		Spec: volcanov1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  lastPintaJobStatus.NumMasters + lastPintaJobStatus.NumReplicas,
			Volumes:       m.job.Spec.Volumes,
			Tasks:         []volcanov1alpha1.TaskSpec{masterSpec, replicaSpec},
			Plugins: map[string][]string{
				"env": {},
				"svc": {},
				"ssh": {},
			},
		},
	}, nil
}

func (m *mpi) ReconcileVCJob(vcJob *volcanov1alpha1.Job) (bool, error) {
	if !(len(vcJob.Spec.Tasks) == 2 && vcJob.Spec.Tasks[0].Name == "master" && vcJob.Spec.Tasks[1].Name == "replica") {
		return false, fmt.Errorf("unexpected Volcano Job tasks during reconciliation, job was created incorrectly")
	}

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(m.job.Status) > 0 {
		lastPintaJobStatus = m.job.Status[0]
	}

	if vcJob.Spec.Tasks[0].Replicas == lastPintaJobStatus.NumMasters && vcJob.Spec.Tasks[1].Replicas == lastPintaJobStatus.NumReplicas {
		return false, nil
	}

	vcJob.Spec.Tasks[0].Replicas = lastPintaJobStatus.NumMasters
	vcJob.Spec.Tasks[1].Replicas = lastPintaJobStatus.NumReplicas
	return true, nil
}
