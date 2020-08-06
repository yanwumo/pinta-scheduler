package api

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type JobID types.UID

type JobInfo struct {
	UID       JobID
	Name      string
	Namespace string

	Priority int32

	NumMasters  int32
	NumReplicas int32

	CreationTimestamp metav1.Time

	Job *pintav1.PintaJob
}

func NewJobInfo(uid JobID, job *pintav1.PintaJob) *JobInfo {
	jobInfo := &JobInfo{
		UID:       uid,
		Name:      job.Name,
		Namespace: job.Namespace,

		NumMasters:  job.Spec.NumMasters,
		NumReplicas: job.Spec.NumReplicas,

		CreationTimestamp: job.GetCreationTimestamp(),

		Job: job,
	}
	return jobInfo
}

func (ji *JobInfo) Clone() *JobInfo {
	info := &JobInfo{
		UID:         ji.UID,
		Name:        ji.Name,
		Namespace:   ji.Namespace,
		Priority:    ji.Priority,
		NumMasters:  ji.NumMasters,
		NumReplicas: ji.NumReplicas,
		Job:         ji.Job.DeepCopy(),
	}

	ji.CreationTimestamp.DeepCopyInto(&info.CreationTimestamp)

	return info
}
