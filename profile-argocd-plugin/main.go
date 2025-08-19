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

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cluster-inventory-api/argocd-profile-plugin/profileclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clusterinventory "sigs.k8s.io/cluster-inventory-api/apis/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clusterinventory.AddToScheme(scheme))
}

type server struct {
	profileClient *profileclient.ProfileClient
}

func main() {
	log.Printf("Starting ClusterProfile argocd plugin...\n")
	portNum := os.Getenv("PORT")
	if portNum == "" {
		log.Fatal("ENV var PORT not found")
	}
	// Create the profile client.
	ctx := context.Background()
	var err error
	profileClient, err := profileclient.NewProfileClient(ctx, scheme)
	if err != nil {
		log.Printf("Error creating client: %v\n", err)
		log.Fatal(err)
	}
	s := server{profileClient: profileClient}
	http.HandleFunc("/api/v1/getparams.execute", s.Reply)

	// Start the service.
	log.Println("Started on port", portNum)
	err = http.ListenAndServe(portNum, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// PluginRequest is the request object sent to the generator plugin service.
type PluginRequest struct {
	// ApplicationSetName is the appSetName of the ApplicationSet, used for logging.
	ApplicationSetName string `json:"applicationSetName"`
	// Input contains the parameter values for this plugin, specified in the ApplicationSet spec.
	Input Input `json:"input"`
}

// Input contains the parameter values for this plugin, specified in the ApplicationSet spec.
type Input struct {
	Parameters ParametersRequest `json:"parameters"`
}

// ParametersRequest has the parameters for the plugin.
type ParametersRequest struct {
	ClusterProfileNamespaces []string             `json:"clusterProfileNamespaces"`
	Selector                 *metav1.LabelSelector `json:"selector,omitempty"`
}

// PluginResponse is the response object returned by the plugin service.
type PluginResponse struct {
	// Output contains the outputs returned by the plugin.
	Output Output `json:"output"`
}

// Output contains the outputs returned by the plugin.
type Output struct {
	Parameters []profileclient.Result `json:"parameters"`
}

// Reply is the handler for the generator plugin.
func (s *server) Reply(w http.ResponseWriter, r *http.Request) {
	// Decode incoming plugin request.
	var request PluginRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate parameters.
	log.Printf("Plugin request received for ApplicationSet: %q", request.ApplicationSetName)
	namespaces := request.Input.Parameters.ClusterProfileNamespaces
	if len(namespaces) == 0 {
		http.Error(w, "Missing required parameter ClusterProfileNamespaces", http.StatusBadRequest)
		return
	}
	selector := request.Input.Parameters.Selector
	if selector == nil {
		// Default to everything.
		selector = &metav1.LabelSelector{}
	}

	// Get the plugin results (list of secrets for each selected ClusterProfile).
	res, err := s.profileClient.PluginResults(r.Context(), namespaces, selector)
	if err != nil {
		log.Printf("Plugin error: %v\n", err)
		http.Error(w, "Plugin error", http.StatusInternalServerError)
		return
	}

	// Marshal plugin response.
	response := PluginResponse{
		Output{
			Parameters: res,
		},
	}
	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		log.Printf("Error writing HTTP reply: %v\n", err)
		http.Error(w, "Error writing HTTP reply", http.StatusInternalServerError)
	}

	log.Printf("Plugin response sent: %+v", response)
	log.Println("-------------------------------------------")
}
