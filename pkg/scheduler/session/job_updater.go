package session

import (
	"context"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const (
	jobUpdaterWorker = 16
)

type jobUpdater struct {
	ssn      *Session
	jobQueue []*info.JobInfo
}

func newJobUpdater(ssn *Session) *jobUpdater {
	queue := make([]*info.JobInfo, 0, len(ssn.Jobs))
	for _, jobInfo := range ssn.Jobs {
		queue = append(queue, jobInfo)
	}

	ju := &jobUpdater{
		ssn:      ssn,
		jobQueue: queue,
	}
	return ju
}

func (ju *jobUpdater) UpdateAll() {
	workqueue.ParallelizeUntil(context.TODO(), jobUpdaterWorker, len(ju.jobQueue), ju.updateJob)
}

func (ju *jobUpdater) updateJob(index int) {
	var err error
	jobInfo := ju.jobQueue[index]
	job := jobInfo.Job
	pinta := ju.ssn.cache.PintaClient().PintaV1()

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(job.Status) > 0 {
		lastPintaJobStatus = job.Status[0]
	}
	// Update job status
	// Ignore jobs without changes
	if jobInfo.NumMasters == lastPintaJobStatus.NumMasters && jobInfo.NumReplicas == lastPintaJobStatus.NumReplicas {
		return
	}

	job.Status = append([]pintav1.PintaJobStatus{
		{
			State:              lastPintaJobStatus.State,
			LastTransitionTime: metav1.Now(),
			NumMasters:         jobInfo.NumMasters,
			NumReplicas:        jobInfo.NumReplicas,
		},
	}, job.Status...)

	_, err = pinta.PintaJobs(jobInfo.Namespace).UpdateStatus(context.TODO(), job, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Commit failed when updating job status: %v", err)
	}
}
