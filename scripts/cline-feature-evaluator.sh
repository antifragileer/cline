#!/bin/bash
#
# Cline Feature Evaluator - Gap Analysis and Remediation Script
#
# This script evaluates golang CLI features against the NodeJS implementation:
# 1. Reviews implementation of golang CLI features vs NodeJS reference
# 2. Creates remediation manifest with execution hierarchy and prompts
# 3. Executes cline remediation sessions per issue (parallel where possible)
# 4. Validates remediation with automated testing
# 5. Compares NodeJS and golang CLI outputs for parity
# 6. Iterates until all features meet definition of done
#
# Usage: ./cline-feature-evaluator.sh [OPTIONS] <path/to/execution-manifest.json>
#

set -euo pipefail

# Track background processes for cleanup
declare -a BACKGROUND_PIDS=()

# Safe array length helper
array_length() {
    local arr_name=$1
    eval "echo \${#${arr_name}[@]}"
}

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Code directories
GOLANG_CLI_DIR="${PROJECT_ROOT}/golang-cli"
NODEJS_CLI_DIR="${PROJECT_ROOT}/cli"

# Default values
DRY_RUN=false
MAX_RETRIES=3
RETRY_DELAY=5
EXECUTION_MANIFEST_FILE=""
REMEDIATION_MANIFEST_FILE=""
CLINE_BIN="cline"
VERBOSE=false
PARALLELISM=4
SKIP_VALIDATION=false
PHASE=""
SKIP_COMPARISON=false
SKIP_TESTS=false
FORCE_REMEDIATION=false

# Colors for output
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly CYAN='\033[0;36m'
    readonly MAGENTA='\033[0;35m'
    readonly ORANGE='\033[0;33m'
    readonly NC='\033[0m'
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly CYAN=''
    readonly MAGENTA=''
    readonly ORANGE=''
    readonly NC=''
fi

# Logging functions
log_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$1" >&2
}

log_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1" >&2
}

log_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1" >&2
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

log_step() {
    printf "${CYAN}[STEP]${NC} %s\n" "$1" >&2
}

log_review() {
    printf "${MAGENTA}[REVIEW]${NC} %s\n" "$1" >&2
}

log_compare() {
    printf "${ORANGE}[COMPARE]${NC} %s\n" "$1" >&2
}

# Display usage information
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <path/to/execution-manifest.json>

Evaluate golang CLI features against NodeJS implementation and remediate gaps.

Options:
    -h, --help                 Show this help message
    -d, --dry-run              Show what would be done without executing
    -r, --remediation-manifest Path to existing remediation manifest
    -p, --parallelism N        Number of parallel remediation executions (default: 4)
    --max-retries N            Maximum retry attempts per remediation (default: 3)
    --retry-delay N            Delay between retries in seconds (default: 5)
    -c, --cline-bin BIN        Path to cline binary (default: cline)
    -v, --verbose              Enable verbose output
    --skip-validation          Skip post-remediation validation
    --skip-comparison          Skip NodeJS vs golang output comparison
    --skip-tests               Skip running automated tests
    --phase N                  Execute only specific phase number
    --force-remediation        Force remediation even if feature appears complete

Examples:
    $(basename "$0") .oxenated/docs/planning/cline-cli-golang-migration-prd-feature-manifest-execution.json
    $(basename "$0") --dry-run path/to/execution-manifest.json
    $(basename "$0") --remediation-manifest path/to/remediation-manifest.json
    $(basename "$0") --parallelism 2 --phase 1 path/to/execution-manifest.json

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -r|--remediation-manifest)
                REMEDIATION_MANIFEST_FILE="$2"
                shift 2
                ;;
            -p|--parallelism)
                PARALLELISM="$2"
                shift 2
                ;;
            --max-retries)
                MAX_RETRIES="$2"
                shift 2
                ;;
            --retry-delay)
                RETRY_DELAY="$2"
                shift 2
                ;;
            -c|--cline-bin)
                CLINE_BIN="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --skip-validation)
                SKIP_VALIDATION=true
                shift
                ;;
            --skip-comparison)
                SKIP_COMPARISON=true
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --phase)
                PHASE="$2"
                shift 2
                ;;
            --force-remediation)
                FORCE_REMEDIATION=true
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                EXECUTION_MANIFEST_FILE="$1"
                shift
                ;;
        esac
    done

    if [[ -z "${EXECUTION_MANIFEST_FILE:-}" ]] && [[ -z "${REMEDIATION_MANIFEST_FILE}" ]]; then
        log_error "Execution manifest file path is required (unless using --remediation-manifest)"
        usage
        exit 1
    fi
}

