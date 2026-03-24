#!/bin/bash
#
# verify_independence.sh - Independence Verification Script for Go CLI
#
# This script verifies that the Go CLI remains independent of Node.js/TypeScript
# dependencies. It performs comprehensive checks to ensure:
#
#   1. No embedded JavaScript/TypeScript in the binary
#   2. No Node.js runtime dependencies
#   3. Static binary linking
#   4. No imports from cli/src/
#   5. No npm dependencies (package-lock.json, yarn.lock, etc.)
#
# Usage:
#   ./scripts/verify_independence.sh [options]
#
# Options:
#   -b, --binary PATH       Path to the built binary (auto-detected if not provided)
#   -p, --project PATH      Path to the golang-cli project root (auto-detected)
#   -j, --json              Output results in JSON format
#   -v, --verbose           Enable verbose output
#   -h, --help              Show this help message
#
# Exit Codes:
#   0 - All independence checks passed
#   1 - One or more checks failed
#   2 - Invalid arguments or configuration error
#
# Feature: FEAT-INFRA-DIST-014-INDEPENDENCE-001

set -euo pipefail

# Colors for terminal output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT=""
BINARY_PATH=""
JSON_OUTPUT=false
VERBOSE=false
FAILED_CHECKS=0
TOTAL_CHECKS=0

# ============================================================================
# Helper Functions
# ============================================================================

