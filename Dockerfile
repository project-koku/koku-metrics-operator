FROM --platform=${BUILDPLATFORM:-linux/amd64} brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.22 AS builder

USER root

# cachito
COPY $REMOTE_SOURCE $REMOTE_SOURCE_DIR
WORKDIR $REMOTE_SOURCE_DIR/app

RUN go version
RUN GOOS=linux GO111MODULE=on go build -ldflags "-w -s -X github.com/project-koku/koku-metrics-operator/internal/controller.GitCommit=6b4d72a4a629527c1de086b416faf6d226fe587a" -v -o bin/costmanagement-metrics-operator ${REMOTE_SOURCE_DIR}/app/cmd/main.go

FROM registry.redhat.io/ubi8/ubi-micro:latest AS base-env

WORKDIR /
COPY --from=builder $REMOTE_SOURCE_DIR/app/bin/costmanagement-metrics-operator /usr/bin/costmanagement-metrics-operator
COPY --from=builder /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/ca-bundle.crt
COPY --from=builder /etc/pki/ca-trust/extracted/openssl/ca-bundle.trust.crt /etc/ssl/certs/ca-bundle.trust.crt

ENTRYPOINT ["/usr/bin/costmanagement-metrics-operator"]

LABEL \
    com.redhat.component="costmanagement-metrics-operator-container"  \
    description="Red Hat Cost Management Metrics Operator"  \
    io.k8s.description="Operator to deploy and manage instances of Cost Management Metrics"  \
    io.k8s.display-name="Cost Management Metrics Operator"  \
    io.openshift.tags="cost,cost-management,prometheus,servicetelemetry,operators"  \
    maintainer="Cost Management <cost-mgmt@redhat.com>"  \
    name="costmanagement-metrics-operator"  \
    summary="Red Hat Cost Management Metrics Operator"  \
    version="3.3.1"