# Check if cline CLI is available
check_cline_cli() {
    log_info "Checking cline CLI availability..."
    
    if ! command -v "$CLINE_BIN" &> /dev/null; then
        log_error "cline CLI not found: $CLINE_BIN"
        log_error "Please ensure cline is installed and in PATH"
        exit 1
    fi
    
    local version
    version=$($CLINE_BIN version 2>/dev/null | head -1 || echo "unknown")
    log_info "Found cline: $version"
}

# Validate JSON file
validate_json() {
    local file="$1"
    if ! jq empty "$file" 2>/dev/null; then
        return 1
    fi
    return 0
}

# Clean and fix common JSON issues
clean_json_file() {
    local input_file="$1"
    local output_file="$2"
    
    if jq . "$input_file" > "$output_file" 2>/dev/null; then
        return 0
    fi
    
    log_warn "JSON has syntax errors, attempting to clean..."
    
    sed -e 's/[\x00-\x08\x0b\x0c\x0e-\x1f]//g' \
        -e 's/\\n"/"/g' \
        -e 's/,\s*}/}/g' \
        -e 's/,\s*]/]/g' \
        "$input_file" > "$output_file.tmp"
    
    if jq . "$output_file.tmp" > "$output_file" 2>/dev/null; then
        rm -f "$output_file.tmp"
        return 0
    fi
    
    rm -f "$output_file.tmp"
    return 1
}

# Get PRD file path from manifest
get_prd_file() {
    local manifest_file="$1"
    local prd_file
    
    prd_file=$(jq -r '.metadata.source_prd // .metadata.source // empty' "$manifest_file" 2>/dev/null || true)
    
    if [[ -n "$prd_file" && -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Infer from manifest filename
    prd_file="${manifest_file%-feature-manifest-execution.json}.md"
    prd_file="${prd_file%-execution.json}.md"
    
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    local manifest_dir
    manifest_dir=$(dirname "$manifest_file")
    
    local base_name
    base_name=$(basename "$manifest_file" .json | sed 's/-feature-manifest-execution$//' | sed 's/-execution$//')
    prd_file="${manifest_dir}/${base_name}.md"
    
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    echo "$prd_file"
}

# Find corresponding NodeJS implementation for a feature
find_nodejs_reference() {
    local feature_id="$1"
    local feature_name="$2"
    
    local references=()
    
    case "$feature_id" in
        # Storage layer
        FEAT-INFRA-STORAGE-*)
            references+=("src/shared/storage/")
            ;;
        # gRPC/Core
        FEAT-INFRA-CORE-*)
            references+=("src/core/")
            references+=("src/shared/proto/")
            ;;
        # API Providers
        FEAT-INFRA-API-*)
            references+=("src/api/")
            ;;
        # CLI Commands
        FEAT-DEV-CLI-*)
            references+=("cli/src/")
            ;;
        # Authentication
        FEAT-DEV-AUTH-*)
            references+=("cli/src/")
            references+=("src/core/auth/")
            ;;
        # UI components
        FEAT-DEV-UI-*)
            references+=("cli/src/components/")
            references+=("cli/src/")
            ;;
        # Task management
        FEAT-DEV-TASK-*)
            references+=("src/core/task/")
            ;;
        # Automation modes
        FEAT-AUTO-MODE-*|FEAT-AUTO-EXEC-*|FEAT-AUTO-OUT-*)
            references+=("cli/src/")
            ;;
        # Enterprise config
        FEAT-ENT-CONFIG-*)
            references+=("src/core/config/")
            references+=("src/shared/")
            ;;
        # Security
        FEAT-ENT-SEC-*)
            references+=("src/core/")
            ;;
        # Audit
        FEAT-ENT-AUDIT-*)
            references+=("src/core/")
            ;;
        # Distribution
        FEAT-INFRA-DIST-*)
            references+=("scripts/")
            ;;
    esac
    
    # Check which references actually exist
    local existing=()
    for ref in "${references[@]}"; do
        if [[ -d "${NODEJS_CLI_DIR}/${ref}" ]] || [[ -d "${PROJECT_ROOT}/${ref}" ]]; then
            existing+=("$ref")
        fi
    done
    
    printf '%s\n' "${existing[@]}"
}

