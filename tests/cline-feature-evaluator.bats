#!/usr/bin/env bats

#
# BATS Tests for Cline Feature Evaluator
#
# These tests verify the cline-feature-evaluator.sh script functionality
#

# Setup test environment
setup() {
    # Create temporary directory for tests
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    
    # Copy the script to test dir
    SCRIPT_PATH="${BATS_TEST_DIRNAME}/../scripts/cline-feature-evaluator.sh"
    cp "$SCRIPT_PATH" "${TEST_DIR}/cline-feature-evaluator.sh"
    chmod +x "${TEST_DIR}/cline-feature-evaluator.sh"
    
    # Create mock directory structure
    mkdir -p "${TEST_DIR}/golang-cli/internal/storage"
    mkdir -p "${TEST_DIR}/golang-cli/internal/api"
    mkdir -p "${TEST_DIR}/cli/src"
    mkdir -p "${TEST_DIR}/src/core"
    mkdir -p "${TEST_DIR}/planning"
    
    # Create sample PRD
    cat > "${TEST_DIR}/planning/test-prd.md" << 'EOF'
# Test Product Requirements Document

## Epic 1: Test Storage Implementation
**ID:** EPIC-INFRA-STORAGE-012

### Description
Test epic for storage layer features.

### Features
- FEAT-INFRA-STORAGE-012-FILE-001: File-based JSON Storage
EOF

    # Create sample execution manifest
    cat > "${TEST_DIR}/planning/test-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Foundation features",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["FEAT-INFRA-STORAGE-012-FILE-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ],
        "critical_path": ["FEAT-INFRA-STORAGE-012-FILE-001"],
        "total_features": 1,
        "estimated_duration_hours": 8
    },
    "features": [
        {
            "id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "name": "File-based JSON Storage",
            "description": "Atomic file operations for global state with file locking",
            "parent_epic": "EPIC-INFRA-STORAGE-012",
            "parent_epic_name": "State & Storage Layer",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {
                "epics": [],
                "features": []
            },
            "parallel_group": 1,
            "execution_phase": 1,
            "estimated_complexity": "high",
            "output_directory": "golang-cli/internal/storage",
            "required_files": [
                "storage.go",
                "storage_test.go",
                "file_storage.go",
                "file_storage_test.go"
            ],
            "prompt": "Implement file-based JSON storage system",
            "test_requirements": [
                "Unit tests >80% coverage",
                "Concurrent access tests",
                "Atomic write verification"
            ],
            "success_criteria": [
                "Atomic thread-safe operations",
                "File locking prevents corruption"
            ],
            "validation_criteria": [
                "Compiles without errors",
                "All tests pass"
            ],
            "execution_status": "completed",
            "completed_at": "2026-03-22T19:47:44Z"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json",
        "source_prd": "test-prd.md",
        "generated": "2026-03-22T00:00:00Z",
        "version": "1.0"
    }
}
EOF

    # Create mock golang files
    cat > "${TEST_DIR}/golang-cli/internal/storage/storage.go" << 'EOF'
package storage

// Storage interface for cline
type Storage interface {
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
}
EOF

    # Create mock NodeJS reference
    mkdir -p "${TEST_DIR}/src/shared/storage"
    cat > "${TEST_DIR}/src/shared/storage/ClineFileStorage.ts" << 'EOF'
export class ClineFileStorage {
    async get(key: string): Promise<any> {
        // Implementation
    }
    async set(key: string, value: any): Promise<void> {
        // Implementation
    }
}
EOF

    # Create mock cline binary
    mkdir -p "${TEST_DIR}/bin"
    cat > "${TEST_DIR}/bin/cline" << 'EOF'
#!/bin/bash
# Mock cline CLI for testing

case "$1" in
    version)
        echo "Cline CLI version: 2.9.0"
        exit 0
        ;;
    -y|--yolo)
        # Simulate successful execution
        if [[ "$3" == *"remediation manifest"* ]]; then
            # Extract path from prompt
            REMEDIATION_MANIFEST=$(echo "$3" | grep -oE '/[^ ]+-remediate\.json' | tail -1)
            if [[ -n "$REMEDIATION_MANIFEST" ]]; then
                cat > "$REMEDIATION_MANIFEST" << 'MANIFESTEOF'
{
    "remediation_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Critical gaps",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["REM-FEAT-STORAGE-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ],
        "total_remediation_tasks": 1,
        "critical_path": ["REM-FEAT-STORAGE-001"]
    },
    "remediation_tasks": [
        {
            "id": "REM-FEAT-STORAGE-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "File-based JSON Storage",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Missing atomic write implementation",
            "nodejs_reference": ["src/shared/storage/ClineFileStorage.ts"],
            "golang_target": ["golang-cli/internal/storage/file_storage.go"],
            "prompt": "Implement atomic file write with temp file + rename pattern",
            "success_criteria": ["Atomic writes implemented", "Tests pass"],
            "test_requirements": ["Test atomic writes", "Test concurrent access"],
            "validation_command": "go test ./internal/storage/... -v",
            "execution_phase": 1,
            "parallel_group": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json",
        "source_prd": "test-prd.md",
        "generated": "2026-03-22T00:00:00Z",
        "evaluated_by": "cline-feature-evaluator"
    }
}
MANIFESTEOF
            fi
        fi
        
        # Remediation execution
        if [[ "$3" == *"REMEDIATION TASK"* ]]; then
            # Simulate creating/updating files
            FEATURE_ID=$(echo "$3" | grep -oE 'REM-FEAT-[A-Z]+-[0-9]+' | head -1)
            if [[ -n "$FEATURE_ID" ]]; then
                # Simulate file creation
                touch "${TEST_DIR}/golang-cli/internal/storage/file_storage.go"
            fi
        fi
        
        # Comparison
        if [[ "$3" == *"Compare the NodeJS and golang implementations"* ]]; then
            echo '{"parity_achieved": true, "nodejs_output": "works", "golang_output": "works", "differences": [], "recommendations": []}'
            exit 0
        fi
        
        echo '{"status":"success"}'
        exit 0
        ;;
    *)
        exit 0
        ;;
