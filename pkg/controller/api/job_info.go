package api

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

type JobInfo struct {
	Namespace string
	Name      string

	Job   *pintav1.PintaJob
	VCJob *volcanov1alpha1.Job
}

func NewJobInfo(job *pintav1.PintaJob) *JobInfo {
	return &JobInfo{
		Name:      job.Name,
		Namespace: job.Namespace,

		Job: job,
	}
}

func (ji *JobInfo) Clone() *JobInfo {
	job := &JobInfo{
		Namespace: ji.Namespace,
		Name:      ji.Name,

		Job:   ji.Job,
		VCJob: ji.VCJob,
	}

	return job
}

func (ji *JobInfo) SetJob(job *pintav1.PintaJob) {
	ji.Name = job.Name
	ji.Namespace = job.Namespace
	ji.Job = job
}

func (ji *JobInfo) SetVCJob(vcjob *volcanov1alpha1.Job) error {
	ji.VCJob = vcjob
	return nil
}