# Get golang implementation directory for a feature
get_golang_output_dir() {
    local feature_id="$1"
    
    case "$feature_id" in
        FEAT-INFRA-STORAGE-*) echo "${GOLANG_CLI_DIR}/internal/storage" ;;
        FEAT-INFRA-CORE-*) echo "${GOLANG_CLI_DIR}/internal/host" ;;
        FEAT-INFRA-API-*) echo "${GOLANG_CLI_DIR}/internal/api" ;;
        FEAT-DEV-CLI-001-CMD-001|FEAT-DEV-CLI-001-CMD-002|FEAT-DEV-CLI-001-CMD-003|FEAT-DEV-CLI-001-CMD-004) echo "${GOLANG_CLI_DIR}/cmd/cline" ;;
        FEAT-DEV-CLI-*) echo "${GOLANG_CLI_DIR}/internal/cli" ;;
        FEAT-DEV-AUTH-*) echo "${GOLANG_CLI_DIR}/internal/auth" ;;
        FEAT-DEV-UI-*) echo "${GOLANG_CLI_DIR}/internal/tui" ;;
        FEAT-DEV-TASK-*) echo "${GOLANG_CLI_DIR}/internal/task" ;;
        FEAT-AUTO-MODE-*|FEAT-AUTO-EXEC-*|FEAT-AUTO-OUT-*) echo "${GOLANG_CLI_DIR}/internal/mode" ;;
        FEAT-ENT-CONFIG-*) echo "${GOLANG_CLI_DIR}/internal/config" ;;
        FEAT-ENT-SEC-*) echo "${GOLANG_CLI_DIR}/internal/security" ;;
        FEAT-ENT-AUDIT-*) echo "${GOLANG_CLI_DIR}/internal/audit" ;;
        FEAT-INFRA-DIST-014-BUILD-001|FEAT-INFRA-DIST-014-HOMEBREW-002|FEAT-INFRA-DIST-014-NPM-003) echo "${GOLANG_CLI_DIR}/scripts" ;;
        FEAT-INFRA-DIST-014-INDEPENDENCE-001) echo "${GOLANG_CLI_DIR}/tests" ;;
        *) echo "${GOLANG_CLI_DIR}/internal" ;;
    esac
}

