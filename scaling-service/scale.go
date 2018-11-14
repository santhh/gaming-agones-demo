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
	gce "cloud.google.com/go/compute/metadata"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"

	"log"
	"math"
	"strconv"
	"time"
)

type PlayerCount struct {
	TimeSeries []struct {
		Points []struct {
			Value struct {
				Int64Value string `json:"int64Value"`
			} `json:"value"`
		} `json:"points"`
	} `json:"timeSeries"`
}

var (
	agonesClient           = getAgonesClient()
	alreadyFasCreated bool = false
)

func main() {

	query_interval := flag.String("interval", "60", "sd export interval")
	defaultBufferSize := flag.Int("defaultReplica", 2, "default buffer size")
	namespace := flag.String("namespace", "default", "gke namespace")
	metricType := flag.String("metricType", "custom.googleapis.com/gameservers/playercount", "custom metrics name")
	fleetname := flag.String("fleetName", "my-game-server-fleet", "agones game server fleet")
	fleeetautoscalername := flag.String("fasName", "fleet-autoscaler-my-game-server ", "agones game server fleet auto scaler name")
	flag.Parse()
	duration, _ := strconv.Atoi(*query_interval)
	currentBufferSize := *defaultBufferSize
	stackdriverService, err := getStackDriverService()

	if err != nil {
		log.Fatalf("Error getting Stackdriver service: %v", err)
	}

	for {

		alreadyFasCreated, currentBufferSize = checkFasStatus(*namespace, *fleeetautoscalername, *defaultBufferSize)
		playersCount, readyReplicas, err := readTimeSeriesValue(stackdriverService, *metricType, *namespace, *fleetname)
		log.Printf("***Player Count %v *** Ready Replicas %v Error:%v", playersCount, readyReplicas, err)
		if err != nil {
			log.Fatalf("Could not read timeseries data %v", err)
		}

		targetBufferSize := desireBufferSize(playersCount, *defaultBufferSize)

		if targetBufferSize != currentBufferSize {
			log.Printf("**Auto Scaling Should Trigger*** Target Buffer: %v Current Buffer: %v\n", targetBufferSize, currentBufferSize)
			if !alreadyFasCreated {

				alreadyFasCreated = createFas(*namespace, *fleetname, *fleeetautoscalername, *defaultBufferSize)

			}
			versionId := getLatestVersionForScalerUpdate(*namespace, *fleeetautoscalername)
			log.Printf("Latest Resource Id: %v", versionId)
			currentBufferSize = performAutoScale(targetBufferSize, versionId, *namespace, *fleetname, *fleeetautoscalername)
		}
		log.Printf("**Auto Scaling Remain Same*** Target Buffer: %v Current Buffer: %v\n", targetBufferSize, currentBufferSize)
		time.Sleep(time.Duration(duration) * time.Second)
	}

}

func checkFasStatus(namespace string, fleeetautoscalername string, defaultBufferSize int) (bool, int) {

	fleetAutoscalerInterface := agonesClient.StableV1alpha1().FleetAutoscalers(namespace)

	fas, err := fleetAutoscalerInterface.Get(fleeetautoscalername, v1.GetOptions{})

	if err != nil {
		log.Printf("FAS Status: %v", err)
		return false, defaultBufferSize
	}

	bufferSize := fas.Spec.Policy.Buffer.BufferSize.IntValue()
	log.Printf("FAS Status %v", bufferSize)
	return true, bufferSize
}

func createFas(namespace string, fleetname string, fleeetautoscalername string, defaultBufferSize int) bool {

	log.Print("Namespace: %v Fleet: %v Fleet Auto Scaler: %v", namespace, fleetname, fleeetautoscalername)

	fleetAutoscalerInterface := agonesClient.StableV1alpha1().FleetAutoscalers(namespace)
	fas := &v1alpha1.FleetAutoscaler{
		ObjectMeta: v1.ObjectMeta{Name: fleeetautoscalername},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: fleetname,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					BufferSize:  intstr.FromInt(defaultBufferSize),
					MaxReplicas: 50,
					MinReplicas: int32(defaultBufferSize + 1),
				},
			},
		},
	}
	newFleetAutoscaler, err := fleetAutoscalerInterface.Create(fas)

	if err != nil {
		log.Fatalf("Could not create fleet auto scaler %v", err)
		return false
	}
	log.Printf("***Fleet Auto Scalar Created*** %v", newFleetAutoscaler)
	return true
}

