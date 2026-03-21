---
description: BATS (Bash Automated Testing System) best practices and mandatory usage for all bash script unit testing
applies_to: ["**/*.sh", "**/*.bats", "**/test/**/*.sh", "**/tests/**/*.bats", "scripts/**/*.sh"]
priority: critical
---

# BATS Testing Best Practices

## Mandatory BATS Usage

**Rule:** ALL bash script unit testing MUST use BATS (Bash Automated Testing System)
**Rule:** NEVER create bash tests using custom testing frameworks or ad-hoc test scripts
**Rule:** BATS is the ONLY approved testing framework for bash script validation
**Rule:** All bash scripts intended for production use MUST have corresponding BATS test files
**Rule:** Test coverage requirements apply equally to bash scripts as to other languages

## Test File Structure

**Rule:** BATS test files MUST use `.bats` file extension
**Rule:** Test files MUST begin with shebang line: `#!/usr/bin/env bats`
**Rule:** Test files MUST be placed in `test/` or `tests/` directory parallel to source scripts
**Rule:** Test file names MUST match the script being tested with `.bats` extension (e.g., `deploy.sh` → `deploy.bats`)
**Rule:** Organize multiple test files by feature or module when testing complex bash projects
**Rule:** Use subdirectories within `test/` to mirror source directory structure when needed

## Test Case Definition

**Rule:** Every test case MUST use the `@test` decorator followed by descriptive test name
**Rule:** Test names MUST clearly describe the behavior being tested
**Rule:** Use natural language test names explaining expected behavior (e.g., "fails when file does not exist")
**Rule:** Group related tests together in the same file with consistent naming patterns
**Rule:** Keep test cases focused on single behavior or assertion
**Rule:** Test names MUST use double quotes to handle spaces and special characters

## Setup and Teardown Hooks

**Rule:** Use `setup()` function to run code before EACH test case
**Rule:** Use `teardown()` function to run code after EACH test case
**Rule:** Use `setup_file()` to run code once before first test in a file
**Rule:** Use `teardown_file()` to run code once after last test in a file
**Rule:** Use `setup_suite()` to run code once before all tests in test suite
**Rule:** Use `teardown_suite()` to run code once after all tests in test suite
**Rule:** Clean up temporary files, processes, and resources in teardown hooks
**Rule:** Never rely on test execution order for setup state
**Rule:** Ensure teardown hooks run even if tests fail by using trap if needed

## Assertions and the run Command

**Rule:** Use `run` command to execute scripts or commands being tested
**Rule:** The `run` command captures exit status in `$status` variable
**Rule:** The `run` command captures output in `$output` variable
**Rule:** The `run` command captures individual output lines in `${lines[@]}` array
**Rule:** Use standard bash test operators for assertions: `[ ]`, `[[ ]]`, or `test`
**Rule:** Check exit status with: `[ "$status" -eq 0 ]` for success or `[ "$status" -ne 0 ]` for failure
**Rule:** Check output with: `[ "$output" = "expected" ]` or pattern matching
**Rule:** Use `set -e` behavior: all commands in test MUST exit successfully unless using `run`
**Rule:** Wrap commands in `run` when testing expected failures

## Helper Libraries

**Rule:** Use `bats-support` library for enhanced testing capabilities
**Rule:** Use `bats-assert` library for readable assertion helpers
**Rule:** Use `bats-file` library for filesystem-related assertions
**Rule:** Load helper libraries in setup functions using `load` command
**Rule:** Load helper libraries with relative paths: `load 'test_helper/bats-assert/load'`
**Rule:** Use `assert_success` and `assert_failure` from bats-assert for exit status checks
**Rule:** Use `assert_output`, `assert_line`, and `refute_output` for output validation
**Rule:** Use `assert_equal` for value comparisons with clear failure messages
**Rule:** Install helper libraries as git submodules or npm packages in test_helper directory

## Test Organization and Common Setup

**Rule:** Create `test_helper/common-setup.bash` for shared setup logic across test files
**Rule:** Use `load 'test_helper/common-setup'` to include common setup in test files
**Rule:** Set `PROJECT_ROOT` variable in common setup for consistent path handling
**Rule:** Use `$BATS_TEST_FILENAME` instead of `$0` or `${BASH_SOURCE[0]}` for path resolution
**Rule:** Add script directories to PATH in setup to make scripts executable by name
**Rule:** Use `DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" && pwd )"` for test file directory
**Rule:** Centralize common test fixtures and mock data in test_helper directory
**Rule:** Document common setup functions with comments explaining their purpose

## Test Data and Fixtures

**Rule:** Create temporary files using `mktemp` or `mktemp -d` for directories
**Rule:** Store temporary file paths in variables for cleanup in teardown
**Rule:** Never hard-code paths to system directories in tests
**Rule:** Use fixtures for complex test data stored in test_helper/fixtures directory
**Rule:** Clean up ALL temporary resources in teardown hooks
**Rule:** Use `mktemp -u` to generate name without creating file when testing creation logic
**Rule:** Avoid depending on external files that may not exist in CI/CD environments

## Environment and Variables