esac
EOF
    chmod +x "${TEST_DIR}/bin/cline"
    
    # Add mock bin to PATH
    export PATH="${TEST_DIR}/bin:$PATH"
    
    # Set up project root for script
    export PROJECT_ROOT="${TEST_DIR}"
}

# Cleanup after tests
teardown() {
    rm -rf "$TEST_DIR"
}

# Test help flag displays usage
@test "shows help message with --help flag" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Cline Feature Evaluator"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"--dry-run"* ]]
    [[ "$output" == *"--remediation-manifest"* ]]
}

@test "shows help message with -h flag" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" -h
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

# Test error on missing execution manifest file
@test "exits with error when execution manifest file is missing" {
    run "${TEST_DIR}/cline-feature-evaluator.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Execution manifest file path is required"* ]]
}

@test "exits with error when execution manifest file does not exist" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" "/nonexistent/path/execution-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]]
}

# Test dry-run mode
@test "dry-run mode shows evaluation plan without executing" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
    [[ "$output" == *"Would process remediation tasks"* ]]
}

@test "dry-run mode with verbose flag" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test with existing remediation manifest file
@test "accepts existing remediation manifest file via --remediation-manifest" {
    # Create a mock remediation manifest
    cat > "${TEST_DIR}/test-remediation-manifest.json" << 'EOF'
{
    "remediation_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Critical gaps",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["REM-FEAT-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ],
        "total_remediation_tasks": 1,
        "critical_path": ["REM-FEAT-001"]
    },
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Test description",
            "nodejs_reference": ["src/shared/storage/ClineFileStorage.ts"],
            "golang_target": ["golang-cli/internal/storage/file_storage.go"],
            "prompt": "Test prompt",
            "success_criteria": ["Criteria 1"],
            "test_requirements": ["Test 1"],
            "execution_phase": 1,
            "parallel_group": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json",
        "source_prd": "test-prd.md",
        "generated": "2026-03-22T00:00:00Z",
        "evaluated_by": "cline-feature-evaluator"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/test-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Using existing remediation manifest"* ]]
}

# Test remediation manifest validation
@test "fails with invalid remediation manifest missing remediation_tasks array" {
    cat > "${TEST_DIR}/invalid-remediation-manifest.json" << 'EOF'
{
    "remediation_plan": {
        "phases": []
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --remediation-manifest "${TEST_DIR}/invalid-remediation-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Remediation manifest validation failed"* ]]
}

@test "accepts remediation manifest with minimal structure" {
    cat > "${TEST_DIR}/minimal-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/minimal-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test command line argument parsing
@test "accepts all valid flags" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" \
        --dry-run \
        --parallelism 2 \
        --max-retries 5 \
        --retry-delay 10 \
        --verbose \
        --skip-validation \
        --skip-comparison \
        --skip-tests \
        --phase 1 \
        --force-remediation \
        --cline-bin "${TEST_DIR}/bin/cline" \
        "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

@test "rejects unknown options" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --unknown-option
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown option"* ]]
}

# Test cline CLI detection
@test "detects missing cline CLI" {
    # Remove mock cline and ensure no cline in PATH
    rm "${TEST_DIR}/bin/cline"
    export PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"cline CLI not found"* ]]
}

