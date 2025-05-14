/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiKubernetesClusterBindingSpec defines the desired state of MultiKubernetesClusterBinding
type MultiKubernetesClusterBindingSpec struct {
	SourceRef    SourceRef    `json:"sourceRef"`
	PlacementRef PlacementRef `json:"placementRef"`
}

// MultiKubernetesClusterBindingStatus defines the observed state of MultiKubernetesClusterBinding
type MultiKubernetesClusterBindingStatus struct {
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiKubernetesClusterBinding is the Schema for the multikubernetesclusterbindings API
type MultiKubernetesClusterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiKubernetesClusterBindingSpec   `json:"spec,omitempty"`
	Status MultiKubernetesClusterBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiKubernetesClusterBindingList contains a list of MultiKubernetesClusterBinding
type MultiKubernetesClusterBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiKubernetesClusterBinding `json:"items"`
}

type SourceRef struct {
	Name             string           `json:"name"`
	GroupVersionKind GroupVersionKind `json:"groupVersionKind"`
	ContentPath      string           `json:"contentPath"`
}

type PlacementRef struct {
	Name string `json:"name"`
}

type GroupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

func init() {
	SchemeBuilder.Register(&MultiKubernetesClusterBinding{}, &MultiKubernetesClusterBindingList{})
}
