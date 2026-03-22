#!/bin/bash
#
# PRD Feature Orchestrator
#
# This script automates the evaluation of features from an epic manifest using the cline CLI:
# 1. Generates a feature manifest with dependency analysis and execution hierarchy
# 2. Executes cline sessions for each feature in dependency order
# 3. Validates feature outputs and retries if files are missing
# 4. Supports parallel execution where dependencies allow
#
# Usage: ./prd-feature-orchestrator.sh [OPTIONS] <path/to/epic-manifest.json>
#

set -euo pipefail

# Track background processes for cleanup
declare -a BACKGROUND_PIDS=()

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
DRY_RUN=false
MAX_RETRIES=3
RETRY_DELAY=5
FEATURE_MANIFEST_FILE=""
CLINE_BIN="cline"
VERBOSE=false
PARALLELISM=4

# Colors for output (if terminal supports it)
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly CYAN='\033[0;36m'
    readonly NC='\033[0m' # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly CYAN=''
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

# Display usage information
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <path/to/epic-manifest.json>

Automate feature extraction using cline CLI from an epic manifest.

Options:
    -h, --help                 Show this help message
    -d, --dry-run              Show what would be done without executing
    -f, --feature-manifest     Use existing feature manifest file
    -p, --parallelism N        Number of parallel feature executions (default: 4)
    -r, --max-retries N        Maximum retry attempts per feature (default: 3)
    -c, --cline-bin BIN        Path to cline binary (default: cline)
    -v, --verbose              Enable verbose output
    --retry-delay N            Delay between retries in seconds (default: 5)

Examples:
    $(basename "$0") .oxenated/docs/planning/cline-cli-golang-migration-prd-epic-manifest.json
    $(basename "$0") --dry-run path/to/epic-manifest.json
    $(basename "$0") --feature-manifest path/to/feature-manifest.json
    $(basename "$0") --parallelism 2 path/to/epic-manifest.json

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
            -f|--feature-manifest)
                FEATURE_MANIFEST_FILE="$2"
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
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                EPIC_MANIFEST_FILE="$1"
                shift
                ;;
        esac
    done

    if [[ -z "${EPIC_MANIFEST_FILE:-}" ]] && [[ -z "${FEATURE_MANIFEST_FILE}" ]]; then
        log_error "Epic manifest file path is required (unless using --feature-manifest)"
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

# Determine persona from epic ID
get_persona_from_epic_id() {
    local epic_id="$1"
    
    # Format: EPIC-[PERSONA]-[DOMAIN]-[NUMBER]
    # Map persona codes to directory names
    if [[ "$epic_id" =~ ^EPIC-DEV- ]]; then
        echo "dev"
    elif [[ "$epic_id" =~ ^EPIC-AUTO- ]]; then
        echo "automation"
    elif [[ "$epic_id" =~ ^EPIC-ENT- ]]; then
        echo "enterprise"
    elif [[ "$epic_id" =~ ^EPIC-INFRA- ]]; then
        echo "infrastructure"
    else
        # Default to lowercase of the second segment
        echo "$epic_id" | cut -d'-' -f2 | tr '[:upper:]' '[:lower:]'
    fi
}

# Get epic directory path from epic ID and manifest location
get_epic_dir() {
    local epic_id="$1"
    local manifest_file="$2"
    
    local manifest_dir
    local prd_basename
    local persona
    
    manifest_dir=$(dirname "$manifest_file")
    prd_basename=$(basename "$manifest_file" -epic-manifest.json)
    persona=$(get_persona_from_epic_id "$epic_id")
    
    echo "${manifest_dir}/${persona}/epics/${epic_id}"
}

