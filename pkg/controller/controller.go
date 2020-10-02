/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/framework"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	pintascheme "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned/scheme"
	informers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions"
	pintajobinformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions/pintascheduler/v1"
	listers "github.com/qed-usc/pinta-scheduler/pkg/generated/listers/pintascheduler/v1"

	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	volcano "volcano.sh/volcano/pkg/client/clientset/versioned"
	volcanoinformers "volcano.sh/volcano/pkg/client/informers/externalversions"
	vcjobinformers "volcano.sh/volcano/pkg/client/informers/externalversions/batch/v1alpha1"
	volcanolisters "volcano.sh/volcano/pkg/client/listers/batch/v1alpha1"
)

const controllerAgentName = "pinta-scheduler"

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

func init() {
	_ = framework.RegisterController(&Controller{})
}

// Controller is the controller implementation for PintaJob resources
type Controller struct {
	kubeClient  kubernetes.Interface
	vcClient    volcano.Interface
	pintaClient clientset.Interface

	vcJobInformer    vcjobinformers.JobInformer
	pintaJobInformer pintajobinformers.PintaJobInformer

	vcJobLister volcanolisters.JobLister
	vcJobSynced func() bool

	pintaJobLister listers.PintaJobLister
	pintaJobSynced func() bool

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
	workers  int
}

func (c *Controller) Name() string {
	return "pinta-controller"
}

func (c *Controller) Initialize(opt *framework.ControllerOption) error {
	c.kubeClient = opt.KubeClient
	c.pintaClient = opt.PintaClient
	c.vcClient = opt.VolcanoClient

	// Create event broadcaster
	// Add pinta-scheduler types to the default Kubernetes Scheme so Events can be
	// logged for pinta-scheduler types.
	utilruntime.Must(pintascheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: c.kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	vcInformerFactory := volcanoinformers.NewSharedInformerFactory(c.vcClient, time.Second*30)
	c.vcJobInformer = vcInformerFactory.Batch().V1alpha1().Jobs()
	c.vcJobLister = c.vcJobInformer.Lister()
	c.vcJobSynced = c.vcJobInformer.Informer().HasSynced

	pintaInformerFactory := informers.NewSharedInformerFactory(c.pintaClient, time.Second*30)
	c.pintaJobInformer = pintaInformerFactory.Pinta().V1().PintaJobs()
	c.pintaJobLister = c.pintaJobInformer.Lister()
	c.pintaJobSynced = c.pintaJobInformer.Informer().HasSynced

	c.workqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PintaJobs")
	c.recorder = recorder
	c.workers = opt.WorkerNum

	klog.Info("Setting up event handlers")
	// Set up an event handler for when PintaJob resources change
	c.pintaJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueuePintaJob,
		UpdateFunc: func(old, new interface{}) {
			c.enqueuePintaJob(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a PintaJob resource will enqueue that PintaJob resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	c.vcJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newVolcanoJob := new.(*volcanov1alpha1.Job)
			oldVolcanoJob := old.(*volcanov1alpha1.Job)
			if newVolcanoJob.ResourceVersion == oldVolcanoJob.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})

	return nil
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting PintaJob controller")
	go c.vcJobInformer.Informer().Run(stopCh)
	go c.pintaJobInformer.Informer().Run(stopCh)

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.vcJobSynced, c.pintaJobSynced); !ok {
		klog.Errorf("failed to wait for caches to sync")
		return
	}

	klog.Info("Starting workers")
	// Launch two workers to process PintaJob resources
	for i := 0; i < c.workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// PintaJob resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the PintaJob resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the PintaJob resource with this namespace/name
	pintaJob, err := c.pintaJobLister.PintaJobs(namespace).Get(name)
	if err != nil {
		// The PintaJob resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("pintaJob '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	var lastPintaJobStatus pintav1.PintaJobStatus
	if len(pintaJob.Status) > 0 {
		lastPintaJobStatus = pintaJob.Status[0]
	}

	volcanoJobName := pintaJob.Name
	if volcanoJobName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: PintaJob name must be specified", key))
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

func (c *Controller) updatePintaJobStatus(pintaJob *pintav1.PintaJob, volcanoJob *volcanov1alpha1.Job) error {
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

// enqueuePintaJob takes a PintaJob resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than PintaJob.
func (c *Controller) enqueuePintaJob(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the PintaJob resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that PintaJob resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a PintaJob, we should not do anything more
		// with it.
		if ownerRef.Kind != "PintaJob" {
			return
		}

		pintaJob, err := c.pintaJobLister.PintaJobs(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of pintaJob '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueuePintaJob(pintaJob)
		return
	}
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
