package app

import (
	"context"
	"fmt"
	"github.com/qed-usc/pinta-scheduler/cmd/controller/app/options"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/helpers"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/framework"
	pintaclientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	"github.com/qed-usc/pinta-scheduler/pkg/kube"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"os"
	"time"
	vcclientset "volcano.sh/volcano/pkg/client/clientset/versioned"
)

const (
	leaseDuration = 15 * time.Second
	renewDeadline = 10 * time.Second
	retryPeriod   = 5 * time.Second
)

// Run the controller.
func Run(opt *options.ServerOption) error {
	config, err := kube.BuildConfig(opt.KubeClientOptions)
	if err != nil {
		return err
	}

	if err := helpers.StartHealthz(opt.HealthzBindAddress, "pinta-controller"); err != nil {
		return err
	}

	run := startControllers(config, opt)

	if !opt.EnableLeaderElection {
		run(context.TODO())
		return fmt.Errorf("finished without leader elect")
	}

	leaderElectionClient, err := kubeclientset.NewForConfig(rest.AddUserAgent(config, "leader-election"))
	if err != nil {
		return err
	}

	// Prepare event clients.
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: leaderElectionClient.CoreV1().Events(opt.LockObjectNamespace)})
	eventRecorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "pinta-controller"})

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("unable to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())

	rl, err := resourcelock.New(resourcelock.ConfigMapsResourceLock,
		opt.LockObjectNamespace,
		"pinta-controller",
		leaderElectionClient.CoreV1(),
		leaderElectionClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: eventRecorder,
		})
	if err != nil {
		return fmt.Errorf("couldn't create resource lock: %v", err)
	}

	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDeadline,
		RetryPeriod:   retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				klog.Fatalf("leaderelection lost")
			},
		},
	})
	return fmt.Errorf("lost lease")
}

func startControllers(config *rest.Config, opt *options.ServerOption) func(ctx context.Context) {
	controllerOpt := &framework.ControllerOption{}

	controllerOpt.SchedulerName = opt.SchedulerName
	controllerOpt.WorkerNum = opt.WorkerThreads

	controllerOpt.KubeClient = kubeclientset.NewForConfigOrDie(config)
	controllerOpt.VolcanoClient = vcclientset.NewForConfigOrDie(config)
	controllerOpt.PintaClient = pintaclientset.NewForConfigOrDie(config)
	controllerOpt.SharedInformerFactory = informers.NewSharedInformerFactory(controllerOpt.KubeClient, 0)

	return func(ctx context.Context) {
		framework.ForeachController(func(c framework.Controller) {
			if err := c.Initialize(controllerOpt); err != nil {
				klog.Errorf("Failed to initialize controller <%s>: %v", c.Name(), err)
				return
			}

			go c.Run(ctx.Done())
		})

		<-ctx.Done()
	}
}
