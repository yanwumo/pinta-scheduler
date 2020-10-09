package updater

import (
	"context"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	controllercache "github.com/qed-usc/pinta-scheduler/pkg/controller/cache"
	pintajobtype "github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/type"
	pintaclientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	vcclientset "volcano.sh/volcano/pkg/client/clientset/versioned"
)

// Updater reconciles Volcano Job based on the given Pinta JobInfo.
type Updater struct {
	cache       controllercache.Cache
	vcClient    vcclientset.Interface
	pintaClient pintaclientset.Interface

	jobInfo *api.JobInfo
}

func NewVCJobUpdater(
	cache controllercache.Cache,
	vcClient vcclientset.Interface,
	pintaClient pintaclientset.Interface,
	info *api.JobInfo,
) *Updater {
	return &Updater{
		cache:       cache,
		vcClient:    vcClient,
		pintaClient: pintaClient,
		jobInfo:     info,
	}
}

func (u *Updater) GetVCJobStatus() volcanov1alpha1.JobPhase {
	job := u.jobInfo.VCJob

	return job.Status.State.Phase
}

func (u *Updater) GetLastPintaJobStatus() pintav1.PintaJobStatus {
	job := u.jobInfo.Job

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(job.Status) > 0 {
		lastPintaJobStatus = job.Status[0]
	}

	return lastPintaJobStatus
}

func (u *Updater) UpdatePintaJobStatusState(state pintav1.PintaJobState) error {
	pintaJobCopy := u.jobInfo.Job.DeepCopy()

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pintaJobCopy.Status) > 0 {
		lastPintaJobStatus = pintaJobCopy.Status[0]
	}

	pintaJobCopy.Status = append([]pintav1.PintaJobStatus{
		{
			State:              state,
			LastTransitionTime: metav1.Now(),
			NumMasters:         lastPintaJobStatus.NumMasters,
			NumReplicas:        lastPintaJobStatus.NumReplicas,
		},
	}, pintaJobCopy.Status...)

	newPintaJob, err := u.pintaClient.PintaV1().PintaJobs(u.jobInfo.Namespace).UpdateStatus(context.TODO(), pintaJobCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return u.cache.Update(newPintaJob)
}

func (u *Updater) Reconcile() error {
	pintaJob := u.jobInfo.Job
	vcJob := u.jobInfo.VCJob

	if pintaJob.DeletionTimestamp != nil {
		klog.Infof("PintaJob <%s/%s> is terminating, skip reconcile.",
			pintaJob.Namespace, pintaJob.Name)
		return nil
	}

	// PintaJob -> Volcano Job
	pintaJobType := pintajobtype.NewType(pintaJob)
	var newVCJob *volcanov1alpha1.Job
	var err error
	if vcJob == nil {
		newVCJob = pintaJobType.BuildVCJob()
		newVCJob, err = u.vcClient.BatchV1alpha1().Jobs(u.jobInfo.Namespace).Create(context.TODO(), newVCJob, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("PintaJob -> Volcano Job <%v/%v> creation failed: %v", u.jobInfo.Namespace, u.jobInfo.Namespace, err)
			return err
		}
	} else {
		newVCJob = vcJob.DeepCopy()
		changed, err := pintaJobType.ReconcileVCJob(newVCJob)

		if !changed {
			return nil
		}

		newVCJob, err = u.vcClient.BatchV1alpha1().Jobs(u.jobInfo.Namespace).Update(context.TODO(), newVCJob, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("PintaJob -> Volcano Job <%v/%v> reconciliation failed: %v", u.jobInfo.Namespace, u.jobInfo.Namespace, err)
			return err
		}
	}

	return u.cache.UpdateVCJob(newVCJob)
}
