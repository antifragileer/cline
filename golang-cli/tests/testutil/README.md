# Test Utilities for AI Connection Serialization

This package provides utilities to ensure tests that execute real LLM commands don't run in parallel, preventing rate limiting and process accumulation issues.

## Problem

Tests that invoke `cline task`, `cline ask`, or any command that makes live API calls are "AI connection tests". These tests:
- Make real API calls to LLM providers (OpenAI, Anthropic, etc.)
- Consume rate limit quotas
- Spawn long-running processes
- Can cause issues when run in parallel

## Solution

The `ai_test_lock.go` utility provides file-based locking to serialize AI connection tests across all test processes.

## Usage

### For AI Connection Tests

Any test that executes the CLI with a prompt that could trigger LLM execution should acquire the AI lock:

```go
package functional

import (
    "testing"
    "github.com/cline/cline/golang-cli/tests/testutil"
)

func TestTaskExecution(t *testing.T) {
    // Acquire lock to ensure serial execution
    release := testutil.AcquireAILock(t)
    defer release()
    
    // Your test code here...
    // This test will not run concurrently with other AI tests
}
```

### For Non-AI Tests

Tests that don't invoke LLM commands (config, version, help, etc.) don't need the lock:

```go
func TestConfigCommands(t *testing.T) {
    // No lock needed - this test doesn't call LLMs
    // These tests can run in parallel with each other
}
```

### Test Classification

**AI Connection Tests** (Need lock):
- Tests using `cline task` with prompts
- Tests using `cline ask`
- Tests with `-y` (yolo) or `--auto-approve-all` flags
- Tests with `-a` (act) or `-p` (plan) modes
- Tests that resume tasks with prompts

**Local Tests** (No lock needed):
- `cline version`
- `cline --help`
- `cline config get/set/list`
- `cline auth status/list` (without API calls)
- `cline history`
- Internal unit tests

### In Test Setup

For test files with mixed AI and non-AI tests, only acquire the lock for specific subtests:

```go
func TestMixedScenarios(t *testing.T) {
    t.Run("version_command", func(t *testing.T) {
        // No lock needed
        // Test version output
    })
    
    t.Run("task_execution", func(t *testing.T) {
        // Lock needed for AI test
        release := testutil.AcquireAILock(t)
        defer release()
        
        // Test task execution
    })
}
```

## Lock Behavior

- **Blocking**: `AcquireAILock` blocks until the lock is available (up to 10 minute timeout)
- **Process-safe**: Uses exclusive file creation (O_EXCL) for atomic lock acquisition
- **Cross-package**: Lock is shared across all test packages
- **Auto-cleanup**: Lock is released when the test completes (via defer)
- **Debugging**: Lock file contains test name, PID, and timestamp

## Configuration

### Environment Variables

- `CLINE_AI_TEST_TIMEOUT`: Override the default 10-minute lock timeout
- `CLINE_AI_TEST_SKIP_PARALLEL`: If set, skip AI tests instead of waiting for lock

### Makefile Integration

The Makefile has been updated to run AI tests with `-parallel=1`:

```bash
# Functional tests (AI tests) run serially
make test-functional

# Integration tests may include AI tests
make test-integration

# All tests with proper serialization
make test-all
```

## Troubleshooting

### Stale Locks

If a test crashes or is killed, the lock file may be left behind. It's automatically cleaned up if older than the timeout, or you can manually remove it:

```bash
rm /tmp/cline_ai_test.lock
```

### Debugging Lock Issues

Enable verbose test output to see lock acquisition/release messages:

```bash
go test -v ./tests/functional -run TestTaskExecution
```

### Timeout Issues

If tests consistently timeout waiting for the lock:
1. Check for stuck test processes: `ps aux | grep cline`
2. Kill stuck processes: `pkill -f "cline.*test"`
3. Remove stale lock: `rm /tmp/cline_ai_test.lock`

## Best Practices

1. **Always defer release**: Use `defer release()` to ensure lock is released even if test fails
2. **Acquire at test start**: Get the lock as early as possible in the test
3. **Minimize lock duration**: Keep AI tests focused and fast
4. **Document AI tests**: Add comments explaining why a test needs the lock
5. **Use timeouts**: Always use `context.WithTimeout` for CLI commands