.PHONY: all build test test-coverage test-integration test-parity test-e2e test-regression test-all clean lint help

# Default target
all: build

# Build the CLI binary
build:
	@echo "Building CLI binary..."
	go build -o cline ./cmd/cline

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

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build the CLI binary"
	@echo "  test             - Run unit tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-integration - Run integration tests"
	@echo "  test-parity      - Run parity tests"
	@echo "  test-e2e         - Run E2E tests"
	@echo "  test-regression  - Run regression tests"
	@echo "  test-all         - Run all tests"
	@echo "  benchmark        - Run benchmarks"
	@echo "  clean            - Clean build artifacts"
	@echo "  lint             - Run linting"
	@echo "  race             - Run with race detector"
	@echo "  coverage-report  - Generate coverage report"
	@echo "  verify-coverage  - Verify 80% coverage threshold"
	@echo "  ci-test          - Run CI tests"
	@echo "  deps             - Install dependencies"
	@echo "  update-deps      - Update dependencies"
	@echo "  fmt              - Format code"
	@echo "  help             - Show this help"