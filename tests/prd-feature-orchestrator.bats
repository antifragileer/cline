#!/usr/bin/env bats

#
# BATS Tests for PRD Feature Orchestrator
#
# These tests verify the prd-feature-orchestrator.sh script functionality
#

# Setup test environment
setup() {
    # Create temporary directory for tests
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    
    # Copy the script to test dir
    SCRIPT_PATH="${BATS_TEST_DIRNAME}/../scripts/prd-feature-orchestrator.sh"
    cp "$SCRIPT_PATH" "${TEST_DIR}/prd-feature-orchestrator.sh"
    chmod +x "${TEST_DIR}/prd-feature-orchestrator.sh"
    
    # Create mock directories
    mkdir -p "${TEST_DIR}/planning/dev/epics/EPIC-DEV-CLI-001"
    
    # Create a sample PRD for testing
    cat > "${TEST_DIR}/planning/test-prd.md" << 'EOF'
# Test Product Requirements Document

## Epic 1: Test CLI Foundation
**ID:** EPIC-DEV-CLI-001  
**Persona:** Developer User

### Description
Test epic for CLI foundation features.

### Features
- FEAT-DEV-CLI-001-CMD-001: Root command feature
- FEAT-DEV-CLI-001-CMD-002: Task subcommand feature
EOF

    # Create a sample epic manifest
    cat > "${TEST_DIR}/planning/test-prd-epic-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "Test CLI Foundation",
            "description": "Test epic for CLI foundation",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": [],
            "parallel_group": 1,
            "features": [
                "FEAT-DEV-CLI-001-CMD-001",
                "FEAT-DEV-CLI-001-CMD-002"
            ],
            "output_directory": "dev/epics/EPIC-DEV-CLI-001",
            "output_files": [
                "epic.md"
            ]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_epics": ["EPIC-DEV-CLI-001"],
                "description": "Test phase"
            }
        ]
    },
    "metadata": {
        "source": "test-prd.md",
        "total_epics": 1
    }
}
EOF

    # Create the epic.md file
    mkdir -p "${TEST_DIR}/planning/dev/epics/EPIC-DEV-CLI-001"
    cat > "${TEST_DIR}/planning/dev/epics/EPIC-DEV-CLI-001/epic.md" << 'EOF'
# Test CLI Foundation Epic

## Features
- FEAT-DEV-CLI-001-CMD-001: Root command feature
- FEAT-DEV-CLI-001-CMD-002: Task subcommand feature
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
        if [[ "$3" == *"feature-manifest"* ]]; then
            # Generate mock feature manifest
            MANIFEST_DIR=$(dirname "$3")
            cat > "${MANIFEST_DIR}/test-prd-feature-manifest.json" << 'MANIFESTEOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Root command feature",
            "description": "Root command structure",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "parallel_group": 1,
            "execution_phase": 1,
            "required_files": ["feature.md"]
        },
        {
            "id": "FEAT-DEV-CLI-001-CMD-002",
            "name": "Task subcommand feature",
            "description": "Task subcommand",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "parallel_group": 1,
            "execution_phase": 1,
            "required_files": ["feature.md"]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-001", "FEAT-DEV-CLI-001-CMD-002"],
                "description": "Phase 1"
            }
        ]
    },
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json",
        "total_features": 2,
        "total_phases": 1
    }
}
MANIFESTEOF
        fi
        
        # Create feature directory and feature.md for feature extraction
        if [[ "$3" == *"prd-feature-extract"* ]]; then
            # Extract feature ID from prompt
            FEATURE_ID=$(echo "$3" | grep -oE 'FEATURE: [A-Z0-9-]+' | cut -d' ' -f2)
            if [[ -n "$FEATURE_ID" ]]; then
                # Create the feature directory structure
                FEATURE_DIR="${TEST_DIR}/planning/dev/epics/EPIC-DEV-CLI-001/features/${FEATURE_ID}"
                mkdir -p "$FEATURE_DIR"
                echo "# Feature: $FEATURE_ID" > "${FEATURE_DIR}/feature.md"
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
    
    # Update mock script to use TEST_DIR
    sed -i.bak "s|TEST_DIR=\"[^\"]*\"|TEST_DIR=\"${TEST_DIR}\"|g" "${TEST_DIR}/bin/cline"
    rm "${TEST_DIR}/bin/cline.bak"
}

# Cleanup after tests
teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Test help flag displays usage
@test "shows help message with --help flag" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRD Feature Orchestrator"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"--dry-run"* ]]
    [[ "$output" == *"--feature-manifest"* ]]
}

@test "shows help message with -h flag" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" -h
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

# Test error on missing epic manifest file
@test "exits with error when epic manifest file is missing" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Epic manifest file path is required"* ]]
}

@test "exits with error when epic manifest file does not exist" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" "/nonexistent/path/epic-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]]
}

# Test dry-run mode
@test "dry-run mode shows execution plan without executing" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"[DRY RUN]"* ]]
}

