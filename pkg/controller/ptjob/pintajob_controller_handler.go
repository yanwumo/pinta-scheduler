package ptjob

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"reflect"
)

func getJobKeyByReq(req *api.Request) string {
	return fmt.Sprintf("%s/%s", req.Namespace, req.JobName)
}

func (c *PintaJobController) addJob(obj interface{}) {
	job, ok := obj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("obj is not PintaJob")
		return
	}

	req := api.Request{
		Namespace: job.Namespace,
		JobName:   job.Name,
	}

	if err := c.cache.Add(job); err != nil {
		klog.Errorf("Failed to add job <%s/%s>: %v in cache",
			job.Namespace, job.Name, err)
	}
	key := getJobKeyByReq(&req)
	queue := c.getWorkerQueue(key)
	queue.Add(req)
}

func (c *PintaJobController) updateJob(oldObj, newObj interface{}) {
	newJob, ok := newObj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("newObj is not Job")
		return
	}

	oldJob, ok := oldObj.(*pintav1.PintaJob)
	if !ok {
		klog.Errorf("oldJob is not Job")
		return
	}

	// No need to update if ResourceVersion is not changed
	if newJob.ResourceVersion == oldJob.ResourceVersion {
		klog.V(6).Infof("No need to update because job is not modified.")
		return
	}

	if err := c.cache.Update(newJob); err != nil {
		klog.Errorf("UpdateJob - Failed to update job <%s/%s>: %v in cache",
			newJob.Namespace, newJob.Name, err)
	}

	// NOTE: Since we only reconcile job based on Spec, we will ignore other attributes
	// For Job status, it's used internally and always been updated via our controller.
	if reflect.DeepEqual(newJob.Spec, oldJob.Spec) && len(newJob.Status) != 0 && len(oldJob.Status) != 0 && newJob.Status[0].State == oldJob.Status[0].State {
		klog.V(6).Infof("Job update event is ignored since no update in 'Spec'.")
		return
	}

	req := api.Request{
		Namespace: newJob.Namespace,
		JobName:   newJob.Name,
	}
	key := getJobKeyByReq(&req)
	queue := c.getWorkerQueue(key)
	queue.Add(req)
}

func (c *PintaJobController) deleteJob(obj interface{}) {
	job, ok := obj.(*pintav1.PintaJob)
	if !ok {
		// If we reached here it means the Job was deleted but its final state is unrecorded.
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		job, ok = tombstone.Obj.(*pintav1.PintaJob)
		if !ok {
			klog.Errorf("Tombstone contained object that is not a volcano Job: %#v", obj)
			return
		}
	}

	if err := c.cache.Delete(job); err != nil {
		klog.Errorf("Failed to delete job <%s/%s>: %v in cache",
			job.Namespace, job.Name, err)
	}
}

func (c *PintaJobController) addNode(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		klog.Errorf("obj is not Node")
		return
	}

	c.cache.AddNode(node)
}

func (c *PintaJobController) updateNode(oldObj, newObj interface{}) {
	newNode, ok := newObj.(*v1.Node)
	if !ok {
		klog.Errorf("newObj is not Node")
		return
	}

	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		klog.Errorf("oldObj is not Node")
		return
	}

	// No need to update if ResourceVersion is not changed
	if newNode.ResourceVersion == oldNode.ResourceVersion {
		klog.V(6).Infof("No need to update because node is not modified.")
		return
	}

	c.cache.UpdateNode(newNode)
}

func (c *PintaJobController) deleteNode(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		// If we reached here it means the Node was deleted but its final state is unrecorded.
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		node, ok = tombstone.Obj.(*v1.Node)
		if !ok {
			klog.Errorf("Tombstone contained object that is not a node: %#v", obj)
			return
		}
	}

	c.cache.DeleteNode(node)
}
