#!/usr/bin/env bash
set -euo pipefail

# Smoke-test the operator on the throwaway k3d cluster.
# Mirrors docs/local-development.md steps 1-7, minus anything OCP-specific
# (no cluster-monitoring-view, no thanos-querier route, no console.redhat.com auth).
# For full e2e, `oc login` to a real OCP cluster and follow local-development.md.
#
# Usage: bash .devcontainer/deploy-operator.sh

NAMESPACE="${WATCH_NAMESPACE:-koku-metrics-operator}"

echo "=== Deploy operator smoke test (k3d) ==="

echo "--- 1. CRDs ---"
make install

echo "--- 2. Namespace + ServiceAccount ---"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f testing/sa.yaml

echo "--- 3. Build manager binary (vendored deps, no upgrades) ---"
go build -mod=vendor -o bin/manager cmd/main.go

echo "--- 4. Verify manifests ---"
make verify-manifests

echo ""
echo "=========================================="
echo "  Smoke test done."
echo ""
echo "  CRDs + SA are installed, binary builds, manifests verify."
echo "  Reconciliation against Prometheus/OCP is NOT exercised on k3d."
echo ""
echo "  To run the operator locally against this k3d cluster:"
echo "    SECRET_ABSPATH=\$PWD/testing WATCH_NAMESPACE=$NAMESPACE \\"
echo "      make run-quick ENABLE_WEBHOOKS=false"
echo ""
echo "  To deploy the controller image into k3d (needs registry):"
echo "    make docker-build IMG=<registry>/koku-metrics-operator:dev"
echo "    k3d image import <registry>/koku-metrics-operator:dev -c koku-dev"
echo "    make deploy IMG=<registry>/koku-metrics-operator:dev"
echo "=========================================="
