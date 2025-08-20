// Copyright 2025 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package profileclient

import (
	"context"
	"fmt"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterinventory "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSecretNameTemplate = "%s.%s" // namespace.name, from ClusterProfile syncer.
)

// ProfileClient is a client that periodically fetches and caches ClusterProfile information.
type ProfileClient struct {
	// K8s client for fetching ClusterProfiles.
	client client.Client
}

// Result is the plugin response, identifying the cluster.
type Result struct {
	// Name is the name of the Secret corresponding to the cluster.
	// The Secrets are generated in the argocd namespace by the ClusterProfile syncer.
	Name string `json:"name"`
}

// NewProfileClient creates a new ProfileSync and starts its periodic reconciliation.
func NewProfileClient(ctx context.Context, scheme *runtime.Scheme) (*ProfileClient, error) {
	var config *rest.Config
	var err error
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from KUBECONFIG: %w", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in cluster config: %w", err)
		}
	}
	// The client needs to know about the clusterinventory scheme.
	client, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create CRD client: %w", err)
	}

	c := &ProfileClient{
		client: client,
	}
	return c, nil
}

// PluginResults returns the results of the plugin, which are the names of the
// Secrets corresponding to the ClusterProfiles in namespace matching labelSelector.
func (c *ProfileClient) PluginResults(ctx context.Context, namespaces []string, selector *metav1.LabelSelector) ([]Result, error) {
	log.Printf("PluginResults called for namespaces %s and selector %v\n", namespaces, selector)
	profiles, err := c.listClusterProfiles(ctx, namespaces, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster profiles: %w", err)
	}
	var ret []Result
	for _, profile := range profiles {
		ret = append(ret, Result{
			Name: fmt.Sprintf(clusterSecretNameTemplate, profile.Namespace, profile.Name),
		})
	}
	return ret, nil
}

func (c *ProfileClient) listClusterProfiles(ctx context.Context, namespaces []string, selector *metav1.LabelSelector) ([]clusterinventory.ClusterProfile, error) {
	var clusterProfiles []clusterinventory.ClusterProfile
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to convert selector: %w", err)
	}
	for _, namespace := range namespaces {
		profileList := &clusterinventory.ClusterProfileList{}
		listOptions := &client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labelSelector,
		}
		if err := c.client.List(ctx, profileList, listOptions); err != nil {
			return nil, fmt.Errorf("failed to list cluster profiles: %w", err)
		}
		log.Printf("Found %d cluster profiles in namespace %s\n", len(profileList.Items), namespace)
		clusterProfiles = append(clusterProfiles, profileList.Items...)
	}
	return clusterProfiles, nil
}