# Build remediation manifest from execution manifest
build_remediation_manifest() {
    local execution_manifest_file="$1"
    local prd_file="$2"
    
    local manifest_dir
    local base_name
    local remediation_manifest_path
    
    manifest_dir=$(dirname "$execution_manifest_file")
    base_name=$(basename "$execution_manifest_file" .json)
    remediation_manifest_path="${manifest_dir}/${base_name}-remediate.json"
    
    log_info "Building remediation manifest from: $execution_manifest_file"
    log_info "Remediation manifest will be written to: $remediation_manifest_path"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would analyze features and build remediation manifest"
        echo "$remediation_manifest_path"
        return 0
    fi
    
    # Create the prompt for remediation manifest generation
    local prompt
    prompt=$(cat << 'EOF'
You are evaluating the golang CLI implementation against the NodeJS CLI reference implementation.

For each feature in the execution manifest, you MUST perform a gap analysis:

1. **Locate NodeJS Reference Implementation:**
   - Search in cli/src/ for corresponding TypeScript/React code
   - Search in src/core/ for backend logic
   - Search in src/api/ for provider implementations
   - Look for equivalent functionality, not identical code structure

2. **Evaluate Golang Implementation:**
   - Check if golang files exist in the expected directory
   - Review code for functional completeness
   - Verify all required methods/functions are implemented
   - Check for proper error handling

3. **Identify Gaps:**
   - Missing files or functions
   - Incomplete implementations
   - Missing error handling
   - Missing tests
   - Behavioral differences from NodeJS

4. **Create Remediation Tasks:**
   For each gap found, create a remediation task with:
   - Task ID (derived from feature ID)
   - Priority (critical, high, medium, low)
   - Detailed description of the gap
   - Specific files to modify/create
   - Implementation prompt for cline
   - Success criteria for validation
   - Test requirements

CRITICAL: The remediation task prompt must:
- Reference the NodeJS implementation file paths
- Explain what functionality needs to be added/matched
- Provide specific technical requirements
- Include comparison criteria for validation

Output ONLY valid JSON with this structure:
{
  "remediation_plan": {
    "phases": [
      {
        "phase": 1,
        "description": "Critical gaps - core functionality",
        "parallel_groups": [...]
      }
    ],
    "total_remediation_tasks": 0,
    "critical_path": []
  },
  "remediation_tasks": [
    {
      "id": "REM-FEAT-XXX-001",
      "feature_id": "FEAT-XXX",
      "feature_name": "Feature Name",
      "priority": "critical|high|medium|low",
      "gap_type": "missing_files|incomplete_implementation|missing_tests|behavioral_difference",
      "description": "Detailed gap description",
      "nodejs_reference": ["cli/src/path/to/file.ts"],
      "golang_target": ["golang-cli/internal/path/file.go"],
      "prompt": "Detailed implementation prompt for cline...",
      "success_criteria": ["criteria 1", "criteria 2"],
      "test_requirements": ["test 1", "test 2"],
      "validation_command": "go test ./internal/path/...",
      "execution_phase": 1,
      "parallel_group": 1
    }
  ],
  "metadata": {
    "source_execution_manifest": "...",
    "source_prd": "...",
    "generated": "timestamp",
    "evaluated_by": "cline-feature-evaluator"
  }
}

Save the remediation manifest to the specified path.
EOF
)
    
    log_info "Starting cline session for remediation manifest generation..."
    
    local full_prompt
    full_prompt="Read the execution manifest at ${execution_manifest_file}, the PRD at ${prd_file}, and the NodeJS CLI code in ${NODEJS_CLI_DIR} and ${PROJECT_ROOT}/src/. Compare with the golang CLI at ${GOLANG_CLI_DIR}. ${prompt} Save the remediation manifest to ${remediation_manifest_path}"
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Prompt: $full_prompt"
    fi
    
    local output_file
    output_file=$(mktemp)
    local attempt=1
    
    while [[ $attempt -le $MAX_RETRIES ]]; do
        log_info "Attempt $attempt/$MAX_RETRIES to generate remediation manifest..."
        
        if $CLINE_BIN task -y --json "$full_prompt" > "$output_file" 2>&1; then
            if [[ -f "$remediation_manifest_path" ]]; then
                if validate_json "$remediation_manifest_path"; then
                    log_success "Remediation manifest generated and validated: $remediation_manifest_path"
                    rm -f "$output_file"
                    echo "$remediation_manifest_path"
                    return 0
                else
                    log_warn "Generated manifest has JSON syntax errors, attempting to clean..."
                    local cleaned_manifest="${remediation_manifest_path}.clean"
                    if clean_json_file "$remediation_manifest_path" "$cleaned_manifest"; then
                        mv "$cleaned_manifest" "$remediation_manifest_path"
                        log_success "Remediation manifest cleaned and validated: $remediation_manifest_path"
                        rm -f "$output_file"
                        echo "$remediation_manifest_path"
                        return 0
                    else
                        log_error "Could not clean JSON, will retry..."
                        rm -f "$remediation_manifest_path" "$cleaned_manifest"
                    fi
                fi
            else
                log_warn "Remediation manifest file was not created at expected path, will retry..."
            fi
        else
            log_warn "Cline execution failed on attempt $attempt. Output:"
            cat "$output_file" >&2
        fi
        
        attempt=$((attempt + 1))
        if [[ $attempt -le $MAX_RETRIES ]]; then
            log_info "Waiting $RETRY_DELAY seconds before retry..."
            sleep "$RETRY_DELAY"
        fi
    done
    
    rm -f "$output_file"
    log_error "Failed to generate valid remediation manifest after $MAX_RETRIES attempts"
    exit 1
}