# Get PRD file path from manifest
get_prd_file() {
    local manifest_file="$1"
    
    local prd_file
    
    # First try to get from manifest metadata
    prd_file=$(jq -r '.metadata.source // empty' "$manifest_file" 2>/dev/null || true)
    
    if [[ -n "$prd_file" && -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Infer from manifest filename
    prd_file="${manifest_file%-epic-manifest.json}.md"
    
    # Check if the inferred path exists
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Try looking in the same directory
    local manifest_dir
    manifest_dir=$(dirname "$manifest_file")
    prd_file="${manifest_dir}/$(jq -r '.metadata.source // empty' "$manifest_file" 2>/dev/null || echo '')"
    
    if [[ -f "$prd_file" ]]; then
        echo "$prd_file"
        return 0
    fi
    
    # Return the inferred path even if it doesn't exist (will fail validation later)
    echo "${manifest_file%-epic-manifest.json}.md"
}

# Build feature manifest from epic manifest
build_feature_manifest() {
    local epic_manifest_file="$1"
    local prd_file="$2"
    
    local manifest_dir
    local prd_basename
    local feature_manifest_path
    
    manifest_dir=$(dirname "$epic_manifest_file")
    prd_basename=$(basename "$epic_manifest_file" -epic-manifest.json)
    feature_manifest_path="${manifest_dir}/${prd_basename}-feature-manifest.json"
    
    log_info "Building feature manifest from: $epic_manifest_file"
    log_info "Feature manifest will be written to: $feature_manifest_path"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would analyze epics and features to build feature manifest"
        echo "$feature_manifest_path"
        return 0
    fi
    
    # Create the prompt for feature manifest generation
    local prompt
    prompt=$(cat << 'EOF'
Read the epic manifest and analyze all features across all epics to create a feature manifest JSON file.

For each feature in the epic manifest:
1. Extract the feature ID (format: FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN])
2. Map it to its parent epic
3. Identify dependencies:
   - Cross-epic dependencies (features that depend on other epics being complete)
   - Cross-feature dependencies within the same epic
   - Technical prerequisites (data models, APIs, shared components)

Create an execution hierarchy that identifies:
1. Which features can be executed in parallel (no dependencies)
2. Execution phases based on dependency chains
3. Critical path for minimum viable implementation

The feature manifest JSON structure should be:
{
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
      "required_files": [
        "feature.md"
      ],
      "output_directory": "path/to/feature"
    }
  ],
  "execution_plan": {
    "phases": [
      {
        "phase": 1,
        "parallel_features": ["FEAT-XXX", "FEAT-YYY"],
        "description": "Foundation features"
      }
    ],
    "critical_path": ["FEAT-XXX", "FEAT-YYY"],
    "dependency_graph": {
      "FEAT-XXX": ["FEAT-YYY", "FEAT-ZZZ"]
    }
  },
  "metadata": {
    "source_epic_manifest": "path/to/epic-manifest.json",
    "source_prd": "path/to/prd.md",
    "total_features": 42,
    "total_phases": 8,
    "generated": "timestamp"
  }
}

Save the manifest to the specified path and return the file location.
EOF
)
    
    # Execute cline to generate the feature manifest
    log_info "Starting cline session for feature manifest generation..."
    
    local full_prompt
    full_prompt="Read the epic manifest at ${epic_manifest_file} and the PRD at ${prd_file}. ${prompt} Save the feature manifest to ${feature_manifest_path}"
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Prompt: $full_prompt"
    fi
    
    # Run cline in yolo mode for automation
    local output_file
    output_file=$(mktemp)
    
    if ! $CLINE_BIN -y --json "$full_prompt" > "$output_file" 2>&1; then
        log_error "Failed to generate feature manifest. Output:"
        cat "$output_file" >&2
        rm -f "$output_file"
        exit 1
    fi
    
    rm -f "$output_file"
    
    # Check if manifest was created
    if [[ ! -f "$feature_manifest_path" ]]; then
        log_error "Feature manifest file was not created at expected path: $feature_manifest_path"
        exit 1
    fi
    
    log_success "Feature manifest generated: $feature_manifest_path"
    echo "$feature_manifest_path"
}

# Validate the feature manifest JSON structure
validate_feature_manifest() {
    local manifest_file="$1"
    
    log_info "Validating feature manifest structure: $manifest_file"
    
    if [[ ! -f "$manifest_file" ]]; then
        log_error "Feature manifest file not found: $manifest_file"
        return 1
    fi
    
    # Check required fields using jq
    if ! jq -e '.features' "$manifest_file" > /dev/null 2>&1; then
        log_error "Feature manifest missing required 'features' array"
        return 1
    fi
    
    if ! jq -e '.execution_plan.phases' "$manifest_file" > /dev/null 2>&1; then
        log_warn "Feature manifest missing 'execution_plan.phases', will use dependency-based ordering"
    fi
    
    local feature_count
    feature_count=$(jq '.features | length' "$manifest_file")
    log_success "Feature manifest validated: $feature_count features found"
    
    return 0
}

