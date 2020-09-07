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
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	// init pprof server
	_ "net/http/pprof"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/wait"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qed-usc/pinta-scheduler/cmd/scheduler/app"
	"github.com/qed-usc/pinta-scheduler/cmd/scheduler/app/options"
	"github.com/qed-usc/pinta-scheduler/pkg/metrics"

	// Import default policies.
	_ "github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies"
)

var logFlushFreq = pflag.Duration("log-flush-frequency", 5*time.Second, "Maximum number of seconds between log flushes")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	klog.InitFlags(nil)

	s := options.NewServerOption()
	s.AddFlags(pflag.CommandLine)
	s.RegisterOptions()

	cliflag.InitFlags()
	if err := s.CheckOptionOrDie(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
