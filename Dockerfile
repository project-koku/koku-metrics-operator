# Build the manager binary
FROM --platform=${BUILDPLATFORM:-linux/amd64} registry.access.redhat.com/ubi8/go-toolset:1.20.10-10 AS builder
ARG TARGETOS
ARG TARGETARCH

USER root
RUN yum -y update && yum clean all

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Copy git to inject the commit during build
COPY .git .git
# Build
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
echo " injecting GIT COMMIT: $GIT_COMMIT" && \
CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOFLAGS=-mod=vendor \
go build -ldflags "-w -s -X github.com/project-koku/koku-metrics-operator/internal/controller.GitCommit=$GIT_COMMIT" -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

# For terminal access, use this image:
# FROM gcr.io/distroless/base:debug-nonroot

LABEL \
    com.redhat.component="koku-metrics-operator-container" \
    description="Koku Metrics Operator" \
    io.k8s.description="Operator to deploy and manage instances of Koku Metrics" \
    io.k8s.display-name="Koku Metrics Operator" \
    io.openshift.tags="cost,cost-management,prometheus,servicetelemetry,operators" \
    maintainer="Cost Management <cost-mgmt@redhat.com>" \
    name="koku-metrics-operator" \
    summary="Koku Metrics Operator"

WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
