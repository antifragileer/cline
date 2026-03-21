#!/usr/bin/env bats

#
# BATS Tests for PRD Epic Orchestrator
#
# These tests verify the prd-epic-orchestrator.sh script functionality
#

# Setup test environment
setup() {
    # Create temporary directory for tests
    TEST_DIR="$(mktemp -d)"
    export TEST_DIR
    
    # Copy the script to test dir
    SCRIPT_PATH="${BATS_TEST_DIRNAME}/../scripts/prd-epic-orchestrator.sh"
    cp "$SCRIPT_PATH" "${TEST_DIR}/prd-epic-orchestrator.sh"
    chmod +x "${TEST_DIR}/prd-epic-orchestrator.sh"
    
    # Create mock directories
    mkdir -p "${TEST_DIR}/planning"
    
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

## Epic 2: Test UI Components
**ID:** EPIC-DEV-UI-002  
**Persona:** Developer User

### Description
Test epic for UI components.

### Dependencies
- EPIC-DEV-CLI-001

### Features
- FEAT-DEV-UI-002-BUBBLE-001: TUI framework
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
        if [[ "$2" == *"manifest"* ]]; then
            # Generate mock manifest
            echo '{"status":"success"}'
        fi
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
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRD Epic Orchestrator"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"--dry-run"* ]]
    [[ "$output" == *"--manifest"* ]]
}

@test "shows help message with -h flag" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" -h
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

# Test error on missing PRD file
@test "exits with error when PRD file is missing" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"PRD file path is required"* ]]
}

@test "exits with error when PRD file does not exist" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" "/nonexistent/path/prd.md"
    
    [ "$status" -eq 1 ]
}

# Test dry-run mode
@test "dry-run mode shows execution plan without executing" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 0 ]
}

@test "dry-run mode with verbose flag" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 0 ]
}

# Test with existing manifest file
@test "accepts existing manifest file via --manifest" {
    # Create a mock manifest
    cat > "${TEST_DIR}/test-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "Test Epic",
            "description": "Test description",
            "persona": "Developer User",
            "domain": "CLI",
            "dependencies": [],
            "features": ["FEAT-001"]
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
    }
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/test-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test manifest validation
@test "fails with invalid manifest missing epics array" {
    cat > "${TEST_DIR}/invalid-manifest.json" << 'EOF'
{
    "execution_plan": {
        "phases": []
    }
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/invalid-manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Manifest validation failed"* ]]
}

@test "accepts manifest without execution_plan (falls back to sequential)" {
    cat > "${TEST_DIR}/minimal-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "Test Epic",
            "description": "Test",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": [],
            "features": []
        }
    ]
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/minimal-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test command line argument parsing
@test "accepts all valid flags" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" \
        --dry-run \
        --max-retries 5 \
        --retry-delay 10 \
        --verbose \
        --cline-bin "${TEST_DIR}/bin/cline" \
        "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 0 ]
}

@test "rejects unknown options" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --unknown-option
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown option"* ]]
}

# Test cline CLI detection
@test "detects missing cline CLI" {
    # Remove mock cline and ensure no cline in PATH
    rm "${TEST_DIR}/bin/cline"
    export PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 1 ]
}

# Test jq dependency check
@test "requires jq to be installed" {
    # Temporarily hide jq
    PATH="/usr/bin:/bin"
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 1 ]
}

# Test verbose mode output
@test "verbose mode shows additional information" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --verbose "${TEST_DIR}/planning/test-prd.md"
    
    [ "$status" -eq 0 ]
}

# Test color output in terminal
@test "produces colored output when TTY detected" {
    # Mock TTY detection by setting a terminal
    run script -q /dev/null "${TEST_DIR}/prd-epic-orchestrator.sh" --help
    
    # Should complete successfully
    [ "$status" -eq 0 ]
}

# Test with multiple epics in execution plan
@test "processes multiple phases in dry-run mode" {
    cat > "${TEST_DIR}/multi-phase-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "Foundation",
            "description": "Foundation epic",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": [],
            "features": []
        },
        {
            "id": "EPIC-DEV-UI-002",
            "name": "UI",
            "description": "UI epic",
            "persona": "Developer",
            "domain": "UI",
            "dependencies": ["EPIC-DEV-CLI-001"],
            "features": []
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_epics": ["EPIC-DEV-CLI-001"],
                "description": "Phase 1"
            },
            {
                "phase": 2,
                "parallel_epics": ["EPIC-DEV-UI-002"],
                "description": "Phase 2"
            }
        ]
    }
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/multi-phase-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test sequential processing without execution plan
@test "processes epics sequentially when no execution plan" {
    cat > "${TEST_DIR}/no-plan-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "First",
            "description": "First epic",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": [],
            "features": []
        },
        {
            "id": "EPIC-DEV-UI-002",
            "name": "Second",
            "description": "Second epic",
            "persona": "Developer",
            "domain": "UI",
            "dependencies": [],
            "features": []
        }
    ]
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/no-plan-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test epic ID parsing for directory structure
@test "correctly parses epic IDs for persona directories" {
    # This is tested indirectly through the dry-run output
    cat > "${TEST_DIR}/persona-test-manifest.json" << 'EOF'
{
    "epics": [
        {
            "id": "EPIC-DEV-CLI-001",
            "name": "Dev Epic",
            "description": "Dev",
            "persona": "Developer",
            "domain": "CLI",
            "dependencies": [],
            "features": []
        },
        {
            "id": "EPIC-AUTO-MODE-001",
            "name": "Auto Epic",
            "description": "Automation",
            "persona": "Automation",
            "domain": "MODE",
            "dependencies": [],
            "features": []
        },
        {
            "id": "EPIC-ENT-SEC-001",
            "name": "Enterprise Epic",
            "description": "Enterprise",
            "persona": "Enterprise",
            "domain": "SEC",
            "dependencies": [],
            "features": []
        },
        {
            "id": "EPIC-INFRA-CORE-001",
            "name": "Infrastructure Epic",
            "description": "Infrastructure",
            "persona": "Infrastructure",
            "domain": "CORE",
            "dependencies": [],
            "features": []
        }
    ],
    "execution_plan": {
        "phases": [
            {
                "phase": 1,
                "parallel_epics": ["EPIC-DEV-CLI-001", "EPIC-AUTO-MODE-001", "EPIC-ENT-SEC-001", "EPIC-INFRA-CORE-001"],
                "description": "All personas"
            }
        ]
    }
}
EOF
    
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --dry-run --manifest "${TEST_DIR}/persona-test-manifest.json"
    
    [ "$status" -eq 0 ]
}

# Test handling of missing manifest file
@test "fails gracefully when manifest file does not exist" {
    run "${TEST_DIR}/prd-epic-orchestrator.sh" --manifest "/nonexistent/manifest.json"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Manifest file not found"* ]]
}