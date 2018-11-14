### Stackriver monitoring integration for Agones Game Server
This is a sample data exporter service to to send game server status (e.g Ready, Allocated, Unknown) to a stack drvier custom metrics in a GCP project. This service assumes you have agones game server up and running in a GKE cluster.

###  Creating a Custom Metrics

This URL below shows how to create a sample custom metrics. 
https://cloud.google.com/monitoring/custom-metrics/creating-metrics#writing-ts

To create a custom metrics from UI using API explorer, please go to following link:

```
https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors/create?apix=true

```
Please pass the parameter as indicated in the screen shot below:

![image](https://screenshot.googleplex.com/OZhTCXOYkpo)


Otherwise, you can also use the curl statement below to create a custom metrics name : custom.googleapis.com/gameservers/stats

curl --request POST \
  'https://monitoring.googleapis.com/v3/projects/agones-poc/metricDescriptors' \
  --header 'Authorization: Bearer [YOUR_BEARER_TOKEN]' \
  --header 'Accept: application/json' \
  --header 'Content-Type: application/json' \
  --data '{"description":"Agones GameServers Stats","displayName":"gameserver-stats","labels":[{"description":"Game Sever Status","key":"replicaStatus","valueType":"STRING"}],"metricKind":"GAUGE","type":"custom.googleapis.com/gameservers/stats","name":"","unit":"","valueType":"INT64"}' \
  --compressed

You can test the metrics by sending a sample message using timeSeries.create API.

curl --request POST \
  'https://monitoring.googleapis.com/v3/projects/agones-poc/timeSeries' \
  --header 'Authorization: Bearer [YOUR_BEARER_TOKEN]' \
  --header 'Accept: application/json' \
  --header 'Content-Type: application/json' \
  --data '{"timeSeries":[{"metric":{"type":"custom.googleapis.com/gameservers/stats","labels":{"replicaStatus":"allocated"}},"resource":{"type":"global","labels":{"project_id":"agones-poc"}},"points":[{"interval":{"endTime":"2018-10-13T10:00:00-04:00"},"value":{"int64Value":15}}]},{"metric":{"type":""}}]}' \
  --compressed

### Local Build and Deploy

Build the docker image:

```
docker build -t gcr.io/<project_id>/agones-allocation-service:<tag> .
	
```

Push to GCR:

```
docker push gcr.io/<project_id>/agones-allocation-service:<tag> 

```
### Update the YAML file

Please update  monitoring.yaml file with the image and service account that has access to gameserver APIs e.g fleet, gameservers. Please refer to the serviceaccount.yaml file if you don;t have any service account created and assigned.

```
	  spec:
	       serviceAccount: fleet-allocator
	  image: gcr.io/<projet_id>/agones-monitoring-service:<tag>
	  	          name: agones-monitoring-service
	          resources:
```

### Stackdriver dashboard
You should see relicaStatus as ready or allocated to the custom metrics that was created in earlier steps.

![image](https://user-images.githubusercontent.com/27572451/47030150-1f1bee80-d13b-11e8-8047-983cec7cfc22.png)

### How does it work?

It uses K8 go clinet sdk to call kubernetes CRD APIs in 60 seconds interval. Currently it only calls get fleets api to query ready and allocated replica status but can be extended to other APIs to gather more information. 









