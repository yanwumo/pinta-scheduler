package cache

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"reflect"
	"sync"

	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	pintainformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions"
	ptjobinformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions/pinta/v1"
	kubeinformers "k8s.io/client-go/informers/core/v1"
	volcanoclientset "volcano.sh/volcano/pkg/client/clientset/versioned"
	volcanoinformers "volcano.sh/volcano/pkg/client/informers/externalversions"
	vcjobinformers "volcano.sh/volcano/pkg/client/informers/externalversions/batch/v1alpha1"
)

type PintaCache struct {
	sync.Mutex

	kubeClient  *kubernetes.Clientset
	vcClient    *volcanoclientset.Clientset
	pintaClient *clientset.Clientset

	nodeInformer  kubeinformers.NodeInformer
	vcInformer    vcjobinformers.JobInformer
	pintaInformer ptjobinformers.PintaJobInformer

	JobInfoUpdater JobInfoUpdater

	Recorder record.EventRecorder

	Jobs  map[info.JobID]*info.JobInfo
	Nodes map[string]*info.NodeInfo
}

func New(config *rest.Config) Cache {
	return newPintaCache(config)
}

func newPintaCache(config *rest.Config) Cache {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("Kubernetes clientset initialization failed: %v", err))
	}
	vcClient, err := volcanoclientset.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("Volcano clientset initialization failed: %v", err))
	}
	client, err := clientset.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("Pinta clientset initialization failed: %v", err))
	}
	eventClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("failed init eventClient, with err: %v", err))
	}

	sc := &PintaCache{
		Jobs:        make(map[info.JobID]*info.JobInfo),
		Nodes:       make(map[string]*info.NodeInfo),
		kubeClient:  kubeClient,
		vcClient:    vcClient,
		pintaClient: client,
	}

	// Prepare event clients.
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: eventClient.CoreV1().Events("")})
	sc.Recorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "pinta-scheduler"})

	sc.JobInfoUpdater = &defaultJobInfoUpdater{
		pintaClient: client,
	}

	informerFactory := informers.NewSharedInformerFactory(sc.kubeClient, 0)
	sc.nodeInformer = informerFactory.Core().V1().Nodes()
	sc.nodeInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    sc.AddNode,
			UpdateFunc: sc.UpdateNode,
			DeleteFunc: sc.DeleteNode,
		},
		0,
	)

	vcinformers := volcanoinformers.NewSharedInformerFactory(sc.vcClient, 0)
	sc.vcInformer = vcinformers.Batch().V1alpha1().Jobs()
	sc.vcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	ptinformers := pintainformers.NewSharedInformerFactory(sc.pintaClient, 0)
	sc.pintaInformer = ptinformers.Pinta().V1().PintaJobs()
	sc.pintaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    sc.AddJob,
		UpdateFunc: sc.UpdateJob,
		DeleteFunc: sc.DeleteJob,
	})

	return sc
}

func (sc *PintaCache) Run(stopCh <-chan struct{}) {
	go sc.nodeInformer.Informer().Run(stopCh)
	go sc.vcInformer.Informer().Run(stopCh)
	go sc.pintaInformer.Informer().Run(stopCh)
}

// Synchronize the cache with apiserver
func (sc *PintaCache) WaitForCacheSync(stopCh <-chan struct{}) bool {
	return cache.WaitForCacheSync(stopCh,
		func() []cache.InformerSynced {
			informerSynced := []cache.InformerSynced{
				sc.nodeInformer.Informer().HasSynced,
				sc.vcInformer.Informer().HasSynced,
				sc.pintaInformer.Informer().HasSynced,
			}
			return informerSynced
		}()...,
	)
}

func (sc *PintaCache) Snapshot(jobCustomFieldsType reflect.Type) *info.ClusterInfo {
	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	snapshot := info.NewClusterInfo()

	for _, value := range sc.Nodes {
		if !value.Ready() {
			continue
		}

		// Skip master node
		if len(value.Node.Spec.Taints) != 0 {
			continue
		}

		snapshot.Nodes[value.Name] = value.Clone()
	}

	var cloneJobLock sync.Mutex
	var wg sync.WaitGroup

	cloneJob := func(value *info.JobInfo) {
		defer wg.Done()

		clonedJob := value.Clone()
		err := clonedJob.ParseCustomFields(jobCustomFieldsType)
		if err != nil {
			klog.Errorf("Cannot parse custom fields for job %v: %v", clonedJob.Name, err)
		}

		cloneJobLock.Lock()
		snapshot.Jobs[value.UID] = clonedJob
		cloneJobLock.Unlock()
	}

	for _, value := range sc.Jobs {
		wg.Add(1)
		go cloneJob(value)
	}
	wg.Wait()

	klog.V(3).Infof("There are <%d> Jobs and <%d> Nodes in total for scheduling.",
		len(snapshot.Jobs), len(snapshot.Nodes))

	return snapshot
}

// Client returns the kubernetes clientSet
func (sc *PintaCache) Client() kubernetes.Interface {
	return sc.kubeClient
}

// VCClient returns the volcano clientSet
func (sc *PintaCache) VCClient() volcanoclientset.Interface {
	return sc.vcClient
}

// PintaClient returns the Pinta clientSet
func (sc *PintaCache) PintaClient() clientset.Interface {
	return sc.pintaClient
}
