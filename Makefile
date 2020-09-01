PROJECT_NAME=extendeddaemonset
ARTIFACT=controller
ARTIFACT_PLUGIN=kubectl-eds

# 0.0 shouldn't clobber any released builds
DOCKER_REGISTRY?=
PREFIX?=${DOCKER_REGISTRY}datadog/${PROJECT_NAME}
SOURCEDIR="."

SOURCES:=$(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

BUILDINFOPKG=github.com/datadog/extendeddaemonset/version
VERSION?=$(shell git describe --tags --dirty)
TAG?=${VERSION}
GIT_COMMIT?=$(shell git rev-parse HEAD)
DATE=$(shell date +%Y-%m-%d/%H:%M:%S )
GOMOD="-mod=vendor"
LDFLAGS=-ldflags "-w -X ${BUILDINFOPKG}.Tag=${TAG} -X ${BUILDINFOPKG}.Commit=${GIT_COMMIT} -X ${BUILDINFOPKG}.Version=${VERSION} -X ${BUILDINFOPKG}.BuildTime=${DATE} -s"

export GO111MODULE=on

all: test build

vendor:
	go mod vendor

tidy:
	go mod tidy -v

build: ${ARTIFACT}

${ARTIFACT}: ${SOURCES}
	CGO_ENABLED=0 go build ${GOMOD} -i -installsuffix cgo ${LDFLAGS} -o ${ARTIFACT} ./cmd/manager/main.go

build-plugin: ${ARTIFACT_PLUGIN}

${ARTIFACT_PLUGIN}: ${SOURCES}
	CGO_ENABLED=0 go build -i -installsuffix cgo ${LDFLAGS} -o ${ARTIFACT_PLUGIN} ./cmd/${ARTIFACT_PLUGIN}/main.go

container:
	bin/operator-sdk build $(PREFIX):$(TAG)
    ifeq ($(KINDPUSH), true)
	 kind load docker-image $(PREFIX):$(TAG)
    endif

container-ci:
	docker build -t $(PREFIX):$(TAG) --build-arg "TAG=$(TAG)" .

test:
	./go.test.sh

e2e:
	hack/generate-e2etest-manifest.sh
	bin/operator-sdk test local ./test/e2e --global-manifest ./test/e2e/global-manifest.yaml --go-test-flags '-v' --image $(PREFIX):$(TAG)

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f ${ARTIFACT}
	rm -rf ./bin

validate: bin/golangci-lint bin/wwhrd
	bin/golangci-lint run ./...
	hack/verify-license.sh

generate: bin/operator-sdk bin/openapi-gen bin/client-gen bin/informer-gen bin/lister-gen
	bin/operator-sdk generate k8s
	bin/operator-sdk generate crds --crd-version v1beta1
	hack/patch-crds.sh

	bin/openapi-gen --logtostderr=true -o "" -i ./pkg/apis/datadoghq/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/datadoghq/v1alpha1 -h ./hack/boilerplate.go.txt -r "-"
	hack/generate-groups.sh client,lister,informer \
  github.com/datadog/extendeddaemonset/pkg/generated github.com/datadog/extendeddaemonset/pkg/apis datadoghq:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

generate-olm: bin/operator-sdk
	bin/operator-sdk generate packagemanifests --version $(VERSION:v%=%) --update-crds --interactive=false

pre-release: bin/yq
	hack/pre-release.sh $(VERSION) $(RELEASE_CANDIDATE)

CRDS = $(wildcard deploy/crds/*_crd.yaml)
local-load: $(CRDS)
		for f in $^; do kubectl apply -f $$f; done
		kubectl apply -f deploy/
		kubectl delete pod -l name=${PROJECT_NAME}

$(filter %.yaml,$(files)): %.yaml: %yaml
	kubectl apply -f $@

install-tools: bin/golangci-lint bin/operator-sdk bin/yq

bin/yq:
	hack/install-yq.sh

bin/golangci-lint:
	hack/golangci-lint.sh v1.18.0

bin/operator-sdk:
	hack/install-operator-sdk.sh

bin/wwhrd:
	hack/install-wwhrd.sh

bin/openapi-gen:
	go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

bin/client-gen:
	go build -o ./bin/client-gen ./vendor/k8s.io/code-generator/cmd/client-gen

bin/informer-gen:
	go build -o ./bin/informer-gen ./vendor/k8s.io/code-generator/cmd/informer-gen

bin/lister-gen:
	go build -o ./bin/lister-gen ./vendor/k8s.io/code-generator/cmd/lister-gen

license: bin/wwhrd
	hack/license.sh

.PHONY: vendor build push clean test e2e validate local-load install-tools list container container-ci license generate-olm pre-release
