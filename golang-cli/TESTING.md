# Testing Guide

This document describes how to run tests and verify the Go CLI implementation.

## Quick Start

```bash
# Run all tests (optimized)
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run CI pipeline
make ci

# Run comprehensive functional tests (Phase 5)
make test-functional

# Run dual testing framework (Go vs TypeScript parity)
make test-dual

# Run comprehensive parity tests
make test-comprehensive
```

## Performance Optimization

### Build Cache

Go maintains a build cache to speed up subsequent builds and tests. The cache is automatically managed, but you can control it:

```bash
# View cache information
make cache-info

# Clean build cache (if you suspect corruption)
make cache-clean

# Clean all caches (nuclear option)
make cache-clean-all
```

### Test Parallelism

Tests run in parallel by default. The Makefile sets `-parallel=4` to balance speed and resource usage:

```bash
# Run with custom parallelism
go test ./... -parallel=8

# Run sequentially (useful for debugging)
go test ./... -parallel=1
```

### Memory Management

To prevent tests from consuming too much memory:

1. **Limit parallel tests**: The Makefile uses `-parallel=4` by default
2. **Use timeouts**: Tests have appropriate timeouts set
3. **Clean up resources**: Tests clean up temp files in `teardown()`

## Running Tests

### Unit Tests (Fast)

```bash
# Run unit tests (cached for development speed)
make test

# Or directly with Go
go test ./... -count=1 -parallel=4
```

### Integration Tests

```bash
# Run integration tests
make test-integration

# Or directly
go test ./tests/integration -v -count=1 -timeout=5m
```

### E2E Tests

```bash
# Build first, then run E2E tests
make test-e2e

# Or step by step
make build
go test ./tests/e2e -v -count=1 -timeout=10m
```

### Parity Tests

```bash
# Run parity tests
make test-parity

# Or directly
go test ./tests/parity -v -count=1 -timeout=10m
```

### Regression Tests

```bash
# Run regression tests
make test-regression

# Or directly
go test ./tests/regression -v -count=1
```

### All Tests

```bash
# Run complete test suite
make test-all
```

### Specific Test Patterns

```bash
# Run specific test by name
make test-pattern PATTERN=TestNewComponent

# Or directly
go test ./... -v -run TestNewComponent
```

## Test Structure

### Unit Tests

Located in `*_test.go` files alongside source code:

- `internal/config/layer_test.go` - Configuration layer tests
- `internal/storage/storage_test.go` - Storage operations tests
- `internal/tui/app_model_test.go` - TUI component tests
- `internal/tui/welcome_model_test.go` - Welcome screen tests
- `internal/api/client_test.go` - API client tests
- `internal/task/runner_test.go` - Task execution tests
- `internal/mode/detect_test.go` - Mode detection tests

### Integration Tests

Located in `tests/integration/`:

- `grpc_test.go` - gRPC communication tests
- `storage_test.go` - Storage integration tests
- `config_test.go` - Configuration integration tests
- `task_management_test.go` - Task management tests
- `providers_test.go` - Provider integration tests

### E2E Tests

Located in `tests/e2e/`:

- `workflow_test.go` - Full CLI workflow tests
- `comprehensive_commands_test.go` - Command coverage tests
- `cross_platform_test.go` - Cross-platform behavior tests
- `performance_benchmark_test.go` - Performance benchmarks

### Parity Tests

Located in `tests/parity/`:

- `parity_test.go` - TypeScript vs Go feature parity

### Regression Tests

Located in `tests/regression/`:

- `regression_test.go` - Issue regression tests

## Coverage Targets

### Critical Components (>80% Coverage)

| Package | Coverage | Status |
|---------|----------|--------|
| internal/config | 87.1% | ✅ |
| internal/storage | 73.3% | 🔄 |
| internal/state | 80.8% | ✅ |
| internal/security | 85.1% | ✅ |
| internal/exit | 85.8% | ✅ |

### Other Components

| Package | Coverage |
|---------|----------|
| internal/agent | 69.8% |
| internal/audit | 70.9% |
| internal/auth | 64.7% |
| internal/mode | 62.7% |
| internal/api | 46.6% |
| internal/host | 39.4% |
| internal/task | 38.7% |
| internal/tui | 5.6% |

*Note: TUI and API coverage is lower due to UI-heavy code and external API dependencies*

## Coverage Reports

```bash
# Generate coverage report
make coverage-report

# Open HTML report
open coverage.html
```

## Writing Tests

### Unit Test Pattern

```go
func TestNewComponent(t *testing.T) {
    t.Run("creates component with defaults", func(t *testing.T) {
        comp, err := NewComponent(ConfigOptions{})
        require.NoError(t, err)
        assert.NotNil(t, comp)
    })
}
```

### Table-Driven Tests

```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"valid input", "test", "result"},
    {"empty input", "", "default"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := Process(tt.input)
        assert.Equal(t, tt.expected, result)
    })
}
```

### Mocking External Dependencies

```go
type MockStorage struct {
    GetFunc func(key string) (interface{}, bool)
    SetFunc func(key string, value interface{}) error
}

func (m *MockStorage) Get(key string) (interface{}, bool) {
    return m.GetFunc(key)
}
```

### Test Cleanup

Always clean up resources in tests:

```go
func TestWithTempFile(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir) // Clean up after test
    
    // Test code here...
}
```

## Debugging Tests

### Run Specific Test
```bash
make debug-test TEST=TestNewComponent
```

### Run with Verbose Output
```bash
go test -v ./internal/config/...
```

### Run with Race Detector
```bash
make test-race
```

### Profile Tests
```bash
# CPU and memory profiling
make benchmark-profile

# View profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Continuous Integration

Tests run automatically on:
- Every pull request
- Every push to main branch
- Nightly builds

### CI Test Matrix

- Go 1.25+
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)
- Windows (x86_64)

### CI Pipeline

```bash
# Run full CI pipeline locally
make ci
```

This runs:
1. Dependency installation
2. Format checking
3. `go vet` analysis
4. Tests with race detector
5. Coverage verification

## Known Issues & Solutions

### Test Performance

If tests are slow:
1. **Check cache**: Run `make cache-info` to see cache size
2. **Clean if needed**: Run `make cache-clean` if cache is corrupted
3. **Limit parallelism**: Reduce `-parallel` value if system is overloaded
4. **Skip heavy tests**: Use build tags to skip integration tests locally

### Memory Issues

If tests cause OOM:
1. Reduce `-parallel` value in Makefile
2. Add `defer` cleanup to all tests
3. Run tests in smaller batches

### Test Timeouts

If tests timeout:
1. Check for goroutine leaks in your code
2. Ensure all network calls have timeouts
3. Use `context.WithTimeout` for long operations

## Development Workflow

### Watch Mode

For development with automatic rebuilds:

```bash
# Install air (once)
go install github.com/cosmtrek/air@latest

# Start watch mode
make watch
```

### Fast Development Cycle

```bash
# Quick build and test cycle
make dev        # Build with debug info
./cline --help  # Test manually
make test       # Run tests
```

## Best Practices

1. **Always use `-count=1`** in CI to disable test caching
2. **Clean up resources** with `defer` in every test
3. **Use table-driven tests** for multiple test cases
4. **Mock external dependencies** for unit tests
5. **Set appropriate timeouts** for integration tests
6. **Run race detector** before submitting PRs
7. **Maintain >80% coverage** for critical packages