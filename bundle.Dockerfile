FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=costmanagement-metrics-operator
LABEL operators.operatorframework.io.bundle.channels.v1=stable
LABEL operators.operatorframework.io.bundle.channel.default.v1=stable
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.35.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

# Copy files to locations specified by labels.
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/

LABEL name="costmanagement-metrics-operator-bundle" \
    # Openshift specific labels
    io.k8s.display-name="Cost Management Metrics Operator" \
    io.k8s.description="Component required to gather metrics from Prometheus and package them to be uploaded to Red Hat Insights cost management. The operator can work in clusters connected to the Internet and air-gapped (with additional configuration and steps)" \
    io.openshift.build.commit.id="f83f111551b4aa06b49fed864945ab01c45d04d0" \
    io.openshift.build.commit.url="https://github.com/project-koku/koku-metrics-operator/commit/f83f111551b4aa06b49fed864945ab01c45d04d0" \
    io.openshift.build.source-location="https://github.com/project-koku/koku-metrics-operator" \
    io.openshift.maintainer.component="Cost Management Metrics Operator" \
    io.openshift.maintainer.product="OpenShift Container Platform" \
    io.openshift.tags="openshift" \
    # Labels required for release via Konflux 
    com.redhat.component="costmanagement-metrics-operator-bundle-container" \
    com.redhat.delivery.appregistry="false" \
    com.redhat.delivery.operator.bundle="true" \
    com.redhat.openshift.versions="v4.12" \
    maintainer="Cost Management <costmanagement@redhat.com>" \
    summary="Operator required to upload metrics data to the cost management service in console.redhat.com." \
    version="3.3.2" \
    release="1" \
    distribution-scope="public" \
    description="Component required to gather metrics from Prometheus and package them to be uploaded to Red Hat Insights cost management. The operator can work in clusters connected to the Internet and air-gapped (with additional configuration and steps)" \
    url="https://github.com/project-koku/koku-metrics-operator" \
    vendor="Red Hat, Inc."