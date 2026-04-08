# TUI Parity Documentation

**Location:** `golang-cli/tests/parity/`  
**Purpose:** Side-by-side testing framework for Node.js CLI vs GoLang CLI  
**Status:** ✅ IMPLEMENTED

---

## Overview

This directory contains the **parity testing framework** that ensures the GoLang CLI achieves 1:1 feature parity with the Node.js 2.x CLI (the reference implementation).

The PRD mandates a **"Dual Testing Mandate"** - all functionality must be tested in BOTH CLI implementations with byte-for-byte output comparison (excluding timestamps/IDs).

---

## Current Status

✅ **FRAMEWORK IMPLEMENTED** - All 8 framework files created and tested.

### Implemented Files

| File | Status | Description |
|------|--------|-------------|
| `parity_suite.go` | ✅ | Main test orchestrator with scenario execution |
| `node_cli_adapter.go` | ✅ | Node.js CLI adapter with auto-discovery |
| `golang_cli_adapter.go` | ✅ | GoLang CLI adapter with build support |
| `output_comparator.go` | ✅ | Output comparison with 10+ normalizers |
| `scenario_registry.go` | ✅ | 25 test scenarios across 7 categories |
| `report_generator.go` | ✅ | Markdown, JSON, HTML, JUnit XML reports |
| `parity_test.go` | ✅ | Main test entry point with CLI flags |
| `baselines/` | ✅ | Directory for expected outputs |

### Test Statistics

- **Total Scenarios:** 25
- **Categories:** help (5), version (3), task (7), history (3), config (2), auth (2), error (3)
- **Framework Tests:** All passing ✅

---

## Gap Analysis Summary

Based on the comprehensive gap analysis:

### Overall Completion: ~35%

| Category | Completion | Status |
|----------|------------|--------|
| CLI Commands | 40% | Task subcommands stubbed |
| TUI Components | 30% | Settings/History not implemented |
| Task Execution | 35% | gRPC streaming partial |
| Authentication | 40% | OAuth flows incomplete |
| Security | 10% | Permissions not implemented |
| Testing | 60% | ✅ Parity framework implemented |

### Critical Gaps (P0 - Must Fix)

1. **Task execution with gRPC integration** - Core functionality incomplete
2. **Approval workflows in TUI** - Currently auto-approve stubs
3. **Message streaming in TUI** - Partial implementation
4. **JSON output mode** - Basic implementation needs completion
5. **Yolo mode functionality** - Partial implementation
6. **Parity testing framework** - ✅ **COMPLETED - Framework implemented**

---

## Testing Requirements

Per `.clinerules/process-leak-prevention.md`:

> **Rule:** AI connection tests must run serially (one at a time) to prevent rate limiting and process accumulation

**Implementation:**
- Use `-parallel=1` flag for AI tests
- Implement rate limiting (max 50 calls/hour)
- Set timeouts (30s default)
- Track and cleanup spawned processes

---

## Test Categories

### 1. Command Execution Scenarios

| Scenario | Node.js CLI | GoLang CLI | Priority |
|----------|-------------|------------|----------|
| `cline --help` | ✅ | ⚠️ Stub | P0 |
| `cline --version` | ✅ | ✅ | ✅ |
| `cline task "prompt"` | ✅ | ⚠️ Stub | P0 |
| `cline task -y "prompt"` | ✅ | ⚠️ Stub | P0 |
| `cline task --json "prompt"` | ✅ | ⚠️ Stub | P0 |
| `cline history` | ✅ | ⚠️ Stub | P1 |
| `cline config` | ✅ | ⚠️ Stub | P1 |

### 2. TUI Interaction Scenarios

| Component | Node.js CLI (Ink) | GoLang CLI (Bubble Tea) | Priority |
|-----------|-------------------|-------------------------|----------|
| Welcome screen | ✅ Full | ⚠️ Basic | P0 |
| Chat interface | ✅ Full | ⚠️ Basic | P0 |
| Settings panel | ✅ Full | ❌ Missing | P1 |
| History view | ✅ Full | ❌ Missing | P1 |
| Approval prompts | ✅ Full | ⚠️ Stub | P0 |

