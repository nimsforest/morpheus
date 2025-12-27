.PHONY: build install clean test run help

BINARY_NAME=morpheus
BUILD_DIR=bin
GO=go
GOFLAGS=-v

help: ## Show this help message
	@echo "Morpheus - Nims Forest Provisioning Tool"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the morpheus binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/morpheus

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

.DEFAULT_GOAL := help
