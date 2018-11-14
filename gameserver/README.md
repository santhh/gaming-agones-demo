### Agones Game Server 
This is a sample dedicated UDP game server built using Agones 0.5-RC images. (Agones Controller and Agones Client SDK). Game server uses fleet and fleet auto scaler to maintain a pool of game servers ready to be allocated.

### Before Start
First of all, you will need a GKE 1.10+ cluster where agones needs to installed. Below steps show the commands required to install Agones using yaml file

```
gcloud container clusters create agones-cluster --cluster-version=1.10   --no-enable-legacy-authorization   --tags=game-server   --enable-basic-auth --password=sanith678910111213141516--scopes=https://www.googleapis.com/auth/devstorage.read_only,compute-rw,cloud-platform   --num-nodes=3   --machine-type=n1-standard-8
gcloud config set container/cluster agones-cluster
gcloud compute firewall-rules create game-server-firewall   --allow udp:7000-8000   --target-tags game-server   --description "Firewall to allow game server udp traffic"
kubectl create clusterrolebinding cluster-admin-binding   --clusterrole cluster-admin --user `gcloud config get-value account`
kubectl create namespace agones-system
kubectl apply -f $GOPATH/src/github.com/santhh/agones/install/yaml/install.yaml 
kubectl describe --namespace agones-system pods


```

### Build & Deploy

Build the image locally and push it to GKE

```
docker build -t gcr.io/<project_id>/agones-udp-server:v5 .

docker push gcr.io/<project_id>/agones-udp-server:v5

```

Update the fleet.yaml file with correct image. 

```
template:
        spec:
          containers:
          - name: my-game-server-fleet
            image: gcr.io/<project_id>/agones-udp-server:v5


```

Update fleet auto scaler yaml file with required scaling config. Please see in the fleetautoscaler.yaml file for details about bufferSize , minReplica, maxReplica

```
      # Size of a buffer of "ready" game server instances
      # The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
      # as instances are being allocated or terminated
      # it can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
      bufferSize: 5
      # minimum fleet size to be set by this FleetAutoscaler. 
      # if not specified, the actual minimum fleet size will be bufferSize
      minReplicas: 6
      # maximum fleet size that can be set by this FleetAutoscaler
      # required
      maxReplicas: 50
```

And finally, kubectl apply commands. Please update with your own path. You should see READY status when all the servers start successfully

```
kubectl apply -f $GOPATH/src/github.com/santhh/gaming-gameserver/fleet.yaml
kubectl apply -f $GOPATH/src/github.com/santhh/gaming-gameserver/fleetautoscaler.yaml
kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports

```

### Test With Gaming Allocation Service

You can manually create an allocation by the command below and see a single server status changed to Allocated


```

kubectl create -f $GOPATH/src/github.com/santhh/mygameserver/fleet-allocation.yaml -o yaml

kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status
NAME                               STATUS      IP               PORT
my-game-server-fleet-9lplj-6rnv7   Ready       35.238.143.177   [map[name:default port:7083]]
my-game-server-fleet-9lplj-7bc28   Ready       35.238.143.177   [map[name:default port:7021]]
my-game-server-fleet-9lplj-9krj2   Allocated   35.184.39.67     [map[name:default port:7857]]
my-game-server-fleet-9lplj-d8w8z   Ready       35.226.89.12     [map[name:default port:7137]]
my-game-server-fleet-9lplj-f8p9q   Ready       35.226.89.12     [map[name:default port:7483]]
my-game-server-fleet-9lplj-s52cz   Ready       35.184.39.67     [map[name:default port:7928]]

```

However, you can also use gaming allocator service to create an allocation. Plase see details in gaming-allocation-service repo.

Allocation end point (GET)

```
https://35.190.19.82/address

{
    "status": {
        "state": "Allocated",
        "ports": [
            {
                "name": "default",
                "port": 7857
            }
        ],
        "address": "35.184.39.67",
        "nodeName": "gke-agones-cluster-default-pool-b73909a7-fl05"
    }
}
```

Heath Check end point (GET)

```
https://35.190.19.82

healthy

```

This service can be used from match maker services to allocate a game server. It uses k8 client sdk for Go to access agones custom resources programmatically. e.g (gameserver, fleet). 


You will have to SSH into the game server to logout or exit

```
nc -u 35.184.39.67 7857
EXIT

```

### Controlling player in the Game Server

After SSH into game server, you can net cat the IP and Port for the allocated game server. There are four others commands you can use to play around with the game server.

NEW - To add a new player in the game server. You should see a message back like. Example below shows there are three players added to the game server. 

```
NEW
Added a player with id bf3uqk7sj83nlfrrmq8g
ACK From Masud's Game Server: NEW
NEW
Added a player with id bf3uqkvsj83nlfrrmq90
ACK From Masud's Game Server: NEW
NEW
Added a player with id bf3uql7sj83nlfrrmq9g
ACK From Masud's Game Server: NEW

```

PLIST - To see the number of active players. Example below shows the player ids for all three players added in the previous step

```
PLIST
***Player Id:*** bf3uqkvsj83nlfrrmq90***Value:***bf3uqkvsj83nlfrrmq90
***Player Id:*** bf3uql7sj83nlfrrmq9g***Value:***bf3uql7sj83nlfrrmq9g
***Player Id:*** bf3uqk7sj83nlfrrmq8g***Value:***bf3uqk7sj83nlfrrmq8g
ACK From Masud's Game Server: PLIST

```

DONE/{playerId} - This command will remove the player form the game server. It takes playerId as a parameter. Below example shows remove a playerId (bf3uqkvsj83nlfrrmq90) by the command and followed by a PLIST command to confirm the removal.

```
DONE/bf3uqkvsj83nlfrrmq90
Removed a player with id bf3uqkvsj83nlfrrmq90
ACK From Masud's Game Server: DONE/bf3uqkvsj83nlfrrmq90
PLIST
***Player Id:*** bf3uql7sj83nlfrrmq9g***Value:***bf3uql7sj83nlfrrmq9g
***Player Id:*** bf3uqk7sj83nlfrrmq8g***Value:***bf3uqk7sj83nlfrrmq8g

```
PCOUNT - This command will give the count of number of players in the game server used by match maker service. e.g; 2, 9

### Stackdriver Analysis for Number of Players

Gameserver exports data relate to number of active players in regular interval. Interval can me passed in fleet.yaml file. By default it's 10 secs

User will have to create a custom metrics by using API explorer. You can use this link to access the API. 

```
https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors/create?apix=true

```
Screenshot shows the parameter used to create playerCount custom metrics.

![image](https://screenshot.googleplex.com/yeL6E3SRTGJ.png)


Screen shot form the stack driver after the data is exported:

![image](https://screenshot.googleplex.com/rxeF3zLk4Uw)