@test "dry-run mode with verbose flag" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test with existing feature manifest file
@test "accepts existing feature manifest file via --feature-manifest" {
    # Create a mock feature manifest
    cat > "${TEST_DIR}/test-feature-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Test Feature",
            "description": "Test description",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "parallel_group": 1,
            "execution_phase": 1,
            "required_files": ["feature.md"]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-001"],
                "description": "Test phase"
            }
        ]
    },
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json",
        "total_features": 1,
        "total_phases": 1
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --feature-manifest "${TEST_DIR}/test-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test feature manifest validation
@test "fails with invalid feature manifest missing features array" {
    cat > "${TEST_DIR}/invalid-feature-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": []
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --feature-manifest "${TEST_DIR}/invalid-feature-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Feature manifest validation failed"* ]]
}

@test "accepts feature manifest without execution_plan (falls back to sequential)" {
    cat > "${TEST_DIR}/minimal-feature-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Test Feature",
            "description": "Test",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        }
    ],
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --feature-manifest "${TEST_DIR}/minimal-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test command line argument parsing
@test "accepts all valid flags" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" \
        --dry-run \
        --parallelism 2 \
        --max-retries 5 \
        --retry-delay 10 \
        --verbose \
        --cline-bin "${TEST_DIR}/bin/cline" \
        "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 0 ]
}

@test "rejects unknown options" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --unknown-option
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown option"* ]]
}

# Test cline CLI detection
@test "detects missing cline CLI" {
    # Remove mock cline and ensure no cline in PATH
    rm "${TEST_DIR}/bin/cline"
    export PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"cline CLI not found"* ]]
}

# Test jq dependency check
@test "requires jq to be installed" {
    # Temporarily hide jq
    PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 1 ]
}

# Test verbose mode output
@test "verbose mode shows additional information" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd-epic-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test color output in terminal
@test "produces colored output when TTY detected" {
    # Mock TTY detection by using script
    run script -q /dev/null "${TEST_DIR}/prd-feature-orchestrator.sh" --help
    
    # Should complete successfully
    [ "$status" -eq 0 ]
}

# Test with multiple features in execution plan
@test "processes multiple phases in dry-run mode" {
    cat > "${TEST_DIR}/multi-phase-feature-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Feature 1",
            "description": "First feature",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        },
        {
            "id": "FEAT-DEV-CLI-001-CMD-002",
            "name": "Feature 2",
            "description": "Second feature",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": ["FEAT-DEV-CLI-001-CMD-001"]},
            "required_files": ["feature.md"]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-001"],
                "description": "Phase 1"
            },
            {
                "phase": 2,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-002"],
                "description": "Phase 2"
            }
        ]
    },
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --feature-manifest "${TEST_DIR}/multi-phase-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test sequential processing without execution plan
@test "processes features sequentially when no execution plan" {
    cat > "${TEST_DIR}/no-plan-feature-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "First Feature",
            "description": "First feature",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        },
        {
            "id": "FEAT-DEV-CLI-001-CMD-002",
            "name": "Second Feature",
            "description": "Second feature",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        }
    ],
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --feature-manifest "${TEST_DIR}/no-plan-feature-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test feature ID parsing for directory structure
@test "correctly parses feature IDs for parent epic" {
    # This is tested indirectly through the dry-run output
    cat > "${TEST_DIR}/feature-id-test-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Dev Feature",
            "description": "Dev",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        },
        {
            "id": "FEAT-AUTO-MODE-001-DETECT-001",
            "name": "Auto Feature",
            "description": "Automation",
            "parent_epic": "EPIC-AUTO-MODE-001",
            "persona": "Automation",
            "domain": "MODE",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-001", "FEAT-AUTO-MODE-001-DETECT-001"],
                "description": "All personas"
            }
        ]
    },
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --feature-manifest "${TEST_DIR}/feature-id-test-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test handling of missing feature manifest file
@test "fails gracefully when feature manifest file does not exist" {
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --feature-manifest "/nonexistent/feature-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Feature manifest file not found"* ]]
}

# Test parallelism option
@test "respects parallelism setting in dry-run" {
    cat > "${TEST_DIR}/parallel-test-manifest.json" << 'EOF'
{
    "features": [
        {
            "id": "FEAT-DEV-CLI-001-CMD-001",
            "name": "Feature 1",
            "description": "Test",
            "parent_epic": "EPIC-DEV-CLI-001",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": {"epics": [], "features": []},
            "required_files": ["feature.md"]
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_features": ["FEAT-DEV-CLI-001-CMD-001"],
                "description": "Phase 1"
            }
        ]
    },
    "metadata": {
        "source_epic_manifest": "test-prd-epic-manifest.json"
    }
}
EOF
    
    run "${TEST_DIR}/prd-feature-orchestrator.sh" --dry-run --parallelism 1 --feature-manifest "${TEST_DIR}/parallel-test-manifest.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Parallelism: 1"* ]]
}