### 3. Message Type Support

| Message Type | Node.js CLI | GoLang CLI | Priority |
|--------------|-------------|------------|----------|
| `say` | ✅ | ✅ | ✅ |
| `ask` | ✅ | ⚠️ Partial | P0 |
| `tool_use` | ✅ | ⚠️ Partial | P0 |
| `command` | ✅ | ⚠️ Partial | P0 |
| `browser_action` | ✅ | ❌ Missing | P2 |
| `mcp_request` | ✅ | ❌ Missing | P2 |

---

## Implementation Plan

### Phase 0: Testing Infrastructure (Week 1)

**Objective:** Establish parity testing framework

**Tasks:**
1. Create all 8 framework files listed above
2. Implement CLI adapters for both implementations
3. Create baseline capture tool
4. Define 16 core test scenarios
5. Generate test reports

**Exit Criteria:**
- Can run `go test ./tests/parity/` successfully
- Can compare help/version outputs
- Generates readable test reports

### Phase 1-6: Feature Implementation (Weeks 2-12)

See `IMPLEMENTATION_PLAN.md` for complete phased approach.

---

## File Structure (Target)

```
golang-cli/tests/parity/
├── README.md                    # This file
├── parity_suite.go              # Test orchestrator
├── node_cli_adapter.go          # Node.js CLI adapter
├── golang_cli_adapter.go        # GoLang CLI adapter
├── output_comparator.go         # Output comparison
├── scenario_registry.go         # Test scenarios
├── report_generator.go          # Report generation
├── parity_test.go               # Test entry point
└── baselines/                   # Expected outputs
    ├── help.txt
    ├── version.txt
    └── ...
```

---

## Running Parity Tests

### Basic Usage

```bash
# Run all parity tests
cd golang-cli
go test ./tests/parity/ -v

# Run specific category
go test ./tests/parity/ -run TestParity_Help -v

# Run with baseline capture
go test ./tests/parity/ -capture-baselines

# Generate report
go test ./tests/parity/ -generate-report=html
```

### Serial Execution (Required for AI Tests)

```bash
# CRITICAL: Always use -parallel=1 for AI tests
go test ./tests/parity/ -parallel=1 -v
```

---

## Adding New Test Scenarios

1. Define scenario in `scenario_registry.go`:

```go
{
    Name:        "my_new_test",
    Description: "Description of what this tests",
    Category:    CategoryTask,
    Args:        []string{"task", "hello"},
    ExpectExitCode: 0,
}
```

2. Capture baseline (if needed):
```bash
go test ./tests/parity/ -capture-baseline=my_new_test
```

3. Run test:
```bash
go test ./tests/parity/ -run TestParity/my_new_test -v
```

---

## Reference Files

### GoLang CLI
- `golang-cli/cmd/cline/root.go` - Main entry point
- `golang-cli/internal/tui/chat_model.go` - Chat TUI
- `golang-cli/internal/task/runner.go` - Task execution

### Node.js CLI (Reference)
- `cli/src/index.ts` - Main entry point
- `cli/src/components/App.tsx` - Main TUI app
- `cli/src/components/ChatView.tsx` - Chat interface

### Documentation
- `GAP_ANALYSIS.md` - Complete gap analysis (47 gaps)
- `IMPLEMENTATION_PLAN.md` - 12-week phased plan
- `IMPLEMENTATION_PROMPT.md` - Next session instructions

---

## Success Criteria

The parity testing framework is complete when:

- [x] All 8 framework files exist and compile
- [x] Can execute both CLIs and capture output
- [x] Can compare outputs with normalization
- [x] Generates readable test reports (4 formats)
- [x] Runs serially for AI tests (no parallelism)
- [x] Documents all 25 test scenarios
- [x] Baseline capture tool implemented
- [x] CI/CD integration ready (JUnit XML)

---

## Next Steps

**Immediate:** Implement the parity testing framework per `IMPLEMENTATION_PROMPT.md`

**Then:** Use the framework to verify each feature as it's implemented in Phases 1-6.

---

*This document will be updated as the parity testing framework is implemented.*