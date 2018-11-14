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
	coresdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	"agones.dev/agones/sdks/go"
	gce "cloud.google.com/go/compute/metadata"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/rs/xid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Player struct {
	playerId string
}

var (
	playerList = make(map[string]Player)
)

func main() {

	go doSignal()

	port := flag.String("port", "7654", "The port to listen to udp traffic on")
	podName := flag.String("pod-name", "", "pod name")
	namespace := flag.String("namespace", "default", "gke namespace")
	metricName := flag.String("metricName", "gameservers/playercount", "custom metrics name")
	export_interval := flag.String("interval", "10", "sd export interval")
	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}

	duration, _ := strconv.Atoi(*export_interval)

	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)

	if err != nil {
		log.Fatalf("Could not start udp server: %v", err)
	}
	defer conn.Close() // nolint: errcheck

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(s, stop)

	log.Print("Marking this server as ready")
	// This tells Agones that the server is ready to receive connections.
	err = s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}

	go doStackDriverExport(s, *podName, time.Duration(duration), *namespace, *metricName)

	readWriteLoop(conn, stop, s)

}

// doSignal shutsdown on SIGTERM/SIGKILL
func doSignal() {
	stop := signals.NewStopChannel()
	<-stop
	log.Println("Exit signal received. Shutting down.")
	os.Exit(0)
}

func readWriteLoop(conn net.PacketConn, stop chan struct{}, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)

		// check if it's logout request

		if len(strings.SplitAfter(txt, "/")) > 1 {

			deletePlayer(txt, s, conn, sender)
		}

		switch txt {

		case "NEW":
			addPlayer(s, conn, sender)
		case "PLIST":
			listPlayers(s, conn, sender)
		// shuts down the gameserver
		case "EXIT":
			exit(s)

		// turns off the health pings
		case "UNHEALTHY":
			close(stop)

		case "GAMESERVER":
			writeGameServerName(s, conn, sender)

		case "WATCH":
			watchGameServerEvents(s)

		case "LABEL":
			setLabel(s)

		case "ANNOTATION":
			setAnnotation(s)

		case "PCOUNT":
			printPlayerCount(s, conn, sender)

		}

		ack(conn, sender, txt)
	}
}

// readPacket reads a string from the connection
func readPacket(conn net.PacketConn, b []byte) (net.Addr, string) {
	n, sender, err := conn.ReadFrom(b)
	if err != nil {
		log.Fatalf("Could not read from udp stream: %v", err)
	}
	txt := strings.TrimSpace(string(b[:n]))
	log.Printf("Received packet from %v: %v", sender.String(), txt)
	return sender, txt
}

