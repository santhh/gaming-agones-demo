## How to build your GRPC files?

```bash
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. --descriptor_set_out=api_descriptor.pb ./gs_auth.proto
```
## How to run locally?

***Run Server:***
```bash
python gs_auth_server.py --apikey="{YOUR-API-KEY}" --authDomain="{PROJECT_ID}.firebaseapp.com" \
--databaseURL="https://{PROJECT_ID}.firebaseio.com" --projectId="{PROJECT_ID}" \
--storageBucket="{BUCKET}.appspot.com" --messagingSenderId="ID"
```

***Run Client:***
```bash
python gs_auth_server.py --host IP.IP.IP.IP:PORT
```


## To Deploy on Cloud Endpoints:

### 01 - Endpoints configuration
To deploy the Endpoints configuration, you use the gcloud endpoints services deploy command. This command uses Service Infrastructure, Google’s foundational services platform, used by Cloud Endpoints and other services to create and manage APIs and services.

***To Deploy:***
```bash
gcloud endpoints services deploy api_descriptor.pb api_config.yaml
```
***To Delete:***
```bash
gcloud endpoints services delete gs-auth-api.endpoints.agones-poc.cloud.goog
```
***To Undelete:***
```bash
gcloud endpoints services undelete gs-auth-api.endpoints.agones-poc.cloud.goog
```

***How does api_config.yaml looks like?***
```bash
# The configuration schema is defined by service.proto file
type: google.api.Service
config_version: 3
#
# Name of the service configuration.
#
name: gs-auth-api.endpoints.agones-poc.cloud.goog
#
# API title to appear in the user interface (Google Cloud Console).
#
title: Game Server gRPC API
apis:
- name: GameserverAuthentication
#
# API usage restrictions.
#
usage:
  rules:
  # GameserverAuthentication methods can be called without an API Key.
  - selector: GameserverAuthentication.RegisterPlayer
    allow_unregistered_calls: true
  - selector: GameserverAuthentication.LoginPlayer
    allow_unregistered_calls: true
  - selector: GameserverAuthentication.LogoutPlayer
    allow_unregistered_calls: true
  - selector: GameserverAuthentication.GetPlayerInfo
    allow_unregistered_calls: true
  - selector: GameserverAuthentication.AuthenticationTokenRefresh
    allow_unregistered_calls: true
  - selector: GameserverAuthentication.UpdateActivity
    allow_unregistered_calls: true
```

### 02 - Package your Python application:

***Build your Dockerfile:***
```bash
FROM python:2.7

WORKDIR /grpc
ENV PATH "$PATH:/grpc"

COPY gs_auth_server.py /grpc
COPY gs_auth_pb2.py /grpc
COPY gs_auth_pb2_grpc.py /grpc
COPY requirements.txt /grpc

RUN pip install -r requirements.txt

CMD python gs_auth_server.py --apikey="{YOUR-API-KEY}" --authDomain="{PROJECT_ID}.firebaseapp.com" --databaseURL="https://{PROJECT_ID}.firebaseio.com" --projectId="{PROJECT_ID}" --storageBucket="{BUCKET}.appspot.com" --messagingSenderId="ID"

EXPOSE 50051
```

***Build your Docker Image:***
```bash
docker build -t gcr.io/$PROJECT_ID/gs_auth:v1 .
docker images
```
***Push your Docker Image:***
```bash
docker push gcr.io/$PROJECT_ID/gs_auth:v1
```
***Test Locally (Optional)***
```bash
docker run --rm -p 50051:50051 gcr.io/agones-poc/gs_auth:v1
```

### 03 - Deploying the API backend:
So far you have deployed the service configuration to Service Management, but you have not yet deployed the code that will serve the API backend. 

***Deploy***
```bash
kubectl create -f api_deploy.yaml
```

***YAML (api_deploy.yaml)***
```bash
apiVersion: v1
kind: Service
metadata:
  name: esp-grpc-auth2
spec:
  ports:
  - port: 80
    targetPort: 9000
    protocol: TCP
    name: http
  selector:
    app: esp-grpc-auth2
  type: LoadBalancer
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: esp-grpc-auth2
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: esp-grpc-auth2
    spec:
      containers:
      - name: esp
        image: gcr.io/endpoints-release/endpoints-runtime:1
        args: [
          "--http2_port=9000",
          "--backend=grpc://127.0.0.1:50051",
          "--service=gs-auth-api.endpoints.agones-poc.cloud.goog",
          "--rollout_strategy=managed",
        ]
        ports:
          - containerPort: 9000
      - name: python-esp-grpc-auth2
        image: gcr.io/agones-poc/gs_auth:v1
        ports:
          - containerPort: 50051
```

### 04 - Test your API with a client:

```bash
python gs_auth_client.py --host <your-gke-pod-ip-address>:80
```

## Running the server with gRPC <-> HTTP/JSON Transcoding