log_info() {
    if [[ "${VERBOSE}" == true ]]; then
        echo -e "${BLUE}[INFO]${NC} $1" >&2
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

show_help() {
    head -n 30 "$0" | tail -n 28
}

# ============================================================================
# Detection Functions
# ============================================================================

detect_project_root() {
    local dir="${SCRIPT_DIR}"
    
    # Walk up to find golang-cli directory
    while [[ "${dir}" != "/" ]]; do
        if [[ -f "${dir}/go.mod" ]]; then
            echo "${dir}"
            return 0
        fi
        dir="$(dirname "${dir}")"
    done
    
    # Try current directory
    if [[ -f "${PWD}/go.mod" ]]; then
        echo "${PWD}"
        return 0
    fi
    
    return 1
}

detect_binary() {
    local project_root="$1"
    local binary_name="cline"
    
    if [[ "$(uname -s)" == "MINGW"* ]] || [[ "$(uname -s)" == "CYGWIN"* ]] || [[ "$(uname -s)" == "MSYS"* ]]; then
        binary_name="cline.exe"
    fi
    
    # Common locations to check
    local locations=(
        "${project_root}/${binary_name}"
        "${project_root}/../${binary_name}"
        "${project_root}/cmd/cline/${binary_name}"
        "${project_root}/bin/${binary_name}"
        "${project_root}/dist/${binary_name}"
        "${project_root}/build/${binary_name}"
    )
    
    for loc in "${locations[@]}"; do
        if [[ -f "${loc}" ]]; then
            echo "${loc}"
            return 0
        fi
    done
    
    # Try to find in PATH
    if command -v "${binary_name}" &>/dev/null; then
        command -v "${binary_name}"
        return 0
    fi
    
    return 1
}

# ============================================================================
# Verification Functions
# ============================================================================

check_no_embedded_js() {
    local binary_path="$1"
    local check_name="no_embedded_js"
    
    log_info "Checking for embedded JavaScript/TypeScript in binary..."
    
    if [[ -z "${binary_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary path not provided\",\"details\":\"Cannot check for embedded JS without a binary path\"}"
        return 1
    fi
    
    if [[ ! -f "${binary_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary not found\",\"details\":\"Binary path does not exist: ${binary_path}\"}"
        return 1
    fi
    
    # JavaScript pattern detection
    local js_patterns=(
        'function[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\('
        'const[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*='
        'let[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*='
        'var[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*='
        'module\.exports'
        'require[[:space:]]*\('
        'import[[:space:]]+.*[[:space:]]+from[[:space:]]+'
        'export[[:space:]]+(default[[:space:]]+)?'
        'console\.(log|error|warn)'
        'process\.env'
        '__dirname'
        '__filename'
        '=>'
        'async[[:space:]]+function'
        'await[[:space:]]+'
    )
    
    local found_patterns=()
    
    for pattern in "${js_patterns[@]}"; do
        if grep -qP "${pattern}" "${binary_path}" 2>/dev/null; then
            found_patterns+=("${pattern}")
        fi
    done
    
    if [[ ${#found_patterns[@]} -gt 0 ]]; then
        local details="Found ${#found_patterns[@]} JS/TS patterns: ${found_patterns[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Potential JavaScript/TypeScript content detected in binary\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No embedded JavaScript/TypeScript detected\"}"
        return 0
    fi
}

check_no_node_dependencies() {
    local project_root="$1"
    local check_name="no_node_dependencies"
    
    log_info "Checking for Node.js dependencies..."
    
    # Check for node_modules
    if [[ -d "${project_root}/node_modules" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"node_modules directory found in project\",\"details\":\"node_modules exists at: ${project_root}/node_modules\"}"
        return 1
    fi
    
    # Check for package.json
    if [[ -f "${project_root}/package.json" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"package.json found in Go CLI project\",\"details\":\"package.json exists at: ${project_root}/package.json\"}"
        return 1
    fi
    
    # Check for Node.js shebangs in scripts
    local scripts_dir="${project_root}/scripts"
    if [[ -d "${scripts_dir}" ]]; then
        while IFS= read -r -d '' file; do
            if head -1 "${file}" | grep -qE '^#!/usr/bin/(env node|node)'; then
                echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Node.js script found\",\"details\":\"Script ${file} has Node.js shebang\"}"
                return 1
            fi
        done < <(find "${scripts_dir}" -type f -print0 2>/dev/null)
    fi
    
    echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No Node.js dependencies detected\"}"
    return 0
}

check_static_binary() {
    local binary_path="$1"
    local check_name="static_binary"
    
    log_info "Checking for static binary linking..."
    
    if [[ -z "${binary_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary path not provided\",\"details\":\"Cannot check static linking without a binary path\"}"
        return 1
    fi
    
    if [[ ! -f "${binary_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary not found\",\"details\":\"Binary path does not exist: ${binary_path}\"}"
        return 1
    fi
    
    local os_name
    os_name="$(uname -s)"
    
    case "${os_name}" in
        Linux)
            check_static_linux "${binary_path}" "${check_name}"
            return $?
            ;;
        Darwin)
            check_static_darwin "${binary_path}" "${check_name}"
            return $?
            ;;
        CYGWIN*|MINGW*|MSYS*)
            check_static_windows "${binary_path}" "${check_name}"
            return $?
            ;;
        *)
            echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"Static binary check skipped on ${os_name}\"}"
            return 0
            ;;
    esac
}

check_static_linux() {
    local binary_path="$1"
    local check_name="$2"
    
    if ! command -v readelf &>/dev/null; then
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"readelf not available, skipping static check\"}"
        return 0
    fi
    
    # Get dynamic dependencies
    local libs
    libs=$(readelf -d "${binary_path}" 2>/dev/null | grep -E '(NEEDED|Shared library)' || true)
    
    if [[ -z "${libs}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"Binary is statically linked\"}"
        return 0
    fi
    
    # Check for non-standard libraries
    local allowed_libs="^(libc|libpthread|libdl|libm|libresolv)\.so"
    local unallowed_libs=()
    
    while IFS= read -r line; do
        local lib
        lib=$(echo "${line}" | grep -oE '\[.*\]' | tr -d '[]' || true)
        if [[ -n "${lib}" ]] && ! [[ "${lib}" =~ ${allowed_libs} ]]; then
            unallowed_libs+=("${lib}")
        fi
    done <<< "${libs}"
    
    if [[ ${#unallowed_libs[@]} -gt 0 ]]; then
        local details="Unexpected libraries: ${unallowed_libs[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary has non-standard dynamic library dependencies\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"Binary uses only standard system libraries\"}"
        return 0
    fi
}

check_static_darwin() {
    local binary_path="$1"
    local check_name="$2"
    
    if ! command -v otool &>/dev/null; then
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"otool not available, skipping static check\"}"
        return 0
    fi
    
    local libs
    libs=$(otool -L "${binary_path}" 2>/dev/null | tail -n +2 || true)
    
    local allowed_libs="(libSystem\.B\.dylib|libc\+\+\.1\.dylib|libresolv\.9\.dylib)"
    local unallowed_libs=()
    
    while IFS= read -r line; do
        local lib
        lib=$(echo "${line}" | awk '{print $1}' || true)
        if [[ -n "${lib}" ]] && ! [[ "${lib}" =~ ${allowed_libs} ]]; then
            unallowed_libs+=("${lib}")
        fi
    done <<< "${libs}"
    
    if [[ ${#unallowed_libs[@]} -gt 0 ]]; then
        local details="Unexpected libraries: ${unallowed_libs[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary has non-standard dynamic library dependencies\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"Binary uses only standard system libraries\"}"
        return 0
    fi
}

check_static_windows() {
    local binary_path="$1"
    local check_name="$2"
    
    # For Windows, we check with objdump if available
    if command -v objdump &>/dev/null; then
        local libs
        libs=$(objdump -p "${binary_path}" 2>/dev/null | grep -E 'DLL Name:' || true)
        
        local allowed_libs="(KERNEL32|USER32|GDI32|ADVAPI32|SHELL32|WS2_32|MSVCRT)\.(dll|DLL)"
        local unallowed_libs=()
        
        while IFS= read -r line; do
            local lib
            lib=$(echo "${line}" | awk '{print $3}' || true)
            if [[ -n "${lib}" ]] && ! [[ "${lib}" =~ ${allowed_libs} ]]; then
                unallowed_libs+=("${lib}")
            fi
        done <<< "${libs}"
        
        if [[ ${#unallowed_libs[@]} -gt 0 ]]; then
            local details="Unexpected libraries: ${unallowed_libs[*]}"
            echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Binary has non-standard DLL dependencies\",\"details\":\"${details}\"}"
            return 1
        fi
    fi
    
    echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"Binary uses only standard system libraries\"}"
    return 0
}

check_no_cli_imports() {
    local project_root="$1"
    local check_name="no_cli_src_imports"
    
    log_info "Checking for imports from cli/src/..."
    
    local cli_src_path="${project_root}/../cli/src"
    if [[ ! -d "${cli_src_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"cli/src directory does not exist, skipping import check\"}"
        return 0
    fi
    
    # Find Go files with cli/src imports
    local cli_imports=()
    
    while IFS= read -r -d '' file; do
        if grep -q "github.com/cline/cline/cli/src" "${file}" 2>/dev/null; then
            local rel_path
            rel_path=$(realpath --relative-to="${project_root}" "${file}" 2>/dev/null || echo "${file}")
            cli_imports+=("${rel_path}")
        fi
    done < <(find "${project_root}" -name "*.go" ! -name "*_test.go" -print0 2>/dev/null)
    
    if [[ ${#cli_imports[@]} -gt 0 ]]; then
        local details="Files with cli/src imports: ${cli_imports[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Found imports from cli/src directory\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No imports from cli/src directory found\"}"
        return 0
    fi
}

check_no_npm_dependencies() {
    local project_root="$1"
    local check_name="no_npm_dependencies"
    
    log_info "Checking for npm dependency files..."
    
    # Check for various lock files and npm config
    local npm_files=(
        "package-lock.json"
        "yarn.lock"
        "pnpm-lock.yaml"
        "npm-shrinkwrap.json"
        ".npmrc"
    )
    
    for file in "${npm_files[@]}"; do
        if [[ -f "${project_root}/${file}" ]]; then
            echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"${file} found in Go CLI project\",\"details\":\"${file} exists at: ${project_root}/${file}\"}"
            return 1
        fi
    done
    
    echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No npm dependency files found\"}"
    return 0
}

check_no_typescript_files() {
    local project_root="$1"
    local check_name="no_typescript_files"
    
    log_info "Checking for TypeScript files..."
    
    # Find TypeScript files, excluding common directories
    local ts_files=()
    
    while IFS= read -r -d '' file; do
        local rel_path
        rel_path=$(realpath --relative-to="${project_root}" "${file}" 2>/dev/null || echo "${file}")
        ts_files+=("${rel_path}")
    done < <(find "${project_root}" -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.mts" -o -name "*.cts" \) ! -path "*/testdata/*" ! -path "*/vendor/*" ! -path "*/.git/*" -print0 2>/dev/null)
    
    if [[ ${#ts_files[@]} -gt 0 ]]; then
        local details="Found ${#ts_files[@]} TypeScript files: ${ts_files[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"TypeScript files found in Go CLI project\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No TypeScript files found\"}"
        return 0
    fi
}

check_no_package_json() {
    local project_root="$1"
    local check_name="no_package_json"
    
    log_info "Checking for package.json files..."
    
    local pkg_files=()
    
    while IFS= read -r -d '' file; do
        local rel_path
        rel_path=$(realpath --relative-to="${project_root}" "${file}" 2>/dev/null || echo "${file}")
        pkg_files+=("${rel_path}")
    done < <(find "${project_root}" -name "package.json" ! -path "*/testdata/*" ! -path "*/vendor/*" ! -path "*/.git/*" -print0 2>/dev/null)
    
    if [[ ${#pkg_files[@]} -gt 0 ]]; then
        local details="Found ${#pkg_files[@]} package.json files: ${pkg_files[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"package.json files found in Go CLI project\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"No package.json files found\"}"
        return 0
    fi
}

check_go_mod_integrity() {
    local project_root="$1"
    local check_name="go_mod_integrity"
    
    log_info "Checking go.mod integrity..."
    
    local go_mod_path="${project_root}/go.mod"
    if [[ ! -f "${go_mod_path}" ]]; then
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"go.mod not found\",\"details\":\"go.mod does not exist at: ${go_mod_path}\"}"
        return 1
    fi
    
    # Check for suspicious JavaScript-related dependencies
    local suspicious_patterns=(
        "github.com/robertkrimen/otto"
        "github.com/dop251/goja"
        "github.com/traefik/yaegi"
        "esbuild"
        "webpack"
        "typescript"
        "babel"
    )
    
    local found=()
    
    for pattern in "${suspicious_patterns[@]}"; do
        if grep -q "${pattern}" "${go_mod_path}" 2>/dev/null; then
            found+=("${pattern}")
        fi
    done
    
    if [[ ${#found[@]} -gt 0 ]]; then
        local details="Found suspicious patterns: ${found[*]}"
        echo "{\"name\":\"${check_name}\",\"passed\":false,\"message\":\"Suspicious JavaScript-related dependencies found in go.mod\",\"details\":\"${details}\"}"
        return 1
    else
        echo "{\"name\":\"${check_name}\",\"passed\":true,\"message\":\"go.mod integrity verified\"}"
        return 0
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--binary)
                BINARY_PATH="$2"
                shift 2
                ;;
            -p|--project)
                PROJECT_ROOT="$2"
                shift 2
                ;;
            -j|--json)
                JSON_OUTPUT=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
    
    # Detect project root if not provided
    if [[ -z "${PROJECT_ROOT}" ]]; then
        PROJECT_ROOT=$(detect_project_root)
        if [[ -z "${PROJECT_ROOT}" ]]; then
            log_error "Could not detect project root. Please specify with -p option."
            exit 2
        fi
        log_info "Detected project root: ${PROJECT_ROOT}"
    fi
    
    # Verify project root is valid
    if [[ ! -f "${PROJECT_ROOT}/go.mod" ]]; then
        log_error "Invalid project root: ${PROJECT_ROOT}/go.mod not found"
        exit 2
    fi
    
    # Detect binary if not provided
    if [[ -z "${BINARY_PATH}" ]]; then
        BINARY_PATH=$(detect_binary "${PROJECT_ROOT}")
        if [[ -n "${BINARY_PATH}" ]]; then
            log_info "Detected binary: ${BINARY_PATH}"
        else
            log_warn "Could not detect binary. Binary-dependent checks will be skipped."
        fi
    fi
    
    # Run all verification checks
    local results=()
    
    result=$(check_no_embedded_js "${BINARY_PATH}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_no_node_dependencies "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_static_binary "${BINARY_PATH}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_no_cli_imports "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_no_npm_dependencies "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_no_typescript_files "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_no_package_json "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    result=$(check_go_mod_integrity "${PROJECT_ROOT}")
    results+=("${result}")
    ((TOTAL_CHECKS++))
    
    # Count failures
    local failed=0
    for result in "${results[@]}"; do
        if [[ "${result}" == *"\"passed\":false"* ]]; then
            ((failed++))
        fi
    done
    
    FAILED_CHECKS=${failed}
    
    # Output results
    if [[ "${JSON_OUTPUT}" == true ]]; then
        # JSON output
        local json_results=""
        for result in "${results[@]}"; do
            if [[ -n "${json_results}" ]]; then
                json_results="${json_results},"
            fi
            json_results="${json_results}${result}"
        done
        
        local success="true"
        local exit_code=0
        if [[ ${FAILED_CHECKS} -gt 0 ]]; then
            success="false"
            exit_code=1
        fi
        
        cat <<EOF
{
  "success": ${success},
  "results": [${json_results}],
  "summary": "$([ "${success}" == "true" ] && echo "All independence checks passed" || echo "Independence verification failed")",
  "exitCode": ${exit_code}
}
EOF
    else
        # Text output
        echo "============================================================"
        echo "  Independence Verification Report"
        echo "============================================================"
        echo ""
        
        for result in "${results[@]}"; do
            local passed=$(echo "${result}" | grep -o '"passed":true' || true)
            local name=$(echo "${result}" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
            local message=$(echo "${result}" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
            
            if [[ -n "${passed}" ]]; then
                log_success "[${name}] ${message}"
            else
                log_fail "[${name}] ${message}"
                local details=$(echo "${result}" | grep -o '"details":"[^"]*"' | cut -d'"' -f4)
                if [[ -n "${details}" ]]; then
                    echo "       Details: ${details}"
                fi
            fi
            echo ""
        done
        
        echo "------------------------------------------------------------"
        if [[ ${FAILED_CHECKS} -eq 0 ]]; then
            log_success "Summary: All independence checks passed"
        else
            log_fail "Summary: Independence verification failed (${FAILED_CHECKS}/${TOTAL_CHECKS} checks failed)"
        fi
        echo "------------------------------------------------------------"
    fi
    
    # Exit with appropriate code
    if [[ ${FAILED_CHECKS} -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function
main "$@"