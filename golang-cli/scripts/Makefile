# Makefile for Cline CLI cross-platform builds
# Provides convenient targets for building, testing, and releasing

.PHONY: all build build-all clean test lint fmt vet help \
        build-linux-amd64 build-linux-arm64 \
        build-darwin-amd64 build-darwin-arm64 \
        build-windows-amd64 \
        sign checksum archive release \
        install dev run

# Project configuration
BINARY_NAME := cline
PROJECT_ROOT := $(shell cd .. && pwd)
BUILD_DIR := $(PROJECT_ROOT)/build
DIST_DIR := $(PROJECT_ROOT)/dist
SCRIPTS_DIR := $(PROJECT_ROOT)/scripts

# Version information (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')

# Build script
BUILD_SCRIPT := $(SCRIPTS_DIR)/build.sh

# Go commands
GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOFMT := $(GO) fmt
GOVET := $(GO) vet

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Help target
help: ## Display this help message
	@echo "Cline CLI Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'

# Build targets
all: clean build-all checksum archive ## Clean, build all platforms, generate checksums and archives

build: ## Build for current platform only
	@echo "$(BLUE)[INFO]$(NC) Building for current platform..."
	@cd $(PROJECT_ROOT) && $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cline
	@echo "$(GREEN)[SUCCESS]$(NC) Built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for all supported platforms
	@echo "$(BLUE)[INFO]$(NC) Building for all platforms..."
	@VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) $(BUILD_SCRIPT)

# Platform-specific build targets
build-linux-amd64: ## Build for Linux AMD64
	@echo "$(BLUE)[INFO]$(NC) Building for linux/amd64..."
	@$(BUILD_SCRIPT) -p linux/amd64 -v $(VERSION)

build-linux-arm64: ## Build for Linux ARM64
	@echo "$(BLUE)[INFO]$(NC) Building for linux/arm64..."
	@$(BUILD_SCRIPT) -p linux/arm64 -v $(VERSION)

build-darwin-amd64: ## Build for macOS AMD64 (Intel)
	@echo "$(BLUE)[INFO]$(NC) Building for darwin/amd64..."
	@$(BUILD_SCRIPT) -p darwin/amd64 -v $(VERSION)

build-darwin-arm64: ## Build for macOS ARM64 (Apple Silicon)
	@echo "$(BLUE)[INFO]$(NC) Building for darwin/arm64..."
	@$(BUILD_SCRIPT) -p darwin/arm64 -v $(VERSION)

build-windows-amd64: ## Build for Windows AMD64
	@echo "$(BLUE)[INFO]$(NC) Building for windows/amd64..."
	@$(BUILD_SCRIPT) -p windows/amd64 -v $(VERSION)

# Release targets
release: clean build-all checksum archive ## Create a full release build
	@echo "$(GREEN)[SUCCESS]$(NC) Release build complete!"

sign: ## Sign macOS binaries (requires codesign identity)
	@echo "$(BLUE)[INFO]$(NC) Building and signing macOS binaries..."
	@$(BUILD_SCRIPT) -s -p darwin/amd64 -v $(VERSION)
	@$(BUILD_SCRIPT) -s -p darwin/arm64 -v $(VERSION)

checksum: ## Generate checksums for built binaries
	@echo "$(BLUE)[INFO]$(NC) Generating checksums..."
	@$(BUILD_SCRIPT) -c

archive: ## Create release archives
	@echo "$(BLUE)[INFO]$(NC) Creating release archives..."
	@$(BUILD_SCRIPT) -a

# Development targets
install: ## Install the binary to $GOPATH/bin
	@echo "$(BLUE)[INFO]$(NC) Installing $(BINARY_NAME)..."
	@cd $(PROJECT_ROOT) && $(GO) install ./cmd/cline
	@echo "$(GREEN)[SUCCESS]$(NC) Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

dev: ## Build and run for development
	@echo "$(BLUE)[INFO]$(NC) Building for development..."
	@cd $(PROJECT_ROOT) && $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cline
	@echo "$(GREEN)[SUCCESS]$(NC) Built: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "$(BLUE)[INFO]$(NC) Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

run: ## Run the application without building
	@cd $(PROJECT_ROOT) && $(GO) run ./cmd/cline

# Testing targets
test: ## Run all tests
	@echo "$(BLUE)[INFO]$(NC) Running tests..."
	@cd $(PROJECT_ROOT) && $(GOTEST) -v ./...

