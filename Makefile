
KIND_CLUSTER_CONFIG := ./deploy/kind_cluster.yaml
CODEGEN := ./hack/update-codegen.sh
VERIFY := ./hack/verify-codegen.sh

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

.PHONY: verify
verify:
	$(VERIFY)

.PHONY: unit-test
unit-test:
	go clean -testcache
	go list ./... | grep -v e2e | xargs go test -p 8 -v -race

.PHONY: run-local
run-local:
	./hack/install_local.sh

.PHONY: clean-e2e
clean-e2e:
	kubectl delete crds --all
