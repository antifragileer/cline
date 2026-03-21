#!/bin/bash
#
# PRD Epic Orchestrator
#
# This script automates the evaluation of a PRD using the cline CLI to:
# 1. Generate an epic manifest from the PRD
# 2. Execute cline sessions for each epic in the correct order
# 3. Validate epic outputs and retry if files are missing
#
# Usage: ./prd-epic-orchestrator.sh [OPTIONS] <path/to/prd.md>
#

set -euo pipefail

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
DRY_RUN=false
MAX_RETRIES=3
RETRY_DELAY=5
MANIFEST_FILE=""
CLINE_BIN="cline"
VERBOSE=false

# Colors for output (if terminal supports it)
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m' # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
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

# Display usage information
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <path/to/prd.md>

Automate PRD evaluation using cline CLI to extract and process epics.

Options:
    -h, --help              Show this help message
    -d, --dry-run           Show what would be done without executing
    -m, --manifest FILE     Use existing manifest file instead of generating
    -r, --max-retries N     Maximum retry attempts per epic (default: 3)
    -c, --cline-bin BIN     Path to cline binary (default: cline)
    -v, --verbose           Enable verbose output
    --retry-delay N         Delay between retries in seconds (default: 5)

Examples:
    $(basename "$0") .oxenated/docs/planning/cline-cli-golang-migration-prd.md
    $(basename "$0") --dry-run .oxenated/docs/planning/product_requirements.md
    $(basename "$0") --manifest path/to/epic-manifest.json

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
            -m|--manifest)
                MANIFEST_FILE="$2"
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
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                PRD_FILE="$1"
                shift
                ;;
        esac
    done

    if [[ -z "${PRD_FILE:-}" ]] && [[ -z "${MANIFEST_FILE}" ]]; then
        log_error "PRD file path is required (unless using --manifest)"
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

# Generate the epic manifest from PRD using cline
generate_epic_manifest() {
    local prd_file="$1"
    local prd_dir
    local prd_basename
    local manifest_path
    
    prd_dir=$(dirname "$prd_file")
    prd_basename=$(basename "$prd_file" .md)
    manifest_path="${prd_dir}/${prd_basename}-epic-manifest.json"
    
    log_info "Generating epic manifest from: $prd_file"
    log_info "Manifest will be written to: $manifest_path"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would execute cline session to generate manifest"
        echo "$manifest_path"
        return 0
    fi
    
    # Create the prompt for manifest generation
    local prompt
    prompt=$(cat << 'EOF'
Read the PRD document and create an epic manifest JSON file that defines:
1. All epics found in the PRD with their IDs
2. Dependencies between epics (which epics must be completed before others)
3. Execution order and which epics can be executed in parallel
4. For each epic: ID, name, description, dependencies, persona, domain, and features

The manifest JSON structure should be:
{
  "epics": [
    {
      "id": "EPIC-DEV-CLI-001",
      "name": "Command Line Interface Foundation",
      "description": "...",
      "persona": "Developer User",
      "domain": "CLI",
      "dependencies": [],
      "parallel_group": 1,
      "features": ["FEAT-..."]
    }
  ],
  "execution_plan": {
    "phases": [
      {
        "phase": 1,
        "parallel_epics": ["EPIC-DEV-CLI-001", "EPIC-DEV-UI-002"],
        "description": "Foundation phase"
      }
    ]
  }
}

Determine dependencies based on:
- Technical prerequisites (storage layer needed before UI)
- Logical flow (CLI commands needed before task management)
- Infrastructure dependencies (core integration needed for providers)

Save the manifest to the specified path and return the file location.
EOF
)
    
    # Execute cline to generate the manifest
    log_info "Starting cline session for manifest generation..."
    
    local full_prompt
    full_prompt="Read the PRD at ${prd_file} and ${prompt} Save the manifest to ${manifest_path}"
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Prompt: $full_prompt"
    fi
    
    # Run cline in yolo mode for automation
    if ! $CLINE_BIN -y --json "$full_prompt" > /tmp/manifest_output.json 2>&1; then
        log_error "Failed to generate manifest. Output:"
        cat /tmp/manifest_output.json >&2
        exit 1
    fi
    
    # Check if manifest was created
    if [[ ! -f "$manifest_path" ]]; then
        # Try to extract manifest from JSON output
        if [[ -f /tmp/manifest_output.json ]]; then
            # Check if output contains the manifest path or content
            if grep -q "epic-manifest.json" /tmp/manifest_output.json; then
                log_warn "Manifest path referenced in output, checking..."
            fi
        fi
        
        log_error "Manifest file was not created at expected path: $manifest_path"
        exit 1
    fi
    
    log_success "Epic manifest generated: $manifest_path"
    echo "$manifest_path"
}

