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

package cache

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	v1 "k8s.io/api/core/v1"
	"reflect"

	"k8s.io/client-go/kubernetes"

	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	vcclient "volcano.sh/volcano/pkg/client/clientset/versioned"
)

// Cache collects pods/nodes/queues information
// and provides information snapshot
type Cache interface {
	// Run start informer
	Run(stopCh <-chan struct{})

	// Snapshot deep copy overall cache information into snapshot
	Snapshot(jobCustomFieldsType reflect.Type) *info.ClusterInfo

	// WaitForCacheSync waits for all cache synced
	WaitForCacheSync(stopCh <-chan struct{}) bool

	// Client returns the kubernetes clientSet
	Client() kubernetes.Interface

	// VCClient returns the volcano clientSet
	VCClient() vcclient.Interface

	// PintaClient returns the Pinta clientSet
	PintaClient() clientset.Interface
}

type JobInfoUpdater interface {
	UpdateJobNodeType(nodeType string) error
	UpdateJobResourceRequirements(rl v1.ResourceList) error
	UpdateJobStatus(status pintav1.PintaJobStatus) error
}
