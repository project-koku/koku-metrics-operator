# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
PREVIOUS_VERSION ?= 3.1.0
VERSION ?= 3.2.0

# Default bundle image tag
IMAGE_TAG_BASE ?= quay.io/project-koku/koku-metrics-operator
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)
CATALOG_IMG ?= quay.io/project-koku/kmc-test-catalog:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= quay.io/project-koku/koku-metrics-operator:v$(VERSION)

# Options for 'bundle-build'
DEFAULT_CHANNEL ?= beta
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
# CRD_OPTIONS ?= "crd:trivialVersions=true"
CRD_OPTIONS ?= "crd:crdVersions={v1}"

GIT_COMMIT=$(shell git rev-parse HEAD)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

EXTERNAL_PROM_ROUTE=https://$(shell oc get routes thanos-querier -n openshift-monitoring -o "jsonpath={.spec.host}")
IMAGE_SHA=$(shell docker inspect --format='{{index .RepoDigests 0}}' ${IMG})

OS = $(shell go env GOOS)
ARCH = $(shell go env GOARCH)

# DOCKER := $(shell which docker 2>/dev/null)
export DOCKER_DEFAULT_PLATFORM = linux/x86_64

# Set the Operator SDK version to use. By default, what is installed on the system is used.
# This is useful for CI or a project to utilize a specific version of the operator-sdk toolkit.
OPERATOR_SDK_VERSION ?= v1.33.0
OPERATOR_REGISTRY_VERSION ?= v1.34.0

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup-auth
setup-auth:
	@cp testing/auth-secret-template.yaml testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSB1c2VybmFtZQ==/$(shell printf "$(shell echo $(or $(USER),console.redhat.com username))" | base64)/g' testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSBwYXNzd29yZA==/$(shell printf "$(shell echo $(or $(PASS),console.redhat.com password))" | base64)/g' testing/authentication_secret.yaml

.PHONY: add-prom-route
add-prom-route:
	@sed -i "" '/prometheus_config/d' testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '  prometheus_config:' >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '    service_address: $(EXTERNAL_PROM_ROUTE)'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '    skip_tls_verification: true' >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

.PHONY: add-auth
add-auth:
	@sed -i "" '/authentication/d' testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '  authentication:'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '    type: basic'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '    secret_name: dev-auth-secret' >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

.PHONY: local-validate-cert
local-validate-cert:
	@sed -i "" '/upload/d' testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '  upload:'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	@echo '    validate_cert: false'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

.PHONY: add-ci-route
add-ci-route:
	@echo '  api_url: https://ci.console.redhat.com'  >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

.PHONY: add-spec
add-spec:
	@echo 'spec:' >> testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: lint
lint: ## Run pre-commit
	pre-commit run --all-files

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: vendor
vendor: ## Run `go mod vendor`.
	go get -u
	go mod tidy
	go mod vendor

.PHONY: verify-manifests
verify-manifests: ## Verify manifests are up to date.
	./hack/verify-manifests.sh

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.28.0
.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: manifests generate fmt vet vendor ## Build manager binary.
	go build -o bin/manager cmd/main.go

SECRET_ABSPATH ?= ./testing
WATCH_NAMESPACE ?= koku-metrics-operator
# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	kubectl apply -f testing/sa.yaml
	WATCH_NAMESPACE=$(WATCH_NAMESPACE) SECRET_ABSPATH=$(SECRET_ABSPATH) GIT_COMMIT=$(GIT_COMMIT) go run cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
PLATFORM ?= linux/amd64
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build --platform=$(PLATFORM) -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	- $(CONTAINER_TOOL) buildx create --name operator-builder --driver-opt image=moby/buildkit:v0.12.4
	- $(CONTAINER_TOOL) buildx use operator-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --provenance=false --tag ${IMG} .
	- $(CONTAINER_TOOL) buildx rm operator-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: deploy-to-file
deploy-to-file: manifests kustomize ## Create a deployment file
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > testing/deployment.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy-cr
deploy-cr:  ## Deploy a KokuMetricsConfig CR for controller running in K8s cluster.
	@cp testing/kokumetricsconfig-template.yaml testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
ifeq ($(AUTH), basic)
	$(MAKE) setup-auth
	$(MAKE) add-auth
	oc apply -f testing/authentication_secret.yaml
