.PHONY: all build build-all build-signed clean test test-coverage test-integration test-parity test-e2e test-regression test-all lint fmt deps update-deps verify-coverage ci-test benchmark release release-homebrew release-npm release-scoop release-linux help

# Default target
all: build

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build directories
BUILD_DIR := build
DIST_DIR := dist

# Build the CLI binary for current platform
build:
	@echo "Building CLI binary..."
	go build -ldflags "-s -w -X github.com/cline/cline/golang-cli/cmd/cline.Version=$(VERSION) -X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=$(GIT_COMMIT) -X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=$(BUILD_TIME)" -o cline ./cmd/cline

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	./scripts/build.sh --clean --all

# Build signed binaries (requires codesign certificate)
build-signed:
	@echo "Building signed binaries..."
	./scripts/build.sh --clean --all --sign

# Create release archives
release: clean
	@echo "Creating release..."
	mkdir -p $(DIST_DIR)
	./scripts/build.sh --clean --all --checksum --archive
	@echo "Release artifacts created in $(DIST_DIR)/"

# Run unit tests
test:
	@echo "Running unit tests..."
	go test ./tests -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	@echo "Coverage report generated: coverage.out"
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html
	@echo "HTML report: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration -v

# Run parity tests
test-parity:
	@echo "Running parity tests..."
	go test ./tests/parity -v

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	go test ./tests/e2e -v

# Run regression tests
test-regression:
	@echo "Running regression tests..."
	go test ./tests/regression -v

# Run all tests
test-all: test test-integration test-parity test-e2e test-regression
	@echo "All tests completed!"

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test ./... -bench=. -benchmem

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f cline
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html
	go clean

# Run linting
lint:
	@echo "Running linter..."
	go fmt ./...
	go vet ./...

# Race detection
race:
	@echo "Running with race detector..."
	go test ./... -race

# Generate coverage report in multiple formats
coverage-report:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Detailed report: coverage.html"

# Verify coverage threshold (80%)
verify-coverage:
	@echo "Verifying coverage threshold..."
	go test ./... -coverprofile=coverage.out
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below threshold of 80%"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets threshold of 80%"; \
	fi

# Run tests in CI mode (for CI/CD)
ci-test:
	@echo "Running CI tests..."
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -func=coverage.out | grep total

# Run specific test pattern
test-pattern:
	@echo "Running tests matching pattern: $(PATTERN)"
	go test ./... -v -run $(PATTERN)

# Debug specific test
debug-test:
	@echo "Debugging test: $(TEST)"
	go test ./... -v -run $(TEST)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd .. && npm run protos

# Generate Homebrew formula
release-homebrew: release
	@echo "Generating Homebrew formula..."
	go run scripts/generate_homebrew.go -version $(VERSION) -local -binary-dir $(DIST_DIR) -output $(DIST_DIR)/cline.rb

# Generate NPM package structure
release-npm: release
	@echo "Creating NPM package structure..."
	mkdir -p $(DIST_DIR)/npm
	cp scripts/package.json scripts/install.js scripts/index.js $(DIST_DIR)/npm/
	mkdir -p $(DIST_DIR)/npm/bin
	# Copy binaries for each platform to npm package
	cp $(DIST_DIR)/cline-$(VERSION)-darwin-amd64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-darwin-arm64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-linux-amd64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-linux-arm64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	cp $(DIST_DIR)/cline-$(VERSION)-windows-amd64.tar.gz $(DIST_DIR)/npm/bin/ 2>/dev/null || true
	@echo "NPM package structure created in $(DIST_DIR)/npm/"
	@echo "To publish: cd $(DIST_DIR)/npm && npm publish --access public"

# Generate Scoop manifest
release-scoop: release
	@echo "Generating Scoop manifest..."
	go run scripts/generate_scoop.go -version $(VERSION) -local -binary-dir $(DIST_DIR) -output $(DIST_DIR)/cline.json

# Generate Linux packages
release-linux: release
	@echo "Generating Linux packages..."
	go run scripts/generate_linux_packages.go -version $(VERSION) -binary-dir $(BUILD_DIR) -output-dir $(DIST_DIR)

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

# Install locally (development)
install: build
	@echo "Installing cline to $(GOPATH)/bin or $(HOME)/go/bin..."
	go install -ldflags "-s -w -X github.com/cline/cline/golang-cli/cmd/cline.Version=$(VERSION) -X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=$(GIT_COMMIT) -X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=$(BUILD_TIME)" ./cmd/cline

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build the CLI binary for current platform"
	@echo "  make build-all          - Build for all platforms"
	@echo "  make build-signed       - Build signed binaries (requires codesign)"
	@echo "  make install            - Install binary locally (go install)"
	@echo ""
	@echo "Release:"
	@echo "  make release            - Create release archives for all platforms"
	@echo "  make release-homebrew   - Generate Homebrew formula"
	@echo "  make release-npm        - Create NPM package structure"
	@echo "  make release-scoop      - Generate Scoop manifest"
	@echo "  make release-linux      - Generate DEB/RPM packages"
	@echo "  make dist               - Create all distribution packages"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run unit tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make test-parity        - Run parity tests"
	@echo "  make test-e2e           - Run E2E tests"
	@echo "  make test-regression    - Run regression tests"
	@echo "  make test-all           - Run all tests"
	@echo "  make benchmark          - Run benchmarks"
	@echo ""
	@echo "Quality:"
	@echo "  make lint               - Run linting"
	@echo "  make fmt                - Format code"
	@echo "  make race               - Run with race detector"
	@echo "  make coverage-report    - Generate coverage report"
	@echo "  make verify-coverage    - Verify 80% coverage threshold"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make deps               - Install dependencies"
	@echo "  make update-deps        - Update dependencies"
	@echo "  make proto              - Generate protobuf code"
	@echo "  make ci-test            - Run CI tests"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GIT_COMMIT=$(GIT_COMMIT)"
	@echo "  BUILD_TIME=$(BUILD_TIME)"