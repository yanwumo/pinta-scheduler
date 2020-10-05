package api

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
)

type JobInfo struct {
	Namespace string
	Name      string

	Job *pintav1.PintaJob
	//Pods map[string]map[string]*v1.Pod
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
		Job:       ji.Job,

		//Pods: make(map[string]map[string]*v1.Pod),
	}

	//for key, pods := range ji.Pods {
	//	job.Pods[key] = make(map[string]*v1.Pod)
	//	for pn, pod := range pods {
	//		job.Pods[key][pn] = pod
	//	}
	//}

	return job
}

func (ji *JobInfo) SetJob(job *pintav1.PintaJob) {
	ji.Name = job.Name
	ji.Namespace = job.Namespace
	ji.Job = job
}
