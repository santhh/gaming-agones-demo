##Getting Started with Gaming Demo
This project contains sample code to implement a gaming architecture in GKE. It uses dedicated game server agones, stackdriver and a custom match maker service to replicate a scalable architecture.  
![image](https://screenshot.googleplex.com/NGeTRp3cKkt.png)

###Step: 1 Understanding Agones Custom Resource Definition In Kubernetes

[GameServer](https://github.com/GoogleCloudPlatform/agones/blob/master/docs/gameserver_spec.md)
[Fleet](https://github.com/GoogleCloudPlatform/agones/blob/master/docs/fleet_spec.md)
[Fleet AutoScaler](https://github.com/GoogleCloudPlatform/agones/blob/master/docs/fleetautoscaler_spec.md)
We will be using these custom resources programmatically using kubernetes API from a go client sdk. Please see details [here](https://github.com/GoogleCloudPlatform/agones/blob/master/docs/access_api.md).


###Step: 2 Install Agones in GKE Cluster

Please follow this [instruction](https://github.com/GoogleCloudPlatform/agones/blob/master/install/README.md) to install Agones in GKE cluster


###Step 3: Deploy Monitoring Service

Monitoring service uses [Custom Metrics Stackdriver Adapter](https://github.com/GoogleCloudPlatform/k8s-stackdriver/tree/master/custom-metrics-stackdriver-adapter). Please deploy new resource model adapter  by using following command:

```
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-stackdriver/master/custom-metrics-stackdriver-adapter/deploy/production/adapter_new_resource_model.yaml


```
Please follow instruction in the [monitoring service repo](./monitoring-service/README.md) to complete the deployment.  

```
bash-4.4$ kubectl get deployments
NAME                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
custom-metric-sd      1         1         1            1           4d

bash-4.4$ kubectl logs custom-metric-sd-58f765cc4c-h282x 
{"msg":"Created the agones api clientset","severity":"info","source":"main","time":"2018-11-13T16:39:38Z"}
2018/11/13 16:39:38 Finished writing time series for new resource model with Ready Replicas: 4 Allocated Replicas: 4

```


###Step: 4 Deploy sample game server 

Please follow instruction in the [game server repo](./gameserver/README.md)

You can test either by kubectl command or kubernetes api directly:

```
kubectl get gameservers

bash-4.4$ kubectl get gs
NAME                               AGE
my-game-server-fleet-22jwr-2j9z2   2d
my-game-server-fleet-22jwr-lc9xt   2d
my-game-server-fleet-22jwr-m926w   1d
my-game-server-fleet-22jwr-nm2nw   3d

From the log: 

bash-4.4$ kubectl logs my-game-server-fleet-22jwr-2j9z2-8f4pj -c my-game-server-fleet
2018/11/14 21:17:30 Starting UDP server, listening on port 7654
2018/11/14 21:17:30 Creating SDK instance
2018/11/14 21:17:31 Starting Health Ping
2018/11/14 21:17:31 Marking this server as ready
2018/11/14 21:17:41 Finished writing time series for Game Server: my-game-server-fleet-22jwr-2j9z2-8f4pj Number of Current Gamers: 0. 
OR. 
kubectl proxy&. 
http://localhost:8001/apis/stable.agones.dev/v1alpha1/namespaces/default/gameservers/. 

```

###Step: 5 Deploy Allocation Service 

Please follow instruction in the [allocation service repo](./allocation-service/README.md)


```
bash-4.4$ kubectl get deployments
NAME                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
fleet-allocator       1         1         1            1           3d


After it's running, you can test it bu accessing the URL:

http://<ingress_ip>/address

{
    "status": {
        "state": "Allocated",
        "ports": [
            {
                "name": "default",
                "port": 7046
            }
        ],
        "address": "35.184.39.67",
        "nodeName": "gke-agones-cluster-default-pool-b73909a7-fl05"
    }
}


kubectl get gs -o=custom-columns=NAME:.metadata.name,STATUS:.status.state,IP:.status.address,PORT:.status.ports

NAME                               STATUS      IP             PORT
my-game-server-fleet-22jwr-2j9z2   Allocated   35.184.39.67   [map[name:default port:7177]]
my-game-server-fleet-22jwr-lc9xt   Ready       35.184.39.67   [map[name:default port:7622]]
my-game-server-fleet-22jwr-m926w   Ready       35.184.39.67   [map[name:default port:7160]]
my-game-server-fleet-22jwr-nm2nw   Ready       35.184.39.67   [map[name:default port:7134]]

```


###Step 6: Deploy Custom Scaler Service

Please follow instruction in the [scaling service repo](./scaling-service/README.md) to complete the deployment.  

```
bash-4.4$ kubectl get deployments
NAME                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
custom-fleet-scaler   1         1         1            1           4d

bash-4.4$ kubectl logs custom-fleet-scaler-7c9bc69c78-q2th2 
2018/11/17 02:16:37 Created the agones api clientset
2018/11/17 02:16:37 FAS does not exist fleetautoscalers.stable.agones.dev "fleet-autoscaler-my-game-server" not found
2018/11/17 02:16:52 ***Player Count 0 *** Ready Replicas 4 Error:<nil>
2018/11/17 02:16:52 **Auto Scaling Remain Same*** Target Buffer: 4 Current Buffer: 4
```

###Step 7: Deploy Custom Matchmaker Service

Please follow instruction in the [matchmaker repo](./scaling-service/README.md) to complete the deployment.  

