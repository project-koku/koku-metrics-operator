# Current Operator version
VERSION ?= 0.9.0
# Default bundle image tag
BUNDLE_IMG ?= quay.io/project-koku/koku-metrics-operator-controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= quay.io/project-koku/koku-metrics-operator:v$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
# CRD_OPTIONS ?= "crd:trivialVersions=true"
CRD_OPTIONS ?= "crd:crdVersions={v1},trivialVersions=true"

# Use git branch for dev team deployment of pushed branches
GITBRANCH=$(shell git branch --show-current)
GITBRANCH_IMG="quay.io/project-koku/koku-metrics-operator:${GITBRANCH}"
GIT_COMMIT=$(shell git rev-parse HEAD)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

EXTERNAL_PROM_ROUTE=https://$(shell oc get routes thanos-querier -n openshift-monitoring -o "jsonpath={.spec.host}")

help:
	@echo "Please use \`make <target>' where <target> is one of:"
	@echo "--- Bundle Commands ---"
	@echo "  bundle                             Generate bundle manifests and metadata, then validate generated files."
	@echo "      DEFAULT_CHANNEL=<channel>                  @param - Required. The default channel for the bundle."
	@echo "  bundle-build                       Build the bundle image."
	@echo "      BUNDLE_IMG=<quay.io image>                 @param - Optional. The quay.io image name."
	@echo "  bundle-push                        Push the bundle image."
	@echo "      BUNDLE_IMG=<quay.io image>                 @param - Optional. The quay.io image name."
	@echo "--- Setup Commands ---"
	@echo "  manager                            build the manager binary"
	@echo "  docker-build                       build the docker image"
	@echo "      IMG=<quay.io image>                        @param - Required. The quay.io image name."
	@echo "  docker-push                        push the docker image to quay.io"
	@echo "      IMG=<quay.io image>                        @param - Required. The quay.io image name."
	@echo "  deploy                             deploy the latest image you have pushed to your cluster"
	@echo "      IMG=<quay.io image>                        @param - Required. The quay.io image name."
	@echo "  build-deploy                       build and deploy the operator image."
	@echo "      IMG=<quay.io image>                        @param - Required. The quay.io image name."
	@echo "  docker-build-user                  build the docker image"
	@echo "      USER=<quay.io username>                    @param - Required. The quay.io username for building the image."
	@echo "  docker-push-user                   push the docker image to quay.io"
	@echo "      USER=<quay.io username>                    @param - Required. The quay.io username for building the image."
	@echo "  deploy-user                        deploy the latest image you have pushed to your cluster"
	@echo "      USER=<quay.io username>                    @param - Required. The quay.io username for building the image."
	@echo "  build-deploy-user                  build and deploy the operator image."
	@echo "      USER=<quay.io username>                    @param - Required. The quay.io username for building the image."
	@echo "  install                            create and register the CRD"
	@echo "--- General Commands ---"
	@echo "  run                                run the operator locally outside of the cluster"
	@echo "  deploy-cr                          copy and configure the sample CR and deploy it. Will also create auth secret depending on parameters"
	@echo "      AUTH=<basic/token>                         @param - Optional. Must specify basic if you want basic auth. Default is token."
	@echo "      USER=<cloud.rh.com username>               @param - Optional. Must specify USER if you choose basic auth. Default is token."
	@echo "      PASS=<cloud.rh.com username>               @param - Optional. Must specify PASS if you choose basic auth. Default is token."
	@echo "      CI=<true/false>                            @param - Optional. Will replace api_url with CI url. Default is false."
	@echo "  deploy-local-cr                    copy and configure the sample CR to use external prometheus route and deploy it. Will also create auth secret depending on parameters"
	@echo "      AUTH=<basic/token>                         @param - Optional. Must specify basic if you want basic auth. Default is token."
	@echo "      USER=<cloud.rh.com username>               @param - Optional. Must specify USER if you choose basic auth. Default is token."
	@echo "      PASS=<cloud.rh.com username>               @param - Optional. Must specify PASS if you choose basic auth. Default is token."
	@echo "      CI=<true/false>                            @param - Optional. Will replace api_url with CI url. Default is false."
	@echo "--- Testing Commands ---"
	@echo "  test                                run unit tests"
	@echo "  fmt                                 run go fmt"
	@echo "  lint                                run pre-commit"

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Run pre-commit
lint:
	pre-commit run --all-files

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	kubectl apply -f testing/sa.yaml
	GIT_COMMIT=${GIT_COMMIT} go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -


# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config using USER arg
deploy-user: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/${USER}/koku-metrics-operator:v0.0.1
	$(KUSTOMIZE) build config/default | kubectl apply -f -

deploy-branch:
	IMG=${GITBRANCH_IMG} $(MAKE) deploy

# replaces the username and password with your base64 encoded username and password and looks up the token value for you
setup-auth:
	@cp testing/auth-secret-template.yaml testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSB1c2VybmFtZQ==/$(shell printf "$(shell echo $(or $(USER),cloud.redhat.com username))" | base64)/g' testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSBwYXNzd29yZA==/$(shell printf "$(shell echo $(or $(PASS),cloud.redhat.com password))" | base64)/g' testing/authentication_secret.yaml

add-prom-route:
	@sed -i "" '/prometheus_config/d' testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '  prometheus_config:' >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '    service_address: $(EXTERNAL_PROM_ROUTE)'  >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '    skip_tls_verification: true' >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

add-auth:
	@sed -i "" '/authentication/d' testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '  authentication:'  >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '    type: basic'  >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '    secret_name: dev-auth-secret' >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

local-validate-cert:
	@sed -i "" '/upload/d' testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '  upload:'  >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	@echo '    validate_cert: false'  >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

add-ci-route:
	@echo '  api_url: https://ci.cloud.redhat.com'  >> testing/cost-mgmt_v1alpha1_costmanagement.yaml

add-spec:
	@echo 'spec:' >> testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

deploy-cr:
	@cp testing/kokumetricsconfig-template.yaml testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
ifeq ($(AUTH), basic)
	$(MAKE) setup-auth
	$(MAKE) add-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

deploy-local-cr:
	@cp testing/kokumetricsconfig-template.yaml testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml
	$(MAKE) add-prom-route
	$(MAKE) local-validate-cert
ifeq ($(AUTH), basic)
	$(MAKE) setup-auth
	$(MAKE) add-auth
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
ifeq ($(CI), true)
	$(MAKE) add-ci-route
endif
	oc apply -f testing/koku-metrics-cfg_v1alpha1_kokumetricsconfig.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Build the docker image
docker-build-no-test:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# Build, push, and deploy the image
build-deploy: docker-build docker-push deploy

# Build the docker image
docker-build-user: test
	docker build . -t quay.io/${USER}/koku-metrics-operator:v0.0.1

# Push the docker image
docker-push-user:
	docker push quay.io/${USER}/koku-metrics-operator:v0.0.1

# Build, push, and deploy the image
build-deploy-user: docker-build-user docker-push-user deploy-user

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
bundle: manifests kustomize
	mkdir -p koku-metrics-operator/$(VERSION)/
	rm -rf ./bundle koku-metrics-operator/$(VERSION)/
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/project-koku/koku-metrics-operator:v$(VERSION)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
	cp -r ./bundle/ koku-metrics-operator/$(VERSION)/
	cp bundle.Dockerfile koku-metrics-operator/0.9.0/Dockerfile
	scripts/txt_replace.py $(VERSION)

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Push the bundle image.
bundle-push:
	docker push $(BUNDLE_IMG)
