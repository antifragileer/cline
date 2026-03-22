#!/bin/bash
#
# Cline Feature Manifest Executor
#
# This script automates the execution of features from a feature manifest using the cline CLI:
# 1. Generates an execution hierarchy JSON with prompts and parameters for each feature
# 2. Executes cline sessions for each feature in dependency order with parallel execution support
# 3. Validates feature outputs and retries if files are missing or incomplete
# 4. Reviews completed features for completeness and remediates if needed
#
# Usage: ./cline-feature-manifest-executor.sh [OPTIONS] <path/to/feature-manifest.json>
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

# Code implementation directory - where actual code should be written
# This is separate from the execution manifest location (which is for planning docs)
CODE_BASE_DIR="${PROJECT_ROOT}/golang-cli"

# Default values
DRY_RUN=false
MAX_RETRIES=3
RETRY_DELAY=5
EXECUTION_MANIFEST_FILE=""
CLINE_BIN="cline"
VERBOSE=false
PARALLELISM=4
SKIP_VALIDATION=false
PHASE=""
SKIP_REVIEW=false

# Colors for output (if terminal supports it)
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly CYAN='\033[0;36m'
    readonly MAGENTA='\033[0;35m'
    readonly NC='\033[0m' # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly CYAN=''
    readonly MAGENTA=''
    readonly NC=''
fi

# Logging functions - all go to stderr to avoid interfering with function returns
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

# Display usage information
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <path/to/feature-manifest.json>

Automate feature implementation using cline CLI from a feature manifest.

Options:
    -h, --help                 Show this help message
    -d, --dry-run              Show what would be done without executing
    -e, --execution-manifest   Use existing execution manifest file
    -p, --parallelism N        Number of parallel feature executions (default: 4)
    -r, --max-retries N        Maximum retry attempts per feature (default: 3)
    -c, --cline-bin BIN        Path to cline binary (default: cline)
    -v, --verbose              Enable verbose output
    --retry-delay N            Delay between retries in seconds (default: 5)
    --skip-validation          Skip post-execution validation
    --skip-review              Skip review and remediation phase
    --phase N                  Execute only specific phase number

Examples:
    $(basename "$0") .oxenated/docs/planning/cline-cli-golang-migration-prd-feature-manifest.json
    $(basename "$0") --dry-run path/to/feature-manifest.json
    $(basename "$0") --execution-manifest path/to/execution-manifest.json
    $(basename "$0") --parallelism 2 --phase 1 path/to/feature-manifest.json

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
            -e|--execution-manifest)
                EXECUTION_MANIFEST_FILE="$2"
                shift 2
                ;;
            -p|--parallelism)
                PARALLELISM="$2"
                shift 2
                ;;
            -r|--max-retries)
                MAX_RETRIES="$2"
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
            --retry-delay)
                RETRY_DELAY="$2"
                shift 2
                ;;
            --skip-validation)
                SKIP_VALIDATION=true
                shift
                ;;
            --skip-review)
                SKIP_REVIEW=true
                shift
                ;;
            --phase)
                PHASE="$2"
                shift 2
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                FEATURE_MANIFEST_FILE="$1"
                shift
                ;;
        esac
    done

    if [[ -z "${FEATURE_MANIFEST_FILE:-}" ]] && [[ -z "${EXECUTION_MANIFEST_FILE}" ]]; then
        log_error "Feature manifest file path is required (unless using --execution-manifest)"
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

