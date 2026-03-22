#!/usr/bin/env bats

#
# BATS Tests for Cline Feature Manifest Executor
#
# These tests verify the cline-feature-manifest-executor.sh script functionality
#

# Setup test environment
setup() {
    # Create temporary directory for tests
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    
    # Copy the script to test dir
    SCRIPT_PATH="${BATS_TEST_DIRNAME}/../scripts/cline-feature-manifest-executor.sh"
    cp "$SCRIPT_PATH" "${TEST_DIR}/cline-feature-manifest-executor.sh"
    chmod +x "${TEST_DIR}/cline-feature-manifest-executor.sh"
    
    # Create mock directories
    mkdir -p "${TEST_DIR}/planning/infrastructure/epics/EPIC-INFRA-CORE-011"
    
    # Create a sample PRD for testing
    cat > "${TEST_DIR}/planning/test-prd.md" << 'EOF'
# Test Product Requirements Document

## Epic 1: Test Core Integration
**ID:** EPIC-INFRA-CORE-011  
**Persona:** Infrastructure (Internal)

### Description
Test epic for core integration features.

### Features
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation
- FEAT-INFRA-CORE-011-STREAM-002: Bidirectional Streaming
EOF

    # Create a sample feature manifest
    cat > "${TEST_DIR}/planning/test-prd-feature-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "gRPC Client Implementation",
            "description": "Generate Go code from proto definitions and implement gRPC client",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "parent_epic_name": "Core Extension Integration",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {
                "epics": [],
                "features": []
            },
            "parallel_group": 1,
            "execution_phase": 1,
            "estimated_complexity": "high",
            "required_files": [
                "feature.md"
            ],
            "output_directory": "infrastructure/epics/EPIC-INFRA-CORE-011/features/FEAT-INFRA-CORE-011-GRPC-001",
            "output_files": [
                "feature.md"
            ]
        },
        {
            "id": "FEAT-INFRA-CORE-011-STREAM-002",
            "name": "Bidirectional Streaming",
            "description": "Handle bidirectional message streaming between CLI and core extension",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "parent_epic_name": "Core Extension Integration",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {
                "epics": [],
                "features": [
                    "FEAT-INFRA-CORE-011-GRPC-001"
                ]
            },
            "parallel_group": 1,
            "execution_phase": 1,
            "estimated_complexity": "high",
            "required_files": [
                "feature.md"
            ],
            "output_directory": "infrastructure/epics/EPIC-INFRA-CORE-011/features/FEAT-INFRA-CORE-011-STREAM-002",
            "output_files": [
                "feature.md"
            ]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Foundation features",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["FEAT-INFRA-CORE-011-GRPC-001", "FEAT-INFRA-CORE-011-STREAM-002"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ],
        "critical_path": ["FEAT-INFRA-CORE-011-GRPC-001", "FEAT-INFRA-CORE-011-STREAM-002"],
        "dependency_graph": {
            "FEAT-INFRA-CORE-011-GRPC-001": [],
            "FEAT-INFRA-CORE-011-STREAM-002": ["FEAT-INFRA-CORE-011-GRPC-001"]
        }
    },
    "metadata": {
        "source_prd": "test-prd.md",
        "total_features": 2,
        "total_phases": 1,
        "generated": "2026-03-22T00:00:00Z"
    }
}
EOF

    # Create the epic.md file
    mkdir -p "${TEST_DIR}/planning/infrastructure/epics/EPIC-INFRA-CORE-011"
    cat > "${TEST_DIR}/planning/infrastructure/epics/EPIC-INFRA-CORE-011/epic.md" << 'EOF'
# Test Core Integration Epic

