/*
Copyright 2018 The Kubernetes Authors.

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
	"fmt"
	"io/ioutil"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/conf"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies"
)

var defaultSchedulerConf = `
policy: "nop"
`

func loadSchedulerConf(confStr string) (policies.Policy, *conf.Configuration, error) {
	schedulerConf := &conf.SchedulerConfiguration{}

	buf := make([]byte, len(confStr))
	copy(buf, confStr)

	if err := yaml.Unmarshal(buf, schedulerConf); err != nil {
		return nil, nil, err
	}

	policyName := schedulerConf.Policy
	policy, found := policies.GetPolicy(strings.TrimSpace(policyName))
	if !found {
		return nil, nil, fmt.Errorf("failed to found Policy %s", policyName)
	}

	return policy, &schedulerConf.Configuration, nil
}

func readSchedulerConf(confPath string) (string, error) {
	dat, err := ioutil.ReadFile(confPath)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}
