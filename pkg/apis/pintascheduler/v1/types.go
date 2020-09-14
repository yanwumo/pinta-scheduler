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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	volcanov1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PintaJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PintaJobSpec   `json:"spec,omitempty"`
	Status PintaJobStatus `json:"status,omitempty"`
}

type PintaJobSpec struct {
	Type    PintaJobType                 `json:"type,omitempty"`
	Volumes []volcanov1alpha1.VolumeSpec `json:"volumes,omitempty"`
	Master  RoleSpec                     `json:"master,omitempty"`
	Replica RoleSpec                     `json:"replica,omitempty"`
}

type PintaJobType string

const (
	Symmetric    PintaJobType = "symmetric"
	PSWorker     PintaJobType = "ps-worker"
	MPI          PintaJobType = "mpi"
	ImageBuilder PintaJobType = "image-builder"
)

type RoleSpec struct {
	NodeType  string          `json:"nodeType,omitempty"`
	Spec      v1.PodSpec      `json:"spec,omitempty"`
	Resources v1.ResourceList `json:"resources,omitempty"`
}

type PintaJobStatus struct {
	State              PintaJobState `json:"state,omitempty"`
	LastTransitionTime metav1.Time   `json:"lastTransitionTime,omitempty"`
	NumMasters         int32         `json:"numMasters,omitempty"`
	NumReplicas        int32         `json:"numReplicas,omitempty"`
}

type PintaJobState string

const (
	Idle      PintaJobState = "Idle"
	Scheduled PintaJobState = "Scheduled"
	Running   PintaJobState = "Running"
	Preempted PintaJobState = "Preempted"
	Completed PintaJobState = "Completed"
	Failed    PintaJobState = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooList is a list of Foo resources
type PintaJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PintaJob `json:"items"`
}
