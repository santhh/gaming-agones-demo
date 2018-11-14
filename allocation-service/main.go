// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime" // for the logger
	"encoding/json"
	"errors"
	"flag"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"net/http"
)

const generatename = "canada-swarm-gameserver-"

var (
	logger       = runtime.NewLoggerWithSource("main")
	agonesClient = getAgonesClient()
	namespace    string
	fleetname    string
)

// A handler for the web server
type handler func(w http.ResponseWriter, r *http.Request)

// The structure of the json response
type result struct {
	Status v1alpha1.GameServerStatus `json:"status"`
}

// Main will set up an http server, fetch the ip and port of the allocated
// gameserver set, and return json a string of GameServerStatus
func main() {

	flag.StringVar(&namespace, "namespace", "default", "namespace")
	flag.StringVar(&fleetname, "fleetName", "my-game-server-fleet", "agones game server fleet")
	flag.Parse()
	// Serve 200 status on / for k8s health checks
	http.HandleFunc("/", handleRoot)
	logger.Info("****Custom Log:* handle root completed***")
	// Serve 200 status on /healthz for k8s health checks
	http.HandleFunc("/healthz", handleHealthz)
	logger.Info("****Custom Log:* handle health completed***")
	// Return the GameServerStatus of the allocated replica to the authorized client
	http.HandleFunc("/address", getOnly(basicAuth(handleAddress)))
	logger.Info("****Custom Log:* handle data completed***")

	if err := http.ListenAndServe(":8000", nil); err != nil {
		logger.WithError(err).Fatal("HTTP server failed to run")
	} else {
		logger.Info("HTTP server is running on port 8000")
	}

}
func getAgonesClient() *versioned.Clientset {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}
	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	} else {
		logger.Info("Created the agones api clientset")
	}
	return agonesClient
}

// Limit verbs the web server handles
func getOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h(w, r)
			return
		}
		http.Error(w, "Get Only", http.StatusMethodNotAllowed)
	}
}

// Let the web server do basic authentication
func basicAuth(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		key, value, _ := r.BasicAuth()
		if key != "v1GameClientKey" || value != "EAEC945C371B2EC361DE399C2F11E" {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}
		pass(w, r)
	}
}

// Let / return Healthy and status code 200
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /")
	}
}

// Let /healthz return Healthy and status code 200
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /healthz")
	}
}

// Let /address return the GameServerStatus
func handleAddress(w http.ResponseWriter, r *http.Request) {
	status, err := allocate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(&result{status})
	_, err = io.WriteString(w, string(result))
	if err != nil {
		logger.WithError(err).Fatal("Error writing json from /address")
	}
}

// Return the number of game servers available to this fleet for allocation
func checkReadyReplicas() int32 {
	// Get a FleetInterface for this namespace
	fleetInterface := agonesClient.StableV1alpha1().Fleets(namespace)
	// Get our fleet
	fleet, err := fleetInterface.Get(fleetname, v1.GetOptions{})

	if err != nil {
		logger.WithError(err).Info("Get fleet failed")

	}

	return fleet.Status.ReadyReplicas
}

// Move a replica from ready to allocated and return the GameServerStatus
func allocate() (v1alpha1.GameServerStatus, error) {
	var result v1alpha1.GameServerStatus
	// Log the values used in the fleet allocation
	logger.WithField("namespace", namespace).Info("namespace for fa")
	logger.WithField("generatename", generatename).Info("generatename for fa")
	logger.WithField("fleetname", fleetname).Info("fleetname for fa")
	// Find out how many ready replicas the fleet has
	readyReplicas := checkReadyReplicas()
	logger.WithField("readyReplicas", readyReplicas).Info("numer of ready replicas")
	// Return and log an error if there are no ready replicas
	if readyReplicas < 1 {
		logger.WithField("fleetname", fleetname).Info("Insufficient ready replicas, cannot create fleet allocation")
		return result, errors.New("Insufficient ready replicas, cannot create fleet allocation")
	}
	// Get a FleetAllocationInterface for this namespace
	fleetAllocationInterface := agonesClient.StableV1alpha1().FleetAllocations(namespace)

	// Define the fleet allocation
	fa := &v1alpha1.FleetAllocation{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: generatename, Namespace: namespace,
		},
		Spec: v1alpha1.FleetAllocationSpec{FleetName: fleetname},
	}
	// Create a new fleet allocation
	newFleetAllocation, err := fleetAllocationInterface.Create(fa)
	if err != nil {
		// Return and log the error
		logger.WithError(err).Info("Failed to create fleet allocation")
		return result, errors.New("Failed to ceate fleet allocation")
	}
	logger.WithField("Result", newFleetAllocation).Info("Pre Result Check for Status")
	result = newFleetAllocation.Status.GameServer.Status
	return result, nil
}
