# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Use git branch for dev team deployment of pushed branches
GITBRANCH=$(shell git branch --show-current)
GITBRANCH_IMG="quay.io/project-koku/korekuta-operator-go:${GITBRANCH}"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# save the git commit
commit:
	git rev-parse HEAD > commit

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests commit
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	kubectl apply -f config/samples/trusted_ca_certmap.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	cat config/openshift-config/role.yaml | kubectl apply -f -
	cat config/openshift-config/role_binding.yaml | kubectl apply -f -

deploy-branch:
	IMG=${GITBRANCH_IMG} $(MAKE) deploy

# replaces the username and password with your base64 encoded username and password and looks up the token value for you
setup-auth:
	@cp config/samples/authentication_secret.yaml testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSB1c2VybmFtZQ==/$(shell printf "$(shell echo $(or $(USER),cloud.redhat.com username))" | base64)/g' testing/authentication_secret.yaml
	@sed -i "" 's/Y2xvdWQucmVkaGF0LmNvbSBwYXNzd29yZA==/$(shell printf "$(shell echo $(or $(PASS),cloud.redhat.com password))" | base64)/g' testing/authentication_secret.yaml

deploy-cr:
	@cp config/samples/cost-mgmt_v1alpha1_costmanagement.yaml testing/cost-mgmt_v1alpha1_costmanagement.yaml
ifeq ($(AUTH), basic)
	$(MAKE) setup-auth
	@echo 'spec:\n  authentication:\n    type: basic\n    secret_name: dev-auth-secret' >> testing/cost-mgmt_v1alpha1_costmanagement.yaml
	oc apply -f testing/authentication_secret.yaml
else
	@echo "Using default token auth"
endif
	oc apply -f testing/cost-mgmt_v1alpha1_costmanagement.yaml


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

# Push the docker image
docker-push:
	docker push ${IMG}

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
bundle: manifests
	operator-sdk generate kustomize manifests -q
	kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