# Get feature output directory
get_feature_output_dir() {
    local feature_id="$1"
    local feature_manifest_file="$2"
    local epic_manifest_file="$3"
    
    local prd_file
    local parent_epic
    
    prd_file=$(get_prd_file "$epic_manifest_file")
    
    # Get parent epic from feature manifest
    parent_epic=$(jq -r ".features[] | select(.id == \"$feature_id\") | .parent_epic" "$feature_manifest_file" 2>/dev/null || echo '')
    
    if [[ -z "$parent_epic" || "$parent_epic" == "null" ]]; then
        # Try to extract from feature ID
        # Format: FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN]
        # Parent epic: EPIC-[PERSONA]-[DOMAIN]-[NNN]
        parent_epic=$(echo "$feature_id" | sed -E 's/^FEAT-([A-Z]+)-([A-Z]+)-([0-9]+)-.*/EPIC-\1-\2-\3/')
    fi
    
    local epic_dir
    local feature_dir
    
    epic_dir=$(get_epic_dir "$parent_epic" "$epic_manifest_file")
    feature_dir="${epic_dir}/features/${feature_id}"
    
    echo "$feature_dir"
}

# Get epic file path for a feature
get_epic_file_for_feature() {
    local feature_id="$1"
    local feature_manifest_file="$2"
    local epic_manifest_file="$3"
    
    local parent_epic
    local epic_dir
    
    parent_epic=$(jq -r ".features[] | select(.id == \"$feature_id\") | .parent_epic" "$feature_manifest_file" 2>/dev/null || echo '')
    
    if [[ -z "$parent_epic" || "$parent_epic" == "null" ]]; then
        # Try to extract from feature ID
        parent_epic=$(echo "$feature_id" | sed -E 's/^FEAT-([A-Z]+)-([A-Z]+)-([0-9]+)-.*/EPIC-\1-\2-\3/')
    fi
    
    epic_dir=$(get_epic_dir "$parent_epic" "$epic_manifest_file")
    
    echo "${epic_dir}/epic.md"
}

