package session

import (
	"context"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
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

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(job.Status) > 0 {
		lastPintaJobStatus = job.Status[0]
	}
	oldState := lastPintaJobStatus.State
	newState := oldState
	// Update job status
	// Ignore jobs without changes
	if oldState != pintav1.Idle && jobInfo.NumMasters == lastPintaJobStatus.NumMasters && jobInfo.NumReplicas == lastPintaJobStatus.NumReplicas {
		return
	}

	// Idle -> Scheduled
	if oldState == pintav1.Idle && (jobInfo.NumMasters != 0 || jobInfo.NumReplicas != 0) {
		newState = pintav1.Scheduled
	}
	// Scheduled -> Idle
	if oldState == pintav1.Scheduled && (jobInfo.NumMasters == 0 && jobInfo.NumReplicas == 0) {
		newState = pintav1.Idle
	}

	if newState == oldState {
		return
	}

	job.Status = append([]pintav1.PintaJobStatus{
		{
			State:              newState,
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

func (ju *jobUpdater) updateRoleSpec(role *pintav1.RoleSpec) {
	if len(role.Spec.Containers) == 0 {
		return
	}

	// nodeSelector
	// TODO: move nodeSelector setting to controller
	if role.NodeType != "" {
		role.Spec.NodeSelector["pinta.qed.usc.edu/type"] = role.NodeType
	}
	// TODO: move resource translation to controller
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