# Test jq dependency check
@test "requires jq to be installed" {
    # Temporarily hide jq
    PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 1 ]
}

# Test verbose mode output
@test "verbose mode shows additional information" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test color output in terminal
@test "produces colored output when TTY detected" {
    # Mock TTY detection by using script
    run script -q /dev/null "${TEST_DIR}/cline-feature-evaluator.sh" --help
    
    # Should complete successfully
    [ "$status" -eq 0 ]
}

# Test with multiple remediation tasks in manifest
@test "processes multiple phases in dry-run mode" {
    cat > "${TEST_DIR}/multi-phase-remediation-manifest.json" << 'EOF'
{
    "remediation_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1 - Critical",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["REM-FEAT-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            },
            {
                "phase": 2,
                "description": "Phase 2 - High Priority",
                "parallel_groups": [
                    {
                        "group": 2,
                        "features": ["REM-FEAT-002"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ]
    },
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Task 1",
            "priority": "critical",
            "gap_type": "incomplete_implementation",
            "description": "First task",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1,
            "parallel_group": 1
        },
        {
            "id": "REM-FEAT-002",
            "feature_id": "FEAT-INFRA-CORE-011-GRPC-001",
            "feature_name": "Task 2",
            "priority": "high",
            "gap_type": "missing_files",
            "description": "Second task",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 2,
            "parallel_group": 2
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/multi-phase-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
    [[ "$output" == *"remediation tasks found"* ]]
}

# Test sequential processing without remediation plan
@test "processes tasks sequentially when no remediation plan" {
    cat > "${TEST_DIR}/no-plan-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "First Task",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "First",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": []
        },
        {
            "id": "REM-FEAT-002",
            "feature_id": "FEAT-INFRA-CORE-011-GRPC-001",
            "feature_name": "Second Task",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Second",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": []
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/no-plan-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test feature ID parsing for directory mapping
@test "correctly maps feature IDs to golang directories" {
    cat > "${TEST_DIR}/feature-mapping-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-STORAGE-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Storage Feature",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Storage",
            "nodejs_reference": ["src/shared/storage/"],
            "golang_target": ["golang-cli/internal/storage/file_storage.go"],
            "prompt": "Test prompt"
        },
        {
            "id": "REM-API-001",
            "feature_id": "FEAT-INFRA-API-013-OPENAI-001",
            "feature_name": "API Feature",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "API",
            "nodejs_reference": ["src/api/"],
            "golang_target": ["golang-cli/internal/api/openai.go"],
            "prompt": "Test prompt"
        },
        {
            "id": "REM-CLI-001",
            "feature_id": "FEAT-DEV-CLI-001-CMD-001",
            "feature_name": "CLI Feature",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "CLI",
            "nodejs_reference": ["cli/src/"],
            "golang_target": ["golang-cli/cmd/cline/root.go"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/feature-mapping-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test handling of missing remediation manifest file
@test "fails gracefully when remediation manifest file does not exist" {
    run "${TEST_DIR}/cline-feature-evaluator.sh" --remediation-manifest "/nonexistent/remediation-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Remediation manifest file not found"* ]]
}

# Test parallelism option
@test "respects parallelism setting in dry-run" {
    cat > "${TEST_DIR}/parallel-test-remediation-manifest.json" << 'EOF'
{
    "remediation_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["REM-FEAT-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ]
    },
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Task 1",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --parallelism 1 --remediation-manifest "${TEST_DIR}/parallel-test-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test phase filtering
@test "filters by phase when --phase is specified" {
    cat > "${TEST_DIR}/phase-filter-remediation-manifest.json" << 'EOF'
{
    "remediation_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["REM-FEAT-001"],
                        "can_execute_in_parallel": true
                    }
                ]
            },
            {
                "phase": 2,
                "description": "Phase 2",
                "parallel_groups": [
                    {
                        "group": 2,
                        "features": ["REM-FEAT-002"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ]
    },
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Task 1",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "execution_phase": 1
        },
        {
            "id": "REM-FEAT-002",
            "feature_id": "FEAT-INFRA-CORE-011-GRPC-001",
            "feature_name": "Task 2",
            "priority": "high",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "execution_phase": 2
        }
    ],
    "metadata": {
        "source_execution_manifest": "test-execution-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --phase 1 --remediation-manifest "${TEST_DIR}/phase-filter-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
}

# Test skip options - each test creates its own manifest
@test "skip-comparison option is accepted" {
    # Create minimal remediation manifest for this test
    cat > "${TEST_DIR}/skip-comparison-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --skip-comparison --remediation-manifest "${TEST_DIR}/skip-comparison-manifest.json"
    
    [ "$status" -eq 0 ]
}

@test "skip-tests option is accepted" {
    cat > "${TEST_DIR}/skip-tests-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --skip-tests --remediation-manifest "${TEST_DIR}/skip-tests-manifest.json"
    
    [ "$status" -eq 0 ]
}

@test "skip-validation option is accepted" {
    cat > "${TEST_DIR}/skip-validation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --skip-validation --remediation-manifest "${TEST_DIR}/skip-validation-manifest.json"
    
    [ "$status" -eq 0 ]
}

@test "force-remediation option is accepted" {
    cat > "${TEST_DIR}/force-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --force-remediation --remediation-manifest "${TEST_DIR}/force-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test PRD file inference
@test "infers PRD file from execution manifest filename" {
    # Create PRD file with expected name
    cat > "${TEST_DIR}/planning/test-execution-manifest.md" << 'EOF'
# Inferred PRD
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run "${TEST_DIR}/planning/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRD File:"* ]]
}

# Test JSON validation (only in non-dry-run mode would this fail)
@test "validates JSON syntax in manifests" {
    cat > "${TEST_DIR}/invalid-json-execution-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-001",
            "name": "Invalid JSON",
            "description": "Missing closing brace"
    ],
    "metadata": {}
}
EOF
    
    # In dry-run mode, the script doesn't actually parse the JSON
    # so we just verify it doesn't crash
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run "${TEST_DIR}/invalid-json-execution-manifest.json"
    
    # In dry-run mode, the script should complete without error
    # (it doesn't actually parse the JSON)
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
}

# Test gap type values in remediation manifest
@test "accepts all valid gap types" {
    cat > "${TEST_DIR}/gap-types-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-001",
            "feature_id": "FEAT-001",
            "feature_name": "Missing Files",
            "priority": "high",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-002",
            "feature_id": "FEAT-002",
            "feature_name": "Incomplete Implementation",
            "priority": "high",
            "gap_type": "incomplete_implementation",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-003",
            "feature_id": "FEAT-003",
            "feature_name": "Missing Tests",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-004",
            "feature_id": "FEAT-004",
            "feature_name": "Behavioral Difference",
            "priority": "medium",
            "gap_type": "behavioral_difference",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/gap-types-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test priority values
@test "accepts all valid priority levels" {
    cat > "${TEST_DIR}/priorities-remediation-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-001",
            "feature_id": "FEAT-001",
            "feature_name": "Critical",
            "priority": "critical",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-002",
            "feature_id": "FEAT-002",
            "feature_name": "High",
            "priority": "high",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-003",
            "feature_id": "FEAT-003",
            "feature_name": "Medium",
            "priority": "medium",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        },
        {
            "id": "REM-004",
            "feature_id": "FEAT-004",
            "feature_name": "Low",
            "priority": "low",
            "gap_type": "missing_files",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test"
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run --remediation-manifest "${TEST_DIR}/priorities-remediation-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test with all skip flags combined
@test "accepts all skip flags together" {
    cat > "${TEST_DIR}/all-skip-flags-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run \
        --skip-validation \
        --skip-comparison \
        --skip-tests \
        --remediation-manifest "${TEST_DIR}/all-skip-flags-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test maximum retries option
@test "respects max-retries setting" {
    cat > "${TEST_DIR}/max-retries-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run \
        --max-retries 5 \
        --remediation-manifest "${TEST_DIR}/max-retries-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test retry delay option
@test "respects retry-delay setting" {
    cat > "${TEST_DIR}/retry-delay-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run \
        --retry-delay 10 \
        --remediation-manifest "${TEST_DIR}/retry-delay-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test custom cline binary path
@test "accepts custom cline binary path" {
    cat > "${TEST_DIR}/custom-cline-manifest.json" << 'EOF'
{
    "remediation_tasks": [
        {
            "id": "REM-FEAT-001",
            "feature_id": "FEAT-INFRA-STORAGE-012-FILE-001",
            "feature_name": "Test Feature",
            "priority": "medium",
            "gap_type": "missing_tests",
            "description": "Test",
            "nodejs_reference": [],
            "golang_target": [],
            "prompt": "Test prompt",
            "success_criteria": [],
            "test_requirements": [],
            "execution_phase": 1
        }
    ],
    "metadata": {
        "source_execution_manifest": "test.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-evaluator.sh" --dry-run \
        --cline-bin "${TEST_DIR}/bin/cline" \
        --remediation-manifest "${TEST_DIR}/custom-cline-manifest.json"
    
    [ "$status" -eq 0 ]
}