# Get PRD file path from feature manifest
get_prd_file() {
    local manifest_file="$1"
    
    local prd_file
    
    # First try to get from manifest metadata
    prd_file=$(jq -r '.metadata.source_prd // .metadata.source // empty' "$manifest_file" 2>/dev/null || true)
    
    if [[ -n "$prd_file" && -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Infer from manifest filename
    prd_file="${manifest_file%-feature-manifest.json}.md"
    prd_file="${prd_file%-manifest.json}.md"
    
    # Check if the inferred path exists
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Try looking in the same directory
    local manifest_dir
    manifest_dir=$(dirname "$manifest_file")
    
    # Try to find a .md file with similar name
    local base_name
    base_name=$(basename "$manifest_file" .json | sed 's/-feature-manifest$//' | sed 's/-manifest$//')
    prd_file="${manifest_dir}/${base_name}.md"
    
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Return the inferred path even if it doesn't exist
    echo "$prd_file"
}

# Validate JSON file
validate_json() {
    local file="$1"
    if ! jq empty "$file" 2>/dev/null; then
        return 1
    fi
    return 0
}

# Clean and fix common JSON issues from cline output
clean_json_file() {
    local input_file="$1"
    local output_file="$2"
    
    # Use jq to validate and reformat, or sed to fix common issues if jq fails
    if jq . "$input_file" > "$output_file" 2>/dev/null; then
        return 0
    fi
    
    log_warn "JSON has syntax errors, attempting to clean..."
    
    # Try to fix common issues:
    # 1. Remove control characters
    # 2. Fix broken escape sequences
    # 3. Remove trailing commas
    sed -e 's/[\x00-\x08\x0b\x0c\x0e-\x1f]//g' \
        -e 's/\\n"/"/g' \
        -e 's/,\s*}/}/g' \
        -e 's/,\s*]/]/g' \
        "$input_file" > "$output_file.tmp"
    
    # Try again with jq
    if jq . "$output_file.tmp" > "$output_file" 2>/dev/null; then
        rm -f "$output_file.tmp"
        return 0
    fi
    
    rm -f "$output_file.tmp"
    return 1
}

# Build execution manifest from feature manifest
build_execution_manifest() {
    local feature_manifest_file="$1"
    local prd_file="$2"
    
    local manifest_dir
    local prd_basename
    local execution_manifest_path
    
    manifest_dir=$(dirname "$feature_manifest_file")
    prd_basename=$(basename "$feature_manifest_file" -feature-manifest.json)
    prd_basename="${prd_basename%-manifest.json}"
    execution_manifest_path="${manifest_dir}/${prd_basename}-feature-manifest-execution.json"
    
    log_info "Building execution manifest from: $feature_manifest_file"
    log_info "Execution manifest will be written to: $execution_manifest_path"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would analyze features and build execution manifest"
        echo "$execution_manifest_path"
        return 0
    fi
    
    # Create the prompt for execution manifest generation
    local prompt
    prompt=$(cat << 'EOF'
Read the feature manifest and create a detailed execution manifest JSON file that defines:

For each feature in the feature manifest:
1. Create a detailed implementation prompt that includes:
   - Feature ID and name
   - Full description from the manifest
   - Parent epic context
   - Dependencies that must be completed first
   - Required output files
   - Output directory path
   - Complexity level
   - Specific implementation instructions

2. Create an execution hierarchy that:
   - Respects all feature dependencies
   - Groups features by parallel_group for concurrent execution
   - Orders features by execution_phase
   - Identifies the critical path

3. For each feature, generate:
   - A comprehensive prompt for cline to implement the feature
   - Test requirements and validation criteria
   - Success criteria for completion
   - Any specific files that must be created

CRITICAL INSTRUCTIONS:
- Output ONLY valid JSON
- Do not include markdown code blocks or explanations
- Ensure all strings are properly escaped
- Do not include trailing commas
- The output must be parseable by jq

The execution manifest JSON structure should be:
{
  "execution_plan": {
    "phases": [
      {
        "phase": 1,
        "description": "Foundation features",
        "parallel_groups": [
          {
            "group": 1,
            "features": ["FEAT-XXX", "FEAT-YYY"],
            "can_execute_in_parallel": true
          }
        ]
      }
    ],
    "critical_path": ["FEAT-XXX", "FEAT-YYY", "FEAT-ZZZ"],
    "total_features": 42,
    "estimated_duration_hours": 80
  },
  "features": [
    {
      "id": "FEAT-DEV-CLI-001-CMD-001",
      "name": "Root command structure",
      "description": "Cobra root command with global flags",
      "parent_epic": "EPIC-DEV-CLI-001",
      "parent_epic_name": "Command Line Interface Foundation",
      "persona": "Developer User",
      "domain": "CLI",
      "dependencies": {
        "epics": [],
        "features": []
      },
      "parallel_group": 1,
      "execution_phase": 1,
      "estimated_complexity": "medium",
      "output_directory": "path/to/feature",
      "required_files": ["feature.md"],
      "prompt": "Detailed implementation prompt for cline...",
      "test_requirements": ["Unit tests", "Integration tests"],
      "success_criteria": ["All required files created", "Tests passing"],
      "validation_criteria": ["feature.md exists", "Implementation complete"]
    }
  ],
  "metadata": {
    "source_feature_manifest": "path/to/feature-manifest.json",
    "source_prd": "path/to/prd.md",
    "generated": "timestamp",
    "version": "1.0"
  }
}

Save the execution manifest to the specified path and return the file location.
EOF
)
    
    # Execute cline to generate the execution manifest
    log_info "Starting cline session for execution manifest generation..."
    
    local full_prompt
    full_prompt="Read the feature manifest at ${feature_manifest_file} and the PRD at ${prd_file}. ${prompt} Save the execution manifest to ${execution_manifest_path}"
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Prompt: $full_prompt"
    fi
    
    # Run cline in yolo mode for automation
    local output_file
    output_file=$(mktemp)
    local attempt=1
    
    while [[ $attempt -le $MAX_RETRIES ]]; do
        log_info "Attempt $attempt/$MAX_RETRIES to generate execution manifest..."
        
        if $CLINE_BIN -y --json "$full_prompt" > "$output_file" 2>&1; then
            # Check if manifest was created and is valid JSON
            if [[ -f "$execution_manifest_path" ]]; then
                if validate_json "$execution_manifest_path"; then
                    log_success "Execution manifest generated and validated: $execution_manifest_path"
                    rm -f "$output_file"
                    echo "$execution_manifest_path"
                    return 0
                else
                    log_warn "Generated manifest has JSON syntax errors, attempting to clean..."
                    local cleaned_manifest="${execution_manifest_path}.clean"
                    if clean_json_file "$execution_manifest_path" "$cleaned_manifest"; then
                        mv "$cleaned_manifest" "$execution_manifest_path"
                        log_success "Execution manifest cleaned and validated: $execution_manifest_path"
                        rm -f "$output_file"
                        echo "$execution_manifest_path"
                        return 0
                    else
                        log_error "Could not clean JSON, will retry..."
                        rm -f "$execution_manifest_path" "$cleaned_manifest"
                    fi
                fi
            else
                log_warn "Execution manifest file was not created at expected path, will retry..."
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
    log_error "Failed to generate valid execution manifest after $MAX_RETRIES attempts"
    exit 1
}

# Validate the execution manifest JSON structure
validate_execution_manifest() {
    local manifest_file="$1"
    
    log_info "Validating execution manifest structure: $manifest_file"
    
    if [[ ! -f "$manifest_file" ]]; then
        log_error "Execution manifest file not found: $manifest_file"
        return 1
    fi
    
    # Check required fields using jq
    if ! jq -e '.features' "$manifest_file" > /dev/null 2>&1; then
        log_error "Execution manifest missing required 'features' array"
        return 1
    fi
    
    if ! jq -e '.execution_plan.phases' "$manifest_file" > /dev/null 2>&1; then
        log_warn "Execution manifest missing 'execution_plan.phases', will use dependency-based ordering"
    fi
    
    local feature_count
    feature_count=$(jq '.features | length' "$manifest_file")
    log_success "Execution manifest validated: $feature_count features found"
    
    return 0
}

# Get feature implementation directory within the code base
# Maps features to proper Go package structure (NOT epic/feature organization)
get_feature_output_dir() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    
    # Map features to proper Go package directories based on domain/purpose
    # NOT organized by epic - organized by Go project structure
    
    case "$feature_id" in
        # Storage layer - internal/storage/
        FEAT-INFRA-STORAGE-*)
            echo "${CODE_BASE_DIR}/internal/storage"
            ;;
        # gRPC/Core integration - internal/host/
        FEAT-INFRA-CORE-*)
            echo "${CODE_BASE_DIR}/internal/host"
            ;;
        # API Providers - internal/api/
        FEAT-INFRA-API-*)
            echo "${CODE_BASE_DIR}/internal/api"
            ;;
        # CLI Commands - cmd/cline/ and internal/cli/
        FEAT-DEV-CLI-001-CMD-001|FEAT-DEV-CLI-001-CMD-002|FEAT-DEV-CLI-001-CMD-003|FEAT-DEV-CLI-001-CMD-004)
            echo "${CODE_BASE_DIR}/cmd/cline"
            ;;
        FEAT-DEV-CLI-*)
            echo "${CODE_BASE_DIR}/internal/cli"
            ;;
        # Authentication - internal/auth/
        FEAT-DEV-AUTH-*)
            echo "${CODE_BASE_DIR}/internal/auth"
            ;;
        # TUI components - internal/tui/
        FEAT-DEV-UI-*)
            echo "${CODE_BASE_DIR}/internal/tui"
            ;;
        # Task management - internal/task/
        FEAT-DEV-TASK-*)
            echo "${CODE_BASE_DIR}/internal/task"
            ;;
        # Automation/Scripting modes - internal/mode/
        FEAT-AUTO-MODE-*|FEAT-AUTO-EXEC-*|FEAT-AUTO-OUT-*)
            echo "${CODE_BASE_DIR}/internal/mode"
            ;;
        # Enterprise config - internal/config/
        FEAT-ENT-CONFIG-*)
            echo "${CODE_BASE_DIR}/internal/config"
            ;;
        # Security/Permissions - internal/security/
        FEAT-ENT-SEC-*)
            echo "${CODE_BASE_DIR}/internal/security"
            ;;
        # Audit logging - internal/audit/
        FEAT-ENT-AUDIT-*)
            echo "${CODE_BASE_DIR}/internal/audit"
            ;;
        # Distribution scripts - scripts/
        FEAT-INFRA-DIST-014-BUILD-001|FEAT-INFRA-DIST-014-HOMEBREW-002|FEAT-INFRA-DIST-014-NPM-003)
            echo "${CODE_BASE_DIR}/scripts"
            ;;
        # Independence verification - tests/
        FEAT-INFRA-DIST-014-INDEPENDENCE-001)
            echo "${CODE_BASE_DIR}/tests"
            ;;
        # Default to internal/ for anything else
        *)
            echo "${CODE_BASE_DIR}/internal"
            ;;
    esac
}