# Validate that feature files were created
validate_feature_output() {
    local feature_id="$1"
    local feature_dir="$2"
    local feature_manifest_file="$3"
    
    log_info "Validating feature output: $feature_id at $feature_dir"
    
    # Get required files from manifest
    local required_files
    required_files=$(jq -r ".features[] | select(.id == \"$feature_id\") | .required_files // [\"feature.md\"]" "$feature_manifest_file")
    
    local missing_files=()
    local has_required=false
    
    # Check for feature.md specifically
    if [[ -f "${feature_dir}/feature.md" ]]; then
        has_required=true
    fi
    
    # Check all files in the required_files array
    if [[ -n "$required_files" && "$required_files" != "null" ]]; then
        while IFS= read -r file; do
            if [[ ! -f "${feature_dir}/${file}" ]]; then
                missing_files+=("$file")
            fi
        done < <(echo "$required_files" | jq -r '.[]')
    fi
    
    if [[ "$has_required" == true && ${#missing_files[@]} -eq 0 ]]; then
        log_success "All required files present for $feature_id"
        return 0
    elif [[ "$has_required" == true ]]; then
        log_warn "Missing some files for $feature_id: ${missing_files[*]}"
        return 0  # feature.md exists, so partial success
    else
        log_warn "Missing required feature.md for $feature_id"
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
    find "$feature_dir" -type f | sed 's|^'"$feature_dir"'/||' | jq -R . | jq -s .
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
        log_warn "Feature manifest file not found, cannot update: $manifest_file"
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
                    "output_files": $files
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
    local feature_manifest_file="$3"
    local epic_manifest_file="$4"
    local attempt="${5:-1}"
    
    log_info "Executing feature: $feature_id ($feature_name) - Attempt $attempt/$MAX_RETRIES"
    
    local feature_dir
    local epic_file
    
    feature_dir=$(get_feature_output_dir "$feature_id" "$feature_manifest_file" "$epic_manifest_file")
    epic_file=$(get_epic_file_for_feature "$feature_id" "$feature_manifest_file" "$epic_manifest_file")
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would execute: cline -y \"/prd-feature-extract.md EPIC: $epic_file FEATURE: $feature_id\""
        return 0
    fi
    
    # Ensure parent directory exists
    mkdir -p "$(dirname "$feature_dir")"
    
    # Build the prompt for feature extraction
    local prompt
    prompt="/prd-feature-extract.md EPIC: $epic_file FEATURE: $feature_id"
    
    log_info "Running cline with prompt: $prompt"
    
    # Execute cline in yolo mode with JSON output
    local output_file
    output_file=$(mktemp)
    
    # Run cline in background so we can track its PID
    $CLINE_BIN -y --json "$prompt" > "$output_file" 2>&1 &
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
    if validate_feature_output "$feature_id" "$feature_dir" "$feature_manifest_file"; then
        # Update manifest with file paths
        update_manifest_with_files "$feature_manifest_file" "$feature_id" "$feature_dir"
        log_success "Feature $feature_id completed successfully"
        return 0
    else
        # Retry logic
        if [[ $attempt -lt $MAX_RETRIES ]]; then
            log_warn "Retrying $feature_id after $RETRY_DELAY seconds..."
            sleep "$RETRY_DELAY"
            clean_feature_directory "$feature_dir"
            execute_feature "$feature_id" "$feature_name" "$feature_manifest_file" "$epic_manifest_file" $((attempt + 1))
            return $?
        else
            log_error "Max retries reached for $feature_id"
            return 1
        fi
    fi
}

# Process all features according to the execution plan
process_features() {
    local feature_manifest_file="$1"
    local epic_manifest_file="$2"
    
    log_info "Starting feature processing..."
    
    # Read features into array (only if file exists)
    local features_json=""
    local total_features=0
    
    if [[ -f "$feature_manifest_file" ]]; then
        features_json=$(jq -c '.features[]' "$feature_manifest_file" 2>/dev/null || true)
        total_features=$(echo "$features_json" | grep -c '^' || echo "0")
    fi
    
    log_info "Total features to process: $total_features"
    
    # Track processed and failed features
    local processed=0
    local failed=0
    # Note: Using simple arrays instead of associative arrays for bash 3.2 compatibility
    local completed_features=""
    
    # Process by phases if available (only if file exists)
    if [[ -f "$feature_manifest_file" ]] && jq -e '.execution_plan.phases' "$feature_manifest_file" > /dev/null 2>&1; then
        local num_phases
        num_phases=$(jq '.execution_plan.phases | length' "$feature_manifest_file")
        
        log_info "Found $num_phases execution phases"
        
        for ((phase=0; phase<num_phases; phase++)); do
            local phase_num
            phase_num=$(jq -r ".execution_plan.phases[$phase].phase" "$feature_manifest_file")
            local parallel_features
            parallel_features=$(jq -r ".execution_plan.phases[$phase].parallel_features[]" "$feature_manifest_file" 2>/dev/null || true)
            
            local features_in_phase
            features_in_phase=$(echo "$parallel_features" | wc -w)
            
            log_step "Phase $phase_num: Processing $features_in_phase features in parallel"
            
            if [[ "$DRY_RUN" == true ]]; then
                for feature_id in $parallel_features; do
                    local feature_name
                    feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$feature_manifest_file")
                    log_info "[DRY RUN] Would process: $feature_id - $feature_name"
                done
                continue
            fi
            
            # Execute features in parallel using background jobs with semaphore
            local pids=()
            local running=0
            
            for feature_id in $parallel_features; do
                local feature_name
                feature_name=$(jq -r ".features[] | select(.id == \"$feature_id\") | .name" "$feature_manifest_file")
                
                # Run in background
                execute_feature "$feature_id" "$feature_name" "$feature_manifest_file" "$epic_manifest_file" 1 &
                pids+=($!)
                BACKGROUND_PIDS+=($!)
                running=$((running + 1))
                
                # Limit parallelism
                if [[ $running -ge $PARALLELISM ]]; then
                    # Wait for at least one job to complete
                    local first_pid=${pids[0]}
                    if wait "$first_pid"; then
                        processed=$((processed + 1))
                    else
                        failed=$((failed + 1))
                    fi
                    # Remove from tracking
                    BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$first_pid})
                    pids=("${pids[@]:1}")
                    running=$((running - 1))
                fi
            done
            
            # Wait for remaining jobs in phase
            for pid in "${pids[@]}"; do
                if wait "$pid"; then
                    processed=$((processed + 1))
                else
                    failed=$((failed + 1))
                fi
                # Remove from tracking
                BACKGROUND_PIDS=(${BACKGROUND_PIDS[@]/$pid})
            done
            
            log_success "Phase $phase_num completed: $processed features processed, $failed failed"
        done
    else
        # Sequential processing
        log_info "No execution plan found, processing sequentially..."
        
        while IFS= read -r feature_json; do
            local feature_id
            local feature_name
            feature_id=$(echo "$feature_json" | jq -r '.id')
            feature_name=$(echo "$feature_json" | jq -r '.name')
            
            if execute_feature "$feature_id" "$feature_name" "$feature_manifest_file" "$epic_manifest_file"; then
                processed=$((processed + 1))
            else
                failed=$((failed + 1))
                log_error "Failed to process feature: $feature_id"
                # Continue with next feature
            fi
        done <<< "$features_json"
    fi
    
    log_step "Feature processing complete: $processed/$total_features successful, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "PRD Feature Orchestrator"
    log_info "========================"
    
    # Check dependencies
    check_cline_cli
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Generate or load feature manifest
    local feature_manifest_file
    local epic_manifest_file
    
    if [[ -n "$FEATURE_MANIFEST_FILE" ]]; then
        feature_manifest_file="$FEATURE_MANIFEST_FILE"
        epic_manifest_file="${EPIC_MANIFEST_FILE:-}"
        
        # Try to find epic manifest from feature manifest
        if [[ -z "$epic_manifest_file" ]]; then
            epic_manifest_file=$(jq -r '.metadata.source_epic_manifest // empty' "$feature_manifest_file" 2>/dev/null || true)
        fi
        
        log_info "Using existing feature manifest: $feature_manifest_file"
        
        if [[ -n "$epic_manifest_file" && -f "$epic_manifest_file" ]]; then
            log_info "Using epic manifest: $epic_manifest_file"
        fi
    else
        epic_manifest_file="$EPIC_MANIFEST_FILE"
        
        # Validate epic manifest exists
        if [[ "$DRY_RUN" == false ]] && [[ ! -f "$epic_manifest_file" ]]; then
            log_error "Epic manifest file not found: $epic_manifest_file"
            exit 1
        fi
        
        # Get PRD file
        local prd_file
        prd_file=$(get_prd_file "$epic_manifest_file")
        
        log_info "Epic Manifest: $epic_manifest_file"
        log_info "PRD File: $prd_file"
        
        # Build feature manifest
        if [[ "$DRY_RUN" == true ]]; then
            feature_manifest_file=$(build_feature_manifest "$epic_manifest_file" "$prd_file")
            # Skip validation since file wasn't created
            log_info "[DRY RUN] Skipping feature manifest validation"
        else
            feature_manifest_file=$(build_feature_manifest "$epic_manifest_file" "$prd_file")
        fi
    fi
    
    # Validate feature manifest (skip in dry-run mode without existing manifest)
    if [[ "$DRY_RUN" == false ]] || [[ -n "$FEATURE_MANIFEST_FILE" ]]; then
        if ! validate_feature_manifest "$feature_manifest_file"; then
            log_error "Feature manifest validation failed"
            exit 1
        fi
    fi
    
    # Ensure we have epic manifest for feature processing
    if [[ -z "$epic_manifest_file" || ! -f "$epic_manifest_file" ]]; then
        # Try to get from feature manifest metadata
        epic_manifest_file=$(jq -r '.metadata.source_epic_manifest // empty' "$feature_manifest_file" 2>/dev/null || true)
        
        if [[ -z "$epic_manifest_file" || ! -f "$epic_manifest_file" ]]; then
            # Infer from feature manifest filename
            epic_manifest_file="${feature_manifest_file%-feature-manifest.json}-epic-manifest.json"
        fi
    fi
    
    # Process all features
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would process features according to execution plan"
        log_info "[DRY RUN] Parallelism: $PARALLELISM"
        log_info "[DRY RUN] Would execute cline for each feature with prompt: /prd-feature-extract.md EPIC: <epic-path> FEATURE: <feature-id>"
        log_success "Dry run complete - no features were actually processed"
        exit 0
    fi
    
    if ! process_features "$feature_manifest_file" "$epic_manifest_file"; then
        log_error "Feature processing completed with failures"
        exit 1
    fi
    
    log_success "Feature orchestration complete!"
}

# Cleanup function for signal handling
cleanup() {
    local signal=$1
    log_warn "Received signal $signal! Cleaning up..."
    
    # Kill any running cline processes
    for pid in "${BACKGROUND_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Terminating cline process: $pid"
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            kill -KILL "$pid" 2>/dev/null || true
        fi
    done
    
    # Kill any cline processes started by this script
    pkill -f "cline.*prd-feature-extract" 2>/dev/null || true
    pkill -f "cline.*feature-manifest" 2>/dev/null || true
    
    log_info "Cleanup complete"
    # Exit with error code for interruption
    exit 130
}

# Set up signal handlers (only for SIGINT and SIGTERM, not EXIT)
trap 'cleanup SIGINT' SIGINT
trap 'cleanup SIGTERM' SIGTERM

# Run main function
main "$@"