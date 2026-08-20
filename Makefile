# Makefile - ForgeKit Development Commands
# Provides convenient targets for building, testing, and packaging

.PHONY: help build test vet lint build-linux package release-snapshot clean

# Default target
help:
	@echo "ForgeKit - Available targets:"
	@echo "  make build          - Build forge binary (development)"
	@echo "  make test           - Run all tests"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run gofmt -w ."
	@echo "  make build-linux    - Build static Linux binary (CGO_ENABLED=0)"
	@echo "  make package        - Build release artifacts with GoReleaser (snapshot)"
	@echo "  make release-snapshot - Alias for package"
	@echo "  make clean          - Remove build artifacts"
	@echo ""
	@echo "Release (requires goreleaser):"
	@echo "  make package        - Build .deb, .tar.gz, checksums via GoReleaser (snapshot mode)"

# Development build
build:
	go build -o forge ./cmd/forge

# Run all tests
test:
	go test ./...

# Run go vet
vet:
	go vet ./...

# Format code
lint:
	gofmt -w .

# Build static Linux binary (for testing distribution)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o forge-linux-amd64 ./cmd/forge
	@echo "Built: forge-linux-amd64"
	@file forge-linux-amd64

# Build release artifacts with GoReleaser (snapshot mode)
package:
	goreleaser release --snapshot --clean

release-snapshot: package

# Clean build artifacts
clean:
	rm -f forge forge-linux-amd64
	rm -rf dist/
	rm -f *.deb *.tar.gz checksums.txt

# Install GoReleaser locally (for development)
install-goreleaser:
	go install github.com/goreleaser/goreleaser/v2@latest

# Verify goreleaser config
check-goreleaser:
	goreleaser check

# Full CI pipeline (matches GitHub Actions)
ci: lint vet test build

# Help target
.DEFAULT_GOAL := help