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
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	gce "cloud.google.com/go/compute/metadata"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const fleetname = "my-game-server-fleet"

var (
	logger       = runtime.NewLoggerWithSource("main")
	agonesClient = getAgonesClient()
)

func main() {
	// Gather pod information
	namespace := flag.String("namespace", "", "namespace")
	fleetname := flag.String("fleetName", "my-game-server-fleet", "agones game server fleet")
	podName := flag.String("pod-name", "", "pod name")
	metricName := flag.String("metric-name", "gameservers/stats", "custom metric name")
	export_interval := flag.String("interval", "10", "sd export interval")
	flag.Parse()
	duration, _ := strconv.Atoi(*export_interval)
	stackdriverService, err := getStackDriverService()

	if err != nil {
		log.Fatalf("Error getting Stackdriver service: %v", err)
	}

	newModelLabels := getResourceLabelsForNewModel(*namespace, *podName)

	for {

		numberOfReadyReplicas, numberOfAllocatedReplicas := checkReplicaStatus(*namespace, *fleetname)
		err := exportMetric(stackdriverService, *metricName, int64(numberOfReadyReplicas), int64(numberOfAllocatedReplicas), "k8s_pod", newModelLabels)
		if err != nil {
			log.Printf("Failed to write time series data for new resource model: %v\n", err)
		} else {
			log.Printf("Finished writing time series for new resource model with Ready Replicas: %v Allocated Replicas: %v\n", numberOfReadyReplicas, numberOfAllocatedReplicas)
		}

		time.Sleep(time.Duration(duration) * time.Second)

	}
}

func getStackDriverService() (*monitoring.Service, error) {
	oauthClient := oauth2.NewClient(context.Background(), google.ComputeTokenSource(""))
	return monitoring.New(oauthClient)
}

func getResourceLabelsForNewModel(namespace, name string) map[string]string {
	projectId, _ := gce.ProjectID()
	location, _ := gce.InstanceAttributeValue("cluster-location")
	location = strings.TrimSpace(location)
	clusterName, _ := gce.InstanceAttributeValue("cluster-name")
	clusterName = strings.TrimSpace(clusterName)
	return map[string]string{
		"project_id":     projectId,
		"location":       location,
		"cluster_name":   clusterName,
		"namespace_name": namespace,
		"pod_name":       name,
	}
}

func exportMetric(stackdriverService *monitoring.Service, metricName string,
	readyReplica int64, allocatedReplica int64, monitoredResource string, resourceLabels map[string]string) error {
	readydataPoint := &monitoring.Point{
		Interval: &monitoring.TimeInterval{
			EndTime: time.Now().Format(time.RFC3339),
		},
		Value: &monitoring.TypedValue{
			Int64Value: &readyReplica,
		},
	}

	allocateddataPoint := &monitoring.Point{
		Interval: &monitoring.TimeInterval{
			EndTime: time.Now().Format(time.RFC3339),
		},
		Value: &monitoring.TypedValue{
			Int64Value: &allocatedReplica,
		},
	}

	// Write time series data.
	request := &monitoring.CreateTimeSeriesRequest{
		TimeSeries: []*monitoring.TimeSeries{
			{
				Metric: &monitoring.Metric{
					Type:   "custom.googleapis.com/" + metricName,
					Labels: getMetricsResourceLabel("ready"),
				},
				Resource: &monitoring.MonitoredResource{
					Type:   monitoredResource,
					Labels: resourceLabels,
				},
				Points: []*monitoring.Point{
					readydataPoint,
				},
			},

			{
				Metric: &monitoring.Metric{
					Type:   "custom.googleapis.com/" + metricName,
					Labels: getMetricsResourceLabel("allocated"),
				},
				Resource: &monitoring.MonitoredResource{
					Type:   monitoredResource,
					Labels: resourceLabels,
				},
				Points: []*monitoring.Point{
					allocateddataPoint,
				},
			},
		},
	}

	projectName := fmt.Sprintf("projects/%s", resourceLabels["project_id"])
	_, err := stackdriverService.Projects.TimeSeries.Create(projectName, request).Do()
	return err
}

func getMetricsResourceLabel(name string) map[string]string {

	return map[string]string{
		"replicaStatus": name,
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

func checkReplicaStatus(namespace string, fleetname string) (int32, int32) {
	// Get a FleetInterface for this namespace
	fleetInterface := agonesClient.StableV1alpha1().Fleets(namespace)
	// Get our fleet
	fleet, err := fleetInterface.Get(fleetname, v1.GetOptions{})
	if err != nil {
		logger.WithError(err).Info("Get fleet failed")
	}
	return fleet.Status.ReadyReplicas, fleet.Status.AllocatedReplicas
}
