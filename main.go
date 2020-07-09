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

package main

import (
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	informers "github.com/qed-usc/pinta-scheduler/pkg/generated/informers/externalversions"
	"github.com/qed-usc/pinta-scheduler/pkg/signals"
	volcanoclientset "volcano.sh/volcano/pkg/client/clientset/versioned"
	volcanoinformers "volcano.sh/volcano/pkg/client/informers/externalversions"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	pintaClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building pinta clientset: %s", err.Error())
	}

	volcanoClient, err := volcanoclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building volcano clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	pintaInformerFactory := informers.NewSharedInformerFactory(pintaClient, time.Second*30)
	volcanoInformerFactory := volcanoinformers.NewSharedInformerFactory(volcanoClient, time.Second*30)

	controller := NewController(kubeClient, pintaClient, volcanoClient,
		volcanoInformerFactory.Batch().V1alpha1().Jobs(),
		pintaInformerFactory.Pinta().V1().PintaJobs())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	pintaInformerFactory.Start(stopCh)
	volcanoInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
