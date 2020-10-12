package cache

import (
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	"golang.org/x/time/rate"
	"sync"
	"time"

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
