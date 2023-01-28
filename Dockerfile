# Build the manager binary
FROM registry.access.redhat.com/ubi8/go-toolset:1.18.9 as builder

USER root

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we do not need to re-download as much
# and so that source changes do not invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY clusterversion/ clusterversion/
COPY collector/ collector/
COPY controllers/ controllers/
COPY crhchttp/ crhchttp/
COPY dirconfig/ dirconfig/
COPY packaging/ packaging/
COPY sources/ sources/
COPY storage/ storage/
COPY strset/ strset/

# Copy git to inject the commit during build
COPY .git .git
# Build
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
echo " injecting GIT COMMIT: $GIT_COMMIT" && \
CGO_ENABLED=0 GOOS=linux GO111MODULE=on \
go build -ldflags "-w -s -X github.com/project-koku/koku-metrics-operator/controllers.GitCommit=$GIT_COMMIT" -a -o manager main.go

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
