FROM --platform=${BUILDPLATFORM:-linux/amd64} brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.24 AS builder

ARG TARGETOS
ARG TARGETARCH

USER root

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Copy git to inject the commit during build
COPY .git .git

# Use FIPS crypto module at build time
# ARG GOFIPS140=v1.0.0

# Build
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
    echo " injecting GIT COMMIT: $GIT_COMMIT" && \
    CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOFLAGS=-mod=vendor \
    go build -ldflags "-w -s -X github.com/project-koku/koku-metrics-operator/internal/controller.GitCommit=$GIT_COMMIT" -a -o manager cmd/main.go

FROM registry.redhat.io/ubi9/ubi-micro:latest AS base-env

WORKDIR /
COPY --from=builder /workspace/manager /usr/bin/costmanagement-metrics-operator
COPY --from=builder /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/ca-bundle.crt
COPY --from=builder /etc/pki/ca-trust/extracted/openssl/ca-bundle.trust.crt /etc/ssl/certs/ca-bundle.trust.crt
COPY LICENSE /licenses/Apache-2.0.txt

USER 65532:65532

# Enable FIPS mode at runtime
# ENV GODEBUG=fips140=on

LABEL \
    com.redhat.component="costmanagement-metrics-operator-container"  \
    description="Red Hat Cost Management Metrics Operator"  \
    distribution-scope="public" \
    io.k8s.description="Operator to deploy and manage instances of Cost Management Metrics"  \
    io.k8s.display-name="Cost Management Metrics Operator"  \
    io.openshift.tags="cost,cost-management,prometheus,servicetelemetry,operators"  \
    maintainer="Cost Management <cost-mgmt@redhat.com>"  \
    name="costmanagement-metrics-operator"  \
    summary="Red Hat Cost Management Metrics Operator"  \
    version="4.0.0" \
    vendor="Red Hat, Inc."

ENTRYPOINT ["/manager"]
