# Dual Testing Framework

The Dual Testing Framework ensures 100% parity between the Go CLI and TypeScript CLI implementations of Cline. It compares outputs, formats, and behavior between both implementations.

## Overview

This framework provides:
- **Test Harness**: Orchestrates parallel execution of tests against both CLI implementations
- **Output Comparison**: Normalizes and compares outputs with detailed diff reporting
- **Format Parity**: Validates JSON structure, plain text formatting, and error messages
- **Edge Case Testing**: Tests compound commands, large files, network failures, and more
- **Regression Detection**: Compares current results against baselines
- **CI Integration**: JUnit XML output and exit codes for CI pipelines

## Structure

```
tests/dual/
├── harness.go              # Core test harness and execution
├── format_parity_test.go   # JSON, text, and error format tests
├── edge_case_test.go       # Edge cases and stress tests
├── runner.go               # Test runner and comparison tools
└── README.md              # This file
```

## Running Tests

### Basic Usage

```bash
# Run all dual tests
go test ./tests/dual/...

# Run with verbose output
go test -v ./tests/dual/...

# Run specific test patterns
go test -v ./tests/dual/... -run TestVersionJSONFormat
```

### Using the CLI Test Runner

```bash
# Build the test runner
go build -o cline-test ./cmd/cline-test

# Run all tests
./cline-test

# Compare specific command
./cline-test -compare='version --json'

# Run specific test patterns
./cline-test -include='version*,config*'

# Generate JUnit report for CI
./cline-test -format=junit -output=report.xml

# Detect regressions
./cline-test -baseline=baseline.json -format=json -output=report.json
```

## Test Categories

### 1. Standard Dual Tests

Basic command parity tests defined in `harness.go`:

- `version_short` - Short version output
- `version_json` - JSON version output
- `help_root` - Root help command
- `config_list` - List configuration
- `history_json` - History in JSON format
- `mcp_help` - MCP help command
- `invalid_command` - Invalid command handling

### 2. Format Parity Tests

#### JSON Format Tests (`format_parity_test.go`)

- **Structure Validation**: Required fields, type checking
- **Semantic Validation**: Semantic versioning, valid timestamps
- **Deterministic Output**: Same command produces identical output
- **Field Order**: Consistent key ordering

#### Plain Text Tests

- **Help Format**: Usage, commands, flags sections
- **Version Output**: Single line with semantic version
- **Whitespace**: Consistent formatting

#### Error Format Tests

- **Exit Codes**: Non-zero for errors
- **Error Messages**: Descriptive error text
- **Invalid Commands**: Unknown command handling
- **Invalid Flags**: Unknown flag handling
- **Missing Arguments**: Required argument handling

### 3. Edge Case Tests (`edge_case_test.go`)

#### Compound Commands
- Piped input handling
- Large argument handling (10KB+)
- Special character arguments
- Multiple flag combinations

#### Large File Handling
- 1MB+ config files
- 1000+ item history files
- Binary/null characters in config
- Deeply nested JSON (100+ levels)

#### Network Failures
- No network available (offline mode)
- Slow network timeout
- Invalid proxy configurations

#### Concurrent Access
- 10+ simultaneous readers/writers
- Rapid sequential calls (100+)

#### Resource Exhaustion
- Many arguments (100+ flags)
- Long environment variables (100KB+)

#### Signal Handling
- SIGINT (Ctrl+C) handling
- Graceful shutdown

#### Unicode and Encoding
- Chinese, Japanese, Arabic characters
- Emoji (🎉🚀💻🔥)
- Mathematical symbols (∀x ∈ ℝ)
- Mixed content

#### File System Edge Cases
- Long paths (20+ nested directories)
- Special characters in paths (spaces, dashes, dots)
- Symlink handling

## Output Formats

### Text Format (Default)

Human-readable output with:
- Test summary
- Individual test results
- Differences highlighted
- Duration information

### JSON Format

Machine-readable output for automation:
```json
{
  "success": true,
  "total_tests": 10,
  "passed_tests": 10,
  "failed_tests": 0,
  "results": [...]
}
```

### JUnit XML Format

For CI/CD integration:
```xml
<testsuites>
  <testsuite name="dual-tests" tests="10" failures="0" time="5.23">
    <testcase name="version_short" time="0.15"/>
    ...
  </testsuite>
</testsuites>
```

## Difference Detection

The framework detects differences with severity levels:

- **INFO**: Performance variations, minor formatting differences
- **WARNING**: Non-critical differences that don't affect functionality
- **ERROR**: Differences that may affect behavior
- **CRITICAL**: Severe differences requiring immediate attention

Each difference includes:
- Type (ExitCode, Line, Matcher, etc.)
- Field name
- Go and TS values
- Description
- Severity level

## Regression Detection

Compare current results against a baseline:

```bash
# Create baseline
./cline-test -format=json -output=baseline.json

# Detect regressions
./cline-test -baseline=baseline.json
```

Regressions detected:
- Tests that passed but now fail
- New differences in existing tests
- Performance degradation

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Run Dual Tests
  run: |
    go test ./tests/dual/... -v
    go build -o cline-test ./cmd/cline-test
    ./cline-test -format=junit -output=dual-test-results.xml

- name: Upload Test Results
  uses: actions/upload-artifact@v3
  with:
    name: dual-test-results
    path: dual-test-results.xml
```

### Makefile Integration

```makefile
test-dual:
	go test ./tests/dual/... -v

test-dual-ci:
	./cline-test -format=junit -output=reports/dual-tests.xml
```

## Adding New Tests

### Standard Dual Test

```go
func MyCustomTests() []DualTest {
    return []DualTest{
        {
            Name:        "my_test",
            Description: "My custom test",
            Args:        []string{"my", "command", "--flag"},
            Timeout:     10 * time.Second,
            OutputMatchers: []OutputMatcher{
                {
                    Name:     "expected_pattern",
                    Pattern:  regexp.MustCompile(`expected output`),
                    Required: true,
                    InBoth:   true,
                },
            },
        },
    }
}
```

### Format Parity Test

```go
func TestMyFeatureJSONFormat(t *testing.T) {
    tests := []OutputFormatTest{
        {
            Name:           "my_feature_json",
            Args:           []string{"my-feature", "--json"},
            RequiredFields: []string{"field1", "field2"},
            FieldTypes: map[string]string{
                "field1": "string",
                "field2": "number",
            },
        },
    }
    // Run tests...
}
```

### Edge Case Test

```go
func TestMyEdgeCase(t *testing.T) {
    t.Run("my_edge_case", func(t *testing.T) {
        // Test implementation
    })
}
```

## Troubleshooting

### Binary Not Found

If binaries are not auto-detected:

```bash
./cline-test -go-binary=/path/to/cline -ts-binary=/path/to/ts-cline
```

### Timeout Issues

Increase timeout for slow tests:

```bash
./cline-test -timeout=60s
```

### Parallel Test Failures

Run tests sequentially:

```bash
./cline-test -parallel=false
```

## Performance Benchmarks

Run benchmarks:

```bash
go test -bench=. ./tests/dual/...
```

Available benchmarks:
- `BenchmarkGoJSONOutput` - Go CLI JSON performance
- `BenchmarkTSJSONOutput` - TS CLI JSON performance
- `BenchmarkLargeConfigHandling` - Large file handling
- `BenchmarkConcurrentAccess` - Concurrent operations