// ack echoes it back, with an ACK
func ack(conn net.PacketConn, sender net.Addr, txt string) {
	ack := "ACK From Masud's Game Server: " + txt + "\n"
	if _, err := conn.WriteTo([]byte(ack), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	os.Exit(0)
}

// writes the GameServer name to the connection UDP stream
func writeGameServerName(s *sdk.SDK, conn net.PacketConn, sender net.Addr) {
	var gs *coresdk.GameServer
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not retrieve GameServer: %v", err)
	}
	var j []byte
	j, err = json.Marshal(gs)
	if err != nil {
		log.Fatalf("error mashalling GameServer to JSON: %v", err)
	}
	log.Printf("GameServer: %s \n", string(j))
	msg := "NAME: " + gs.ObjectMeta.Name + "\n"
	if _, err = conn.WriteTo([]byte(msg), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}

// watchGameServerEvents creates a callback to log when
// gameserver events occur
func watchGameServerEvents(s *sdk.SDK) {
	err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		j, err := json.Marshal(gs)
		if err != nil {
			log.Fatalf("error mashalling GameServer to JSON: %v", err)
		}
		log.Printf("GameServer Event: %s \n", string(j))
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

// setAnnotation sets a given annotation
func setAnnotation(s *sdk.SDK) {
	log.Print("Setting annotation")

	err := s.SetAnnotation("timestamp", time.Now().UTC().String())
	if err != nil {
		log.Fatalf("could not set annotation: %v", err)
	}
}

// setLabel sets a given label
func setLabel(s *sdk.SDK) {
	log.Print("Setting label")
	// label values can only be alpha, - and .
	err := s.SetLabel("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		log.Fatalf("could not set label: %v", err)
	}
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
func addPlayer(s *sdk.SDK, conn net.PacketConn, sender net.Addr) {
	id := xid.New().String()
	//playerList = make(map[string]Player)
	playerList[id] = Player{id}
	log.Print("Added a player with id: %v", playerList[id].playerId)
	msg := "Added a player with id " + playerList[id].playerId + "\n"
	if _, err := conn.WriteTo([]byte(msg), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}

func printPlayerCount(s *sdk.SDK, conn net.PacketConn, sender net.Addr) {

	msg := strconv.Itoa(countPlayers()) + "\n"
	if _, err := conn.WriteTo([]byte(msg), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}

}

func listPlayers(s *sdk.SDK, conn net.PacketConn, sender net.Addr) {
	for key, value := range playerList {

		msg := "***Player Id:*** " + key + "***Value:***" + value.playerId + "\n"

		if _, err := conn.WriteTo([]byte(msg), sender); err != nil {
			log.Fatalf("Could not write to udp stream: %v", err)
		}
	}
}
func deletePlayer(txt string, s *sdk.SDK, conn net.PacketConn, sender net.Addr) {

	cmd := strings.SplitAfter(txt, "/")
	delete(playerList, cmd[1])
	msg := "Removed a player with id " + cmd[1] + "\n"
	if _, err := conn.WriteTo([]byte(msg), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}

}

func getStackDriverService() (*monitoring.Service, error) {
	oauthClient := oauth2.NewClient(context.Background(), google.ComputeTokenSource(""))
	return monitoring.New(oauthClient)
}

func getResourceLabelsForNewModel(namespace, name string) map[string]string {
	projectId, err := gce.ProjectID()
	if err != nil {
		log.Fatalf("Could not find project id %v", err)
	}
	location, err := gce.InstanceAttributeValue("cluster-location")
	if err != nil {
		log.Fatalf("Could not find location %v", err)
	}
	clusterName, err := gce.InstanceAttributeValue("cluster-name")
	if err != nil {
		log.Fatalf("Could not find cluster name %v", err)
	}

	return map[string]string{
		"project_id":     projectId,
		"location":       location,
		"cluster_name":   clusterName,
		"namespace_name": namespace,
		"pod_name":       name,
	}
}

func exportMetric(stackdriverService *monitoring.Service, metricName string,
	numberofGamers int64, monitoredResource string, resourceLabels map[string]string, metricsLabels map[string]string) error {
	dataPoint := &monitoring.Point{
		Interval: &monitoring.TimeInterval{
			EndTime: time.Now().Format(time.RFC3339),
		},
		Value: &monitoring.TypedValue{
			Int64Value: &numberofGamers,
		},
	}

	// Write time series data.
	request := &monitoring.CreateTimeSeriesRequest{
		TimeSeries: []*monitoring.TimeSeries{
			{
				Metric: &monitoring.Metric{
					Type:   "custom.googleapis.com/" + metricName,
					Labels: metricsLabels,
				},
				Resource: &monitoring.MonitoredResource{
					Type:   monitoredResource,
					Labels: resourceLabels,
				},
				Points: []*monitoring.Point{
					dataPoint,
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
		"gameServer": name,
	}
}

func countPlayers() int {

	return len(playerList)

}

// name - game server name for metrics label
func doStackDriverExport(s *sdk.SDK, name string, interval time.Duration, namespace string, metricName string) {

	stackdriverService, err := getStackDriverService()
	if err != nil {
		log.Fatalf("Error getting Stackdriver service: %v\n", err)
	}

	gs, errGs := s.GameServer()
	if errGs != nil {
		log.Fatalf("Could not retrieve GameServer: %v", errGs)
	}
	metricsLabels := getMetricsResourceLabel(gs.ObjectMeta.Name)
	resourceLabels := getResourceLabelsForNewModel(namespace, name)
	tick := time.Tick(interval * time.Second)
	for {
		//numberofGamers := len(playerList)

		numberofGamers := countPlayers()

		select {
		case <-tick:

			err := exportMetric(stackdriverService, metricName, int64(numberofGamers), "k8s_pod", resourceLabels, metricsLabels)
			if err != nil {
				log.Printf("Failed to write time series data for new resource model: %v\n", err)
			} else {
				log.Printf("Finished writing time series for Game Server: %v Number of Current Gamers: %v\n", resourceLabels["pod_name"], numberofGamers)
			}
			if err != nil {
				log.Fatalf("Could not send SD export, %v", err)
			}

		}
	}

}
