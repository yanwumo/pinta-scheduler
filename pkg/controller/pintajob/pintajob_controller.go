package pintajob

import (
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	controllercache "github.com/qed-usc/pinta-scheduler/pkg/controller/cache"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/framework"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/state"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/pintajob/updater"
	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	pintascheme "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned/scheme"
	informers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions"
	pintajobinformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions/pintascheduler/v1"
	listers "github.com/qed-usc/pinta-scheduler/pkg/generated/listers/pintascheduler/v1"
	"hash"
	"hash/fnv"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"time"
	volcano "volcano.sh/volcano/pkg/client/clientset/versioned"
	volcanoinformers "volcano.sh/volcano/pkg/client/informers/externalversions"
	vcjobinformers "volcano.sh/volcano/pkg/client/informers/externalversions/batch/v1alpha1"
	volcanolisters "volcano.sh/volcano/pkg/client/listers/batch/v1alpha1"
)

func init() {
	_ = framework.RegisterController(&PintaJobController{})
}

const (
	// maxRetries is the number of times a PintaJob will retry before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a PintaJob is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

type PintaJobController struct {
	kubeClient  kubernetes.Interface
	vcClient    volcano.Interface
	pintaClient clientset.Interface

	nodeInformer     coreinformers.NodeInformer
	vcJobInformer    vcjobinformers.JobInformer
	pintaJobInformer pintajobinformers.PintaJobInformer

	nodeLister corelisters.NodeLister
	nodeSynced func() bool

	vcJobLister volcanolisters.JobLister
	vcJobSynced func() bool

	pintaJobLister listers.PintaJobLister
	pintaJobSynced func() bool

	queueList []workqueue.RateLimitingInterface
	cache     controllercache.Cache
	recorder  record.EventRecorder
	workers   uint32
}

func (c *PintaJobController) Name() string {
	return "pintajob-controller"
}

func (c *PintaJobController) Initialize(opt *framework.ControllerOption) error {
	c.kubeClient = opt.KubeClient
	c.pintaClient = opt.PintaClient
	c.vcClient = opt.VolcanoClient

	sharedInformers := opt.SharedInformerFactory
	workers := opt.WorkerNum

	// Initialize event client
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: c.kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(pintascheme.Scheme, v1.EventSource{Component: "pinta-controller"})

	c.queueList = make([]workqueue.RateLimitingInterface, workers)
	c.cache = controllercache.New()
	c.recorder = recorder
	c.workers = workers

	var i uint32
	for i = 0; i < workers; i++ {
		c.queueList[i] = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	}

	c.nodeInformer = sharedInformers.Core().V1().Nodes()
	c.nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNode,
		UpdateFunc: c.updateNode,
		DeleteFunc: c.deleteNode,
	})
	c.nodeLister = c.nodeInformer.Lister()
	c.nodeSynced = c.nodeInformer.Informer().HasSynced

	c.vcJobInformer = volcanoinformers.NewSharedInformerFactory(c.vcClient, 0).Batch().V1alpha1().Jobs()
	c.vcJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVCJob,
		UpdateFunc: c.updateVCJob,
		DeleteFunc: c.deleteVCJob,
	})
	c.vcJobLister = c.vcJobInformer.Lister()
	c.vcJobSynced = c.vcJobInformer.Informer().HasSynced

	c.pintaJobInformer = informers.NewSharedInformerFactory(c.pintaClient, 0).Pinta().V1().PintaJobs()
	c.pintaJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addJob,
		UpdateFunc: c.updateJob,
		DeleteFunc: c.deleteJob,
	})
	c.pintaJobLister = c.pintaJobInformer.Lister()
	c.pintaJobSynced = c.pintaJobInformer.Informer().HasSynced

	return nil
}

func (c *PintaJobController) Run(stopCh <-chan struct{}) {
	go c.nodeInformer.Informer().Run(stopCh)
	go c.vcJobInformer.Informer().Run(stopCh)
	go c.pintaJobInformer.Informer().Run(stopCh)

	cache.WaitForCacheSync(stopCh, c.nodeSynced, c.vcJobSynced, c.pintaJobSynced)

	var i uint32
	for i = 0; i < c.workers; i++ {
		go func(num uint32) {
			wait.Until(
				func() {
					c.worker(num)
				},
				time.Second,
				stopCh)
		}(i)
	}

	go c.cache.Run(stopCh)

	// Re-sync error tasks.
	//go wait.Until(c.processResyncTask, 0, stopCh)

	klog.Infof("PintaJobController is running")
}
func (c *PintaJobController) worker(i uint32) {
	klog.Infof("worker %d start ...... ", i)

	for c.processNextReq(i) {
	}
}

func (c *PintaJobController) belongsToThisRoutine(key string, count uint32) bool {
	var hashVal hash.Hash32
	var val uint32

	hashVal = fnv.New32()
	_, _ = hashVal.Write([]byte(key))

	val = hashVal.Sum32()

	return val%c.workers == count
}

func (c *PintaJobController) getWorkerQueue(key string) workqueue.RateLimitingInterface {
	var hashVal hash.Hash32
	var val uint32

	hashVal = fnv.New32()
	_, _ = hashVal.Write([]byte(key))

	val = hashVal.Sum32()

	queue := c.queueList[val%c.workers]

	return queue
}

func (c *PintaJobController) processNextReq(count uint32) bool {
	queue := c.queueList[count]
	obj, shutdown := queue.Get()
	if shutdown {
		klog.Errorf("Fail to pop item from queue")
		return false
	}

	req := obj.(api.Request)
	defer queue.Done(req)

	key := controllercache.JobKeyByReq(&req)
	if !c.belongsToThisRoutine(key, count) {
		klog.Errorf("The job does not belong to this routine key:%s, worker:%d...... ", key, count)
		queueLocal := c.getWorkerQueue(key)
		queueLocal.Add(req)
		return true
	}

	klog.V(3).Infof("Try to handle request <%v>", req)

	jobInfo, err := c.cache.Get(key)
	if err != nil {
		klog.Errorf("Failed to get job by <%v> from cache: %v", req, err)
		return true
	}

	vcJobUpdater := updater.NewVCJobUpdater(c.cache, c.vcClient, c.pintaClient, jobInfo)

	st := state.NewState(vcJobUpdater)
	if st == nil {
		klog.Errorf("Invalid state of Job <%v/%v>", jobInfo.Job.Namespace, jobInfo.Job.Name)
		return true
	}

	klog.V(4).Infof("Job <%v/%v> executes in <%v> state", jobInfo.Job.Namespace, jobInfo.Job.Name, st.Name())
	if err := st.Execute(); err != nil {
		if queue.NumRequeues(req) < maxRetries {
			klog.V(2).Infof("Failed to handle Job <%s/%s>: %v",
				jobInfo.Job.Namespace, jobInfo.Job.Name, err)
			// If any error, requeue it.
			queue.AddRateLimited(req)
			return true
		}
		//c.recordJobEvent(jobInfo.Job.Namespace, jobInfo.Job.Name, batchv1alpha1.ExecuteAction, fmt.Sprintf(
		//	"Job failed on action %s for retry limit reached", action))
		klog.Warningf("Dropping job <%s/%s> out of the queue: %v because max retries has reached", jobInfo.Job.Namespace, jobInfo.Job.Name, err)
	}

	// If no error, forget it.
	queue.Forget(req)

	return true
}
