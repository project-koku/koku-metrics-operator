FROM mcr.microsoft.com/devcontainers/go:1-1.26-bookworm

# Remove the stale yarn repo that ships with the base image — its GPG key
# is expired and breaks apt-get update in Codespaces.
# (Same fix as cost_ai_grid_poc PoC.)
RUN rm -f /etc/apt/sources.list.d/yarn.list 2>/dev/null; \
    apt-get update && apt-get install -y --no-install-recommends \
    python3-pip \
    jq \
    curl \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# k3d (runs k3s clusters inside Docker — codespace containers lack systemd for bare k3s)
# Same as PoC: gives a throwaway cluster for `make install` / `make deploy` smoke tests.
# NOTE: this is k3s, not OpenShift — full e2e (monitoring stack, thanos-querier,
# ClusterVersion, OLM) still needs a real OCP cluster via `oc login` from inside the codespace.
RUN curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# OpenShift client (oc). Pinned to a stable 4.x stream; override OC_VERSION as needed.
# Provides `oc` + a matching `kubectl`. The devcontainer feature set only ships
# plain kubectl, so we install oc explicitly (operator docs require oc).
ARG OC_VERSION=4.17
RUN ARCH="$(uname -m)" && case "$ARCH" in x86_64) OC_ARCH=x86_64 ;; aarch64|arm64) OC_ARCH=arm64 ;; *) OC_ARCH=x86_64 ;; esac && \
    curl -sL "https://mirror.openshift.com/pub/openshift-v4/${OC_ARCH}/clients/ocp/${OC_VERSION}/openshift-client-linux.tar.gz" \
    | tar xz -C /usr/local/bin oc kubectl \
    && chmod +x /usr/local/bin/oc /usr/local/bin/kubectl

# k9s (same as PoC — handy for watching reconciles)
ARG K9S_VERSION=v0.50.4
RUN curl -sL "https://github.com/derailed/k9s/releases/download/${K9S_VERSION}/k9s_Linux_amd64.tar.gz" \
    | tar xz -C /usr/local/bin k9s \
    && chmod +x /usr/local/bin/k9s

# operator-sdk (matches Makefile OPERATOR_SDK_VERSION v1.41.1)
ARG OPERATOR_SDK_VERSION=v1.41.1
RUN OS=$(go env GOOS 2>/dev/null || echo linux) && ARCH=$(go env GOARCH 2>/dev/null || echo amd64) && \
    curl -sSLo /usr/local/bin/operator-sdk \
    "https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_${OS}_${ARCH}" \
    && chmod +x /usr/local/bin/operator-sdk

# yq (matches Makefile YQ_VERSION v4.52.2 — used by bundle/downstream targets)
ARG YQ_VERSION=v4.52.2
RUN curl -sSLo /usr/local/bin/yq \
    "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64" \
    && chmod +x /usr/local/bin/yq

# pre-commit (Makefile `lint` target runs `pre-commit run --all-files`)
RUN pip3 install --break-system-packages pre-commit

# Custom welcome message
COPY .devcontainer/first-run-notice.txt /usr/local/etc/vscode-dev-containers/first-run-notice.txt
