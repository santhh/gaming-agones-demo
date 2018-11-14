# build
FROM golang:1.10.3 as builder
RUN curl https://glide.sh/get | sh
ENV GOPKG=$GOPKG
ENV GOPATH=$GOPATH 
COPY main.go .
RUN go get cloud.google.com/go/compute/metadata
RUN go get golang.org/x/net/context
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/google
RUN go get google.golang.org/api/monitoring/v3
RUN go get agones.dev/agones/pkg/sdk
RUN go get k8s.io/apimachinery/pkg/apis/meta/v1
RUN go get k8s.io/client-go/rest
COPY glide.* ./
RUN glide -v
RUN glide update --strip-vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .
# final image
FROM alpine:3.7
ENV GOPKG=$GOPKG/
RUN apk add --update ca-certificates
RUN adduser -D service
COPY --from=builder $GOPKG/service /home/service/service
RUN chown -R service /home/service && \
    chmod o+x /home/service/service

USER service
ENTRYPOINT /home/service/service