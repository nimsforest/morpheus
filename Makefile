.PHONY: build install clean test run help hooks check

BINARY_NAME=morpheus
BUILD_DIR=bin
GO=go
GOFLAGS=-v

# Get version from git tags, fallback to "dev" if no tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

help: ## Show this help message
	@echo "Morpheus - Nims Forest Provisioning Tool"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the morpheus binary
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/morpheus

install: build ## Install morpheus to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✓ Installed successfully!"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -cover ./...

test-coverage: ## Generate HTML coverage report
	@echo "Generating coverage report..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

run: build ## Build and run morpheus
	@$(BUILD_DIR)/$(BINARY_NAME)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

lint: fmt vet ## Run linters

hooks: ## Install git pre-commit hooks
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@cp scripts/hooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Pre-commit hook installed"

check: fmt vet ## Run all pre-commit checks (same as hooks)
	@echo "Running pre-commit checks..."
	@$(GO) build ./...
	@echo "✓ Build OK"
	@cp go.mod go.mod.backup
	@cp go.sum go.sum.backup 2>/dev/null || true
	@$(GO) mod tidy
	@if ! diff -q go.mod go.mod.backup > /dev/null 2>&1; then \
		echo "✗ go.mod is not tidy. Run 'go mod tidy'"; \
		mv go.mod.backup go.mod; \
		mv go.sum.backup go.sum 2>/dev/null || true; \
		exit 1; \
	fi
	@mv go.mod.backup go.mod
	@mv go.sum.backup go.sum 2>/dev/null || true
	@echo "✓ go.mod is tidy"
	@echo "✓ All checks passed!"

.DEFAULT_GOAL := help