# Validate the remediation manifest
validate_remediation_manifest() {
    local manifest_file="$1"
    
    log_info "Validating remediation manifest: $manifest_file"
    
    if [[ ! -f "$manifest_file" ]]; then
        log_error "Remediation manifest file not found: $manifest_file"
        return 1
    fi
    
    if ! jq -e '.remediation_tasks' "$manifest_file" > /dev/null 2>&1; then
        log_error "Remediation manifest missing required 'remediation_tasks' array"
        return 1
    fi
    
    local task_count
    task_count=$(jq '.remediation_tasks | length' "$manifest_file")
    log_success "Remediation manifest validated: $task_count remediation tasks found"
    
    return 0
}

# Execute remediation for a single task
execute_remediation() {
    local task_id="$1"
    local task_name="$2"
    local remediation_manifest_file="$3"
    local attempt="${4:-1}"
    
    log_info "Executing remediation: $task_id ($task_name) - Attempt $attempt/$MAX_RETRIES"
    
    # Get task details from manifest
    local task_json
    task_json=$(jq -c ".remediation_tasks[] | select(.id == \"$task_id\")" "$remediation_manifest_file")
    
    local golang_target
    local prompt
    local nodejs_ref
    
    golang_target=$(echo "$task_json" | jq -r '.golang_target[0] // empty')
    prompt=$(echo "$task_json" | jq -r '.prompt // empty')
    nodejs_ref=$(echo "$task_json" | jq -r '.nodejs_reference | join(", ") // empty')
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would execute: cline task -y \"$prompt\""
        log_info "[DRY RUN] Target: $golang_target"
        return 0
    fi
    
    # Build comprehensive remediation prompt
    local full_prompt
    full_prompt="REMEDIATION TASK: $task_id - $task_name

OBJECTIVE:
$prompt

REFERENCE IMPLEMENTATION (NodeJS):
Review these files for the reference behavior:
$nodejs_ref

TARGET IMPLEMENTATION (Go):
Modify/create files in: $golang_target

REQUIREMENTS:
1. Match the functionality of the NodeJS reference implementation
2. Follow Go best practices and idiomatic patterns
3. Maintain compatibility with existing golang-cli code
4. Include proper error handling
5. Add or update tests as needed

DO NOT:
- Change the NodeJS implementation
- Break existing golang-cli functionality
- Add unnecessary abstractions

Execute the remediation and ensure all success criteria are met."

    if [[ "$VERBOSE" == true ]]; then
        log_info "Full prompt: $full_prompt"
    fi
    
    local output_file
    output_file=$(mktemp)
    
    # Run cline in background
    $CLINE_BIN task -y --json "$full_prompt" > "$output_file" 2>&1 &
    local cline_pid=$!
    BACKGROUND_PIDS+=($cline_pid)
    
    local exit_code=0
    if ! wait "$cline_pid"; then
        exit_code=$?
        log_error "Cline execution failed for $task_id (exit code: $exit_code)"
        cat "$output_file" >&2
    fi
    
    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
    rm -f "$output_file"
    
    if [[ $exit_code -ne 0 ]]; then
        return $exit_code
    fi
    
    log_success "Remediation $task_id completed"
    return 0
}

# Validate remediation with tests
validate_remediation_tests() {
    local task_id="$1"
    local task_name="$2"
    local remediation_manifest_file="$3"
    
    log_review "Validating remediation with tests: $task_id"
    
    if [[ "$SKIP_TESTS" == true ]]; then
        log_info "Skipping tests (--skip-tests)"
        return 0
    fi
    
    local task_json
    task_json=$(jq -c ".remediation_tasks[] | select(.id == \"$task_id\")" "$remediation_manifest_file")
    
    local validation_command
    local golang_target
    
    validation_command=$(echo "$task_json" | jq -r '.validation_command // empty')
    golang_target=$(echo "$task_json" | jq -r '.golang_target[0] // empty')
    
    if [[ -z "$validation_command" ]]; then
        # Default validation: run go test on target directory
        local target_dir
        target_dir=$(dirname "$golang_target")
        validation_command="cd ${GOLANG_CLI_DIR} && go test ./${target_dir#${GOLANG_CLI_DIR}/}... -v"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run validation: $validation_command"
        return 0
    fi
    
    log_info "Running validation: $validation_command"
    
    local output_file
    output_file=$(mktemp)
    
    if eval "$validation_command" > "$output_file" 2>&1; then
        log_success "Validation passed for $task_id"
        rm -f "$output_file"
        
        # Update manifest with validation status
        update_task_status "$remediation_manifest_file" "$task_id" "validated"
        return 0
    else
        log_error "Validation failed for $task_id"
        cat "$output_file" >&2
        rm -f "$output_file"
        return 1
    fi
}