else ifeq ($(AUTH), service-account)
	$(MAKE) setup-sa-auth
	$(MAKE) add-sa-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

.PHONY: deploy-local-cr
deploy-local-cr:  ## Deploy a KokuMetricsConfig CR for controller running on local host.
	@cp testing/kokumetricsconfig-template.yaml testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
	$(MAKE) add-prom-route
	$(MAKE) local-validate-cert
ifeq ($(AUTH), basic)
	$(MAKE) setup-auth
	$(MAKE) add-auth
	oc apply -f testing/authentication_secret.yaml
else ifeq ($(AUTH), service-account)
	$(MAKE) setup-sa-auth
	$(MAKE) add-sa-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml

SECRET_NAME = $(shell oc get secrets -o name | grep -m 1 koku-metrics-controller-manager-token-)
.PHONY: get-token-and-cert
get-token-and-cert:  ## Get a token from a running K8s cluster for local development.
	printf "%s" "$(shell oc whoami --show-token)" > $(SECRET_ABSPATH)/token
	oc get -o template $(SECRET_NAME) -o go-template=='{{index .data "service-ca.crt"|base64decode}}' > $(SECRET_ABSPATH)/service-ca.crt

##@ Build Bundle and Test Catalog

NAMESPACE ?= ""
.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	mkdir -p koku-metrics-operator/$(VERSION)/
	rm -rf ./bundle koku-metrics-operator/$(VERSION)/
	operator-sdk generate kustomize manifests
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_SHA}
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
	cp -r ./bundle/ koku-metrics-operator/$(VERSION)/
	cp bundle.Dockerfile koku-metrics-operator/$(VERSION)/Dockerfile
	scripts/txt_replace.py $(VERSION) $(PREVIOUS_VERSION) ${IMAGE_SHA} --namespace=${NAMESPACE}

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	cd koku-metrics-operator/$(VERSION) && $(CONTAINER_TOOL) build -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(CONTAINER_TOOL) push $(BUNDLE_IMG)

.PHONY: test-catalog
test-catalog: opm ## Build a test-catalog
	$(OPM) index add --from-index quay.io/project-koku/kmc-test-catalog:v${PREVIOUS_VERSION} --bundles ${BUNDLE_IMG} --tag ${CATALOG_IMG} --container-tool docker

.PHONY: test-catalog-push
test-catalog-push: ## Push the test-catalog
	$(CONTAINER_TOOL) push ${CATALOG_IMG}


##@ Generate downstream file changes

#### Updates code for downstream release
REMOVE_FILES = koku-metrics-operator/
UPSTREAM_LOWERCASE = koku
UPSTREAM_UPPERCASE = Koku
DOWNSTREAM_LOWERCASE = costmanagement
DOWNSTREAM_UPPERCASE = CostManagement
.PHONY: downstream
downstream: ## Generate the code changes necessary for the downstream image.
	rm -rf $(REMOVE_FILES)
	# sed replace everything but the Makefile
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec sed -i -- 's/$(UPSTREAM_UPPERCASE)/$(DOWNSTREAM_UPPERCASE)/g' {} +
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec sed -i -- 's/$(UPSTREAM_LOWERCASE)/$(DOWNSTREAM_LOWERCASE)/g' {} +
	# fix the cert
	- sed -i -- 's/ca-certificates.crt/ca-bundle.crt/g' internal/crhchttp/http_cloud_dot_redhat.go
	- sed -i -- 's/isCertified bool = false/isCertified bool = true/g' internal/packaging/packaging.go
	# clean up the other files
	- git clean -fx
	# mv the sample to the correctly named file
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec rename -f -- 's/$(UPSTREAM_UPPERCASE)/$(DOWNSTREAM_UPPERCASE)/g' {} +
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec rename -f -- 's/$(UPSTREAM_LOWERCASE)/$(DOWNSTREAM_LOWERCASE)/g' {} +
	$(MAKE) generate
	$(MAKE) manifests

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.1.1
CONTROLLER_TOOLS_VERSION ?= v0.13.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: opm
OPM = $(LOCALBIN)/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPERATOR_REGISTRY_VERSION)/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

.PHONY: operator-sdk
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif
