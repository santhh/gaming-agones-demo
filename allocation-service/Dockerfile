# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# build
FROM golang:1.10.3 as builder
RUN curl https://glide.sh/get | sh
ENV GOPATH=/Users/masudhasan/Documents/workspace-sts-3.9.5.RELEASE/go
WORKDIR $GOPATH/src/github.com/santhh/gaming-allocation-service
COPY main.go .
RUN go get agones.dev/agones/pkg/sdk
RUN go get k8s.io/apimachinery/pkg/apis/meta/v1
RUN go get k8s.io/client-go/rest
COPY glide.* ./
RUN glide -v
RUN glide update --strip-vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .

# final image
FROM alpine:3.7
ENV GOPATH=/Users/masudhasan/Documents/workspace-sts-3.9.5.RELEASE/go
RUN apk add --update ca-certificates
RUN adduser -D service
COPY --from=builder $GOPATH/src/github.com/santhh/gaming-allocation-service \
                    /home/service
RUN chown -R service /home/service && \
    chmod o+x /home/service/service

USER service
ENTRYPOINT /home/service/service
