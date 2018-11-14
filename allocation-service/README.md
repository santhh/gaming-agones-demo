### Agones Game Server Allocation Service
This is a sample service to allocate a dedicated agones game server from fleet. After it's deployed correctly in a GKE cluster, you can access the service in GKE ingress IP for the cluster.  

This service can be used from match maker services to allocate a game server. It uses k8 client sdk for Go to access agones custom resources programmatically. e.g (gameserver, fleet). 

Allocation end point (GET)

````
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
````

Heath Check end point (GET)

````
https://35.190.19.82

healthy

````

Please refer to gaming game-server repo for sample UDP gameserver fleet and fleet auto scalaer yaml files for details.

###  Build & Deploy

Update glide.yaml file with your go root and project src location for package element

````
package: github.com/santhh/gaming-allocation-service # e.g. github.com/foo/bar
import:
- package: k8s.io/client-go
  version: v8.0.0

````
For go build. You can use brew to install glide if required

````
glide update --strip-vendor

````


Update docker file as well with correct location

````
ENV GOPATH=/Users/masudhasan/Documents/workspace-sts-3.9.5.RELEASE/go
WORKDIR $GOPATH/src/github.com/santhh/gaming-allocation-service

````

Build docker image for your GCP project. 

````
docker build -t gcr.io/<project_id>/agones-allocation-service:v12 .

````
Push the image to GCR

````
docker push gcr.io/<project_id>/agones-allocation-service:v12
````

To deply in GKE. Update the YAML files with the tag you build on previous steps. Then perform following kubectl commands with your local path

````
kubectl apply -f $GOPATH/src/github.com/santhh/allocator-service/service_account.yaml

kubectl create -f $GOPATH/src/github.com/santhh/allocator-service/allocator-service.yaml

kubectl create -f $GOPATH/src/github.com/santhh/allocator-service/allocator-ingress.yaml

kubectl get ingress fleet-allocator-ingress (to get the ingress IP for service)

````

### Security

Service uses basic auth and only operates over https. U/P : v1GameClientKey/EAEC945C371B2EC361DE399C2F11E









