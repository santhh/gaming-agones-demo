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
ENV GOPKG=$GOPKG
ENV GOPATH=$GOPATH
WORKDIR $GOPKG/
COPY server.go .
RUN go version
RUN go get agones.dev/agones/pkg/sdk
RUN go get github.com/rs/xid
RUN go get cloud.google.com/go/compute/metadata
RUN go get golang.org/x/net/context
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/google
RUN go get google.golang.org/api/monitoring/v3
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# final image
FROM alpine:3.7
ENV GOPKG=$GOPKG/
RUN apk add --update ca-certificates
RUN adduser -D server
COPY --from=builder $GOPKG/server /home/server/server
RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER server
ENTRYPOINT /home/server/server