***Update your .proto file (gs_auth_02.proto)***
```bash
// TODO (Patrick): (COPYRIGHT NOTICE)

// Definitions for Game Authentication service

syntax = "proto3";

import "google/api/annotations.proto"; 

service GameserverAuthentication {
  rpc RegisterPlayer (PlayerRegistrationRequest) returns (PlayerRegistrationResponse) {
    option (google.api.http) = {
      post: "/createplayer" 
      body: "*"
    };  
  }
  rpc LoginPlayer (LoginRequest) returns (LoginResponse) {
    option (google.api.http) = {
      post: "/loginplayer" 
      body: "*"
    };
  }
  rpc LogoutPlayer(LogoutRequest) returns (LogoutResponse) {
    option (google.api.http) = { get: "/logout/{email}" };
  }
  rpc GetPlayerInfo (PlayerInfoRequest) returns (PlayerInfoResponse) {
    option (google.api.http) = { get: "/getplayerinfo/{idToken}" };
  }
  rpc AuthenticationTokenRefresh (AuthenticationTokenRefreshRequest) returns (AuthenticationTokenRefreshResponse) {
    option (google.api.http) = { get: "/authenticationtokenrefresh/{idToken}" };
  }
  rpc UpdateActivity(UpdateActivityRequest) returns (UpdateActivityResponse) {
    option (google.api.http) = { get: "/updateactivity/{email}" };
  }
}


message PlayerRegistrationRequest{
  string email = 1;
  string password = 2;
}

message PlayerRegistrationResponse{
}

message LoginRequest {
  string email = 1;
  string password = 2;
}

message LoginResponse {
  string idToken = 1;
  string email = 2;
  string refToken = 3;
  bool loggedIn = 4;
  float time = 5;
}

message LogoutRequest{
  string email = 1;
}

message LogoutResponse{
}

message PlayerInfoRequest {
  string idToken = 1;
}

message PlayerInfoResponse{
  string email = 1;
  string passwordUpdatedAt = 2;
  string emailVerified = 3;
  bool presence = 4;
  float lastActivity = 5;
}

message AuthenticationTokenRefreshRequest{
  string idToken = 1;
}

message AuthenticationTokenRefreshResponse{
  string idToken = 1;
}

message UpdateActivityRequest{
  string email = 1;
}

message UpdateActivityResponse{
}
```

***Do some housekeeping:***
```bash
git clone https://github.com/googleapis/googleapis
GOOGLEAPIS_DIR=./googleapis
```

***Generate your new .pb file for http access (api_descriptor_http.pb)***
```bash
python -m grpc_tools.protoc --include_imports --include_source_info --proto_path=${GOOGLEAPIS_DIR} --proto_path=. --descriptor_set_out=api_descriptor_http.pb ./gs_auth_02.proto  
```

***Deploy to Service Infrastructure Cloud Endpoints***

Upload "api_descriptor_http.pb" to CloudShell and deploy it:
```bash
gcloud endpoints services deploy api_config.yaml api_descriptor_http.pb
```

***Then, change your api_deploy.yaml to service HTTP1 requests to another port (added new service / new container port / new http_port in args):***

```bash
apiVersion: v1
kind: Service
metadata:
  name: esp-grpc-auth
spec:
  ports:
  - port: 80
    targetPort: 9000
    protocol: TCP
    name: http
  selector:
    app: esp-grpc-auth
  type: LoadBalancer
---
apiVersion: v1
kind: Service
metadata:
  name: esp-grpc-auth-http
spec:
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: esp-grpc-auth
  type: LoadBalancer
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: esp-grpc-auth
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: esp-grpc-auth
    spec:
      containers:
      - name: esp
        image: gcr.io/endpoints-release/endpoints-runtime:1
        args: [
          "--http2_port=9000",
          "--http_port=8080",
          "--backend=grpc://127.0.0.1:50051",
          "--service=gs-auth-api.endpoints.agones-poc.cloud.goog",
          "--rollout_strategy=managed",
        ]
        ports:
          - containerPort: 9000
      - name: python-esp-grpc-auth
        image: gcr.io/agones-poc/gs_auth:v1
        ports:
          - containerPort: 50051
            containerPort: 8080
```

***And, finally:***

```bash
kubectl create -f api_deploy.yaml
```

### REST API Endpoints:

```bash

POST "http://ip.ip.ip.ip/createplayer"
POST "http://ip.ip.ip.ip/loginplayer" 
GET: "http://ip.ip.ip.ip/logout/{email}"
GET: "http://ip.ip.ip.ip/getplayerinfo/{idToken}"
GET: "http://ip.ip.ip.ip/authenticationtokenrefresh/{idToken}"
GET: "http://ip.ip.ip.ip/updateactivity/{email}"

```

## gRPC Authentication API using Firebase

## gRPC functions and protocol buffers

## gRPC sample client