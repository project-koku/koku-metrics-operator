# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
PREVIOUS_VERSION ?= 4.0.0
VERSION ?= 4.1.0

MIN_KUBE_VERSION = 1.24.0
MIN_OCP_VERSION = 4.12

# Default bundle image tag
IMAGE_TAG_BASE ?= quay.io/project-koku/koku-metrics-operator
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)
PREVIOUS_BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(PREVIOUS_VERSION)
CATALOG_IMG ?= quay.io/project-koku/kmc-test-catalog:v$(VERSION)
DOWNSTREAM_IMAGE_TAG ?= registry-proxy.engineering.redhat.com/rh-osbs/costmanagement-metrics-operator:$(VERSION)

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
	@sed -i "" '/prometheus_config/d' testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '  prometheus_config:' >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '    service_address: $(EXTERNAL_PROM_ROUTE)'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '    skip_tls_verification: true' >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: add-auth
add-auth:
	@sed -i "" '/authentication/d' testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '  authentication:'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '    type: basic'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '    secret_name: dev-auth-secret' >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: local-validate-cert
local-validate-cert:
	@sed -i "" '/upload/d' testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '  upload:'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	@echo '    validate_cert: false'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: add-ci-route
add-ci-route:
	@echo '  api_url: https://ci.console.redhat.com'  >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: add-spec
add-spec:
	@echo 'spec:' >> testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

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
vendor: ## Update deps, tidy and vendor modules.
	go get -u ./...
	go mod tidy
	go mod vendor

.PHONY: verify-manifests
verify-manifests: ## Verify manifests are up to date.
	./hack/verify-manifests.sh

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.x
.PHONY: test
test: envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-qemu
test-qemu: envtest-not-local ## Run tests - specific for multiarch in github action
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST_NOT_LOCAL) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./...

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
deploy-cr:  ## Deploy a CostManagementMetricsConfig CR for controller running in K8s cluster.
	@cp testing/costmanagementmetricsconfig-template.yaml testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
ifeq ($(AUTH), service-account)
	$(MAKE) setup-sa-auth
	$(MAKE) add-sa-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: deploy-local-cr
deploy-local-cr:  ## Deploy a CostManagementMetricsConfig CR for controller running on local host.
	@cp testing/costmanagementmetricsconfig-template.yaml testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml
	$(MAKE) add-prom-route
	$(MAKE) local-validate-cert
ifeq ($(AUTH), service-account)
	$(MAKE) setup-sa-auth
	$(MAKE) add-sa-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml

.PHONY: get-token-and-cert
get-token-and-cert: ## Get a token and the cluster's service CA certificate from a running K8s cluster for local development. The --duration flag is optional but useful in development for longer-lived tokens.
	oc create token koku-metrics-controller-manager -n koku-metrics-operator --duration=8760h > $(SECRET_ABSPATH)/token
	oc get configmap kube-root-ca.crt -n koku-metrics-operator -o jsonpath='{.data.ca\.crt}' > $(SECRET_ABSPATH)/service-ca.crt

##@ Build Bundle and Test Catalog

.PHONY: bundle
bundle: operator-sdk manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	rm -rf ./bundle
	$(OPERATOR_SDK) generate kustomize manifests
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_SHA)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

	$(YQ) -i '.annotations."com.redhat.openshift.versions" = "$(MIN_OCP_VERSION)"' bundle/metadata/annotations.yaml
	$(YQ) -i '(.annotations."com.redhat.openshift.versions" | key) head_comment="OpenShift specific annotations."' bundle/metadata/annotations.yaml
	$(YQ) -i '.metadata.annotations.containerImage = "$(IMAGE_SHA)"' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.description |= load_str("docs/csv-description.md")' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.minKubeVersion = "$(MIN_KUBE_VERSION)"' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.relatedImages = [{"name": "koku-metrics-operator", "image": "$(IMAGE_SHA)"}]' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.replaces = "koku-metrics-operator.v$(PREVIOUS_VERSION)"' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
ifdef NAMESPACE
	$(YQ) -i '.metadata.namespace = "$(NAMESPACE)"' bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml
endif

	$(OPERATOR_SDK) bundle validate bundle/ --select-optional name=multiarch
	$(OPERATOR_SDK) bundle validate bundle/ --select-optional suite=operatorframework

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(CONTAINER_TOOL) build --platform linux/x86_64 -t $(BUNDLE_IMG) -f bundle.Dockerfile .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(CONTAINER_TOOL) push $(BUNDLE_IMG)

.PHONY: bundle-deploy-previous
bundle-deploy-previous: operator-sdk ## Deploy previous bundle into a cluster.
	$(OPERATOR_SDK) run bundle $(PREVIOUS_BUNDLE_IMG) --namespace=koku-metrics-operator --install-mode=OwnNamespace

.PHONY: bundle-deploy
bundle-deploy: operator-sdk ## Deploy current bundle into a cluster.
	$(OPERATOR_SDK) run bundle $(BUNDLE_IMG) --namespace=koku-metrics-operator --install-mode=OwnNamespace --security-context-config restricted

.PHONY: bundle-deploy-upgrade
bundle-deploy-upgrade: operator-sdk ## Test a bundle upgrade. The previous bundle must have been deployed first.
	$(OPERATOR_SDK) run bundle-upgrade $(BUNDLE_IMG) --namespace=koku-metrics-operator

.PHONY: bundle-deploy-cleanup
bundle-deploy-cleanup: operator-sdk ## Delete the entirety of the deployed bundle
	$(OPERATOR_SDK) cleanup koku-metrics-operator --delete-crds --delete-all --namespace=koku-metrics-operator

