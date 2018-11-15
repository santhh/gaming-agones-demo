### Gaming Match Maker Service Using S2 Geometry Library

This is a sample match maker service that uses S2 Geometry library to locate a cell id based on latitude and longitude of players location. It works as follows:

- Service takes latitude and longitude as path variable for the api. e.g. /matchmaker/v1/allocate/lat/43.652690/lon/-79.373590. 
- Based on the latitude and longitude, service finds level 10 cell id. 
- if the cell id already exists, service assigns a game server that has player count less than the maximum players allowed. if no such game server is found it creates a new one for the existing cell id by calling agones allocation service.  
- if the cell id does not exist, service calls agones allocation service to allocate a new game server 
 
Response received from service:

```
GET : http://104.197.83.43/matchmaker/v1/allocate/lat/43.652690/lon/-79.373590

{
    "status": {
        "cellId": "-8514957794590326784",
        "address": "35.226.89.12",
        "ports": [
            {
                "port": "7393"
            }
        ]
    }
}

```

### Build & Deploy

Below maven goal can be used to build, create and push the image to google container registry. S2  libraries are built as a jar and is part of resource/lib folder (s2-geometry-java.jar). It uses jib to build and push the image.

```
install:install-file -Dfile=<clone_path>/matchmaker/src/main/resources/lib/s2-geometry-java.jar -DgroupId=s2.libs -DartifactId=s2.libs -Dversion=1.0 -Dpackaging=jar

compile jib:build -Dproject_id={project_id}-Dimage_tag={tag}

```



Create a GKE cluster and deploy the service

```
gcloud beta container --project "agones-poc" clusters create "matchmaker" --zone "us-central1-a" --username "admin" --cluster-version "1.9.7-gke.6" --machine-type "n1-standard-16" --image-type "COS" --disk-type "pd-standard" --disk-size "100" --scopes "https://www.googleapis.com/auth/cloud-platform" --num-nodes "1" --enable-cloud-logging --enable-cloud-monitoring --network "projects/agones-poc/global/networks/default" --subnetwork "projects/agones-poc/regions/us-central1/subnetworks/default" --addons HorizontalPodAutoscaling,HttpLoadBalancing,KubernetesDashboard --enable-autoupgrade --enable-autorepair

gcloud config set container/cluster matchmaker


gcloud container clusters get-credentials matchmaker --zone us-central1-a --project agones-poc

```

Service will need  three configurations 
1. Max Users :  this parameter is used to check if a game server has reached to maximum number of players allowed.
2. Allocation Service URL: Please see read me for gaming-allocation-service to get the service URL.
3. Datastore kind: Name of the data store kind to be used

```
kubectl create configmap gaming-matchmaker-config\
--from-literal=max_users=12 \
--from-literal=allocation_service_url=http://35.244.220.217/address \
--from-literal=data_store_kind=match_maker_data

```
Deploy the service 

```
kubectl apply -f <clone_path>/gaming-matchmaker-service/kubernetes/match-maker-deploy.yaml

```
Expose the deployment

```
kubectl expose deployment gaming-matchmaker --port=80 --target-port=8080 \
        --name=gaming-matchmaker-service --type=LoadBalancer
```

### Testing 

Cell Id # 1: http://104.197.83.43/matchmaker/v1/allocate/lat/43.652690/lon/-79.373590

```
{
    "status": {
        "cellId": "-8514957794590326784",
        "address": "35.226.89.12",
        "ports": [
            {
                "port": "7393"
            }
        ]
    }
}

```

Cell Id # 2: http://104.197.83.43/matchmaker/v1/allocate/lat/43.545360/lon/-79.742280. 
http://104.197.83.43/matchmaker/v1/allocate/lat/43.545050/lon/-79.742600

These locations are close to each other and return same cell id and game server:

```
{
    "status": {
        "cellId": "-8634692411831877632",
        "address": "35.184.39.67",
        "ports": [
            {
                "port": "7171"
            }
        ]
    }
}

```

Screen shot from datastore:

![image](https://screenshot.googleplex.com/HGJeKSbfhR4.png)


![image](https://screenshot.googleplex.com/SytbB8UbtkY.png)






