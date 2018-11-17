# Build the manager binary
FROM golang:1.10.3 as builder
RUN curl https://glide.sh/get | sh

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/cluster-api-provider-baiducloud
COPY . .

# Build
RUN glide up
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager sigs.k8s.io/cluster-api-provider-baiducloud/cmd/manager

