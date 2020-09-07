
KIND_CLUSTER_CONFIG := ./deploy/kind_cluster.yaml
CODEGEN := ./hack/update-codegen.sh
VERIFY := ./hack/verify-codegen.sh

IMAGE_REPO := pintaqed/pinta-scheduler
IMAGE_TAG ?= "dev"

.PHONY: vendor
vendor:
	go mod tidy && go mod vendor

# Generate deepcopy, conversion, clients, listers, and informers
codegen:
	# Clients, listers, and informers
	$(CODEGEN)

# Generates everything.
.PHONY: gen-all
gen-all: codegen vendor

.PHONY: build
build:
	go build -o ./bin/pinta-controller -mod=vendor ./cmd/controller/
	go build -o ./bin/pinta-scheduler -mod=vendor ./cmd/scheduler/

.PHONY: verify
verify:
	$(VERIFY)

.PHONY: unit-test
unit-test:
	go clean -testcache
	go list ./... | grep -v e2e | xargs go test -p 8 -v -race

.PHONY: monitoring
monitoring:
	./hack/install_monitoring.sh

.PHONY: expose-prometheus
expose-prometheus:
	# Using helm chart
	$(eval POD_NAME:=$(shell kubectl get pods --namespace monitoring -l "app=prometheus,component=server" -o jsonpath="{.items[0].metadata.name}"))
	kubectl --namespace monitoring port-forward $(POD_NAME) 9090
 	# kubectl --namespace monitoring port-forward svc/prometheus-k8s 9090

.PHONY: run-local
run-local: build
	./hack/install_local.sh

.PHONY: clean-all
clean-all:
	kubectl delete --ignore-not-found=true -f https://raw.githubusercontent.com/volcano-sh/volcano/v1.0.1/installer/volcano-development.yaml
	helm uninstall -n monitoring prometheus
	kubectl delete --ignore-not-found=true -f ./deploy/dev --all

# This is a automated proeccess on docker hub at pintaqed/pinta-scheduler.
container:
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) . --squash
	docker push $(IMAGE_REPO):$(IMAGE_TAG)