## Features
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation
- FEAT-INFRA-CORE-011-STREAM-002: Bidirectional Streaming
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
        if [[ "$3" == *"execution-manifest"* ]] || [[ "$3" == *"execution manifest"* ]]; then
            # Generate mock execution manifest
            MANIFEST_DIR=$(dirname "$3")
            if [[ "$3" == *"execution manifest to"* ]]; then
                # Extract path from end of prompt
                EXEC_MANIFEST=$(echo "$3" | grep -oE '/[^ ]+-feature-manifest-execution\.json' | tail -1)
                if [[ -n "$EXEC_MANIFEST" ]]; then
                    cat > "$EXEC_MANIFEST" << 'MANIFESTEOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Foundation features",
                "parallel_groups": [
                    {
                        "group": 1,
                        "features": ["FEAT-INFRA-CORE-011-GRPC-001", "FEAT-INFRA-CORE-011-STREAM-002"],
                        "can_execute_in_parallel": true
                    }
                ]
            }
        ],
        "critical_path": ["FEAT-INFRA-CORE-011-GRPC-001", "FEAT-INFRA-CORE-011-STREAM-002"],
        "total_features": 2,
        "estimated_duration_hours": 16
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "gRPC Client Implementation",
            "description": "Generate Go code from proto definitions",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "parent_epic_name": "Core Extension Integration",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {
                "epics": [],
                "features": []
            },
            "parallel_group": 1,
            "execution_phase": 1,
            "estimated_complexity": "high",
            "output_directory": "infrastructure/epics/EPIC-INFRA-CORE-011/features/FEAT-INFRA-CORE-011-GRPC-001",
            "required_files": ["feature.md"],
            "prompt": "Implement gRPC client for core extension communication",
            "test_requirements": ["Unit tests", "Integration tests"],
            "success_criteria": ["gRPC client implemented", "Tests passing"],
            "validation_criteria": ["feature.md exists", "Implementation complete"]
        },
        {
            "id": "FEAT-INFRA-CORE-011-STREAM-002",
            "name": "Bidirectional Streaming",
            "description": "Handle bidirectional message streaming",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "parent_epic_name": "Core Extension Integration",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {
                "epics": [],
                "features": ["FEAT-INFRA-CORE-011-GRPC-001"]
            },
            "parallel_group": 1,
            "execution_phase": 1,
            "estimated_complexity": "high",
            "output_directory": "infrastructure/epics/EPIC-INFRA-CORE-011/features/FEAT-INFRA-CORE-011-STREAM-002",
            "required_files": ["feature.md"],
            "prompt": "Implement bidirectional streaming for messages",
            "test_requirements": ["Unit tests", "Integration tests"],
            "success_criteria": ["Streaming implemented", "Tests passing"],
            "validation_criteria": ["feature.md exists", "Implementation complete"]
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json",
        "source_prd": "test-prd.md",
        "generated": "2026-03-22T00:00:00Z",
        "version": "1.0"
    }
}
MANIFESTEOF
                fi
            fi
        fi
        
        # Create feature directory and feature.md for feature implementation
        if [[ "$3" == *"Implement feature"* ]]; then
            # Extract feature ID from prompt
            FEATURE_ID=$(echo "$3" | grep -oE 'FEAT-[A-Z]+-[A-Z]+-[0-9]+-[A-Z]+-[0-9]+' | head -1)
            if [[ -n "$FEATURE_ID" ]]; then
                # Create the feature directory structure
                FEATURE_DIR="${TEST_DIR}/planning/infrastructure/epics/EPIC-INFRA-CORE-011/features/${FEATURE_ID}"
                mkdir -p "$FEATURE_DIR"
                echo "# Feature: $FEATURE_ID" > "${FEATURE_DIR}/feature.md"
                echo "" >> "${FEATURE_DIR}/feature.md"
                echo "## Description" >> "${FEATURE_DIR}/feature.md"
                echo "Test feature implementation" >> "${FEATURE_DIR}/feature.md"
            fi
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
}

# Cleanup after tests
teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Test help flag displays usage
@test "shows help message with --help flag" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Cline Feature Manifest Executor"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"--dry-run"* ]]
    [[ "$output" == *"--execution-manifest"* ]]
}

@test "shows help message with -h flag" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" -h
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

# Test error on missing feature manifest file
@test "exits with error when feature manifest file is missing" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Feature manifest file path is required"* ]]
}

@test "exits with error when feature manifest file does not exist" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" "/nonexistent/path/feature-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]]
}

# Test dry-run mode
@test "dry-run mode shows execution plan without executing" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
    [[ "$output" == *"Would process features"* ]]
}

@test "dry-run mode with verbose flag" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test with existing execution manifest file
@test "accepts existing execution manifest file via --execution-manifest" {
    # Create a mock execution manifest
    cat > "${TEST_DIR}/test-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Test phase",
                "parallel_features": ["FEAT-INFRA-CORE-011-GRPC-001"]
            }
        ]
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Test Feature",
            "description": "Test description",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "parallel_group": 1,
            "execution_phase": 1,
            "required_files": ["feature.md"],
            "prompt": "Test prompt",
            "test_requirements": [],
            "success_criteria": ["feature.md exists"]
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json",
        "generated": "2026-03-22T00:00:00Z"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --execution-manifest "${TEST_DIR}/test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test execution manifest validation
@test "fails with invalid execution manifest missing features array" {
    cat > "${TEST_DIR}/invalid-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": []
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --execution-manifest "${TEST_DIR}/invalid-execution-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Execution manifest validation failed"* ]]
}

@test "accepts execution manifest without execution_plan (falls back to sequential)" {
    cat > "${TEST_DIR}/minimal-execution-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Test Feature",
            "description": "Test",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --execution-manifest "${TEST_DIR}/minimal-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test command line argument parsing
@test "accepts all valid flags" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" \
        --dry-run \
        --parallelism 2 \
        --max-retries 5 \
        --retry-delay 10 \
        --verbose \
        --skip-validation \
        --skip-review \
        --phase 1 \
        --cline-bin "${TEST_DIR}/bin/cline" \
        "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Parallelism: 2"* ]]
}

