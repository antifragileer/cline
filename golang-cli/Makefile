.PHONY: all build build-all build-signed clean test test-coverage test-integration test-parity test-e2e test-regression test-all lint fmt deps update-deps verify-coverage ci-test benchmark release release-homebrew release-npm release-scoop release-linux help dev watch

# Default target
all: build

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := go vet

# Build flags for optimization
# -s: Omit symbol table and debug info
# -w: Omit DWARF symbol table
# -trimpath: Remove all file system paths from the resulting executable
LDFLAGS := -s -w \
    -X github.com/cline/cline/golang-cli/cmd/cline.Version=$(VERSION) \
    -X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=$(GIT_COMMIT) \
    -X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=$(BUILD_TIME)

# Test flags for performance
# -count=1: Disable test caching for CI, enable for local dev
# -p: Number of parallel tests (defaults to GOMAXPROCS)
TEST_FLAGS := -count=1 -parallel=4
RACE_FLAGS := -race

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build directories
BUILD_DIR := build
DIST_DIR := dist
TMP_DIR := tmp

# Go build cache settings
GOCACHE ?= $(shell $(GOCMD) env GOCACHE)
GOMODCACHE ?= $(shell $(GOCMD) env GOMODCACHE)

# =============================================================================
# Development Targets
# =============================================================================

# Build the CLI binary for current platform (optimized)
build:
	@echo "Building CLI binary..."
	$(GOBUILD) \
		-ldflags "$(LDFLAGS)" \
		-trimpath \
		-o cline \
		./cmd/cline

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	./scripts/build.sh --clean --all

# Build signed binaries (requires codesign certificate)
build-signed:
	@echo "Building signed binaries..."
	./scripts/build.sh --clean --all --sign

# Development build (faster, with debug info)
dev:
	@echo "Building development binary..."
	$(GOBUILD) \
		-gcflags="all=-N -l" \
		-o cline \
		./cmd/cline

# Watch mode for development (requires air: go install github.com/cosmtrek/air@latest)
watch:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest && air; \
	fi

# =============================================================================
# Testing Targets (Optimized)
# =============================================================================

# Run unit tests with caching enabled (fast for development)
test:
	@echo "Running unit tests..."
	@./scripts/test.sh --unit

# Run tests with coverage (optimized)
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) ./... -count=1 -parallel=4 -coverprofile=coverage.out -covermode=atomic
	@echo "Coverage report generated: coverage.out"
	$(GOCMD) tool cover -func=coverage.out | grep total
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "HTML report: coverage.html"

# Run tests with race detection (slower but thorough)
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) ./... $(RACE_FLAGS) -count=1

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@./scripts/test.sh --integration

# Run parity tests
test-parity:
	@echo "Running parity tests..."
	$(GOTEST) -v ./tests/parity -count=1 -timeout=10m -parallel=2

# Run E2E tests (requires built binary)
test-e2e: build
	@echo "Running E2E tests..."
	@./scripts/test.sh --e2e

# Run regression tests
test-regression:
	@echo "Running regression tests..."
	$(GOTEST) -v ./tests/regression -count=1 -timeout=5m

# Run all tests (comprehensive)
test-all:
	@echo "Running all tests..."
	@./scripts/test.sh --all

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) ./... -bench=. -benchmem -count=3 -run=^#

# Run benchmarks with profiling
benchmark-profile:
	@echo "Running benchmarks with profiling..."
	$(GOTEST) ./... -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof -run=^#
	@echo "Profiles generated: cpu.prof, mem.prof"

# Run specific test pattern
test-pattern:
	@echo "Running tests matching pattern: $(PATTERN)"
	$(GOTEST) ./... -v -run $(PATTERN)

# Debug specific test
debug-test:
	@echo "Debugging test: $(TEST)"
	$(GOTEST) ./... -v -run $(TEST)

# =============================================================================
# Quality & Linting
# =============================================================================

# Run linting
lint:
	@echo "Running linter..."
	$(GOFMT) -w .
	$(GOVET) ./...

