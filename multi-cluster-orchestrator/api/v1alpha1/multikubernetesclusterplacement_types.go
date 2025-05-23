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

// MultiKubernetesClusterPlacementSpec defines the desired state of MultiKubernetesClusterPlacement
type MultiKubernetesClusterPlacementSpec struct {
	// ClusterSelectorRules defines a pipeline of rules which consume an ordered list
	// of clusters and output an ordered list of clusters to generate the list of
	// current target clusters.
	ClusterSelectorRules []PlacementClusterSelectorRule `json:"clusterSelectorRules"`
	// Scaling defines the scaling configuration of the placement. If not specified,
	// the placement will use all eligible clusters.
	Scaling Scaling `json:"scaling,omitempty"`
}

// Scaling defines the scaling configuration of the placement. If not specified,
// the placement will use all eligible clusters. Only one of the scaling configurations
// can be specified.
type Scaling struct {
	// AutoscaleForCapacity defines the configuration for scaling based on ability to obtain capacity.
	AutoscaleForCapacity *AutoscaleForCapacity `json:"autoscaleForCapacity,omitempty"`
}

// AutoscaleForCapacity defines the configuration for scaling based on ability to obtain capacity.
type AutoscaleForCapacity struct {
	// MinClustersBelowCapacityCeiling defines the minimum required number of
	// clusters that are both currently active and able to scale up.
	// +kubebuilder:validation:Minimum=1
	MinClustersBelowCapacityCeiling int `json:"minClustersBelowCapacityCeiling"`
	// MaxClusters defines the maximum number of clusters that can be used to
	// run the workload. If zero or not specified, no maximum is enforced.
	MaxClusters int `json:"maxClusters,omitempty"`
	// WorkloadDetails describes the deployment which is being autoscaled as
	// part of the workload. This information is used to query metrics from each
	// target cluster about the deployment's current ability to scale up/down on
	// that cluster.
	WorkloadDetails *WorkloadDetails `json:"workloadDetails,omitempty"`
	// Indicates that the controller should assume clusters in the SCALING_IN
	// state will be proactively drained by a draining system.
	// TODO: this is hacky, we should revisit it
	UseDraining bool `json:"useDraining,omitempty"`
}

// WorkloadDetails describes the deployment which scaling decisions should be based on.
type WorkloadDetails struct {
	// Namespace is the namespace of the workload to be placed.
	Namespace string `json:"namespace,omitempty"`
	// DeploymentName is the name of the deployment to be placed.
	DeploymentName string `json:"deploymentName"`
	// HPAName is the name of the HPA to be placed.
	HPAName string `json:"hpaName"`
}

// MultiKubernetesClusterPlacementStatus defines the observed state of MultiKubernetesClusterPlacement
type MultiKubernetesClusterPlacementStatus struct {
	// Clusters is the list of clusters consumed by the workload delivery system
	Clusters               []PlacementCluster `json:"clusters,omitempty"`
	LastAdditionTime       metav1.Time        `json:"lastAdditionTime,omitempty"`
	LastClusterRemovalTime metav1.Time        `json:"lastClusterRemovalTime,omitempty"`
}

// +kubebuilder:validation:Enum=ACTIVE;SCALING_IN;UNHEALTHY;EVICTING
type PlacementClusterState string

const (
	// Active cluster running the workload
	StateActive PlacementClusterState = "ACTIVE"
	// Cluster running the workload but pending to be removed due to scale in
	StateScalingIn PlacementClusterState = "SCALING_IN"
	// Cluster running the workload but pending to be removed due to no longer
	// being eligible to run the workload
	StateEvicting PlacementClusterState = "EVICTING"
	// Cluster running the workload but pending to be removed due to the
	// workload being unhealthy on the cluster
	StateUnhealthy PlacementClusterState = "UNHEALTHY"
)

// PlacementCluster describes a cluster on which the workload should be placed.
// This is consumed by workload delivery systems.
type PlacementCluster struct {
	Name               string                `json:"name"`
	Namespace          string                `json:"namespace,omitempty"`
	State              PlacementClusterState `json:"state"`
	LastTransitionTime metav1.Time           `json:"lastTransitionTime"`
	// Whether the cluster is currently at its capacity ceiling. For informational/debugging purposes only. TODO: remove this
	AtCapacityCeiling *bool `json:"atCapacityCeiling,omitempty"`
	// TODO: To make draining pluggable, this should likely be a separate CRD owned by a separate controller
	Draining *PlacementClusterDraining `json:"draining,omitempty"`
}

type PlacementClusterDraining struct {
	// Desired maximum replicas for a draining cluster. The drainer plugin should
	// reconcile the HPA's max replicas field to this value.
	DesiredMaxReplicas int `json:"desiredMaxReplicas"`
	// The last time DesiredMaxReplicas was decreased.
	LastReplicaCountDecrease metav1.Time `json:"lastReplicaCountDecrease"`
	// For informational/debugging purposes only. TODO: remove this
	CurrentReplicaCount int `json:"currentReplicaCount,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiKubernetesClusterPlacement is the Schema for the multikubernetesclusterplacements API
type MultiKubernetesClusterPlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiKubernetesClusterPlacementSpec   `json:"spec,omitempty"`
	Status MultiKubernetesClusterPlacementStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiKubernetesClusterPlacementList contains a list of MultiKubernetesClusterPlacement
type MultiKubernetesClusterPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiKubernetesClusterPlacement `json:"items"`
}

// +kubebuilder:validation:Enum=all-clusters;cluster-list;cluster-name-regex
type PlacementClusterSelectorRuleType string

const (
	RuleTypeAllClusters      PlacementClusterSelectorRuleType = "all-clusters"
	RuleTypeClusterList      PlacementClusterSelectorRuleType = "cluster-list"
	RuleTypeClusterNameRegex PlacementClusterSelectorRuleType = "cluster-name-regex"
)

// PlacementClusterSelectorRule defines a rule which takes an ordered list of
// clusters and returns a list of clusters based on the rule type and arguments.
type PlacementClusterSelectorRule struct {
	// Type specifies the rule type and may be one of:
	// - all-clusters: all clusters (as defined by the ClusterProfiles)
	// - cluster-list: a user-provided comma-separated ordered list of clusters in
	//   the format cluster-inventory-ns/cluster-name (for each cluster, a
	//   ClusterProfile with the given name must exist in the given namespace)
	// - cluster-name-regex: a user-provided regular expression to match cluster
	//   names against (the available cluster names for matching are defined by
	//   the ClusterProfiles)
	Type PlacementClusterSelectorRuleType `json:"type"`
	// Arguments are specific to each rule type. Here are example usages for the
	// supported arguments:
	// - cluster-list:
	//   clusters: "cluster-inventory-ns/cluster1,cluster-inventory-ns/cluster2,cluster-inventory-ns/cluster3"
	// - cluster-name-regex
	//   regex: "cluster-inventory-ns/cluster\d+"
	Arguments map[string]string `json:"arguments,omitempty"`
}

func init() {
	SchemeBuilder.Register(&MultiKubernetesClusterPlacement{}, &MultiKubernetesClusterPlacementList{})
}
