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

package conf

// SchedulerConfiguration defines the configuration of scheduler.
type SchedulerConfiguration struct {
	// policies defines the policies list of scheduler in order
	Policy string `yaml:"policy"`
	// Configurations is configuration for policies
	Configuration Configuration `yaml:"configuration"`
}

// Configuration is configuration of policy
type Configuration struct {
	// Arguments defines the different arguments that can be given to specified policy
	Arguments map[string]string `yaml:"arguments"`
}
