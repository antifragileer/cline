#!/bin/bash
# Optimized test runner for Go CLI
# Handles timeouts, parallelization, and resource limits

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test configuration
PARALLEL="${PARALLEL:-4}"
TIMEOUT="${TIMEOUT:-2m}"
VERBOSE="${VERBOSE:-0}"
RACE="${RACE:-0}"
SHORT="${SHORT:-0}"

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1" >&2; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" >&2; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Usage
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] [PACKAGE]

Optimized test runner with timeouts and resource limits

Options:
    -h, --help          Show this help
    -p, --parallel N    Number of parallel tests (default: $PARALLEL)
    -t, --timeout DUR   Test timeout (default: $TIMEOUT)
    -v, --verbose       Verbose output
    -r, --race          Enable race detector (slower)
    -s, --short         Run short tests only
    -a, --all           Run all test packages
    --unit              Run unit tests only (fast)
    --integration       Run integration tests only
    --e2e               Run E2E tests only

Examples:
    $(basename "$0")                    # Run unit tests
    $(basename "$0") --unit             # Same as above
    $(basename "$0") --all              # Run all tests
    $(basename "$0") -p 2 -t 5m         # Limit parallelism and timeout
    $(basename "$0") ./internal/config  # Test specific package
EOF
}

# Parse arguments
PACKAGE=""
RUN_ALL=0
RUN_UNIT=0
RUN_INTEGRATION=0
RUN_E2E=0

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -p|--parallel)
            PARALLEL="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        -r|--race)
            RACE=1
            shift
            ;;
        -s|--short)
            SHORT=1
            shift
            ;;
        -a|--all)
            RUN_ALL=1
            shift
            ;;
        --unit)
            RUN_UNIT=1
            shift
            ;;
        --integration)
            RUN_INTEGRATION=1
            shift
            ;;
        --e2e)
            RUN_E2E=1
            shift
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            PACKAGE="$1"
            shift
            ;;
    esac
done

# Build test flags
TEST_FLAGS=()
TEST_FLAGS+=("-parallel=$PARALLEL")
TEST_FLAGS+=("-timeout=$TIMEOUT")
TEST_FLAGS+=("-count=1")

[[ "$VERBOSE" == "1" ]] && TEST_FLAGS+=("-v")
[[ "$RACE" == "1" ]] && TEST_FLAGS+=("-race")
[[ "$SHORT" == "1" ]] && TEST_FLAGS+=("-short")

# Run tests with timeout protection
run_tests() {
    local pkg=$1
    local name=$2
    
    log_info "Running $name tests..."
    log_info "Package: $pkg"
    log_info "Flags: ${TEST_FLAGS[*]}"
    
    local start_time=$(date +%s)
    
    # Run with a hard timeout using timeout command
    if command -v timeout >/dev/null 2>&1; then
        timeout "$TIMEOUT" go test "$pkg" "${TEST_FLAGS[@]}" || {
            local exit_code=$?
            if [[ $exit_code -eq 124 ]]; then
                log_error "Tests timed out after $TIMEOUT"
                return 1
            fi
            return $exit_code
        }
    else
        # macOS doesn't have timeout by default
        go test "$pkg" "${TEST_FLAGS[@]}"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    log_success "$name tests completed in ${duration}s"
}

# Main execution
cd "$PROJECT_ROOT"

# Determine what to run
if [[ "$RUN_ALL" == "1" ]]; then
    log_info "Running all tests (this may take a while)..."
    run_tests "./..." "all"
elif [[ "$RUN_INTEGRATION" == "1" ]]; then
    run_tests "./tests/integration" "integration"
elif [[ "$RUN_E2E" == "1" ]]; then
    # E2E tests require built binary
    if [[ ! -f "$PROJECT_ROOT/cline" ]]; then
        log_warn "Binary not found, building first..."
        make build
    fi
    run_tests "./tests/e2e" "E2E"
elif [[ "$RUN_UNIT" == "1" ]] || [[ -z "$PACKAGE" ]]; then
    # Default: run unit tests (internal packages only, fast)
    log_info "Running unit tests (fast)..."
    
    # Test internal packages first (usually faster)
    for pkg in internal/*; do
        if [[ -d "$pkg" ]] && [[ -n "$(find "$pkg" -name '*_test.go' 2>/dev/null)" ]]; then
            log_info "Testing $pkg..."
            run_tests "./$pkg" "$pkg" || true  # Continue on failure
        fi
    done
    
    # Test cmd packages
    for pkg in cmd/*; do
        if [[ -d "$pkg" ]] && [[ -n "$(find "$pkg" -name '*_test.go' 2>/dev/null)" ]]; then
            log_info "Testing $pkg..."
            run_tests "./$pkg" "$pkg" || true
        fi
    done
    
    log_success "Unit tests completed"
else
    # Run specific package
    run_tests "$PACKAGE" "$PACKAGE"
fi