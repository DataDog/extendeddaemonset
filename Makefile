PROJECT_NAME=extendeddaemonset
ARTIFACT=controller
ARTIFACT_PLUGIN=kubectl-eds

# 0.0 shouldn't clobber any released builds
DOCKER_REGISTRY?=
PREFIX?=${DOCKER_REGISTRY}datadog/${PROJECT_NAME}
SOURCEDIR="."

SOURCES:=$(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

BUILDINFOPKG=github.com/datadog/extendeddaemonset/version
GIT_TAG?=$(shell git tag -l --contains HEAD | tail -1)
TAG?=${GIT_TAG}
TAG_HASH=$(shell git tag | tail -1)_$(shell git rev-parse --short HEAD)
VERSION?=$(if $(GIT_TAG),$(GIT_TAG),$(TAG_HASH))
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
	./bin/operator-sdk build $(PREFIX):$(TAG)
    ifeq ($(KINDPUSH), true)
	 kind load docker-image $(PREFIX):$(TAG)
    endif

container-ci:
	docker build -t $(PREFIX):$(TAG) --build-arg "TAG=$(TAG)" .

test:
	./go.test.sh

e2e:
	operator-sdk test local --verbose ./test/e2e --image $(PREFIX):$(TAG)

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f ${ARTIFACT}

validate:
	./bin/golangci-lint run ./...
	./hack/verify-license.sh

generate: bin/operator-sdk bin/openapi-gen
	./bin/operator-sdk generate k8s
	./bin/operator-sdk generate crds
	./bin/openapi-gen --logtostderr=true -o "" -i ./pkg/apis/datadoghq/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/datadoghq/v1alpha1 -h ./LICENSE -r "-"


CRDS = $(wildcard deploy/crds/*_crd.yaml)
local-load: $(CRDS)
		for f in $^; do kubectl apply -f $$f; done
		kubectl apply -f deploy/
		kubectl delete pod -l name=${PROJECT_NAME}

$(filter %.yaml,$(files)): %.yaml: %yaml
	kubectl apply -f $@

install-tools: bin/golangci-lint bin/operator-sdk bin/openapi-gen

bin/golangci-lint:
	./hack/golangci-lint.sh v1.18.0

bin/operator-sdk:
	./hack/install-operator-sdk.sh

bin/wwhrd:
	./hack/install-wwhrd.sh

bin/openapi-gen:
	go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

license: bin/wwhrd
	./hack/license.sh

.PHONY: vendor build push clean test e2e validate local-load install-tools list container container-ci license
