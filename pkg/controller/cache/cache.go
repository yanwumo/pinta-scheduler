package cache

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	"sync"
	"time"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"

	"golang.org/x/time/rate"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type jobCache struct {
	sync.Mutex

	nodes     map[string]*info.NodeInfo
	nodeTypes map[string]*info.NodeTypeInfo

	jobs        map[string]*api.JobInfo
	deletedJobs workqueue.RateLimitingInterface
}

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

func New() Cache {
	queue := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 180*time.Second),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)

	return &jobCache{
		nodes:       map[string]*info.NodeInfo{},
		nodeTypes:   map[string]*info.NodeTypeInfo{},
		jobs:        map[string]*api.JobInfo{},
		deletedJobs: workqueue.NewRateLimitingQueue(queue),
	}
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

func (jc *jobCache) addOrUpdateNode(node *v1.Node) {
	// Build NodeInfo from node
	nodeInfo := info.NewNodeInfo(node)

	// Check if the node already exists in the cache
	oldNodeInfo, found := jc.nodes[node.Name]
	if found && !oldNodeInfo.Allocatable.EqualStrict(nodeInfo.Allocatable) {
		// If the node resource has changed, recalculate node type map
		jc.deleteNode(node)
	}

	// Add new NodeInfo to the cache
	jc.nodes[node.Name] = nodeInfo

	// Update node type map
	nodeType, found := jc.nodeTypes[nodeInfo.Type]
	if found {
		nodeType.AddNode(nodeInfo)
	} else {
		jc.nodeTypes[nodeInfo.Type] = info.NewNodeTypeInfo(nodeInfo)
	}
}

func (jc *jobCache) deleteNode(node *v1.Node) {
	// Delete NodeInfo from the cache
	delete(jc.nodes, node.Name)

	// Recalculate node type map
	jc.nodeTypes = map[string]*info.NodeTypeInfo{}
	for _, nodeInfo := range jc.nodes {
		nodeType, found := jc.nodeTypes[nodeInfo.Type]
		if found {
			nodeType.AddNode(nodeInfo)
		} else {
			jc.nodeTypes[nodeInfo.Type] = info.NewNodeTypeInfo(nodeInfo)
		}
	}
}

func (jc *jobCache) AddNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.addOrUpdateNode(node)
}

func (jc *jobCache) UpdateNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.addOrUpdateNode(node)
}

func (jc *jobCache) DeleteNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.deleteNode(node)
}

func (jc *jobCache) Run(stopCh <-chan struct{}) {
	wait.Until(jc.worker, 0, stopCh)
}

func (jc *jobCache) worker() {
	for jc.processCleanupJob() {
	}
}

func (jc *jobCache) processCleanupJob() bool {
	obj, shutdown := jc.deletedJobs.Get()
	if shutdown {
		return false
	}
	defer jc.deletedJobs.Done(obj)

	job, ok := obj.(*api.JobInfo)
	if !ok {
		klog.Errorf("failed to convert %v to *apis.JobInfo", obj)
		return true
	}

	jc.Mutex.Lock()
	defer jc.Mutex.Unlock()

	if jobTerminated(job) {
		jc.deletedJobs.Forget(obj)
		key := keyFn(job.Namespace, job.Name)
		delete(jc.jobs, key)
		klog.V(3).Infof("Job <%s> was deleted.", key)
	} else {
		// Retry
		jc.deleteJob(job)
	}
	return true
}

func (jc *jobCache) deleteJob(job *api.JobInfo) {
	klog.V(3).Infof("Try to delete Job <%v/%v>",
		job.Namespace, job.Name)

	jc.deletedJobs.AddRateLimited(job)
}
