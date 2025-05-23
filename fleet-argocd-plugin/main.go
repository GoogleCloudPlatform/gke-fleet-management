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

package main

import (
	"context"
	"encoding/json"
	"fleet-management-tools/argocd-sync/fleetclient"
	"fmt"      // formatting and printing values to the console.
	"log"      // logging messages to the console.
	"net/http" // Used for build HTTP servers and clients.
	"os"
)

var fleetSync *fleetclient.FleetSync

func main() {
	log.Println("Starting GKE Fleet argocd plugin...")
	projectNum := os.Getenv("FLEET_PROJECT_NUMBER")
	if projectNum == "" {
		log.Fatal("ENV var FLEET_PROJECT_NUMBER not found")
	}
	portNum := os.Getenv("PORT")
	if portNum == "" {
		log.Fatal("ENV var PORT not found")
	}
	// Start fleet client.
	ctx := context.Background()
	var err error
	fleetSync, err = fleetclient.NewFleetSync(ctx, projectNum)
	if err != nil {
		fmt.Printf("Error creating fleet client: %v\n", err)
		log.Fatal(err)
	}
	http.HandleFunc("/api/v1/getparams.execute", Reply)
	// Spinning up the server.
	log.Println("Started on port", portNum)
	fmt.Println("To close connection CTRL+C :-)")
	err = http.ListenAndServe(portNum, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// PluginRequest is the request object sent to the plugin generator service.
type PluginRequest struct {
	// ApplicationSetName is the appSetName of the ApplicationSet for which we're requesting parameters. Useful for logging in
	// the plugin service.
	ApplicationSetName string `json:"applicationSetName"`
	// Input is the map of parameters set in the ApplicationSet spec for this generator.
	Input Input `json:"input"`
}

// Input is the map of parameters set in the ApplicationSet spec for this generator.
type Input struct {
	Parameters ParametersRequest `json:"parameters"`
}

// ParametersRequest is the input variables.
type ParametersRequest struct {
	FleetProjectNumber string `json:"fleetProjectNumber"`
	ScopeID            string `json:"scopeId"`
}

// PluginResponse is the response object returned by the plugin generator service.
type PluginResponse struct {
	// Output is the map of outputs returned by the plugin.
	Output Output `json:"output"`
}

// Output is the map of outputs returned by the plugin generator.
type Output struct {
	Parameters []fleetclient.Result `json:"parameters"`
}

// Reply is the handler for the fleet plugin generator.
func Reply(w http.ResponseWriter, r *http.Request) {
	// Decode incoming plugin request.
	var request PluginRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Validate parameters.
	projectNum := request.Input.Parameters.FleetProjectNumber
	if projectNum == "" {
		http.Error(w, "Missing required parameter FleetProjectNumber", http.StatusBadRequest)
		return
	}
	if projectNum != fleetSync.ProjectNum {
		http.Error(w, "Invalid fleetProjectNumber in request, doesn't match FLEET_PROJECT_NUMBER specified in the Fleet plugin", http.StatusBadRequest)
		return
	}
	scopeID := request.Input.Parameters.ScopeID
	res, err := fleetSync.PluginResults(context.Background(), scopeID)
	if err != nil {
		fmt.Printf("Error rendering result: %v\n", err)
		http.Error(w, "Error rendering result", http.StatusInternalServerError)
	}
	// Encode plugin response.
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
		fmt.Printf("Error writing HTTP reply: %v\n", err)
		http.Error(w, "Error writing HTTP reply", http.StatusInternalServerError)
	}

	fmt.Printf("%+v\n", response)
	fmt.Println("-------------------------------------------")
}