##@ Generate downstream file changes

#### Updates code for downstream release
REMOVE_FILES = koku-metrics-operator/ config/scorecard/
UPSTREAM_LOWERCASE = koku
UPSTREAM_UPPERCASE = Koku
DOWNSTREAM_LOWERCASE = costmanagement
DOWNSTREAM_UPPERCASE = CostManagement
.PHONY: downstream
downstream: operator-sdk ## Generate the code changes necessary for the downstream image.
	rm -rf $(REMOVE_FILES)
	# sed replace everything but the Makefile
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec sed -i '' 's/$(UPSTREAM_UPPERCASE)/$(DOWNSTREAM_UPPERCASE)/g' {} +
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec sed -i '' 's/$(UPSTREAM_LOWERCASE)/$(DOWNSTREAM_LOWERCASE)/g' {} +

	- LC_ALL=C find internal/* -type f -exec sed -i '' '/^\/\/ +kubebuilder:rbac:groups/ s/$(UPSTREAM_LOWERCASE)/$(DOWNSTREAM_LOWERCASE)/g' {} +
	- sed -i '' 's/isCertified bool = false/isCertified bool = true/g' internal/packaging/packaging.go
	# clean up the other files
	# - git clean -fx
	# mv the sample to the correctly named file
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec rename -f -- 's/$(UPSTREAM_UPPERCASE)/$(DOWNSTREAM_UPPERCASE)/g' {} +
	- LC_ALL=C find api/v1beta1 config/* docs/* -type f -exec rename -f -- 's/$(UPSTREAM_LOWERCASE)/$(DOWNSTREAM_LOWERCASE)/g' {} +

	$(YQ) -i '.projectName = "costmanagement-metrics-operator"' PROJECT
	$(YQ) -i '.resources.[0].group = "costmanagement-metrics-cfg"' PROJECT
	$(YQ) -i '.resources.[0].kind = "CostManagementMetricsConfig"' PROJECT
	$(YQ) -i 'del(.resources[] | select(. == "../scorecard"))' config/manifests/kustomization.yaml

	$(MAKE) manifests

	mkdir -p costmanagement-metrics-operator/$(VERSION)/
	rm -rf ./bundle costmanagement-metrics-operator/$(VERSION)/

	$(OPERATOR_SDK) generate kustomize manifests
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(DOWNSTREAM_IMAGE_TAG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle --overwrite --version $(VERSION) --default-channel=stable --channels=stable

	$(YQ) -i '.annotations."com.redhat.openshift.versions" = "$(MIN_OCP_VERSION)"' bundle/metadata/annotations.yaml
	$(YQ) -i '(.annotations."com.redhat.openshift.versions" | key) head_comment="OpenShift specific annotations."' bundle/metadata/annotations.yaml

	$(YQ) -i '.metadata.annotations.repository = "https://github.com/project-koku/koku-metrics-operator"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.annotations.certified = "true"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.annotations.support = "Red Hat"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.annotations."operators.openshift.io/valid-subscription" = "[\"OpenShift Kubernetes Engine\", \"OpenShift Container Platform\", \"OpenShift Platform Plus\"]"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.annotations.containerImage = "$(DOWNSTREAM_IMAGE_TAG)"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.name = "costmanagement-metrics-operator.$(VERSION)"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.install.spec.deployments.[0].spec.template.spec.containers.[0].command = ["/usr/bin/costmanagement-metrics-operator"]' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.description |= load_str("docs/csv-description.md")' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.displayName = "Cost Management Metrics Operator"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.minKubeVersion = "$(MIN_KUBE_VERSION)"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.replaces = "costmanagement-metrics-operator.$(PREVIOUS_VERSION)"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.links[0].url = "https://github.com/project-koku/koku-metrics-operator"' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml

	sed -i '' 's/CostManagement Metrics Operator/Cost Management Metrics Operator/g' bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml

	# scripts/update_bundle_dockerfile.py
	cat downstream-assets/bundle.Dockerfile.txt >> bundle.Dockerfile
	sed -i '' '/^COPY / s/bundle\///g' bundle.Dockerfile
	sed -i '' 's/MIN_OCP_VERSION/$(MIN_OCP_VERSION)/g' bundle.Dockerfile
	sed -i '' 's/REPLACE_VERSION/$(VERSION)/g' bundle.Dockerfile

	cp -r ./bundle/ costmanagement-metrics-operator/$(VERSION)/
	cp bundle.Dockerfile costmanagement-metrics-operator/$(VERSION)/Dockerfile

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
ENVTEST_NOT_LOCAL ?= $(shell go env GOPATH)/bin/$(shell go env GOOS)_$(shell go env GOARCH)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.3.0
CONTROLLER_TOOLS_VERSION ?= v0.16.5
SETUP_ENVTEST_VERSION ?= v0.0.0-20240318095156-c7e1dc9b5302
YQ_VERSION ?= v4.2.0

# Set the Operator SDK version to use. By default, what is installed on the system is used.
# This is useful for CI or a project to utilize a specific version of the operator-sdk toolkit.
OPERATOR_SDK_VERSION ?= v1.41.1

.PHONY: yq
YQ ?= $(LOCALBIN)/yq
yq: ## Download yq locally if necessary.
ifeq (,$(wildcard $(YQ)))
ifeq (, $(shell which yq 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(YQ)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(YQ) https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$${OS}_${{ARCH}} && chmod +x $(YQ)
	}
else
YQ = $(shell which yq)
endif
endif

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
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

.PHONY: envtest-not-local
envtest-not-local: ## Download envtest-setup for qemu unit tests - specific to github action.
	test -s setup-envtest || go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

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
