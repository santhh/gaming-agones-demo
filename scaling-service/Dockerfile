# build
FROM golang:1.10.3 as builder
RUN curl https://glide.sh/get | sh
ENV GOPATH=/Users/masudhasan/Documents/workspace-sts-3.9.5.RELEASE/go
WORKDIR $GOPATH/src/github.com/santhh/gaming-scaling-service
COPY scale.go .
RUN go get cloud.google.com/go/compute/metadata
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/google
RUN go get google.golang.org/api/monitoring/v3
RUN go get k8s.io/apimachinery/pkg/apis/meta/v1
RUN go get k8s.io/client-go/rest
RUN go get agones.dev/agones/pkg/sdk
COPY glide.* ./
RUN glide -v
RUN glide update --strip-vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .
# final image
FROM alpine:3.7
ENV GOPATH=/Users/masudhasan/Documents/workspace-sts-3.9.5.RELEASE/go
RUN apk add --update ca-certificates
RUN adduser -D service
COPY --from=builder $GOPATH/src/github.com/santhh/gaming-scaling-service /home/service
RUN chown -R service /home/service && \
    chmod o+x /home/service/service

USER service
ENTRYPOINT /home/service/service
