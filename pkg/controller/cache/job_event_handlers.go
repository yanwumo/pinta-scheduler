package cache

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

func keyFn(ns, name string) string {
	return fmt.Sprintf("%s/%s", ns, name)
}

func JobKeyByName(namespace string, name string) string {
	return keyFn(namespace, name)
}

func JobKeyByReq(req *api.Request) string {
	return keyFn(req.Namespace, req.JobName)
}

func JobKey(job *pintav1.PintaJob) string {
	return keyFn(job.Namespace, job.Name)
}

func VCJobKey(vcjob *volcanov1alpha1.Job) string {
	return keyFn(vcjob.Namespace, vcjob.Name)
}

func jobTerminated(job *api.JobInfo) bool {
	return job.Job == nil
}

func (jc *jobCache) Get(key string) (*api.JobInfo, error) {
	jc.Lock()
	defer jc.Unlock()

	job, found := jc.jobs[key]
	if !found {
		return nil, fmt.Errorf("failed to find job <%s>", key)
	}

	if job.Job == nil {
		return nil, fmt.Errorf("job <%s> is not ready", key)
	}

	return job.Clone(), nil
}

func (jc *jobCache) GetStatus(key string) (*pintav1.PintaJobStatus, error) {
	jc.Lock()
	defer jc.Unlock()

	job, found := jc.jobs[key]
	if !found {
		return nil, fmt.Errorf("failed to find job <%s>", key)
	}

	status := job.Job.Status[0]

	return &status, nil
}

func (jc *jobCache) Add(job *pintav1.PintaJob) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(job)
	if jobInfo, found := jc.jobs[key]; found {
		if jobInfo.Job == nil {
			jobInfo.SetJob(job)
			return nil
		}
		return fmt.Errorf("duplicated jobInfo <%v>", key)
	}

	jc.jobs[key] = api.NewJobInfo(job)

	return nil
}

func (jc *jobCache) Update(obj *pintav1.PintaJob) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(obj)
	job, found := jc.jobs[key]
	if !found {
		return fmt.Errorf("failed to find job <%v>", key)
	}
	job.Job = obj

	return nil
}

func (jc *jobCache) Delete(obj *pintav1.PintaJob) error {
	jc.Lock()
	defer jc.Unlock()

	key := JobKey(obj)
	jobInfo, found := jc.jobs[key]
	if !found {
		return fmt.Errorf("failed to find job <%v>", key)
	}
	jobInfo.Job = nil
	jc.deleteJob(jobInfo)

	return nil
}

func (jc *jobCache) addOrUpdateVCJob(vcjob *volcanov1alpha1.Job) error {
	key := VCJobKey(vcjob)
	job, found := jc.jobs[key]
	if !found {
		job = &api.JobInfo{
			VCJob: vcjob,
		}
		jc.jobs[key] = job
	}

	return job.SetVCJob(vcjob)
}

func (jc *jobCache) AddVCJob(vcjob *volcanov1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	return jc.addOrUpdateVCJob(vcjob)
}

func (jc *jobCache) UpdateVCJob(vcjob *volcanov1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	return jc.addOrUpdateVCJob(vcjob)
}

func (jc *jobCache) DeleteVCJob(vcjob *volcanov1alpha1.Job) error {
	jc.Lock()
	defer jc.Unlock()

	key := VCJobKey(vcjob)
	job, found := jc.jobs[key]
	if found {
		job.VCJob = nil
	}

	return nil
}
