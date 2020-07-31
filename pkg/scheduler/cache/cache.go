package cache

import (
	"context"
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sync"

	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	pintainformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions"
	ptjobinformers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
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

	Jobs  map[api.JobID]*api.JobInfo
	Nodes map[string]*api.NodeInfo
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

	sc := &PintaCache{
		Jobs:        make(map[api.JobID]*api.JobInfo),
		Nodes:       make(map[string]*api.NodeInfo),
		kubeClient:  kubeClient,
		vcClient:    vcClient,
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

func (sc *PintaCache) Snapshot() *api.ClusterInfo {
	sc.Mutex.Lock()
	defer sc.Mutex.Unlock()

	snapshot := api.NewClusterInfo()

	for _, value := range sc.Nodes {
		if !value.Ready() {
			continue
		}

		snapshot.Nodes[value.Name] = value.Clone()
	}

	var cloneJobLock sync.Mutex
	var wg sync.WaitGroup

	cloneJob := func(value *api.JobInfo) {
		defer wg.Done()

		clonedJob := value.Clone()

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

func (sc *PintaCache) Commit(snapshot *api.ClusterInfo) {
	pinta := sc.pintaClient.PintaV1()
	for _, jobID := range snapshot.Changes {
		jobInfo := snapshot.Jobs[jobID]
		//job, err := pinta.PintaJobs(jobInfo.Namespace).Get(context.TODO(), jobInfo.Name, metav1.GetOptions{})
		//if err != nil {
		//	klog.Errorf("Commit failed when getting job: %v", err)
		//	continue
		//}
		job := jobInfo.Job
		job.Spec.NumMasters = jobInfo.NumMasters
		job.Spec.NumReplicas = jobInfo.NumReplicas
		job, err := pinta.PintaJobs(jobInfo.Namespace).Update(context.TODO(), job, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Commit failed when updating job: %v", err)
		}
		if job.Status == pintav1.Idle && (job.Spec.NumMasters != 0 || job.Spec.NumReplicas != 0) {
			job.Status = pintav1.Scheduled
			_, err = pinta.PintaJobs(jobInfo.Namespace).UpdateStatus(context.TODO(), job, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Commit failed when updating job status: %v", err)
			}
		}
	}
}
