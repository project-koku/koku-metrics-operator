#!/usr/bin/env bash
set -euo pipefail

echo "=== koku-metrics-operator — Codespace Setup ==="

# ── 1. Wait for Docker-in-Docker (same as PoC) ──
echo "Waiting for Docker daemon..."
for i in $(seq 1 30); do
    if docker info >/dev/null 2>&1; then
        echo "Docker ready."
        break
    fi
    sleep 2
done
if ! docker info >/dev/null 2>&1; then
    echo "ERROR: Docker daemon not available after 60s"
    exit 1
fi

# ── 2. Create k3d cluster (same mechanism as PoC, renamed) ──
# Throwaway k3s for `make install` / `make deploy` smoke tests and fast
# reconcile loops. NOT a substitute for OCP e2e (no monitoring stack,
# no thanos-querier, no ClusterVersion, no OLM CSV patching).
if ! k3d cluster list 2>/dev/null | grep -q 'koku-dev'; then
    echo "Creating k3d cluster..."
    k3d cluster create koku-dev \
        --wait \
        --timeout 120s \
        --agents 0 \
        --k3s-arg "--disable=traefik@server:0"
else
    echo "k3d cluster 'koku-dev' already exists."
fi

echo "Waiting for k3s node to be Ready..."
for i in $(seq 1 60); do
    if kubectl get nodes 2>/dev/null | grep -q ' Ready'; then
        echo "k3s node ready."
        break
    fi
    sleep 2
done

# ── 3. Go dependencies (vendor/ is committed, so this is fast) ──
echo "Downloading Go modules..."
go mod download

# ── 4. Build operator tooling via Makefile (pins versions) ──
# Installs bin/kustomize (v5.8.1), bin/controller-gen (v0.20.0),
# bin/setup-envtest (release-0.22) into ./bin.
echo "Installing kustomize / controller-gen / setup-envtest..."
make kustomize controller-gen envtest

# ── 5. Fetch envtest binaries (kube-apiserver + etcd for `make test`) ──
# Unit tests do NOT need the k3d cluster — they use KUBEBUILDER_ASSETS.
echo "Fetching envtest binaries..."
make envtest 2>/dev/null || true
ENVTEST_K8S_VERSION="$(go list -m -f '{{ .Version }}' k8s.io/api | awk -F'[v.]' '{printf "1.%d", $3}')"
./bin/setup-envtest use "$ENVTEST_K8S_VERSION" --bin-dir ./bin -p path >/dev/null || true

# ── 6. pre-commit hooks (Makefile `lint` target depends on it) ──
if [ -f .pre-commit-config.yaml ]; then
    echo "Installing pre-commit hooks..."
    pre-commit install --install-hooks || true
fi

# ── 7. Verify tools ──
echo ""
echo "--- Tool Versions ---"
go version
oc version --client 2>/dev/null || oc version --client=true 2>/dev/null || true
kubectl version --client 2>/dev/null || true
kustomize version 2>/dev/null || ./bin/kustomize version 2>/dev/null || true
controller-gen --version 2>/dev/null || ./bin/controller-gen --version 2>/dev/null || true
operator-sdk version 2>/dev/null || true
k3d version
k9s version --short 2>/dev/null || k9s version || true
gh version --short 2>/dev/null || gh --version | head -1
docker version --format 'Docker {{.Client.Version}}'

echo ""
echo "--- Cluster Status (k3d throwaway) ---"
kubectl get nodes
kubectl cluster-info || true

echo ""
echo "=========================================="
echo "  Codespace ready!"
echo ""
echo "  k3d cluster 'koku-dev' is running (k3s, smoke tests only)."
echo ""
echo "  Quick checks (no cluster needed):"
echo "    make fmt && make test"
echo "    make verify-manifests"
echo ""
echo "  Smoke test on k3d:"
echo "    bash .devcontainer/deploy-operator.sh"
echo ""
echo "  Full e2e needs a real OCP cluster:"
echo "    oc login --token=<token> --server=<server>"
echo "    make get-token-and-cert"
echo "    make run ENABLE_WEBHOOKS=false"
echo "    make deploy-local-cr"
echo ""
echo "  k9s is available — just run: k9s"
echo "=========================================="