# Get parent epic file path for a feature
get_epic_file_for_feature() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    local feature_manifest_file="$3"
    
    local parent_epic
    local epic_file
    
    # Try to get parent epic from execution manifest
    parent_epic=$(jq -r ".features[] | select(.id == \"$feature_id\") | .parent_epic" "$execution_manifest_file" 2>/dev/null || echo '')
    
    if [[ -z "$parent_epic" || "$parent_epic" == "null" ]]; then
        # Try to extract from feature ID
        parent_epic=$(echo "$feature_id" | sed -E 's/^FEAT-([A-Z]+)-([A-Z]+)-([0-9]+)-.*/EPIC-\1-\2-\3/')
    fi
    
    # Construct epic file path
    local manifest_dir
    local persona
    
    manifest_dir=$(dirname "$execution_manifest_file")
    
    if [[ "$parent_epic" =~ ^EPIC-DEV- ]]; then
        persona="dev"
    elif [[ "$parent_epic" =~ ^EPIC-AUTO- ]]; then
        persona="automation"
    elif [[ "$parent_epic" =~ ^EPIC-ENT- ]]; then
        persona="enterprise"
    elif [[ "$parent_epic" =~ ^EPIC-INFRA- ]]; then
        persona="infrastructure"
    else
        persona="general"
    fi
    
    epic_file="${manifest_dir}/${persona}/epics/${parent_epic}/epic.md"
    
    echo "$epic_file"
}