# Compare NodeJS vs golang outputs
compare_implementations() {
    local task_id="$1"
    local task_name="$2"
    local remediation_manifest_file="$3"
    
    log_compare "Comparing NodeJS vs golang: $task_id"
    
    if [[ "$SKIP_COMPARISON" == true ]]; then
        log_info "Skipping comparison (--skip-comparison)"
        return 0
    fi
    
    local task_json
    task_json=$(jq -c ".remediation_tasks[] | select(.id == \"$task_id\")" "$remediation_manifest_file")
    
    local feature_id
    feature_id=$(echo "$task_json" | jq -r '.feature_id')
    
    # Build comparison prompt
    local comparison_prompt
    comparison_prompt="Compare the NodeJS and golang implementations for feature $feature_id.

NodeJS Implementation:
- Review the TypeScript/React code in cli/src/ and src/core/

Golang Implementation:
- Review the Go code in golang-cli/

Task:
1. Identify the equivalent functionality in both implementations
2. Execute a test scenario in the NodeJS CLI to observe behavior
3. Execute the same scenario in the golang CLI
4. Compare outputs for functional parity
5. Identify any behavioral differences

Return a JSON result:
{
  \"parity_achieved\": true/false,
  \"nodejs_output\": \"description of observed behavior\",
  \"golang_output\": \"description of observed behavior\",
  \"differences\": [\"list of differences\"],
  \"recommendations\": [\"suggested fixes\"]
}"

    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run comparison for $task_id"
        return 0
    fi
    
    local output_file
    output_file=$(mktemp)
    
    # Run cline for comparison
    $CLINE_BIN task -y --json "$comparison_prompt" > "$output_file" 2>&1 &
    local cline_pid=$!
    BACKGROUND_PIDS+=($cline_pid)
    
    if ! wait "$cline_pid"; then
        log_warn "Comparison failed for $task_id"
        rm -f "$output_file"
        BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
        return 1
    fi
    
    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
    
    # Parse comparison result
    local comparison_result
    comparison_result=$(cat "$output_file" 2>/dev/null || echo '{"parity_achieved": false}')
    rm -f "$output_file"
    
    local parity_achieved
    parity_achieved=$(echo "$comparison_result" | jq -r '.parity_achieved // false')
    
    # Update manifest with comparison result
    local temp_manifest
    temp_manifest=$(mktemp)
    
    jq --arg task_id "$task_id" \
       --argjson comparison "$comparison_result" \
       '
       .remediation_tasks = [
           (.remediation_tasks[] | 
            if .id == $task_id then
                . + {
                    "comparison_result": $comparison,
                    "comparison_completed": now | todate
                }
            else
                .
            end
           )
       ]
       ' "$remediation_manifest_file" > "$temp_manifest"
    
    mv "$temp_manifest" "$remediation_manifest_file"
    
    if [[ "$parity_achieved" == "true" ]]; then
        log_success "Parity achieved for $task_id"
        return 0
    else
        log_warn "Parity NOT achieved for $task_id - further remediation needed"
        return 1
    fi
}

# Update task status in manifest
update_task_status() {
    local manifest_file="$1"
    local task_id="$2"
    local status="$3"
    
    local temp_manifest
    temp_manifest=$(mktemp)
    
    jq --arg task_id "$task_id" \
       --arg status "$status" \
       --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
       '
       .remediation_tasks = [
           (.remediation_tasks[] | 
            if .id == $task_id then
                . + {
                    "status": $status,
                    "status_updated_at": $timestamp
                }
            else
                .
            end
           )
       ]
       ' "$manifest_file" > "$temp_manifest"
    
    mv "$temp_manifest" "$manifest_file"
}

