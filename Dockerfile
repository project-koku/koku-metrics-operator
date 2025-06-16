FROM --platform=${BUILDPLATFORM:-linux/amd64} brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.24_test AS builder

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

# Set the GOEXPERIMENT to enable the strict FIPS runtime check.
ENV GOEXPERIMENT=strictfipsruntime

# Build
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
    echo " injecting GIT COMMIT: $GIT_COMMIT" && \
    CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOFLAGS=-mod=vendor \
    go build -ldflags "-w -s -X github.com/project-koku/koku-metrics-operator/internal/controller.GitCommit=$GIT_COMMIT" -a -o manager cmd/main.go

# ubi9-micro-openssl
FROM registry.access.redhat.com/ubi9/ubi AS ubi-micro-build
RUN mkdir -p /mnt/rootfs
RUN rpm --root /mnt/rootfs --import /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
RUN yum install --installroot /mnt/rootfs --releasever 9 --setopt install_weak_deps=false --nodocs -y openssl; yum clean all
RUN rm -rf /mnt/rootfs/var/cache/*

FROM registry.access.redhat.com/ubi9/ubi-micro AS ubi9-micro
COPY --from=ubi-micro-build /mnt/rootfs/ /
CMD /usr/bin/openssl

WORKDIR /
COPY --from=builder /workspace/manager /manager
COPY --from=builder /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/ca-bundle.crt
COPY --from=builder /etc/pki/ca-trust/extracted/openssl/ca-bundle.trust.crt /etc/ssl/certs/ca-bundle.trust.crt
COPY LICENSE /licenses/Apache-2.0.txt

USER 65532:65532


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