# Format code only
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

# Check formatting (CI)
fmt-check:
	@echo "Checking formatting..."
	@unformatted=$$($(GOFMT) -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run static analysis
static-analysis:
	@echo "Running static analysis..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

# =============================================================================
# Coverage Targets
# =============================================================================

# Generate coverage report in multiple formats
coverage-report:
	@echo "Generating coverage report..."
	$(GOTEST) ./... -count=1 -coverprofile=coverage.out -covermode=atomic
	$(GOCMD) tool cover -func=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage Summary:"
	@$(GOCMD) tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Detailed report: coverage.html"

# Verify coverage threshold (80%)
verify-coverage:
	@echo "Verifying coverage threshold..."
	$(GOTEST) ./... -count=1 -coverprofile=coverage.out -covermode=atomic
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below threshold of 80%"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets threshold of 80%"; \
	fi

# =============================================================================
# CI/CD Targets
# =============================================================================

# Run tests in CI mode (for CI/CD)
ci-test:
	@echo "Running CI tests..."
	$(GOTEST) ./... -v -race -count=1 -coverprofile=coverage.out -covermode=atomic
	$(GOCMD) tool cover -func=coverage.out | grep total

# Full CI pipeline
ci: deps fmt-check vet ci-test verify-coverage
	@echo "CI pipeline completed successfully!"

# =============================================================================
# Build Cache Management
# =============================================================================

# Show cache info
cache-info:
	@echo "Go build cache: $(GOCACHE)"
	@echo "Go module cache: $(GOMODCACHE)"
	@du -sh $(GOCACHE) 2>/dev/null || echo "Cache directory not accessible"
	@du -sh $(GOMODCACHE) 2>/dev/null || echo "Module cache directory not accessible"

# Clean build cache
cache-clean:
	@echo "Cleaning Go build cache..."
	$(GOCMD) clean -cache

# Clean module cache (aggressive)
cache-clean-mod:
	@echo "Cleaning Go module cache..."
	$(GOCMD) clean -modcache

# Clean all caches
cache-clean-all:
	@echo "Cleaning all Go caches..."
	$(GOCMD) clean -cache -modcache -testcache

# =============================================================================
# Maintenance
# =============================================================================

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f cline cline-go
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(TMP_DIR)
	rm -f coverage.out coverage.html cpu.prof mem.prof
	$(GOCLEAN)

# Deep clean (includes caches)
clean-all: clean cache-clean-all

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Verify dependencies
verify-deps:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd .. && npm run protos

# =============================================================================
# Release Targets
# =============================================================================

# Create release archives
release: clean
	@echo "Creating release..."
	mkdir -p $(DIST_DIR)
	./scripts/build.sh --clean --all --checksum --archive
	@echo "Release artifacts created in $(DIST_DIR)/"

# Generate Homebrew formula
release-homebrew: release
	@echo "Generating Homebrew formula..."
	$(GOCMD) run ./cmd/generate-homebrew/main.go -version $(VERSION) -local -binary-dir $(DIST_DIR) -output $(DIST_DIR)/cline.rb

# Generate NPM package structure
release-npm: release
	@echo "Creating NPM package structure..."
	mkdir -p $(DIST_DIR)/npm
	cp npm-wrapper/package.json npm-wrapper/index.js npm-wrapper/platform.js npm-wrapper/install.js $(DIST_DIR)/npm/
	mkdir -p $(DIST_DIR)/npm/bin
	cp $(DIST_DIR)/cline-$(VERSION)-darwin-amd64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-darwin-arm64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-linux-amd64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-linux-arm64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-windows-amd64.zip $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	@echo "NPM package structure created in $(DIST_DIR)/npm/"
	@echo "To publish: cd $(DIST_DIR)/npm && npm publish --access public"

# Generate Scoop manifest
release-scoop: release
	@echo "Generating Scoop manifest..."
	$(GOCMD) run ./cmd/scoop-generate/main.go -version $(VERSION) -local -binary-dir $(DIST_DIR) -output $(DIST_DIR)/cline.json

# Generate Linux packages
release-linux: release
	@echo "Generating Linux packages..."
	@echo "Linux packages would be generated here (DEB/RPM)"
	@echo "Skipping Linux package generation - requires additional tooling"

# Create all distribution packages
dist: release release-homebrew release-npm release-scoop release-linux
	@echo ""
	@echo "=========================================="
	@echo "All distribution packages created!"
	@echo "=========================================="
	@echo ""
	@echo "Artifacts in $(DIST_DIR)/:"
	@ls -la $(DIST_DIR)/ 2>/dev/null || echo "  (directory listing not available)"
	@echo ""
	@echo "Distribution methods:"
	@echo "  - GitHub Releases: Upload $(DIST_DIR)/cline-$(VERSION)-*.tar.gz"
	@echo "  - Homebrew:        Upload $(DIST_DIR)/cline.rb to homebrew-tap"
	@echo "  - NPM:             cd $(DIST_DIR)/npm && npm publish"
	@echo "  - Scoop:           Upload $(DIST_DIR)/cline.json to scoop-bucket"
	@echo "  - Linux packages:  Upload $(DIST_DIR)/*.deb and *.rpm to package repo"
	@echo ""

# =============================================================================
# Installation
# =============================================================================

# Install locally (development)
install: build
	@echo "Installing cline to $(GOPATH)/bin or $(HOME)/go/bin..."
	$(GOCMD) install -ldflags "$(LDFLAGS)" ./cmd/cline

# Uninstall local installation
uninstall:
	@echo "Uninstalling cline..."
	rm -f $(GOPATH)/bin/cline $(HOME)/go/bin/cline

# =============================================================================
# Help
# =============================================================================

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build the CLI binary for current platform (optimized)"
	@echo "  make dev                - Build development binary (with debug info)"
	@echo "  make watch              - Watch mode for development (requires air)"
	@echo "  make build-all          - Build for all platforms"
	@echo "  make build-signed       - Build signed binaries (requires codesign)"
	@echo "  make install            - Install binary locally (go install)"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run unit tests"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make test-parity        - Run parity tests"
	@echo "  make test-e2e           - Run E2E tests"
	@echo "  make test-regression    - Run regression tests"
	@echo "  make test-all           - Run all tests"
	@echo "  make benchmark          - Run benchmarks"
	@echo "  make benchmark-profile  - Run benchmarks with CPU/memory profiling"
	@echo ""
	@echo "Quality:"
	@echo "  make lint               - Run linting"
	@echo "  make fmt                - Format code"
	@echo "  make fmt-check          - Check formatting (CI)"
	@echo "  make vet                - Run go vet"
	@echo "  make static-analysis    - Run staticcheck"
	@echo "  make coverage-report    - Generate coverage report"
	@echo "  make verify-coverage    - Verify 80% coverage threshold"
	@echo "  make ci-test            - Run CI tests"
	@echo "  make ci                 - Full CI pipeline"
	@echo ""
	@echo "Cache Management:"
	@echo "  make cache-info         - Show cache information"
	@echo "  make cache-clean        - Clean build cache"
	@echo "  make cache-clean-mod    - Clean module cache"
	@echo "  make cache-clean-all    - Clean all caches"
	@echo ""
	@echo "Release:"
	@echo "  make release            - Create release archives for all platforms"
	@echo "  make release-homebrew   - Generate Homebrew formula"
	@echo "  make release-npm        - Create NPM package structure"
	@echo "  make release-scoop      - Generate Scoop manifest"
	@echo "  make release-linux      - Generate DEB/RPM packages"
	@echo "  make dist               - Create all distribution packages"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make clean-all          - Deep clean (includes caches)"
	@echo "  make deps               - Install dependencies"
	@echo "  make update-deps        - Update dependencies"
	@echo "  make verify-deps        - Verify dependencies"
	@echo "  make proto              - Generate protobuf code"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GIT_COMMIT=$(GIT_COMMIT)"
	@echo "  BUILD_TIME=$(BUILD_TIME)"