# Process all remediation tasks
process_remediations() {
    local remediation_manifest_file="$1"
    
    log_info "Starting remediation processing..."
    
    local tasks_json
    tasks_json=$(jq -c '.remediation_tasks[]' "$remediation_manifest_file" 2>/dev/null || true)
    local total_tasks
    total_tasks=$(echo "$tasks_json" | grep -c '^' || echo "0")
    
    log_info "Total remediation tasks: $total_tasks"
    
    local completed=0
    local failed=0
    local needs_remediation=()
    
    # Filter by phase if specified
    if [[ -n "$PHASE" ]]; then
        log_info "Filtering to phase $PHASE only"
        tasks_json=$(echo "$tasks_json" | jq -c "select(.execution_phase == $PHASE)")
    fi
    
    # Process by phases if available
    if jq -e '.remediation_plan.phases' "$remediation_manifest_file" > /dev/null 2>&1; then
        local num_phases
        num_phases=$(jq '.remediation_plan.phases | length' "$remediation_manifest_file")
        
        log_info "Found $num_phases remediation phases"
        
        for ((phase_idx=0; phase_idx<num_phases; phase_idx++)); do
            local phase_num
            phase_num=$(jq -r ".remediation_plan.phases[$phase_idx].phase" "$remediation_manifest_file")
            
            if [[ -n "$PHASE" && "$phase_num" != "$PHASE" ]]; then
                log_info "Skipping phase $phase_num (filtering to phase $PHASE)"
                continue
            fi
            
            log_step "Phase $phase_num: Processing remediation tasks"
            
            # Get tasks for this phase
            local phase_tasks
            phase_tasks=$(jq -r ".remediation_tasks[] | select(.execution_phase == $phase_num) | .id" "$remediation_manifest_file")
            
            # Process tasks in parallel
            process_parallel_tasks "$phase_tasks" "$remediation_manifest_file"
        done
    else
        # Sequential processing
        while IFS= read -r task_json; do
            local task_id
            local task_name
            task_id=$(echo "$task_json" | jq -r '.id')
            task_name=$(echo "$task_json" | jq -r '.name // .task_name // "Unknown"')
            
            if execute_remediation "$task_id" "$task_name" "$remediation_manifest_file"; then
                # Validate with tests
                if validate_remediation_tests "$task_id" "$task_name" "$remediation_manifest_file"; then
                    # Compare implementations
                    if compare_implementations "$task_id" "$task_name" "$remediation_manifest_file"; then
                        completed=$((completed + 1))
                        update_task_status "$remediation_manifest_file" "$task_id" "completed"
                    else
                        needs_remediation+=("$task_id")
                        update_task_status "$remediation_manifest_file" "$task_id" "comparison_failed"
                    fi
                else
                    update_task_status "$remediation_manifest_file" "$task_id" "tests_failed"
                    failed=$((failed + 1))
                fi
            else
                update_task_status "$remediation_manifest_file" "$task_id" "execution_failed"
                failed=$((failed + 1))
            fi
        done <<< "$tasks_json"
    fi
    
    # Handle tasks that need further remediation
    if [[ ${#needs_remediation[@]} -gt 0 ]]; then
        log_step "Handling tasks needing further remediation..."
        
        for task_id in "${needs_remediation[@]}"; do
            local task_name
            task_name=$(jq -r ".remediation_tasks[] | select(.id == \"$task_id\") | .name // .task_name" "$remediation_manifest_file")
            
            log_warn "Task $task_id needs further remediation"
            
            # Re-execute remediation with updated prompt
            local retry_prompt
            retry_prompt="The comparison for $task_id failed to achieve parity with NodeJS. Review the comparison result in the remediation manifest and fix the behavioral differences. Focus on making the golang implementation match the NodeJS behavior exactly."
            
            if [[ "$DRY_RUN" == false ]]; then
                $CLINE_BIN task -y "$retry_prompt" 2>&1 | head -50
                
                # Re-compare
                if compare_implementations "$task_id" "$task_name" "$remediation_manifest_file"; then
                    log_success "Parity achieved for $task_id on retry"
                    completed=$((completed + 1))
                    update_task_status "$remediation_manifest_file" "$task_id" "completed_after_retry"
                else
                    log_error "Still no parity for $task_id after retry"
                    failed=$((failed + 1))
                fi
            fi
        done
    fi
    
    log_step "Remediation complete: $completed/$total_tasks completed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Process tasks in parallel with semaphore
process_parallel_tasks() {
    local task_ids="$1"
    local remediation_manifest_file="$2"
    
    local pids=()
    local running=0
    
    for task_id in $task_ids; do
        local task_name
        task_name=$(jq -r ".remediation_tasks[] | select(.id == \"$task_id\") | .name // .task_name" "$remediation_manifest_file")
        
        # Execute remediation
        execute_remediation "$task_id" "$task_name" "$remediation_manifest_file" 1 &
        pids+=($!)
        BACKGROUND_PIDS+=($!)
        running=$((running + 1))
        
        # Limit parallelism
        if [[ $running -ge $PARALLELISM ]]; then
            local first_pid=${pids[0]}
            wait "$first_pid" || true
            BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$first_pid})
            pids=("${pids[@]:1}")
            running=$((running - 1))
        fi
    done
    
    # Wait for remaining
    for pid in "${pids[@]}"; do
        wait "$pid" || true
        BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$pid})
    done
    
    log_info "Parallel batch complete"
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Cline Feature Evaluator"
    log_info "======================"
    
    # Check dependencies
    check_cline_cli
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Load or generate remediation manifest
    local remediation_manifest_file
    
    if [[ -n "$REMEDIATION_MANIFEST_FILE" ]]; then
        remediation_manifest_file="$REMEDIATION_MANIFEST_FILE"
        log_info "Using existing remediation manifest: $remediation_manifest_file"
        
        if ! validate_remediation_manifest "$remediation_manifest_file"; then
            log_error "Remediation manifest validation failed"
            exit 1
        fi
    else
        # Validate execution manifest
        if [[ ! -f "$EXECUTION_MANIFEST_FILE" ]]; then
            log_error "Execution manifest file not found: $EXECUTION_MANIFEST_FILE"
            exit 1
        fi
        
        local prd_file
        prd_file=$(get_prd_file "$EXECUTION_MANIFEST_FILE")
        
        log_info "Execution Manifest: $EXECUTION_MANIFEST_FILE"
        log_info "PRD File: $prd_file"
        
        # Build remediation manifest
        if [[ "$DRY_RUN" == true ]]; then
            remediation_manifest_file=$(build_remediation_manifest "$EXECUTION_MANIFEST_FILE" "$prd_file")
            log_info "[DRY RUN] Would validate remediation manifest"
        else
            remediation_manifest_file=$(build_remediation_manifest "$EXECUTION_MANIFEST_FILE" "$prd_file")
            
            if ! validate_remediation_manifest "$remediation_manifest_file"; then
                log_error "Remediation manifest validation failed"
                exit 1
            fi
        fi
    fi
    
    # Process remediations
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would process remediation tasks"
        log_info "[DRY RUN] Would execute: execute_remediation for each task"
        log_info "[DRY RUN] Would validate: validate_remediation_tests for each task"
        log_info "[DRY RUN] Would compare: compare_implementations for each task"
        log_info "[DRY RUN] Would iterate until parity achieved"
        log_success "Dry run complete - no remediations were actually executed"
        exit 0
    fi
    
    if ! process_remediations "$remediation_manifest_file"; then
        log_error "Remediation processing completed with failures"
        log_info "Review the remediation manifest for details: $remediation_manifest_file"
        exit 1
    fi
    
    log_success "Feature evaluation complete! All remediations processed."
    log_info "Remediation manifest: $remediation_manifest_file"
}

# Cleanup function for signal handling
cleanup() {
    local signal=$1
    log_warn "Received signal $signal! Cleaning up..."
    
    if [[ ${#BACKGROUND_PIDS[@]} -gt 0 ]]; then
        for pid in "${BACKGROUND_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                log_info "Terminating cline process: $pid"
                kill -TERM "$pid" 2>/dev/null || true
                sleep 1
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    pkill -f "cline.*remediation" 2>/dev/null || true
    
    log_info "Cleanup complete"
    exit 130
}

# Set up signal handlers
trap 'cleanup SIGINT' SIGINT
trap 'cleanup SIGTERM' SIGTERM

# Run main function
main "$@"