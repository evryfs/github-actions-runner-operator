# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

TAG := $(shell git describe --tags --always)
TAG_WITHOUT_PREFIX := $(shell echo $(TAG) | sed s/^v//)
IMG ?= quay.io/evryfs/github-actions-runner-operator:$(TAG)
GHCR_IMG ?= ghcr.io/evryfs/github-actions-runner-operator:${TAG}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,generateEmbeddedObjectMeta=true"

# Default bundle image tag
BUNDLE_IMG ?= quay.io/evryfs/github-actions-runner-operator-bundle:$(TAG)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
KUBEBUILDER_ASSETS=/tmp/envtest_assets.d
CONTROLLER_RUNTIME_VERSION=v0.8.3
K8S_VERSION=1.22.0
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
test: generate fmt vet manifests
	mkdir -p ${KUBEBUILDER_ASSETS}
	curl -sSL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-${GOOS}-${GOARCH}.tar.gz" | tar xvz -C ${KUBEBUILDER_ASSETS} --strip-components=2
	KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS} go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

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
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker pull ${GHCR_IMG}
	docker tag ${GHCR_IMG} ${IMG}
	docker push ${IMG}
	docker push $(BUNDLE_IMG)

docker-push-ghcr:
	docker tag ${IMG} ${GHCR_IMG}
	docker push ${GHCR_IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2 ;\
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
	go get sigs.k8s.io/kustomize/kustomize/v4@v4.4.0 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version $(TAG_WITHOUT_PREFIX) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

load-kind-image:
	docker pull ${GHCR_IMG}
	docker tag ${GHCR_IMG} ${IMG}
	kind load docker-image ${IMG} --name chart-testing
