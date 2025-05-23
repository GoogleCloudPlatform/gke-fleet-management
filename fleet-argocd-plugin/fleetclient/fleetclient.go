// Copyright 2024 Google LLC
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
package fleetclient

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	fleet "google.golang.org/api/gkehub/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// Fleet API service poll interval.
	reconcileInterval = 10 * time.Second
	// Template for the Kubernetes Secret name, {{.MembershipID}}.{{.Region}}.{{.ProjectNum}}.
	clusterSecretNameTemplate = "%s.%s.%s"
	// Template for the Kubernetes Secret manifest.
	clusterSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.Name}}
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: cluster
  annotations:
    fleet.gke.io/managed-by-fleet-plugin: "true"
type: Opaque
stringData:
  name: {{.Name}}
  server: {{.ConnectGatewayURL}}
  config: |
    {
      "execProviderConfig": {
        "command": "argocd-k8s-auth",
        "args": ["gcp"],
        "apiVersion": "client.authentication.k8s.io/v1beta1"
      },
      "tlsClientConfig": {
        "insecure": false,
        "caData": ""
      }
    }
`
)

// FleetSync is a client that periodically polls the GKE Fleet API and caches fleet information.
type FleetSync struct {
	svc *fleet.Service
	// GCP project number of fleet host project.
	ProjectNum string
	// A cached map from Membership full resource name to a list of Scope IDs.
	MembershipTenancyMapCache map[string][]string
	// A cached map from Scope IDs to a list of Membership full resource names.
	ScopeTenancyMapCache map[string][]string
}

// NewFleetSync creates a new FleetSync and starts its periodical reconciliation.
func NewFleetSync(ctx context.Context, projectNum string) (*FleetSync, error) {
	service, err := fleet.NewService(ctx)
	if err != nil {
		return nil, err
	}
	c := &FleetSync{
		svc:        service,
		ProjectNum: projectNum,
	}

	// Build the initial fleet topology before handling RPCs.
	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.startReconcile(ctx)
	return c, nil
}

func (c *FleetSync) startReconcile(ctx context.Context) {
	go func() {
		for {
			time.Sleep(reconcileInterval)
			if err := c.Refresh(ctx); err != nil {
				fmt.Printf("Error refreshing fleet: %v\n", err)
			}
		}
	}()
}

// Result encapsulates the response from the fleet service.
type Result struct {
	ServerURL string `json:"server"`
	Name      string `json:"name"`
	NameShort string `json:"nameShort"`
}

// PluginResults returns the results of the plugin.
func (c *FleetSync) PluginResults(ctx context.Context, scopeID string) ([]Result, error) {
	if c.MembershipTenancyMapCache == nil || c.ScopeTenancyMapCache == nil {
		return nil, fmt.Errorf("fleet is empty")
	}
	var results []Result

	// Scope mode. Only include memberships in the specified scope.
	if scopeID != "" {
		if c.ScopeTenancyMapCache[scopeID] == nil {
			return nil, fmt.Errorf("unknown scope ID to the Fleet plugin: %s", scopeID)
		}
		for _, name := range c.ScopeTenancyMapCache[scopeID] {
			results = append(results, resultFromMembership(name, c.ProjectNum))
		}
		return results, nil
	}

	// Include all member clusters in the Fleet.
	for name := range c.MembershipTenancyMapCache {
		results = append(results, resultFromMembership(name, c.ProjectNum))
	}
	return results, nil
}

func resultFromMembership(name, projectNum string) Result {
	parts := strings.Split(name, "/")
	region, membershipID := parts[3], parts[5]
	return Result{
		ServerURL: connectGatewayURL(projectNum, region, membershipID),
		Name:      fmt.Sprintf(clusterSecretNameTemplate, membershipID, region, projectNum),
		NameShort: fmt.Sprint(membershipID),
	}
}

func connectGatewayURL(projectNum, region, membershipID string) string {
	if region == "global" {
		return fmt.Sprintf("https://connectgateway.googleapis.com/v1/projects/%s/locations/%s/gkeMemberships/%s", projectNum, region, membershipID)
	}
	return fmt.Sprintf("https://%s-connectgateway.googleapis.com/v1/projects/%s/locations/%s/gkeMemberships/%s", region, projectNum, region, membershipID)
}

// Refresh polls fleet API, rebuilds the local cached fleet topology map, and updates cluster secrets.
func (c *FleetSync) Refresh(ctx context.Context) error {
	mems, err := c.listMemberships(ctx, c.ProjectNum)
	if err != nil {
		return fmt.Errorf("failed to list memberships: %w", err)
	}

	scopes, err := c.listScopes(ctx, c.ProjectNum)
	if err != nil {
		return fmt.Errorf("failed to list scopes: %w", err)
	}

	mbs, err := c.listMembershipBindings(ctx, c.ProjectNum)
	if err != nil {
		return fmt.Errorf("failed to list membership bindings: %w", err)
	}

	// Build one map from Memberships to a list of Scopes that the membership cluster is associated with,
	// and one reverse indexed map from Scopes to Memberships.
	memTenancyMap := make(map[string][]string)
	for _, mem := range mems {
		membershipName := mem.Name
		memTenancyMap[membershipName] = make([]string, 0)
	}

	scopeTenancyMap := make(map[string][]string)
	for _, s := range scopes {
		scopeID := s.Name
		scopeTenancyMap[scopeID] = make([]string, 0)
	}

	for _, binding := range mbs {
		// bindingName is in the format of
		// `projects/{project}/locations/{location}/memberships/{membership}/bindings/{membershipbinding}`
		bindingName := binding.Name
		parts := strings.Split(bindingName, "/")
		if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "memberships" || parts[6] != "bindings" {
			fmt.Printf("Invalid binding resource name format: %s\n", bindingName)
			continue
		}

		// Add the scope to the list for this membership
		membership := strings.Join(parts[:6], "/")
		scopeParts := strings.Split(binding.Scope, "/")
		if len(scopeParts) == 0 {
			fmt.Printf("Invalid scope in binding (%s): %s\n", bindingName, binding.Scope)
			continue
		}

		scope := scopeParts[len(scopeParts)-1]
		memTenancyMap[membership] = append(memTenancyMap[membership], scope)
		scopeTenancyMap[scope] = append(scopeTenancyMap[scope], membership)
	}

	// Refresh cache.
	c.MembershipTenancyMapCache = memTenancyMap
	c.ScopeTenancyMapCache = scopeTenancyMap

	// Update cluster Secrets.
	if err := c.reconcileClusterSecrets(ctx); err != nil {
		return fmt.Errorf("failed to reconcile cluster secrets: %w", err)
	}
	return nil
}

func (c *FleetSync) reconcileClusterSecrets(ctx context.Context) error {
	// Create a Kubernetes clientset to apply resources.
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Construct a map of cluster secrets, from name to manifest.
	clusterSecrets := make(map[string]string)
	for membership := range c.MembershipTenancyMapCache {
		parts := strings.Split(membership, "/")
		secretName := fmt.Sprintf(clusterSecretNameTemplate, parts[5], parts[3], c.ProjectNum)
		param := struct {
			Name              string
			ConnectGatewayURL string
		}{
			Name:              secretName,
			ConnectGatewayURL: connectGatewayURL(c.ProjectNum, parts[3], parts[5]),
		}
		tmpl, err := template.New("secret").Parse(clusterSecretTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse template: %w", err)
		}
		var secretManifest bytes.Buffer
		err = tmpl.Execute(&secretManifest, param)
		if err != nil {
			fmt.Println("Error creating Secret manifest:", err)
			continue
		}
		clusterSecrets[secretName] = secretManifest.String()
	}
	fmt.Printf("Reconciling Cluster Secrets: %v\n", clusterSecrets)

	// Apply the Secret to the cluster.
	err = applySecrets(ctx, clientset, clusterSecrets)
	if err != nil {
		return fmt.Errorf("failed to apply secret: %w", err)
	}

	// Prune cluster secrets that are no longer existing in the Fleet.
	return pruneSecrets(ctx, clientset, clusterSecrets)
}

func applySecrets(ctx context.Context, clientset *kubernetes.Clientset, clusterSecrets map[string]string) error {
	secretsClient := clientset.CoreV1().Secrets("argocd")
	for _, manifest := range clusterSecrets {
		secret, err := secretFromManifest(manifest)
		if err != nil {
			return fmt.Errorf("error converting manifest %q to a k8s secret: %v", manifest, err)
		}
		_, err = secretsClient.Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			// Check if "already exists", then update.
			if !errors.IsAlreadyExists(err) {
				return fmt.Errorf("error creating secret: %v", err)
			}
			_, err = secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("error updating secret: %v", err)
			}
		}
	}
	fmt.Println("Successfully applied Secrets.")
	return nil
}

func pruneSecrets(ctx context.Context, clientset *kubernetes.Clientset, clusterSecrets map[string]string) error {
	secretsClient := clientset.CoreV1().Secrets("argocd")
	listOptions := metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=cluster",
	}

	existingSecrets, err := secretsClient.List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, secret := range existingSecrets.Items {
		// Skip secrets that are not managed by the fleet plugin.
		if secret.Annotations["fleet.gke.io/managed-by-fleet-plugin"] != "true" {
			continue
		}
		if _, exists := clusterSecrets[secret.Name]; !exists {
			// Secret no longer corresponds to a membership, delete it.
			err := secretsClient.Delete(ctx, secret.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}
		}
	}

	fmt.Println("Successfully pruned Secrets.")
	return nil
}

func secretFromManifest(manifest string) (*corev1.Secret, error) {
	// Universal deserializer can handle various Kubernetes object formats
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding to scheme: %v", err)
	}

	decode := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode
	obj, _, err := decode([]byte(manifest), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding manifest %q: %v", manifest, err)
	}
	// Type assertion to ensure it's a corev1.Secret
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("decoded object is not of type Secret")
	}
	return secret, nil
}

// listMemberships fetches the memberships under a given parent.
func (c *FleetSync) listMemberships(ctx context.Context, project string) ([]*fleet.Membership, error) {
	var ret []*fleet.Membership
	parent := fmt.Sprintf("projects/%s/locations/-", project)
	call := c.svc.Projects.Locations.Memberships.List(parent)
	err := call.Pages(ctx, func(resp *fleet.ListMembershipsResponse) error {
		ret = append(ret, resp.Resources...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// listScopes fetches the scopes under a given parent.
func (c *FleetSync) listScopes(ctx context.Context, project string) ([]*fleet.Scope, error) {
	var ret []*fleet.Scope
	parent := fmt.Sprintf("projects/%s/locations/global", project)
	call := c.svc.Projects.Locations.Scopes.List(parent)
	err := call.Pages(ctx, func(resp *fleet.ListScopesResponse) error {
		ret = append(ret, resp.Scopes...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// listMembershipBindings fetches the membership bindings under a given parent.
func (c *FleetSync) listMembershipBindings(ctx context.Context, project string) ([]*fleet.MembershipBinding, error) {
	var ret []*fleet.MembershipBinding
	parent := fmt.Sprintf("projects/%s/locations/-/memberships/-", project)
	call := c.svc.Projects.Locations.Memberships.Bindings.List(parent)
	err := call.Pages(ctx, func(resp *fleet.ListMembershipBindingsResponse) error {
		ret = append(ret, resp.MembershipBindings...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
