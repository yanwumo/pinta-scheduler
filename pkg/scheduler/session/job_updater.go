package session

import (
	"context"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const (
	jobUpdaterWorker = 16
)

type jobUpdater struct {
	ssn      *Session
	jobQueue []*api.JobInfo
}

func newJobUpdater(ssn *Session) *jobUpdater {
	queue := make([]*api.JobInfo, 0, len(ssn.Jobs))
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

	// Update job spec
	if job.Spec.Master.Spec.NodeSelector == nil || job.Spec.Replica.Spec.NodeSelector == nil {
		job.Spec.Master.Spec.NodeSelector = map[string]string{}
		job.Spec.Replica.Spec.NodeSelector = map[string]string{}
		ju.updateRoleSpec(&job.Spec.Master)
		ju.updateRoleSpec(&job.Spec.Replica)

		job, err = pinta.PintaJobs(jobInfo.Namespace).Update(context.TODO(), job, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Commit failed when updating job: %v", err)
		}
	}

	// Update job status
	// Ignore jobs without changes
	if job.Status.State != pintav1.Idle && jobInfo.NumMasters == job.Status.NumMasters && jobInfo.NumReplicas == job.Status.NumReplicas {
		return
	}

	job.Status.NumMasters = jobInfo.NumMasters
	job.Status.NumReplicas = jobInfo.NumReplicas
	// Idle -> Scheduled
	if job.Status.State == pintav1.Idle && (jobInfo.NumMasters != 0 || jobInfo.NumReplicas != 0) {
		job.Status.State = pintav1.Scheduled
		job.Status.LastTransitionTime = metav1.Now()
	}
	// Scheduled -> Idle
	if job.Status.State == pintav1.Scheduled && (jobInfo.NumMasters == 0 && jobInfo.NumReplicas == 0) {
		job.Status.State = pintav1.Idle
		job.Status.LastTransitionTime = metav1.Now()
	}

	_, err = pinta.PintaJobs(jobInfo.Namespace).UpdateStatus(context.TODO(), job, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Commit failed when updating job status: %v", err)
	}
}

func (ju *jobUpdater) updateRoleSpec(role *pintav1.RoleSpec) {
	// nodeSelector
	if role.NodeType != "" {
		role.Spec.NodeSelector["pinta.qed.usc.edu/type"] = role.NodeType
	}
	fractionNode, found := role.Resources["node"]
	if found && !fractionNode.IsZero() {
		one, _ := resource.ParseQuantity("1")
		if !fractionNode.Equal(one) {
			klog.Errorf("resources.node != 1, treating it as 1")
		}
		if len(role.Resources) != 1 {
			klog.Errorf("resources.node cannot be specified together with other resource types, ignoring other resource types")
		}
		oneNodeResource := ju.ssn.NodeTypes[role.NodeType].Resource
		role.Spec.Containers[0].Resources.Limits = oneNodeResource.ToResourceList()
	} else {
		role.Spec.Containers[0].Resources.Limits = role.Resources
	}
}