# Get feature prompt from execution manifest
get_feature_prompt() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    
    local prompt
    
    prompt=$(jq -r ".features[] | select(.id == \"$feature_id\") | .prompt" "$execution_manifest_file" 2>/dev/null || echo '')
    
    if [[ -z "$prompt" || "$prompt" == "null" ]]; then
        # Generate default prompt
        local feature_name
        local feature_desc
        
        feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$execution_manifest_file" 2>/dev/null || echo "$feature_id")
        feature_desc=$(jq -r ".features[] | select(.id == \"$feature_id\") | .description" "$execution_manifest_file" 2>/dev/null || echo '')
        
        prompt="Implement feature $feature_id: $feature_name"
        if [[ -n "$feature_desc" && "$feature_desc" != "null" ]]; then
            prompt="$prompt. Description: $feature_desc"
        fi
        prompt="$prompt. Create all required files including feature.md documentation."
    fi
    
    echo "$prompt"
}

# Get required files for a feature
get_required_files() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    
    local required_files
    
    required_files=$(jq -r ".features[] | select(.id == \"$feature_id\") | .required_files // [\"feature.md\"]" "$execution_manifest_file" 2>/dev/null || echo '["feature.md"]')
    
    echo "$required_files"
}

# Get success criteria for a feature
get_success_criteria() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    
    local criteria
    
    criteria=$(jq -r ".features[] | select(.id == \"$feature_id\") | .success_criteria // [\"feature.md exists\"]" "$execution_manifest_file" 2>/dev/null || echo '["feature.md exists"]')
    
    echo "$criteria"
}

# Get test requirements for a feature
get_test_requirements() {
    local feature_id="$1"
    local execution_manifest_file="$2"
    
    local tests
    
    tests=$(jq -r ".features[] | select(.id == \"$feature_id\") | .test_requirements // []" "$execution_manifest_file" 2>/dev/null || echo '[]')
    
    echo "$tests"
}