# Validate the manifest JSON structure
validate_manifest() {
    local manifest_file="$1"
    
    log_info "Validating manifest structure: $manifest_file"
    
    if [[ ! -f "$manifest_file" ]]; then
        log_error "Manifest file not found: $manifest_file"
        return 1
    fi
    
    # Check required fields using jq
    if ! jq -e '.epics' "$manifest_file" > /dev/null 2>&1; then
        log_error "Manifest missing required 'epics' array"
        return 1
    fi
    
    if ! jq -e '.execution_plan.phases' "$manifest_file" > /dev/null 2>&1; then
        log_warn "Manifest missing 'execution_plan.phases', will use dependency-based ordering"
    fi
    
    local epic_count
    epic_count=$(jq '.epics | length' "$manifest_file")
    log_success "Manifest validated: $epic_count epics found"
    
    return 0
}

# Get the expected output directory for an epic based on its ID
get_epic_output_dir() {
    local epic_id="$1"
    local prd_file="$2"
    local prd_dir
    
    prd_dir=$(dirname "$prd_file")
    
    # Parse epic ID to determine persona directory
    # Format: EPIC-[PERSONA]-[DOMAIN]-[NUMBER]
    local persona=""
    
    if [[ "$epic_id" =~ ^EPIC-DEV- ]]; then
        persona="dev"
    elif [[ "$epic_id" =~ ^EPIC-AUTO- ]]; then
        persona="automation"
    elif [[ "$epic_id" =~ ^EPIC-ENT- ]]; then
        persona="enterprise"
    elif [[ "$epic_id" =~ ^EPIC-INFRA- ]]; then
        persona="infrastructure"
    else
        # Default based on common patterns in the PRD
        persona="general"
    fi
    
    # Create directory structure
    local output_dir="${prd_dir}/${persona}/epics/${epic_id}"
    echo "$output_dir"
}

