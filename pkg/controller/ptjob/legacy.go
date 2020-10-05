package ptjob

import (
	"context"
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a PintaJob is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a PintaJob fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by PintaJob"
	// MessageResourceSynced is the message used for an Event fired when a PintaJob
	// is synced successfully
	MessageResourceSynced = "PintaJob synced successfully"
)

func (c *PintaJobController) legacyProcessPintaJob(pintaJob *pintav1.PintaJob) error {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pintaJob.Status) > 0 {
		lastPintaJobStatus = pintaJob.Status[0]
	}

	volcanoJobName := pintaJob.Name
	if volcanoJobName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("PintaJob name must be specified"))
		return nil
	}

	// Get the Volcano job with the name specified in PintaJob.spec
	volcanoJob, err := c.vcJobLister.Jobs(pintaJob.Namespace).Get(volcanoJobName)
	// If the resource doesn't exist, we'll create it
	// Idle is the only state that PintaJob doesn't have a Volcano Job mapping
	if errors.IsNotFound(err) {
		err = nil
		volcanoJob = nil
		if lastPintaJobStatus.State != pintav1.Idle && lastPintaJobStatus.State != "" {
			volcanoJob, err = c.vcClient.BatchV1alpha1().Jobs(pintaJob.Namespace).Create(context.TODO(), newVCJob(pintaJob), metav1.CreateOptions{})
		}
	} else {
		if lastPintaJobStatus.State == pintav1.Idle || lastPintaJobStatus.State == "" {
			err = c.vcClient.BatchV1alpha1().Jobs(pintaJob.Namespace).Delete(context.TODO(), volcanoJobName, metav1.DeleteOptions{})
			volcanoJob = nil
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this PintaJob resource, we should log
	// a warning to the event recorder and return error msg.
	if volcanoJob != nil && !metav1.IsControlledBy(volcanoJob, pintaJob) {
		msg := fmt.Sprintf(MessageResourceExists, volcanoJob.Name)
		c.recorder.Event(pintaJob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// If this number of the replicas on the PintaJob resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if volcanoJob != nil {
		consistent := true
		for i, task := range volcanoJob.Spec.Tasks {
			if task.Name == "ps" || task.Name == "master" {
				if task.Replicas != lastPintaJobStatus.NumMasters {
					consistent = false
					volcanoJob.Spec.Tasks[i].Replicas = lastPintaJobStatus.NumMasters
				}
			} else if task.Name == "worker" || task.Name == "replica" {
				if task.Replicas != lastPintaJobStatus.NumReplicas {
					consistent = false
					volcanoJob.Spec.Tasks[i].Replicas = lastPintaJobStatus.NumReplicas
				}
			} else if task.Name == "image-builder" {
				if task.Replicas != 1 {
					consistent = false
					volcanoJob.Spec.Tasks[i].Replicas = 1
				}
			}
		}
		if !consistent {
			// Calculate total number of replicas
			volcanoJob.Spec.MinAvailable = 0
			for _, task := range volcanoJob.Spec.Tasks {
				volcanoJob.Spec.MinAvailable += task.Replicas
			}
			volcanoJob, err = c.vcClient.BatchV1alpha1().Jobs(pintaJob.Namespace).Update(context.TODO(), volcanoJob, metav1.UpdateOptions{})
		}
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the PintaJob resource to reflect the
	// current state of the world
	err = c.updatePintaJobStatus(pintaJob, volcanoJob)
	if err != nil {
		return err
	}

	c.recorder.Event(pintaJob, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *PintaJobController) updatePintaJobStatus(pintaJob *pintav1.PintaJob, volcanoJob *volcanov1alpha1.Job) error {
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pintaJob.Status) > 0 {
		lastPintaJobStatus = pintaJob.Status[0]
	}
	oldState := lastPintaJobStatus.State
	newState := oldState
	if oldState == "" {
		newState = pintav1.Idle
	}
	if volcanoJob != nil {
		switch volcanoJob.Status.State.Phase {
		case volcanov1alpha1.Running:
			if lastPintaJobStatus.NumReplicas == 0 {
				newState = pintav1.Preempted
			} else {
				newState = pintav1.Running
			}
		case volcanov1alpha1.Completed:
			newState = pintav1.Completed
		case volcanov1alpha1.Failed:
			newState = pintav1.Failed
		}
	}

	if newState == oldState {
		return nil
	}

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	pintaJobCopy := pintaJob.DeepCopy()
	pintaJobCopy.Status = append([]pintav1.PintaJobStatus{
		{
			State:              newState,
			LastTransitionTime: metav1.Now(),
			NumMasters:         lastPintaJobStatus.NumMasters,
			NumReplicas:        lastPintaJobStatus.NumReplicas,
		},
	}, pintaJobCopy.Status...)
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the PintaJob resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	//
	// Subresource is enabled.
	_, err := c.pintaClient.PintaV1().PintaJobs(pintaJob.Namespace).UpdateStatus(context.TODO(), pintaJobCopy, metav1.UpdateOptions{})
	return err
}

// newVCJob creates a new Volcano Job for a PintaJob resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the PintaJob resource that 'owns' it.
func newVCJob(pintaJob *pintav1.PintaJob) *volcanov1alpha1.Job {
	//labels := map[string]string{
	//	"app":        "pinta-job",
	//	"controller": pintaJob.Name,
	//}
	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pintaJob.Status) > 0 {
		lastPintaJobStatus = pintaJob.Status[0]
	}
	var tasks []volcanov1alpha1.TaskSpec
	switch pintaJob.Spec.Type {
	case pintav1.PSWorker:
		tasks = []volcanov1alpha1.TaskSpec{
			{
				Name:     "ps",
				Replicas: lastPintaJobStatus.NumMasters,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Master.Spec,
				},
				Policies: nil,
			},
			{
				Name:     "worker",
				Replicas: lastPintaJobStatus.NumReplicas,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Replica.Spec,
				},
				Policies: []volcanov1alpha1.LifecyclePolicy{
					{
						Event:  "TaskCompleted",
						Action: "CompleteJob",
					},
				},
			},
		}
	case pintav1.MPI:
		tasks = []volcanov1alpha1.TaskSpec{
			{
				Name:     "master",
				Replicas: lastPintaJobStatus.NumMasters,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Master.Spec,
				},
				Policies: []volcanov1alpha1.LifecyclePolicy{
					{
						Event:  "TaskCompleted",
						Action: "CompleteJob",
					},
				},
			},
			{
				Name:     "replica",
				Replicas: lastPintaJobStatus.NumReplicas,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Replica.Spec,
				},
				Policies: nil,
			},
		}
	case pintav1.Symmetric:
		tasks = []volcanov1alpha1.TaskSpec{
			{
				Name:     "replica",
				Replicas: lastPintaJobStatus.NumReplicas,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Replica.Spec,
				},
				Policies: []volcanov1alpha1.LifecyclePolicy{
					{
						Event:  "TaskCompleted",
						Action: "CompleteJob",
					},
				},
			},
		}
	case pintav1.ImageBuilder:
		tasks = []volcanov1alpha1.TaskSpec{
			{
				Name:     "image-builder",
				Replicas: 1,
				Template: corev1.PodTemplateSpec{
					Spec: pintaJob.Spec.Replica.Spec,
				},
				Policies: nil,
			},
		}
	default:
		return nil
	}
	// Calculate total number of replicas
	var sumReplicas int32 = 0
	for _, task := range tasks {
		sumReplicas += task.Replicas
	}
	return &volcanov1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pintaJob.Name,
			Namespace: pintaJob.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pintaJob, pintav1.SchemeGroupVersion.WithKind("PintaJob")),
			},
		},
		Spec: volcanov1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  sumReplicas,
			Volumes:       pintaJob.Spec.Volumes,
			Tasks:         tasks,
			Policies:      nil,
			Plugins: map[string][]string{
				"env": {},
				"svc": {},
			},
			Queue:                   "",
			MaxRetry:                0,
			TTLSecondsAfterFinished: nil,
			PriorityClassName:       "",
		},
	}
}
