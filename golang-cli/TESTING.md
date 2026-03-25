# Go CLI Testing Guide

This document describes the testing infrastructure for the Cline Go CLI implementation.

## Overview

The testing infrastructure consists of multiple test suites:

1. **Unit Tests** - Individual function and component tests
2. **Integration Tests** - gRPC, storage, and API provider tests
3. **Parity Tests** - Side-by-side comparison with TypeScript CLI
4. **E2E Tests** - End-to-end workflow tests
5. **Regression Tests** - Tests for known issues to prevent recurrence
6. **Independence Tests** - Tests to verify TypeScript independence

## Running Tests

### Run All Tests

```bash
go test ./tests/... -v
```

### Run Specific Test Suites

```bash
# Unit tests
go test ./tests -v

# Integration tests
go test ./tests/integration -v

# Parity tests
go test ./tests/parity -v

# E2E tests
go test ./tests/e2e -v

# Regression tests
go test ./tests/regression -v
```

### Run with Coverage

```bash
# Run all tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Check coverage threshold (80%)
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total
```

## Test Structure

### Test Organization

```
golang-cli/
├── tests/
│   ├── coverage_test.go          # Coverage analysis framework
│   ├── integration/
│   │   ├── grpc_test.go          # gRPC service tests
│   │   ├── storage_test.go       # Storage implementation tests
│   │   └── providers_test.go     # API provider tests
│   ├── parity/
│   │   └── parity_test.go        # CLI parity tests
│   ├── e2e/
│   │   └── workflow_test.go      # End-to-end workflow tests
│   └── regression/
│       └── regression_test.go    # Regression tests
```

## Coverage Requirements

### Target Coverage: >80%

All packages must achieve at least 80% test coverage. The coverage analyzer in `tests/coverage_test.go` enforces this requirement.

### Coverage Categories

1. **Unit Test Coverage** - Core logic and utilities
2. **Integration Coverage** - gRPC, storage, provider interactions
3. **E2E Coverage** - Complete user workflows

## Writing Tests

### Unit Test Example

```go
func TestNewCoverageAnalyzer(t *testing.T) {
    t.Run("creates analyzer with correct settings", func(t *testing.T) {
        analyzer := NewCoverageAnalyzer("/project", 85.0)

        if analyzer.ProjectRoot != "/project" {
            t.Errorf("ProjectRoot = %s, want /project", analyzer.ProjectRoot)
        }
        if analyzer.Threshold != 85.0 {
            t.Errorf("Threshold = %f, want 85.0", analyzer.Threshold)
        }
    })
}
```

### Integration Test Example

```go
func TestFileStorage(t *testing.T) {
    t.Run("sets and gets values", func(t *testing.T) {
        tempDir, err := os.MkdirTemp("", "storage-test-*")
        require.NoError(t, err)
        defer os.RemoveAll(tempDir)

        filePath := filepath.Join(tempDir, "test.json")
        store := storage.NewFileStorage(filePath)
        require.NoError(t, store.Initialize())

        err = store.Set("key1", "value1")
        require.NoError(t, err)

        val, err := store.Get("key1")
        require.NoError(t, err)
        assert.Equal(t, "value1", val)
    })
}
```

### Regression Test Example

```go
func TestRegressionConfigPersistence(t *testing.T) {
    binary := FindBinary()
    if binary == "" {
        t.Skip("CLI binary not found")
    }

    tempDir, err := os.MkdirTemp("", "regression-config-*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    t.Run("config survives binary restart", func(t *testing.T) {
        // First run: set config
        cmd1 := exec.Command(binary, "config", "set", "key", "value")
        output1, err := cmd1.CombinedOutput()
        require.NoError(t, err)

        // Second run: verify config persists
        cmd2 := exec.Command(binary, "config", "get", "key")
        output2, err := cmd2.CombinedOutput()
        require.NoError(t, err)

        assert.Contains(t, string(output2), "value")
    })
}
```

## CI/CD Integration

### Test Runner Script

Use the provided test runner script:

```bash
./scripts/run-tests.sh
```

### Makefile Targets

```makefile
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	go test ./tests/integration -v

test-parity:
	go test ./tests/parity -v

test-e2e:
	go test ./tests/e2e -v

test-regression:
	go test ./tests/regression -v

test-all: test test-integration test-parity test-e2e test-regression
```

## Performance Testing

### Benchmarks

Run benchmarks with:

```bash
# Run all benchmarks
go test ./... -bench=.

# Run specific benchmark
go test ./tests/integration -bench=BenchmarkStorage

# Run with memory profiling
go test ./tests/integration -bench=. -benchmem
```

## Debugging Tests

### Verbose Output

```bash
go test ./tests/integration -v -run TestFileStorage
```

### Race Detection

```bash
go test ./... -race
```

### Timeout Control

```bash
# Default timeout is 10 minutes
go test ./tests/e2e -timeout 5m
```

## Test Independence from TypeScript

### Independence Verification

Tests in `tests/regression/` verify the Go CLI can operate independently:

1. No shared configuration files
2. No TypeScript runtime dependencies
3. No parent directory file access
4. Self-contained binary

### Verifying Independence

```bash
# Build Go binary
go build -o cline ./cmd/cline

# Move to isolated location
cp cline /tmp/test-cline
cd /tmp

# Run tests
./test-cline version
./test-cline config list
```

## Known Issues and Regressions

### Tracking Known Issues

Add known issues to `KnownIssues` map in `tests/regression/regression_test.go`:

```go
var KnownIssues = map[string]*RegressionTest{
    "CLI-001": {
        ID:          "CLI-001",
        Description: "Config corruption on concurrent access",
        RelatedBug:  "https://github.com/cline/cline/issues/123",
        Category:    "storage",
        Steps: []RegressionStep{
            // Test steps
        },
    },
}
```

## Best Practices

1. **Always clean up** - Use `defer os.RemoveAll(tempDir)` for temporary files
2. **Skip when unavailable** - Use `t.Skip()` when dependencies are missing
3. **Table-driven tests** - Use test tables for multiple test cases
4. **Subtests** - Use `t.Run()` for organized test output
5. **Parallel execution** - Use `t.Parallel()` for independent tests
6. **Timeout handling** - Set appropriate timeouts for E2E tests
7. **Environment isolation** - Use temporary directories for test data
8. **Binary detection** - Automatically find test binaries

## Test Data

### Test Fixtures

Place test fixtures in `tests/fixtures/`:

```
tests/
├── fixtures/
│   ├── config/
│   │   └── sample.json
│   ├── history/
│   │   └── sample.json
│   └── providers/
│       └── responses/
```

### Mock Servers

Integration tests use `httptest` for mocking API providers:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(mockResponse)
}))
defer server.Close()
```

## Contributing

When adding new features:

1. Write unit tests for new functions
2. Add integration tests for new APIs
3. Update parity tests if behavior changes
4. Add regression tests for bug fixes
5. Ensure coverage remains above 80%