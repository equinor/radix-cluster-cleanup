
.PHONY: lint
lint: bootstrap
	cd radix-cluster-cleanup && golangci-lint run


HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)

bootstrap:
ifndef HAS_GOLANGCI_LINT
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
endif