# Validate that epic files were created
validate_epic_output() {
    local epic_id="$1"
    local epic_dir="$2"
    
    log_info "Validating epic output: $epic_id at $epic_dir"
    
    local required_files=(
        "epic.md"
    )
    
    local missing_files=()
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "${epic_dir}/${file}" ]]; then
            missing_files+=("$file")
        fi
    done
    
    if [[ ${#missing_files[@]} -eq 0 ]]; then
        log_success "All required files present for $epic_id"
        return 0
    else
        log_warn "Missing files for $epic_id: ${missing_files[*]}"
        return 1
    fi
}

# Clean epic directory for retry
clean_epic_directory() {
    local epic_dir="$1"
    
    log_warn "Cleaning epic directory for retry: $epic_dir"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would remove: $epic_dir"
        return 0
    fi
    
    if [[ -d "$epic_dir" ]]; then
        rm -rf "$epic_dir"
        log_info "Removed: $epic_dir"
    fi
}

# Execute cline session for a single epic
execute_epic() {
    local epic_id="$1"
    local epic_name="$2"
    local prd_file="$3"
    local attempt="${4:-1}"
    
    log_info "Executing epic: $epic_id ($epic_name) - Attempt $attempt/$MAX_RETRIES"
    
    local epic_dir
    epic_dir=$(get_epic_output_dir "$epic_id" "$prd_file")
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would execute: cline -y \"/prd-epic-extract.md PRD: $prd_file EPIC: $epic_id\""
        return 0
    fi
    
    # Ensure parent directory exists
    mkdir -p "$(dirname "$epic_dir")"
    
    # Build the prompt for epic extraction
    local prompt
    prompt="/prd-epic-extract.md PRD: $prd_file EPIC: $epic_id"
    
    log_info "Running cline with prompt: $prompt"
    
    # Execute cline in yolo mode with JSON output
    local output_file="/tmp/epic_${epic_id}_output.json"
    if ! $CLINE_BIN -y --json "$prompt" > "$output_file" 2>&1; then
        log_error "Cline execution failed for $epic_id"
        cat "$output_file" >&2
        return 1
    fi
    
    # Validate output
    if validate_epic_output "$epic_id" "$epic_dir"; then
        log_success "Epic $epic_id completed successfully"
        return 0
    else
        # Retry logic
        if [[ $attempt -lt $MAX_RETRIES ]]; then
            log_warn "Retrying $epic_id after $RETRY_DELAY seconds..."
            sleep "$RETRY_DELAY"
            clean_epic_directory "$epic_dir"
            execute_epic "$epic_id" "$epic_name" "$prd_file" $((attempt + 1))
            return $?
        else
            log_error "Max retries reached for $epic_id"
            return 1
        fi
    fi
}

# Get execution order from manifest
get_execution_order() {
    local manifest_file="$1"
    
    # Try to use execution_plan first, otherwise use dependencies
    if jq -e '.execution_plan.phases' "$manifest_file" > /dev/null 2>&1; then
        jq -r '.execution_plan.phases[] | .parallel_epics[]' "$manifest_file"
    else
        # Fall back to epic order with dependency resolution
        jq -r '.epics[] | select(.dependencies | length == 0) | .id' "$manifest_file"
    fi
}

# Process all epics according to the execution plan
process_epics() {
    local manifest_file="$1"
    local prd_file="$2"
    
    # Skip processing in dry-run mode without existing manifest
    if [[ "$DRY_RUN" == true ]] && [[ ! -f "$manifest_file" ]]; then
        log_info "[DRY RUN] Would read manifest and process epics"
        log_info "[DRY RUN] Manifest would be at: $manifest_file"
        return 0
    fi
    
    log_info "Starting epic processing..."
    
    # Read epics into array
    local epics_json
    epics_json=$(jq -c '.epics[]' "$manifest_file")
    
    # Process by phases if available
    if jq -e '.execution_plan.phases' "$manifest_file" > /dev/null 2>&1; then
        local num_phases
        num_phases=$(jq '.execution_plan.phases | length' "$manifest_file")
        
        log_info "Found $num_phases execution phases"
        
        for ((phase=0; phase<num_phases; phase++)); do
            local phase_num
            phase_num=$(jq -r ".execution_plan.phases[$phase].phase" "$manifest_file")
            local parallel_epics
            parallel_epics=$(jq -r ".execution_plan.phases[$phase].parallel_epics[]" "$manifest_file" 2>/dev/null || true)
            
            log_info "Phase $phase_num: Processing $(echo "$parallel_epics" | wc -w) epics in parallel"
            
            if [[ "$DRY_RUN" == true ]]; then
                for epic_id in $parallel_epics; do
                    local epic_name
                    epic_name=$(jq -r ".epics[] | select(.id == \"$epic_id\") | .name" "$manifest_file")
                    log_info "[DRY RUN] Would process: $epic_id - $epic_name"
                done
                continue
            fi
            
            # Execute epics in parallel using background jobs
            local pids=()
            for epic_id in $parallel_epics; do
                local epic_name
                epic_name=$(jq -r ".epics[] | select(.id == \"$epic_id\") | .name" "$manifest_file")
                
                # Run in background
                execute_epic "$epic_id" "$epic_name" "$prd_file" 1 &
                pids+=($!)
            done
            
            # Wait for all parallel jobs to complete
            local failed=0
            for pid in "${pids[@]}"; do
                if ! wait "$pid"; then
                    failed=$((failed + 1))
                fi
            done
            
            if [[ $failed -gt 0 ]]; then
                log_error "Phase $phase_num completed with $failed failures"
                # Continue to next phase but track failure
            else
                log_success "Phase $phase_num completed successfully"
            fi
        done
    else
        # Sequential processing
        log_info "No execution plan found, processing sequentially..."
        
        while IFS= read -r epic_json; do
            local epic_id
            local epic_name
            epic_id=$(echo "$epic_json" | jq -r '.id')
            epic_name=$(echo "$epic_json" | jq -r '.name')
            
            if ! execute_epic "$epic_id" "$epic_name" "$prd_file"; then
                log_error "Failed to process epic: $epic_id"
                # Continue with next epic
            fi
        done <<< "$epics_json"
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "PRD Epic Orchestrator"
    log_info "====================="
    
    # Check dependencies
    check_cline_cli
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Generate or load manifest
    local manifest_file
    if [[ -n "$MANIFEST_FILE" ]]; then
        manifest_file="$MANIFEST_FILE"
        log_info "Using existing manifest: $manifest_file"
    else
        if [[ "$DRY_RUN" == true ]]; then
            # In dry-run mode, just show what would happen
            manifest_file=$(generate_epic_manifest "$PRD_FILE")
            # Skip validation since file wasn't created
            log_info "[DRY RUN] Skipping manifest validation"
        else
            manifest_file=$(generate_epic_manifest "$PRD_FILE")
        fi
    fi
    
    # Validate manifest (skip in dry-run mode without existing manifest)
    if [[ "$DRY_RUN" == false ]] || [[ -n "$MANIFEST_FILE" ]]; then
        if ! validate_manifest "$manifest_file"; then
            log_error "Manifest validation failed"
            exit 1
        fi
    fi
    
    # Get the PRD file path from manifest location if needed
    if [[ -z "${PRD_FILE:-}" ]]; then
        PRD_FILE=$(jq -r '.prd_file // empty' "$manifest_file" 2>/dev/null || true)
        if [[ -z "$PRD_FILE" ]]; then
            # Infer from manifest filename
            PRD_FILE="${manifest_file%-epic-manifest.json}.md"
        fi
    fi
    
    log_info "PRD File: $PRD_FILE"
    log_info "Manifest: $manifest_file"
    
    # Check PRD file exists (skip in dry-run mode)
    if [[ "$DRY_RUN" == false ]] && [[ ! -f "$PRD_FILE" ]]; then
        log_error "PRD file not found: $PRD_FILE"
        exit 1
    fi
    
    # Process all epics
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would process epics according to execution plan"
    fi
    
    process_epics "$manifest_file" "$PRD_FILE"
    
    log_success "Epic orchestration complete!"
}

# Run main function
main "$@"