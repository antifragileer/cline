# Testing Guide

This document describes how to run tests and verify the Go CLI implementation.

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test ./... -cover
```

### Run Tests with Coverage Report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Run Tests for Specific Package
```bash
go test ./internal/config/...
go test ./internal/storage/...
go test ./internal/tui/...
```

### Run Tests with Verbose Output
```bash
go test -v ./internal/config/...
```

### Run Benchmarks
```bash
go test -bench=. ./...
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

### E2E Tests

Located in `tests/e2e/`:

- `workflow_test.go` - Full CLI workflow tests

### Parity Tests

Located in `tests/parity/`:

- `feature_parity_test.go` - TypeScript vs Go feature parity
- `behavior_parity_test.go` - Behavior comparison tests

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

## Continuous Integration

Tests run automatically on:
- Every pull request
- Every push to main branch
- Nightly builds

### CI Test Matrix

- Go 1.21+
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)
- Windows (x86_64)

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

## Debugging Tests

### Run Specific Test
```bash
go test -run TestNewComponent ./internal/config/...
```

### Run with Race Detector
```bash
go test -race ./...
```

### Generate HTML Coverage Report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Known Issues

- Storage tests may take time due to file I/O
- TUI tests require terminal emulator
- Some tests require network access (marked with `//go:build integration`)