func getLatestVersionForScalerUpdate(namespace string, fleeetautoscalername string) string {
	fleetAutoscalerInterface := agonesClient.StableV1alpha1().FleetAutoscalers(namespace)

	fas, err := fleetAutoscalerInterface.Get(fleeetautoscalername, v1.GetOptions{})

	if err != nil {
		log.Fatalf("Could not update fleet auto scaler %v", err)
	}
	return fas.ResourceVersion

}

func desireBufferSize(pc int, defaultBufferSize int) int {

	desireReplicas := float64(pc) / float64(defaultBufferSize)
	desireBufferSize := int(math.Ceil(desireReplicas))
	if desireBufferSize < defaultBufferSize {
		desireBufferSize = defaultBufferSize
	}
	return desireBufferSize

}
func performAutoScale(desireBufferSize int, versionId string, namespace string, fleetname string, fleeetautoscalername string) int {

	fleetAutoscalerInterface := agonesClient.StableV1alpha1().FleetAutoscalers(namespace)
	fas := &v1alpha1.FleetAutoscaler{
		ObjectMeta: v1.ObjectMeta{Name: fleeetautoscalername, ResourceVersion: versionId},
		Spec: v1alpha1.FleetAutoscalerSpec{
			FleetName: fleetname,
			Policy: v1alpha1.FleetAutoscalerPolicy{
				Type: v1alpha1.BufferPolicyType,
				Buffer: &v1alpha1.BufferPolicy{
					BufferSize:  intstr.FromInt(desireBufferSize),
					MaxReplicas: 50,
				},
			},
		},
	}
	newFleetAutoscaler, err := fleetAutoscalerInterface.Update(fas)

	if err != nil {
		log.Fatalf("Could not update fleet auto scaler %v", err)
	}

	value := newFleetAutoscaler.Spec.Policy.Buffer.BufferSize
	currentBufferSize := value.IntValue()
	log.Printf("Successfully Updated Buffer Size to %v", currentBufferSize)
	return currentBufferSize

}
func getStackDriverService() (*monitoring.Service, error) {
	oauthClient := oauth2.NewClient(context.Background(), google.ComputeTokenSource(""))
	return monitoring.New(oauthClient)
}
func projectResource() string {

	projectId, err := gce.ProjectID()
	if err != nil {
		log.Fatalf("Could not find project id %v", err)
	}
	return "projects/" + projectId
}

func readTimeSeriesValue(s *monitoring.Service, metricType string, namespace string, fleetname string) (int, int32, error) {

	endTime := time.Now().UTC()

	resp, err := s.Projects.TimeSeries.List(projectResource()).
		Filter(fmt.Sprintf("metric.type=\"%s\"", metricType)).
		IntervalEndTime(endTime.Format(time.RFC3339Nano)).
		AggregationAlignmentPeriod("60s").
		AggregationCrossSeriesReducer("REDUCE_SUM").
		AggregationPerSeriesAligner("ALIGN_SUM").
		AggregationGroupByFields("resource.labels.cluster_name").
		Fields("timeSeries.points.value").
		Do()

	if err != nil {
		return 0, 0, fmt.Errorf("Could not read time series value, %v ", err)
	}

	numberOfPlayers := PlayerCount{}
	json.Unmarshal(formatResource(resp), &numberOfPlayers)
	playerCount, _ := strconv.Atoi(numberOfPlayers.TimeSeries[0].Points[0].Value.Int64Value)
	return playerCount, checkReadyReplicas(namespace, fleetname), nil
}
func formatResource(resource interface{}) []byte {
	b, err := json.MarshalIndent(resource, "", "    ")
	if err != nil {
		panic(err)
	}

	return b
}
func checkReadyReplicas(namespace string, fleetname string) int32 {
	// Get a FleetInterface for this namespace
	fleetInterface := agonesClient.StableV1alpha1().Fleets(namespace)
	// Get our fleet
	fleet, err := fleetInterface.Get(fleetname, v1.GetOptions{})
	if err != nil {
		log.Printf("Get fleet failed %v", err)
	}
	return fleet.Status.ReadyReplicas
}
func getAgonesClient() *versioned.Clientset {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Could not create in cluster config %v", err)
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not create the agones api clientset %v", err)

	} else {
		log.Println("Created the agones api clientset")
	}
	return agonesClient
}
