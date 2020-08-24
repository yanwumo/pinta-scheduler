package cache

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func getJobID(job *pintav1.PintaJob) api.JobID {
	return api.JobID(fmt.Sprintf("%s/%s", job.Namespace, job.Name))
}

// Assumes that lock is already acquired.
func (sc *PintaCache) addJob(job *pintav1.PintaJob) error {
	if job.Status.State == pintav1.Completed {
		return nil
	}

	jobID := getJobID(job)
	ji := api.NewJobInfo(jobID, job)
	if sc.Jobs[jobID] != nil {
		sc.Jobs[jobID] = ji
	} else {
		sc.Jobs[jobID] = ji
	}
	return nil
}

// Assumes that lock is already acquired.
func (sc *PintaCache) updateJob(oldJob, newJob *pintav1.PintaJob) error {
	if err := sc.deleteJob(oldJob); err != nil {
		return err
	}

	if len(getController(newJob)) == 0 {
		newJob.OwnerReferences = oldJob.OwnerReferences
	}

	return sc.addJob(newJob)
}

// Assumes that lock is already acquired.
func (sc *PintaCache) deleteJob(job *pintav1.PintaJob) error {
	delete(sc.Jobs, getJobID(job))
	return nil
}

// AddJob add job to scheduler cache
func (sc *PintaCache) AddJob(obj interface{}) {
	job, ok := obj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("Cannot convert to *v1alpha1.Job: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.addJob(job)
	if err != nil {
		klog.Errorf("Failed to add job <%s/%s> into cache: %v",
			job.Namespace, job.Name, err)
		return
	}
	klog.V(3).Infof("Added job <%s/%v> into cache.", job.Namespace, job.Name)
}

// UpdateJob update job to scheduler cache
func (sc *PintaCache) UpdateJob(oldObj, newObj interface{}) {
	oldJob, ok := oldObj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *pintav1.PintaJob: %v", oldObj)
		return
	}
	newJob, ok := newObj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("Cannot convert newObj to *pintav1.PintaJob: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.updateJob(oldJob, newJob)
	if err != nil {
		klog.Errorf("Failed to update job %v in cache: %v", oldJob.Name, err)
		return
	}

	klog.V(3).Infof("Updated job <%s/%v> in cache.", oldJob.Namespace, oldJob.Name)
}

// DeleteJob delete job from scheduler cache
func (sc *PintaCache) DeleteJob(obj interface{}) {
	var job *pintav1.PintaJob
	switch t := obj.(type) {
	case *pintav1.PintaJob:
		job = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		job, ok = t.Obj.(*pintav1.PintaJob)
		if !ok {
			klog.Errorf("Cannot convert to *v1alpha1.Job: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *v1alpha1.Job: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.deleteJob(job)
	if err != nil {
		klog.Errorf("Failed to delete job %v from cache: %v", job.Name, err)
		return
	}

	klog.V(3).Infof("Deleted job <%s/%v> from cache.", job.Namespace, job.Name)
}

// Assumes that lock is already acquired.
func (sc *PintaCache) addNode(node *v1.Node) error {
	if sc.Nodes[node.Name] != nil {
		sc.Nodes[node.Name].SetNode(node)
	} else {
		sc.Nodes[node.Name] = api.NewNodeInfo(node)
	}
	return nil
}

// Assumes that lock is already acquired.
func (sc *PintaCache) updateNode(oldNode, newNode *v1.Node) error {
	if sc.Nodes[newNode.Name] != nil {
		sc.Nodes[newNode.Name].SetNode(newNode)
		return nil
	}

	return fmt.Errorf("node <%s> does not exist", newNode.Name)
}

// Assumes that lock is already acquired.
func (sc *PintaCache) deleteNode(node *v1.Node) error {
	if _, ok := sc.Nodes[node.Name]; !ok {
		return fmt.Errorf("node <%s> does not exist", node.Name)
	}
	delete(sc.Nodes, node.Name)
	return nil
}

// AddNode add node to scheduler cache
func (sc *PintaCache) AddNode(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		klog.Errorf("Cannot convert to *v1.Node: %v", obj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.addNode(node)
	if err != nil {
		klog.Errorf("Failed to add node %s into cache: %v", node.Name, err)
		return
	}
}

// UpdateNode update node to scheduler cache
func (sc *PintaCache) UpdateNode(oldObj, newObj interface{}) {
	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		klog.Errorf("Cannot convert oldObj to *v1.Node: %v", oldObj)
		return
	}
	newNode, ok := newObj.(*v1.Node)
	if !ok {
		klog.Errorf("Cannot convert newObj to *v1.Node: %v", newObj)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.updateNode(oldNode, newNode)
	if err != nil {
		klog.Errorf("Failed to update node %v in cache: %v", oldNode.Name, err)
		return
	}
}

// DeleteNode delete node from scheduler cache
func (sc *PintaCache) DeleteNode(obj interface{}) {
	var node *v1.Node
	switch t := obj.(type) {
	case *v1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*v1.Node)
		if !ok {
			klog.Errorf("Cannot convert to *v1.Node: %v", t.Obj)
			return
		}
	default:
		klog.Errorf("Cannot convert to *v1.Node: %v", t)
		return
	}

	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	err := sc.deleteNode(node)
	if err != nil {
		klog.Errorf("Failed to delete node %s from cache: %v", node.Name, err)
		return
	}
}

func getController(obj interface{}) types.UID {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	controllerRef := metav1.GetControllerOf(accessor)
	if controllerRef != nil {
		return controllerRef.UID
	}

	return ""
}
