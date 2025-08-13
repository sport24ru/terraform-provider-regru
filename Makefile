.PHONY: help build test clean release install

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build the provider binary"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  release   - Build release binaries"
	@echo "  install   - Install provider to local Terraform plugins directory"

# Build the provider
build:
	go build -o terraform-provider-regru

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f terraform-provider-regru
	rm -rf dist/

# Build release binaries
release:
	goreleaser build --snapshot --clean

# Install provider locally
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/sport24ru/regru/1.0.0/darwin_arm64/
	cp terraform-provider-regru ~/.terraform.d/plugins/registry.terraform.io/sport24ru/regru/1.0.0/darwin_arm64/

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Update dependencies
deps:
	go mod tidy
	go mod download 