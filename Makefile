.PHONY: staticcheck
staticcheck:
	staticcheck `go list ./... | grep -v "pkg/client"` &&     go vet `go list ./... | grep -v "pkg/client"`
