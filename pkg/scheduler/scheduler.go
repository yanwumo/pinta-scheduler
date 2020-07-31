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

package scheduler

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/conf"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	pintacache "github.com/qed-usc/pinta-scheduler/pkg/scheduler/cache"
)

// Scheduler watches for new unscheduled pods for volcano. It attempts to find
// nodes that they fit on and writes bindings back to the api server.
type Scheduler struct {
	cache          pintacache.Cache
	policy         policies.Policy
	configuration  *conf.Configuration
	schedulerConf  string
	schedulePeriod time.Duration
}

// NewScheduler returns a scheduler
func NewScheduler(
	config *rest.Config,
	schedulerConf string,
	period time.Duration,
) (*Scheduler, error) {
	scheduler := &Scheduler{
		schedulerConf:  schedulerConf,
		cache:          pintacache.New(config),
		schedulePeriod: period,
	}

	return scheduler, nil
}

// Run runs the Scheduler
func (pc *Scheduler) Run(stopCh <-chan struct{}) {
	// Start cache for policy.
	go pc.cache.Run(stopCh)
	pc.cache.WaitForCacheSync(stopCh)

	go wait.Until(pc.runOnce, pc.schedulePeriod, stopCh)
}

func (pc *Scheduler) runOnce() {
	klog.V(4).Infof("Start scheduling ...")
	defer klog.V(4).Infof("End scheduling ...")

	pc.loadSchedulerConf()

	snapshot := pc.cache.Snapshot()
	pc.policy.Execute(snapshot)
	pc.cache.Commit(snapshot)
}

func (pc *Scheduler) loadSchedulerConf() {
	var err error

	// Load configuration of scheduler
	schedConf := defaultSchedulerConf
	if len(pc.schedulerConf) != 0 {
		if schedConf, err = readSchedulerConf(pc.schedulerConf); err != nil {
			klog.Errorf("Failed to read scheduler configuration '%s', using default configuration: %v",
				pc.schedulerConf, err)
			schedConf = defaultSchedulerConf
		}
	}

	pc.policy, pc.configuration, err = loadSchedulerConf(schedConf)
	if err != nil {
		panic(err)
	}
}