**Rule:** Set required environment variables in setup functions
**Rule:** Save and restore modified environment variables in tests
**Rule:** Use local variables within test functions to avoid global state pollution
**Rule:** Export variables only when necessary for tested script execution
**Rule:** Document required environment variables in test file comments
**Rule:** Use BATS-provided variables: `$BATS_TEST_FILENAME`, `$BATS_TEST_DIRNAME`, `$BATS_TEST_NAME`
**Rule:** Never rely on user's shell environment for test execution

## Mocking and Stubbing

**Rule:** Mock external commands by creating stub functions or scripts in setup
**Rule:** Place mock scripts earlier in PATH than real commands
**Rule:** Create mock scripts in temporary directory added to PATH
**Rule:** Verify mock invocations by logging calls to temporary file
**Rule:** Clean up mock functions and scripts in teardown
**Rule:** Use function definitions to override commands: `function docker() { echo "mocked"; }`
**Rule:** Document what is being mocked and why in test comments

## CI/CD Integration

**Rule:** BATS produces TAP (Test Anything Protocol) output compatible with CI systems
**Rule:** Run BATS tests in CI/CD pipeline using `bats test/` or `bats tests/`
**Rule:** Use `--tap` flag for explicit TAP format output in CI environments
**Rule:** Use `--formatter junit` for JUnit XML output when required by CI system
**Rule:** Run tests with `--timing` flag to identify slow tests
**Rule:** Use `--jobs` flag for parallel test execution when tests are independent
**Rule:** Set `BATS_RUN_SKIPPED=1` to run tests marked with `skip` in specific CI runs
**Rule:** Ensure all dependencies (bats-core, helper libraries) are installed in CI environment
**Rule:** Cache BATS and helper library installations in CI for faster builds

## Parallel Test Execution

**Rule:** Design tests to be completely independent and parallelizable
**Rule:** Avoid shared state between tests that prevents parallel execution
**Rule:** Use unique temporary files and directories per test
**Rule:** Run tests in parallel with `bats --jobs <number>` flag
**Rule:** Start with `--jobs 4` and adjust based on system resources
**Rule:** Ensure parallel tests don't conflict with ports, files, or system resources

## Test Coverage and Quality

**Rule:** Write tests for all critical bash script functionality
**Rule:** Test both success and failure scenarios for each script function
**Rule:** Test edge cases, boundary conditions, and error handling
**Rule:** Test command-line argument parsing and validation
**Rule:** Test script behavior with missing dependencies or permissions
**Rule:** Use descriptive failure messages in assertions for debugging
**Rule:** Keep tests maintainable by avoiding excessive mocking or complex setup
**Rule:** Refactor tests when they become difficult to understand or maintain

## Debugging Tests

**Rule:** Use `echo` statements in tests for debugging (visible in TAP output on failure)
**Rule:** Run single test file with: `bats test/specific-test.bats`
**Rule:** Run specific test by line number: `bats test/file.bats:42`
**Rule:** Use `--print-output-on-failure` flag to see output from passing tests
**Rule:** Add `skip "debugging"` to focus on specific test temporarily
**Rule:** Never commit tests with `skip` unless documenting known issues
**Rule:** Use `set -x` in test or setup for detailed execution trace

## Performance Optimization

**Rule:** Minimize expensive operations in setup and teardown hooks
**Rule:** Use `setup_file` and `teardown_file` for expensive one-time setup
**Rule:** Avoid unnecessary subshells and command executions in tests
**Rule:** Cache computed values in setup when used by multiple tests
**Rule:** Profile tests with `--timing` flag to identify slow tests
**Rule:** Optimize slow tests by reducing external command calls

## Anti-Patterns to Avoid

**Rule:** NEVER test implementation details, test behavior and outcomes
**Rule:** NEVER create tests that depend on execution order
**Rule:** NEVER use sleep for synchronization, use proper waiting mechanisms
**Rule:** NEVER hardcode absolute paths to files or directories
**Rule:** NEVER skip cleanup in teardown hooks
**Rule:** NEVER commit commented-out tests, remove them or document with skip
**Rule:** NEVER write overly complex tests that are hard to understand
**Rule:** NEVER mock everything, test real script behavior when possible
**Rule:** NEVER ignore test failures in CI/CD pipeline
**Rule:** NEVER use global variables without cleanup
**Rule:** NEVER test bash scripts without BATS
**Rule:** NEVER create custom bash testing frameworks when BATS exists

## Documentation and Maintenance

**Rule:** Add comments explaining complex test setup or assertions
**Rule:** Document test file purpose at top with comments
**Rule:** Keep test files up to date when modifying tested scripts
**Rule:** Remove obsolete tests when removing script features
**Rule:** Include README in test directory explaining how to run tests
**Rule:** Document required BATS version and helper library versions

## Installation and Dependencies

**Rule:** Install BATS using package manager or npm: `npm install -g bats`
**Rule:** Include BATS installation instructions in project README
**Rule:** Pin BATS version in package.json or similar dependency file
**Rule:** Install helper libraries as git submodules in test/test_helper/
**Rule:** Document all testing dependencies in project documentation
**Rule:** Provide setup script to install BATS and dependencies for new developers

## Enforcement

**Rule:** Code reviews MUST verify bash scripts have BATS tests
**Rule:** CI/CD pipeline MUST run all BATS tests on every commit
**Rule:** CI/CD MUST fail build if any BATS tests fail
**Rule:** Pull requests adding bash scripts MUST include BATS tests
**Rule:** This is a CRITICAL priority rule with NO exceptions permitted