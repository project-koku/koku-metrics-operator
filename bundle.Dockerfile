FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=costmanagement-metrics-operator
LABEL operators.operatorframework.io.bundle.channels.v1=stable
LABEL operators.operatorframework.io.bundle.channel.default.v1=stable
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.41.1
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

# Copy files to locations specified by labels.
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/

# Openshift specific labels
LABEL io.k8s.display-name="Cost Management Metrics Operator"
LABEL io.k8s.description="Component required to gather metrics from Prometheus, package and upload them to the cost management service in the cloud. The operator can work in clusters connected to the Internet and air-gapped (with additional configuration and steps)"
LABEL io.openshift.build.commit.id="d0bc2c257b15ad2f6d73b4e77badbd9706d1a694"
LABEL io.openshift.build.commit.url="https://github.com/project-koku/koku-metrics-operator/commit/d0bc2c257b15ad2f6d73b4e77badbd9706d1a694"
LABEL io.openshift.build.source-location="https://github.com/project-koku/koku-metrics-operator"
LABEL io.openshift.maintainer.component="Cost Management Metrics Operator"
LABEL io.openshift.maintainer.product="OpenShift Container Platform"
LABEL io.openshift.tags="openshift"
LABEL com.redhat.component="costmanagement-metrics-operator-bundle-container"
LABEL com.redhat.delivery.appregistry="false"
LABEL com.redhat.delivery.operator.bundle="true"
LABEL com.redhat.openshift.versions="v4.12"
LABEL name="costmanagement/costmanagement-metrics-operator-bundle"
LABEL cpe="cpe:/a:redhat:cost_management:4::el9"
LABEL maintainer="<costmanagement@redhat.com>"
LABEL summary="Operator required to upload metrics data to the cost management service in console.redhat.com."
LABEL version="4.2.0"
LABEL release="4.2.0"
LABEL distribution-scope="public"
LABEL description="Component required to gather metrics from Prometheus and package them to be uploaded to Red Hat Insights cost management. The operator can work in clusters connected to the Internet and air-gapped (with additional configuration and steps)"
LABEL url="https://github.com/project-koku/koku-metrics-operator"
LABEL vendor="Red Hat, Inc."
