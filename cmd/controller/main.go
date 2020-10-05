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
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/qed-usc/pinta-scheduler/cmd/controller/app"
	"github.com/qed-usc/pinta-scheduler/cmd/controller/app/options"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	cliflag "k8s.io/component-base/cli/flag"
	"volcano.sh/volcano/pkg/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	_ "github.com/qed-usc/pinta-scheduler/pkg/controller/ptjob"
	"github.com/qed-usc/pinta-scheduler/pkg/metrics"
)

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

var (
	masterURL  string
	kubeconfig string
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	klog.InitFlags(nil)

	s := options.NewServerOption()
	s.AddFlags(pflag.CommandLine)

	cliflag.InitFlags()

	if s.PrintVersion {
		version.PrintVersionAndExit()
	}
	if err := s.CheckOptionOrDie(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	// The default klog flush interval is 30 seconds, which is frighteningly long.
	go wait.Until(klog.Flush, *logFlushFreq, wait.NeverStop)
	defer klog.Flush()

	// start serving prometheus on 8080
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metrics.RegisterPintaJob()
	go func() {
		klog.Info("Listening and serving metrics at port 8080...")
		err := http.ListenAndServe(":8080", metricsMux)
		if err != nil {
			klog.Errorf("Metrics (http) serving failed: %v", err)
		}
	}()

	if err := app.Run(s); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