# Validate that feature implementation files were created
validate_feature_output() {
    local feature_id="$1"
    local feature_dir="$2"
    local execution_manifest_file="$3"
    
    log_info "Validating feature output: $feature_id at $feature_dir"
    
    # Get required files from manifest
    local required_files
    required_files=$(get_required_files "$feature_id" "$execution_manifest_file")
    
    local missing_files=()
    local has_implementation=false
    
    # Check for Go source files (.go) or test files (.go) as evidence of implementation
    if [[ -d "$feature_dir" ]]; then
        if find "$feature_dir" -name "*.go" -type f 2>/dev/null | grep -q .; then
            has_implementation=true
        fi
    fi
    
    # Also check all explicitly required files from the manifest
    if [[ -n "$required_files" && "$required_files" != "null" ]]; then
        while IFS= read -r file; do
            if [[ ! -f "${feature_dir}/${file}" ]]; then
                missing_files+=("$file")
            fi
        done < <(echo "$required_files" | jq -r '.[]')
    fi
    
    if [[ "$has_implementation" == true && ${#missing_files[@]} -eq 0 ]]; then
        log_success "Implementation files present for $feature_id"
        return 0
    elif [[ "$has_implementation" == true ]]; then
        log_warn "Missing some files for $feature_id: ${missing_files[*]}"
        return 0  # Implementation exists, so partial success
    else
        log_warn "No implementation files (.go) found for $feature_id in $feature_dir"
        return 1
    fi
}

# Get list of all files in feature directory
get_feature_files() {
    local feature_dir="$1"
    
    if [[ ! -d "$feature_dir" ]]; then
        echo "[]"
        return
    fi
    
    # Find all files and convert to JSON array
    find "$feature_dir" -type f 2>/dev/null | sed 's|^'"$feature_dir"'/||' | jq -R . | jq -s .
}

# Update manifest with output file paths for a feature
update_manifest_with_files() {
    local manifest_file="$1"
    local feature_id="$2"
    local feature_dir="$3"
    
    log_info "Updating manifest with file paths for $feature_id"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would update manifest with files from: $feature_dir"
        return 0
    fi
    
    if [[ ! -f "$manifest_file" ]]; then
        log_warn "Execution manifest file not found, cannot update: $manifest_file"
        return 1
    fi
    
    # Get list of files in feature directory
    local files_json
    files_json=$(get_feature_files "$feature_dir")
    
    # Update the feature entry with output_files field
    local temp_manifest
    temp_manifest=$(mktemp)
    
    jq --arg feature_id "$feature_id" \
       --arg feature_dir "$feature_dir" \
       --argjson files "$files_json" \
       '
       .features = [
           (.features[] | 
            if .id == $feature_id then
                . + {
                    "output_directory": $feature_dir,
                    "output_files": $files,
                    "execution_status": "completed",
                    "completed_at": now | todate
                }
            else
                .
            end
           )
       ]
       ' "$manifest_file" > "$temp_manifest"
    
    if [[ $? -eq 0 ]]; then
        mv "$temp_manifest" "$manifest_file"
        log_success "Updated manifest with output paths for $feature_id"
    else
        log_error "Failed to update manifest for $feature_id"
        rm -f "$temp_manifest"
        return 1
    fi
}

# Clean feature directory for retry
clean_feature_directory() {
    local feature_dir="$1"
    
    log_warn "Cleaning feature directory for retry: $feature_dir"
    
    # SAFETY CHECK: Never delete files in .oxenated/docs/planning/ (planning docs)
    if [[ "$feature_dir" == *".oxenated/docs/planning"* ]]; then
        log_error "SAFETY: Refusing to delete planning documentation directory: $feature_dir"
        log_error "The script should only delete implementation directories in golang-cli/"
        return 1
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would remove: $feature_dir"
        return 0
    fi
    
    if [[ -d "$feature_dir" ]]; then
        rm -rf "$feature_dir"
        log_info "Removed: $feature_dir"
    fi
}

# Execute cline session for a single feature
execute_feature() {
    local feature_id="$1"
    local feature_name="$2"
    local execution_manifest_file="$3"
    local attempt="${4:-1}"
    
    log_info "Executing feature: $feature_id ($feature_name) - Attempt $attempt/$MAX_RETRIES"
    
    local feature_dir
    local feature_prompt
    
    feature_dir=$(get_feature_output_dir "$feature_id" "$execution_manifest_file")
    feature_prompt=$(get_feature_prompt "$feature_id" "$execution_manifest_file")
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would execute: cline -y \"$feature_prompt\""
        log_info "[DRY RUN] Output directory: $feature_dir"
        return 0
    fi
    
    # Ensure parent directory exists
    mkdir -p "$(dirname "$feature_dir")"
    
    log_info "Running cline for feature: $feature_id"
    log_info "Output directory: $feature_dir"
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Prompt: $feature_prompt"
    fi
    
    # Execute cline in yolo mode with JSON output
    local output_file
    output_file=$(mktemp)
    
    # Create a comprehensive prompt that guides Cline to implement code
    # NOT to rewrite requirements or create feature.md documentation
    local full_prompt
    full_prompt="Implement feature $feature_id: $feature_name in the Go CLI codebase. 
    
IMPORTANT INSTRUCTIONS:
1. $feature_prompt
2. Write all implementation files to: $feature_dir
3. This is CODE IMPLEMENTATION - do NOT create documentation files
4. Do NOT rewrite requirements - implement the actual Go code
5. Create Go source files (.go) with proper package structure
6. Include unit tests (.go files with _test suffix) where appropriate
7. Follow Go best practices and the existing project structure in golang-cli/
8. The code should be organized by functional domain (storage, api, cli, etc.) NOT by epic/feature hierarchy
9. Add code to the existing package at $feature_dir - do NOT create new subdirectories for features"
    
    # Run cline in background so we can track its PID
    $CLINE_BIN -y --json "$full_prompt" > "$output_file" 2>&1 &
    local cline_pid=$!
    BACKGROUND_PIDS+=($cline_pid)
    
    # Wait for cline to complete
    local exit_code=0
    if ! wait "$cline_pid"; then
        exit_code=$?
        log_error "Cline execution failed for $feature_id (exit code: $exit_code)"
        cat "$output_file" >&2
    fi
    
    # Remove from tracking
    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
    rm -f "$output_file"
    
    if [[ $exit_code -ne 0 ]]; then
        return $exit_code
    fi
    
    # Validate output
    if validate_feature_output "$feature_id" "$feature_dir" "$execution_manifest_file"; then
        # Update manifest with file paths
        update_manifest_with_files "$execution_manifest_file" "$feature_id" "$feature_dir"
        log_success "Feature $feature_id completed successfully"
        return 0
    else
        # Retry logic
        if [[ $attempt -lt $MAX_RETRIES ]]; then
            log_warn "Retrying $feature_id after $RETRY_DELAY seconds..."
            sleep "$RETRY_DELAY"
            clean_feature_directory "$feature_dir"
            execute_feature "$feature_id" "$feature_name" "$execution_manifest_file" $((attempt + 1))
            return $?
        else
            log_error "Max retries reached for $feature_id"
            return 1
        fi
    fi
}

# Review a completed feature for completeness
review_feature() {
    local feature_id="$1"
    local feature_name="$2"
    local execution_manifest_file="$3"
    
    log_review "Reviewing feature: $feature_id ($feature_name)"
    
    local feature_dir
    local success_criteria
    local test_requirements
    
    feature_dir=$(get_feature_output_dir "$feature_id" "$execution_manifest_file")
    success_criteria=$(get_success_criteria "$feature_id" "$execution_manifest_file")
    test_requirements=$(get_test_requirements "$feature_id" "$execution_manifest_file")
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would review: $feature_id"
        return 0
    fi
    
    if [[ "$SKIP_REVIEW" == true ]]; then
        log_info "Skipping review for $feature_id (--skip-review)"
        return 0
    fi
    
    # Build review prompt
    local review_prompt
    review_prompt="Review the implementation of feature $feature_id ($feature_name) in directory $feature_dir. "
    review_prompt+="Check if all required files are present and complete. "
    review_prompt+="Success criteria: $success_criteria. "
    review_prompt+="Test requirements: $test_requirements. "
    review_prompt+="If the feature is incomplete, missing files, or tests are failing, identify what needs to be fixed. "
    review_prompt+="Return a JSON response with: { \"complete\": true/false, \"missing_files\": [], \"issues\": [], \"recommendations\": [] }"
    
    local output_file
    output_file=$(mktemp)
    
    # Run cline for review
    $CLINE_BIN -y --json "$review_prompt" > "$output_file" 2>&1 &
    local cline_pid=$!
    BACKGROUND_PIDS+=($cline_pid)
    
    if ! wait "$cline_pid"; then
        log_warn "Review failed for $feature_id, assuming incomplete"
        rm -f "$output_file"
        BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
        return 1
    fi
    
    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
    
    # Parse review result
    local review_result
    review_result=$(cat "$output_file" 2>/dev/null || echo '{"complete": false}')
    rm -f "$output_file"
    
    # Check if complete
    local is_complete
    is_complete=$(echo "$review_result" | jq -r '.complete // false')
    
    if [[ "$is_complete" == "true" ]]; then
        log_success "Feature $feature_id passed review"
        
        # Update manifest with review status
        local temp_manifest
        temp_manifest=$(mktemp)
        
        jq --arg feature_id "$feature_id" \
           --arg review "$review_result" \
           '
           .features = [
               (.features[] | 
                if .id == $feature_id then
                    . + {
                        "review_status": "passed",
                        "review_result": $review
                    }
                else
                    .
                end
               )
           ]
           ' "$execution_manifest_file" > "$temp_manifest"
        
        mv "$temp_manifest" "$execution_manifest_file"
        return 0
    else
        log_warn "Feature $feature_id failed review"
        
        # Update manifest with review status
        local temp_manifest
        temp_manifest=$(mktemp)
        
        jq --arg feature_id "$feature_id" \
           --arg review "$review_result" \
           '
           .features = [
               (.features[] | 
                if .id == $feature_id then
                    . + {
                        "review_status": "failed",
                        "review_result": $review
                    }
                else
                    .
                end
               )
           ]
           ' "$execution_manifest_file" > "$temp_manifest"
        
        mv "$temp_manifest" "$execution_manifest_file"
        return 1
    fi
}

# Remediate a feature that failed review
remediate_feature() {
    local feature_id="$1"
    local feature_name="$2"
    local execution_manifest_file="$3"
    
    log_review "Remediating feature: $feature_id ($feature_name)"
    
    local feature_dir
    local review_result
    
    feature_dir=$(get_feature_output_dir "$feature_id" "$execution_manifest_file")
    review_result=$(jq -r ".features[] | select(.id == \"$feature_id\") | .review_result" "$execution_manifest_file" 2>/dev/null || echo '{}')
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would remediate: $feature_id"
        return 0
    fi
    
    # Build remediation prompt
    local remediation_prompt
    remediation_prompt="The feature $feature_id ($feature_name) in directory $feature_dir needs remediation. "
    remediation_prompt+="Review findings: $review_result. "
    remediation_prompt+="Please fix the identified issues, complete missing files, and ensure all tests pass. "
    remediation_prompt+="Maintain all existing files and only add/modify what's needed to meet the success criteria."
    
    local output_file
    output_file=$(mktemp)
    
    # Run cline for remediation
    $CLINE_BIN -y --json "$remediation_prompt" > "$output_file" 2>&1 &
    local cline_pid=$!
    BACKGROUND_PIDS+=($cline_pid)
    
    local exit_code=0
    if ! wait "$cline_pid"; then
        exit_code=$?
        log_error "Remediation failed for $feature_id"
        cat "$output_file" >&2
    fi
    
    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$cline_pid})
    rm -f "$output_file"
    
    if [[ $exit_code -eq 0 ]]; then
        # Re-review
        log_info "Re-reviewing feature after remediation: $feature_id"
        if review_feature "$feature_id" "$feature_name" "$execution_manifest_file"; then
            log_success "Feature $feature_id remediated successfully"
            return 0
        else
            log_error "Feature $feature_id still failing after remediation"
            return 1
        fi
    else
        return $exit_code
    fi
}

# Process all features according to the execution plan
process_features() {
    local execution_manifest_file="$1"
    
    log_info "Starting feature processing..."
    
    # Read features into array (only if file exists)
    local features_json=""
    local total_features=0
    
    if [[ -f "$execution_manifest_file" ]]; then
        features_json=$(jq -c '.features[]' "$execution_manifest_file" 2>/dev/null || true)
        total_features=$(echo "$features_json" | grep -c '^' || echo "0")
    fi
    
    log_info "Total features to process: $total_features"
    
    # Track processed and failed features
    local processed=0
    local failed=0
    local needs_review=""
    
    # Filter by phase if specified
    local phase_filter=""
    if [[ -n "$PHASE" ]]; then
        phase_filter="select(.execution_phase == $PHASE)"
        log_info "Filtering to phase $PHASE only"
    fi
    
    # Process by phases if available (only if file exists)
    if [[ -f "$execution_manifest_file" ]] && jq -e '.execution_plan.phases' "$execution_manifest_file" > /dev/null 2>&1; then
        local num_phases
        num_phases=$(jq '.execution_plan.phases | length' "$execution_manifest_file")
        
        log_info "Found $num_phases execution phases"
        
        for ((phase_idx=0; phase_idx<num_phases; phase_idx++)); do
            local phase_num
            phase_num=$(jq -r ".execution_plan.phases[$phase_idx].phase" "$execution_manifest_file")
            
            # Skip if filtering by phase and this isn't it
            if [[ -n "$PHASE" && "$phase_num" != "$PHASE" ]]; then
                log_info "Skipping phase $phase_num (filtering to phase $PHASE)"
                continue
            fi
            
            local parallel_groups
            parallel_groups=$(jq -c ".execution_plan.phases[$phase_idx].parallel_groups[]" "$execution_manifest_file" 2>/dev/null || true)
            
            if [[ -z "$parallel_groups" ]]; then
                # Try parallel_features format instead
                local parallel_features
                parallel_features=$(jq -r ".execution_plan.phases[$phase_idx].parallel_features[]" "$execution_manifest_file" 2>/dev/null || true)
                
                if [[ -n "$parallel_features" ]]; then
                    log_step "Phase $phase_num: Processing features in parallel"
                    
                    if [[ "$DRY_RUN" == true ]]; then
                        for feature_id in $parallel_features; do
                            local feature_name
                            feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$execution_manifest_file")
                            log_info "[DRY RUN] Would process: $feature_id - $feature_name"
                        done
                        continue
                    fi
                    
                    # Execute features in parallel with semaphore
                    process_parallel_features "$parallel_features" "$execution_manifest_file"
                fi
            else
                log_step "Phase $phase_num: Processing parallel groups"
                
                local group_idx=0
                while IFS= read -r group_json; do
                    group_idx=$((group_idx + 1))
                    local parallel_features
                    parallel_features=$(echo "$group_json" | jq -r '.features[]' 2>/dev/null || true)
                    
                    if [[ -n "$parallel_features" ]]; then
                        log_info "Processing parallel group $group_idx"
                        
                        if [[ "$DRY_RUN" == true ]]; then
                            for feature_id in $parallel_features; do
                                local feature_name
                                feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$execution_manifest_file")
                                log_info "[DRY RUN] Would process: $feature_id - $feature_name"
                            done
                            continue
                        fi
                        
                        process_parallel_features "$parallel_features" "$execution_manifest_file"
                    fi
                done <<< "$parallel_groups"
            fi
            
            log_success "Phase $phase_num completed"
        done
    else
        # Sequential processing
        log_info "No execution plan found, processing sequentially..."
        
        while IFS= read -r feature_json; do
            local feature_id
            local feature_name
            feature_id=$(echo "$feature_json" | jq -r '.id')
            feature_name=$(echo "$feature_json" | jq -r '.name')
            
            if [[ -n "$PHASE" ]]; then
                local feature_phase
                feature_phase=$(echo "$feature_json" | jq -r '.execution_phase')
                if [[ "$feature_phase" != "$PHASE" ]]; then
                    continue
                fi
            fi
            
            if execute_feature "$feature_id" "$feature_name" "$execution_manifest_file"; then
                processed=$((processed + 1))
                needs_review="$needs_review $feature_id"
            else
                failed=$((failed + 1))
                log_error "Failed to process feature: $feature_id"
            fi
        done <<< "$features_json"
    fi
    
    # Review and remediation phase
    if [[ "$SKIP_REVIEW" == false && -n "$needs_review" && "$DRY_RUN" == false ]]; then
        log_step "Starting review and remediation phase"
        
        for feature_id in $needs_review; do
            local feature_name
            feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$execution_manifest_file")
            
            if ! review_feature "$feature_id" "$feature_name" "$execution_manifest_file"; then
                # Try remediation
                if ! remediate_feature "$feature_id" "$feature_name" "$execution_manifest_file"; then
                    log_error "Feature $feature_id failed review and remediation"
                    failed=$((failed + 1))
                    processed=$((processed - 1))
                fi
            fi
        done
    fi
    
    log_step "Feature processing complete: $processed/$total_features successful, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Process features in parallel with semaphore
process_parallel_features() {
    local feature_ids="$1"
    local execution_manifest_file="$2"
    
    local pids=()
    local running=0
    local completed=0
    local failed_count=0
    
    for feature_id in $feature_ids; do
        local feature_name
        feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$execution_manifest_file")
        
        # Run in background
        execute_feature "$feature_id" "$feature_name" "$execution_manifest_file" 1 &
        pids+=($!)
        BACKGROUND_PIDS+=($!)
        running=$((running + 1))
        
        # Limit parallelism
        if [[ $running -ge $PARALLELISM ]]; then
            # Wait for at least one job to complete
            local first_pid=${pids[0]}
            if wait "$first_pid"; then
                completed=$((completed + 1))
            else
                failed_count=$((failed_count + 1))
            fi
            # Remove from tracking
            BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$first_pid})
            pids=("${pids[@]:1}")
            running=$((running - 1))
        fi
    done
    
    # Wait for remaining jobs
    for pid in "${pids[@]}"; do
        if wait "$pid"; then
            completed=$((completed + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        # Remove from tracking
        BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$pid})
    done
    
    log_info "Parallel batch complete: $completed completed, $failed_count failed"
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Cline Feature Manifest Executor"
    log_info "==============================="
    
    # Check dependencies
    check_cline_cli
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Generate or load execution manifest
    local execution_manifest_file
    local feature_manifest_file
    
    if [[ -n "$EXECUTION_MANIFEST_FILE" ]]; then
        execution_manifest_file="$EXECUTION_MANIFEST_FILE"
        feature_manifest_file="${FEATURE_MANIFEST_FILE:-}"
        
        # Try to find feature manifest from execution manifest
        if [[ -z "$feature_manifest_file" ]]; then
            feature_manifest_file=$(jq -r '.metadata.source_feature_manifest // empty' "$execution_manifest_file" 2>/dev/null || true)
        fi
        
        log_info "Using existing execution manifest: $execution_manifest_file"
        
        if [[ -n "$feature_manifest_file" && -f "$feature_manifest_file" ]]; then
            log_info "Feature manifest: $feature_manifest_file"
        fi
    else
        feature_manifest_file="$FEATURE_MANIFEST_FILE"
        
        # Validate feature manifest exists
        if [[ "$DRY_RUN" == false ]] && [[ ! -f "$feature_manifest_file" ]]; then
            log_error "Feature manifest file not found: $feature_manifest_file"
            exit 1
        fi
        
        # Get PRD file
        local prd_file
        prd_file=$(get_prd_file "$feature_manifest_file")
        
        log_info "Feature Manifest: $feature_manifest_file"
        log_info "PRD File: $prd_file"
        
        # Build execution manifest
        if [[ "$DRY_RUN" == true ]]; then
            execution_manifest_file=$(build_execution_manifest "$feature_manifest_file" "$prd_file")
            # Skip validation since file wasn't created
            log_info "[DRY RUN] Skipping execution manifest validation"
        else
            execution_manifest_file=$(build_execution_manifest "$feature_manifest_file" "$prd_file")
        fi
    fi
    
    # Validate execution manifest (skip in dry-run mode without existing manifest)
    if [[ "$DRY_RUN" == false ]] || [[ -n "$EXECUTION_MANIFEST_FILE" ]]; then
        if ! validate_execution_manifest "$execution_manifest_file"; then
            log_error "Execution manifest validation failed"
            exit 1
        fi
    fi
    
    # Process all features
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would process features according to execution plan"
        log_info "[DRY RUN] Parallelism: $PARALLELISM"
        log_info "[DRY RUN] Would execute cline for each feature with generated prompts"
        log_info "[DRY RUN] Would review and remediate if needed"
        log_success "Dry run complete - no features were actually processed"
        exit 0
    fi
    
    if ! process_features "$execution_manifest_file"; then
        log_error "Feature processing completed with failures"
        exit 1
    fi
    
    log_success "Feature manifest execution complete!"
    log_info "Execution manifest: $execution_manifest_file"
}

# Cleanup function for signal handling
cleanup() {
    local signal=$1
    log_warn "Received signal $signal! Cleaning up..."
    
    # Kill any running cline processes - handle empty array safely
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
    
    # Kill any cline processes started by this script
    pkill -f "cline.*feature.*implement" 2>/dev/null || true
    pkill -f "cline.*execution.*manifest" 2>/dev/null || true
    
    log_info "Cleanup complete"
    # Exit with error code for interruption
    exit 130
}

# Set up signal handlers (only for SIGINT and SIGTERM, not EXIT)
trap 'cleanup SIGINT' SIGINT
trap 'cleanup SIGTERM' SIGTERM

# Run main function
main "$@"