test-short: ## Run short tests only
	@echo "$(BLUE)[INFO]$(NC) Running short tests..."
	@cd $(PROJECT_ROOT) && $(GOTEST) -short ./...

test-race: ## Run tests with race detector
	@echo "$(BLUE)[INFO]$(NC) Running tests with race detector..."
	@cd $(PROJECT_ROOT) && $(GOTEST) -race ./...

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)[INFO]$(NC) Running tests with coverage..."
	@cd $(PROJECT_ROOT) && $(GOTEST) -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage report: coverage.html"

test-build: ## Run build-specific tests
	@echo "$(BLUE)[INFO]$(NC) Running build tests..."
	@cd $(PROJECT_ROOT)/scripts && $(GOTEST) -v -run TestBuild ./...

# Code quality targets
fmt: ## Format Go code
	@echo "$(BLUE)[INFO]$(NC) Formatting code..."
	@cd $(PROJECT_ROOT) && $(GOFMT) ./...

vet: ## Run go vet
	@echo "$(BLUE)[INFO]$(NC) Running go vet..."
	@cd $(PROJECT_ROOT) && $(GOVET) ./...

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)[INFO]$(NC) Running linter..."
	@which golangci-lint > /dev/null 2>&1 || (echo "$(RED)[ERROR]$(NC) golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@cd $(PROJECT_ROOT) && golangci-lint run ./...

check: fmt vet ## Run fmt and vet

# Cleanup targets
clean: ## Clean build artifacts
	@echo "$(BLUE)[INFO]$(NC) Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(PROJECT_ROOT)/coverage.out $(PROJECT_ROOT)/coverage.html
	@echo "$(GREEN)[SUCCESS]$(NC) Build artifacts cleaned"

clean-all: clean ## Clean all artifacts including Go cache
	@echo "$(BLUE)[INFO]$(NC) Cleaning Go cache..."
	@cd $(PROJECT_ROOT) && $(GO) clean -cache -testcache
	@echo "$(GREEN)[SUCCESS]$(NC) All artifacts cleaned"

# Utility targets
version: ## Display version information
	@echo "Cline CLI Build System"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Directory: $(BUILD_DIR)"
	@echo "Dist Directory: $(DIST_DIR)"

info: ## Display build environment information
	@echo "$(BLUE)Build Environment$(NC)"
	@echo "=================="
	@echo "Go Version: $(shell $(GO) version)"
	@echo "Go Path: $(shell $(GO) env GOPATH)"
	@echo "Project Root: $(PROJECT_ROOT)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "$(BLUE)Supported Platforms$(NC)"
	@echo "===================="
	@echo "  - linux/amd64"
	@echo "  - linux/arm64"
	@echo "  - darwin/amd64"
	@echo "  - darwin/arm64"
	@echo "  - windows/amd64"
	@echo ""
	@echo "$(BLUE)Build Tools$(NC)"
	@echo "============"
	@echo "codesign: $(shell which codesign 2>/dev/null || echo 'not found')"
	@echo "shasum: $(shell which shasum 2>/dev/null || echo 'not found')"
	@echo "sha256sum: $(shell which sha256sum 2>/dev/null || echo 'not found')"
	@echo "zip: $(shell which zip 2>/dev/null || echo 'not found')"
	@echo "tar: $(shell which tar 2>/dev/null || echo 'not found')"

# CI/CD targets
ci-build: clean build-all checksum ## CI build target
	@echo "$(GREEN)[SUCCESS]$(NC) CI build complete"

ci-test: test test-build ## CI test target
	@echo "$(GREEN)[SUCCESS]$(NC) CI tests complete"

ci-release: clean release ## CI release target
	@echo "$(GREEN)[SUCCESS]$(NC) CI release complete"

# Docker targets (if Docker is available)
docker-build: ## Build using Docker
	@echo "$(BLUE)[INFO]$(NC) Building with Docker..."
	@docker build -t cline-cli-builder -f $(PROJECT_ROOT)/Dockerfile $(PROJECT_ROOT)
	@docker run --rm -v $(BUILD_DIR):/output cline-cli-builder

# Verify build artifacts
verify: ## Verify built binaries
	@echo "$(BLUE)[INFO]$(NC) Verifying build artifacts..."
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$binary" ]; then \
			echo "Verifying: $$binary"; \
			file "$$binary" || true; \
			if [[ "$$binary" == *darwin* ]]; then \
				codesign --verify --verbose "$$binary" 2>/dev/null && echo "  Signature: valid" || echo "  Signature: not signed or invalid"; \
			fi; \
		fi; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Verification complete"