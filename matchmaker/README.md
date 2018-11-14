Create a GKE Cluster

```
gcloud beta container --project "agones-poc" clusters create "matchmaker" --zone "us-central1-a" --username "admin" --cluster-version "1.9.7-gke.6" --machine-type "n1-standard-16" --image-type "COS" --disk-type "pd-standard" --disk-size "100" --scopes "https://www.googleapis.com/auth/cloud-platform" --num-nodes "1" --enable-cloud-logging --enable-cloud-monitoring --network "projects/agones-poc/global/networks/default" --subnetwork "projects/agones-poc/regions/us-central1/subnetworks/default" --addons HorizontalPodAutoscaling,HttpLoadBalancing,KubernetesDashboard --enable-autoupgrade --enable-autorepair

gcloud config set container/cluster matchmaker


gcloud container clusters get-credentials matchmaker --zone us-central1-a --project agones-poc

gcloud container clusters get-credentials matchmaker --zone us-central1-a --project agones-poc


kubectl create configmap gaming-matchmaker-config\
--from-literal=max_users=12 \
--from-literal=allocation_service_url=http://35.244.220.217/address \
--from-literal=data_store_kind=match_maker_data

kubectl apply -f <clone_path>/gaming-matchmaker-service/kubernetes/match-maker-deploy.yaml

kubectl expose deployment gaming-matchmaker --port=80 --target-port=8080 \
        --name=gaming-matchmaker-service --type=LoadBalancer


```