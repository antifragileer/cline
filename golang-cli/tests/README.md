# Smoke Tests and Binary Verification

This directory contains comprehensive smoke tests, independence verification, cross-platform build tests, and CI/CD pipeline tests for the Cline Go CLI.

## Files Created

### 1. smoke_test.go
Smoke tests for the Go CLI binary:
- **Binary execution**: Verifies the binary can be executed
- **Help text display**: Checks help output format and content
- **Version output**: Tests version command with --short and --json flags
- **Config command**: Tests config command functionality
- **History command**: Tests history command functionality
- **Binary size verification**: Ensures binary is under 100MB limit
- **No external dependencies**: Verifies binary runs with minimal environment

### 2. independence_test.go
Independence verification for the Go CLI:
- No embedded JavaScript/TypeScript in binary
- No Node.js dependencies (node_modules, package.json)
- Static binary linking verification
- No imports from cli/src/
- No npm dependency files (package-lock.json, yarn.lock, etc.)
- No TypeScript files in project
- go.mod integrity checks

### 3. cross_platform_test.go
Cross-platform build verification:
- Supports: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Automated cross-compilation testing
- Binary size validation per platform
- Build report generation

### 4. ci_cd_test.go
CI/CD pipeline tests:
- Build stage verification
- Unit test execution
- Smoke test integration
- Independence check integration
- Cross-platform build testing
- Binary size validation

### 5. test_runner.go
Unified test runner with CLI interface:
- Run all tests or specific test suites
- JSON and human-readable output formats
- Auto-detection of project root and binary paths
- Integration with all test types

### 6. scripts/verify_independence.sh
Bash script for independence verification:
- Comprehensive 8-point independence check
- JSON and text output formats
- Auto-detection of binary and project paths
- Detailed reporting with pass/fail status

## Running Tests

```bash
# Run all tests
go test ./tests/... -v

# Run smoke tests only
go test ./tests/... -v -run TestSmoke

# Run independence checks only
go test ./tests/... -v -run TestIndependence

# Run cross-platform builds
go test ./tests/... -v -run TestCrossPlatform

# Run CI/CD pipeline tests
go test ./tests/... -v -run TestCICD

# Use the bash script
./scripts/verify_independence.sh -v
```

## Binary Size Limit
Maximum allowed binary size: **100 MB**

## Cross-Platform Targets
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

## Success Criteria
All tests must pass:
- [x] Binary executes successfully
- [x] Help text displays correctly
- [x] Version output is correct
- [x] Config command works
- [x] History command works
- [x] Binary size is under 100MB
- [x] Binary runs without external dependencies
- [x] No embedded JS/TS in binary
- [x] No Node.js dependencies
- [x] Static binary linking
- [x] No imports from cli/src/
- [x] Cross-platform builds succeed
- [x] CI/CD pipeline passes
