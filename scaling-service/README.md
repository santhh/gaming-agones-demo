### Scaling Game Server using Stack Driver Custom Metrics 
This is a sample service to scale game server using a custom metrics received from agones game server. This service uses gaming-gameserver and gaming-monitoring service to create number of players in the agones server. After the players are created, it scales ready game servers up and down based on the number of players currently playing. 

### Before Start
This service assumes you have gaming game server, gaming monitoring service and gaming allocation service  deployed as described in the README for other repos. You should confirm if pods are up and running. From the example below you can see there are 4 replicas of game servers, one allocation service and one monitoring service running

```
custom-metric-sd-6b9c75f7d8-dscr5        1/1       Running   0          17d
fleet-allocator-78574bbf9-zf84l          1/1       Running   0          21d
my-game-server-fleet-p64bw-bkcsh-rz44p   2/2       Running   0          16m
my-game-server-fleet-p64bw-lcgvz-jsjjp   2/2       Running   0          5m
my-game-server-fleet-p64bw-mvvvg-mfdrl   2/2       Running   0          16m
my-game-server-fleet-p64bw-rl8s6-5pv4n   2/2       Running   0          5m

```

You can also check if the fleet auto scalaer is avaiable for the cluster. If there is, please delete the default auto scaler

```
kubectl get fas 

kubectl delete fas <name>

```

### Build & Deploy

Build the image locally and push it to GKE

```
docker build -t gcr.io/<project_id>/agones-scaling-service:v<tag> .

docker push gcr.io/<project_id>/agones-scaling-service:v<tag>

```

Deploy the service using kubectl create command after updating scaling.yaml file as requied for the namespace. Alos, please create a service account before deploying the service as showed below. (service-account.yaml).

```
kubectl apply -f $GOPATH/src/<path>gaming-scaling-service/service-account.yaml

kubectl create -f $GOPATH/src/<path>/gaming-scaling-service/scaling.yaml


```

you should see a pod created like below and in the log: 

```
NAME                                     READY     STATUS    RESTARTS   AGE
custom-fleet-scaler-8586d4bbc5-c4kvb     1/1       Running   0          25m

kubectl logs custom-fleet-scaler-8586d4bbc5-c4kvb 
2018/11/05 16:40:21 ***Player Count 0 *** Ready Replicas 4 Error:<nil>
2018/11/05 16:40:21 **Auto Scaling Remain Same*** Target Buffer: 4 Current Buffer: 4

```



### Testing 

You can manually create an allocation by the command below and see a single server status changed to Allocated


You you can also use gaming allocator service to create an allocation. Plase see details in gaming-allocation-service repo.

Allocation end point (GET)

```
https://<cluster_ing_ip>/address

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

You will have to SSH into the game server to create a player by using NEW command. Please review README for gaming-gameserver and how to sue PLIST, NEW , PCOUNT and DONE/{player_id} commands.

```
nc -u <ip> <port>
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

###  How the scaling works?

1. Currently the default setup is to have 4 ready game server replicas to start with. 
2. As load is generated to create more players, it does following:  
	a. Change the status to read gameserver to allocated  
	b. Create 12 users / game server  
	c. Timing tested with 10 secs to have a allocated server and create each user in every 2.5 secs  
3. As games ervers sends number of player stats in the stack driver custom stats, soon you should see in the log:

```
2018/11/05 16:19:04 ***Player Count 19 *** Ready Replicas 0 Error:<nil>
2018/11/05 16:19:04 **Auto Scaling Should Trigger*** Target Buffer: 5 Current Buffer: 4
2018/11/05 16:19:04 Namespace: %v Fleet: %v Fleet Auto Scaler: %vdefaultmy-game-server-fleetfleet-autoscaler-my-game-server
2018/11/05 16:19:04 ***Fleet Auto Scalar Created***{fleet-autoscaler-my-game-server  default /apis/stable.agones.dev/v1alpha1/namespaces/default/fleetautoscalers/fleet-autoscaler-my-game-server 836d4016-e116-11e8-b318-42010af00025 4269307 1 2018-11-05 16:19:04 +0000 UTC <nil> <nil> map[] map[] [{stable.agones.dev/v1alpha1 Fleet my-game-server-fleet b6279bb9-e115-11e8-b318-42010af00025 0xc42051b560 0xc42051b54f}] nil [] } {my-game-server-fleet {Buffer 0xc4201c3e30}} {0 0 <nil> false false}}
2018/11/05 16:19:04 Latest Resource Id: 4269307
2018/11/05 16:19:04 Successfully Updated Buffer Size to 5

```

As you can see  number of ready replicas increased to 5  It does a very simple custom logic to maintain 1:4 ratio between gameserver : numberOfPlayers

```
//for plater count 19. e.g: '19/4 =4.7 ceiling to 5'
	desireReplicas := float64(pc) / float64(defaultBufferSize)
	desireBufferSize := int(math.Ceil(desireReplicas))
	if desireBufferSize < defaultBufferSize {
		desireBufferSize = defaultBufferSize
	}
	return desireBufferSize

```

This service uses buffer size for agones fleet auto scaler feature to maintain the desire replicas based on custom metrics (e.g. number of Players for our case)

It automatically creates fleet auto scaler from the fleet details as specified in the yaml file. Maximum replicas are set to 50 (ready + allocated) at this time.
