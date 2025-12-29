

.PHONY: test
test:
	cd radix-cluster-cleanup && go test -cover `go list ./...`
	
.PHONY: lint
lint: bootstrap
	cd radix-cluster-cleanup && golangci-lint run


HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)

bootstrap:
ifndef HAS_GOLANGCI_LINT
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
endif
