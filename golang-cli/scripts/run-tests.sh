#!/bin/bash
#
# Test runner script for Cline Go CLI
# Runs all test suites and generates reports
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Test configuration
COVERAGE_THRESHOLD=80
TEST_TIMEOUT=10m
VERBOSE=false
CI_MODE=false

# Help message
show_help() {
    cat << EOF
Usage: run-tests.sh [OPTIONS]

Run test suites for Cline Go CLI

Options:
    -h, --help          Show this help message
    -v, --verbose       Verbose output
    -c, --coverage      Run with coverage (default)
    -n, --no-coverage   Run without coverage
    -t, --timeout SEC   Test timeout (default: 10m)
    --ci                CI mode (race detection, coverage)
    --unit              Run only unit tests
    --integration       Run only integration tests
    --parity            Run only parity tests
    --e2e               Run only E2E tests
    --regression        Run only regression tests
    --all               Run all tests (default)
    --verify            Verify 80% coverage threshold

Examples:
    ./scripts/run-tests.sh                    # Run all tests
    ./scripts/run-tests.sh --unit             # Run only unit tests
    ./scripts/run-tests.sh --ci               # Run in CI mode
    ./scripts/run-tests.sh --verify           # Verify coverage threshold
EOF
}

# Parse arguments
RUN_UNIT=true
RUN_INTEGRATION=true
RUN_PARITY=true
RUN_E2E=true
RUN_REGRESSION=true
RUN_COVERAGE=true
VERIFY_COVERAGE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            RUN_COVERAGE=true
            shift
            ;;
        -n|--no-coverage)
            RUN_COVERAGE=false
            shift
            ;;
        -t|--timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --ci)
            CI_MODE=true
            shift
            ;;
        --unit)
            RUN_UNIT=true
            RUN_INTEGRATION=false
            RUN_PARITY=false
            RUN_E2E=false
            RUN_REGRESSION=false
            shift
            ;;
        --integration)
            RUN_UNIT=false
            RUN_INTEGRATION=true
            RUN_PARITY=false
            RUN_E2E=false
            RUN_REGRESSION=false
            shift
            ;;
        --parity)
            RUN_UNIT=false
            RUN_INTEGRATION=false
            RUN_PARITY=true
            RUN_E2E=false
            RUN_REGRESSION=false
            shift
            ;;
        --e2e)
            RUN_UNIT=false
            RUN_INTEGRATION=false
            RUN_PARITY=false
            RUN_E2E=true
            RUN_REGRESSION=false
            shift
            ;;
        --regression)
            RUN_UNIT=false
            RUN_INTEGRATION=false
            RUN_PARITY=false
            RUN_E2E=false
            RUN_REGRESSION=true
            shift
            ;;
        --all)
            RUN_UNIT=true
            RUN_INTEGRATION=true
            RUN_PARITY=true
            RUN_E2E=true
            RUN_REGRESSION=true
            shift
            ;;
        --verify)
            VERIFY_COVERAGE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Change to project root
cd "${PROJECT_ROOT}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

echo -e "${YELLOW}====================================${NC}"
echo -e "${YELLOW}Cline Go CLI Test Runner${NC}"
echo -e "${YELLOW}====================================${NC}"
echo ""

# Build the binary first if needed for E2E/parity tests
if [ "$RUN_E2E" = true ] || [ "$RUN_PARITY" = true ] || [ "$RUN_REGRESSION" = true ]; then
    echo -e "${YELLOW}Building CLI binary...${NC}"
    go build -o cline ./cmd/cline
    echo -e "${GREEN}✓ Binary built successfully${NC}"
    echo ""
fi

# Track results
declare -A RESULTS
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run test suite
run_test_suite() {
    local name=$1
    local package=$2
    local extra_flags=$3

    echo -e "${YELLOW}Running $name tests...${NC}"

    local flags="-v"
    if [ "$VERBOSE" = true ]; then
        flags="-v"
    fi

    if [ "$CI_MODE" = true ]; then
        flags="$flags -race"
    fi

    if [ -n "$extra_flags" ]; then
        flags="$flags $extra_flags"
    fi

    local start_time=$(date +%s)

    if go test "$package" $flags -timeout "$TEST_TIMEOUT"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✓ $name tests passed (${duration}s)${NC}"
        RESULTS["$name"]="PASSED"
        ((PASSED_TESTS++))
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}✗ $name tests failed (${duration}s)${NC}"
        RESULTS["$name"]="FAILED"
        ((FAILED_TESTS++))
    fi

    ((TOTAL_TESTS++))
    echo ""
}

# Run unit tests
if [ "$RUN_UNIT" = true ]; then
    run_test_suite "Unit" "./tests"
fi

# Run integration tests
if [ "$RUN_INTEGRATION" = true ]; then
    run_test_suite "Integration" "./tests/integration"
fi

# Run parity tests
if [ "$RUN_PARITY" = true ]; then
    run_test_suite "Parity" "./tests/parity"
fi

# Run E2E tests
if [ "$RUN_E2E" = true ]; then
    run_test_suite "E2E" "./tests/e2e"
fi

# Run regression tests
if [ "$RUN_REGRESSION" = true ]; then
    run_test_suite "Regression" "./tests/regression"
fi

# Generate coverage report
if [ "$RUN_COVERAGE" = true ]; then
    echo -e "${YELLOW}Generating coverage report...${NC}"

    if go test ./... -coverprofile=coverage.out -timeout "$TEST_TIMEOUT"; then
        echo -e "${GREEN}✓ Coverage data generated${NC}"

        # Display coverage summary
        echo ""
        echo -e "${YELLOW}Coverage Summary:${NC}"
        go tool cover -func=coverage.out | tail -1

        # Generate HTML report
        go tool cover -html=coverage.out -o coverage.html
        echo -e "${GREEN}✓ HTML report: coverage.html${NC}"

        # Check coverage threshold
        if [ "$VERIFY_COVERAGE" = true ]; then
            echo ""
            echo -e "${YELLOW}Verifying coverage threshold (${COVERAGE_THRESHOLD}%)...${NC}"

            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
            if (( $(echo "$COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
                echo -e "${RED}✗ Coverage $COVERAGE% is below threshold of $COVERAGE_THRESHOLD%${NC}"
                exit 1
            else
                echo -e "${GREEN}✓ Coverage $COVERAGE% meets threshold of $COVERAGE_THRESHOLD%${NC}"
            fi
        fi
    else
        echo -e "${RED}✗ Failed to generate coverage report${NC}"
    fi
    echo ""
fi

# Print summary
echo -e "${YELLOW}====================================${NC}"
echo -e "${YELLOW}Test Summary${NC}"
echo -e "${YELLOW}====================================${NC}"

for name in "${!RESULTS[@]}"; do
    status=${RESULTS[$name]}
    if [ "$status" = "PASSED" ]; then
        echo -e "${GREEN}✓ $name: $status${NC}"
    else
        echo -e "${RED}✗ $name: $status${NC}"
    fi
done

echo ""
echo -e "Total: $TOTAL_TESTS | Passed: $PASSED_TESTS | Failed: $FAILED_TESTS"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi