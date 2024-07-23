# Build the manager binary
FROM registry.access.redhat.com/ubi9/ubi:9.4-1123.1719560047 as builder

RUN yum -y install \
 golang \
 gpgme-devel

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY pkg pkg

# Build
RUN GOOS=linux GOARCH=amd64 go build -tags exclude_graphdriver_btrfs,btrfs_noversion -a -o metrics-server main.go

# Use ubi-micro as minimal base image to package the manager binary
# See https://catalog.redhat.com/software/containers/ubi9/ubi-micro/615bdf943f6014fa45ae1b58
FROM registry.access.redhat.com/ubi9/ubi:9.4
COPY policy.json /etc/containers/
COPY --from=builder /metrics-server /bin/

# It is mandatory to set these labels
LABEL name="Konflux Release Service"
LABEL description="Konflux Release Availability Metrics Service"
LABEL io.k8s.description="Konflux Release Availability Metrics Service"
LABEL io.k8s.display-name="release-availability-metrics"
LABEL summary="Konflux Release Availability Metrics Service"
LABEL com.redhat.component="release-availability-service"

USER 65532:65532

ENTRYPOINT ["/bin/metrics-server"]