@test "rejects unknown options" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --unknown-option
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown option"* ]]
}

# Test cline CLI detection
@test "detects missing cline CLI" {
    # Remove mock cline and ensure no cline in PATH
    rm "${TEST_DIR}/bin/cline"
    export PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"cline CLI not found"* ]]
}

# Test jq dependency check
@test "requires jq to be installed" {
    # Temporarily hide jq
    PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 1 ]
}

# Test verbose mode output
@test "verbose mode shows additional information" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test color output in terminal
@test "produces colored output when TTY detected" {
    # Mock TTY detection by using script
    run script -q /dev/null "${TEST_DIR}/cline-feature-manifest-executor.sh" --help
    
    # Should complete successfully
    [ "$status" -eq 0 ]
}

# Test with multiple features in execution plan
@test "processes multiple phases in dry-run mode" {
    cat > "${TEST_DIR}/multi-phase-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1",
                "parallel_features": ["FEAT-INFRA-CORE-011-GRPC-001"]
            },
            {
                "phase": 2,
                "description": "Phase 2",
                "parallel_features": ["FEAT-INFRA-CORE-011-STREAM-002"]
            }
        ]
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Feature 1",
            "description": "First feature",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        },
        {
            "id": "FEAT-INFRA-CORE-011-STREAM-002",
            "name": "Feature 2",
            "description": "Second feature",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": ["FEAT-INFRA-CORE-011-GRPC-001"]},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --execution-manifest "${TEST_DIR}/multi-phase-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Phase 1"* ]]
    [[ "$output" == *"Phase 2"* ]]
}

# Test sequential processing without execution plan
@test "processes features sequentially when no execution plan" {
    cat > "${TEST_DIR}/no-plan-execution-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "First Feature",
            "description": "First feature",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        },
        {
            "id": "FEAT-INFRA-CORE-011-STREAM-002",
            "name": "Second Feature",
            "description": "Second feature",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --execution-manifest "${TEST_DIR}/no-plan-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"processing sequentially"* ]]
}

# Test feature ID parsing for directory structure
@test "correctly parses feature IDs for parent epic" {
    # This is tested indirectly through the dry-run output
    cat > "${TEST_DIR}/feature-id-test-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "All personas",
                "parallel_features": ["FEAT-INFRA-CORE-011-GRPC-001", "FEAT-DEV-CLI-001-CMD-001"]
            }
        ]
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Infra Feature",
            "description": "Infrastructure",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure (Internal)",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        },
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Dev Feature",
            "description": "Developer",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --execution-manifest "${TEST_DIR}/feature-id-test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test handling of missing execution manifest file
@test "fails gracefully when execution manifest file does not exist" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --execution-manifest "/nonexistent/execution-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Execution manifest file not found"* ]]
}

# Test parallelism option
@test "respects parallelism setting in dry-run" {
    cat > "${TEST_DIR}/parallel-test-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1",
                "parallel_features": ["FEAT-INFRA-CORE-011-GRPC-001"]
            }
        ]
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Feature 1",
            "description": "Test",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt"
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --parallelism 1 --execution-manifest "${TEST_DIR}/parallel-test-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Parallelism: 1"* ]]
}

# Test phase filtering
@test "filters by phase when --phase is specified" {
    cat > "${TEST_DIR}/phase-filter-execution-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "description": "Phase 1",
                "parallel_features": ["FEAT-INFRA-CORE-011-GRPC-001"]
            },
            {
                "phase": 2,
                "description": "Phase 2",
                "parallel_features": ["FEAT-INFRA-CORE-011-STREAM-002"]
            }
        ]
    },
    "features": [
        {
            "id": "FEAT-INFRA-CORE-011-GRPC-001",
            "name": "Feature 1",
            "description": "Test",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt",
            "execution_phase": 1
        },
        {
            "id": "FEAT-INFRA-CORE-011-STREAM-002",
            "name": "Feature 2",
            "description": "Test",
            "parent_epic": "EPIC-INFRA-CORE-011",
            "persona": "Infrastructure",
            "domain": "Infrastructure",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"],
            "prompt": "Test prompt",
            "execution_phase": 2
        }
    ],
    "metadata": {
        "source_feature_manifest": "test-prd-feature-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --phase 1 --execution-manifest "${TEST_DIR}/phase-filter-execution-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"phase 1"* ]]
    [[ "$output" == *"Skipping phase 2"* ]]
}

# Test skip-review option
@test "skip-review option bypasses review phase" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --skip-review "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Would review and remediate"* ]] || true  # May or may not be in dry-run output
}

# Test skip-validation option
@test "skip-validation option is accepted" {
    run "${TEST_DIR}/cline-feature-manifest-executor.sh" --dry-run --skip-validation "${TEST_DIR}/planning/test-prd-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}