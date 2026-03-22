# Yolo Mode Implementation

## Feature ID
FEAT-AUTO-EXEC-006-YOLO-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808-L845]

## Epic Context
**Parent Epic:** EPIC-AUTO-EXEC-006 - Automated Execution & Yolo Mode [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L1]
**Target Persona:** DevOps/Automation User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L132-L140]
**Epic Objective:** Enable fully automated, unattended execution of Cline tasks for CI/CD pipelines, batch processing, and integration with other tools [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L813]
**Business Impact:** Provides secure, controlled automation for enterprise environments, enabling DevOps engineers and automation specialists to run reliable, scriptable AI-assisted operations without manual intervention [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L801]

## Feature Overview
**Purpose:** Implement fully automated task execution that suppresses all confirmation prompts and auto-approves tool executions, allowing the CLI to run unsupervised from start to finish [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L801]
**Scope:** 
- Auto-approval of all tool executions when `-y` flag is provided
- Suppression of all confirmation prompts
- Automatic task completion without user intervention
- Integration with JSON output mode for scripting workflows
- Reliable exit codes for CI/CD pipeline integration
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808-L845]

**PRD References:** REQ-006 (Implement yolo mode for automated execution) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L803]
**Dependencies:** 
- EPIC-DEV-CLI-001: Command Line Interface Foundation (provides `-y` flag parsing) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- EPIC-DEV-TASK-003: Task Management (provides tool approval workflow hooks) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- EPIC-AUTO-MODE-005: Plain Text & Scripting Modes (provides non-interactive mode foundation) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808-L814 - IAOOI section for this feature]

**Inputs:**
- `--yolo` flag from command line arguments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808]
- Auto-approve configuration from settings [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808]
- Task context and prompt [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808]
- Tool execution requests from AI agent [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808]

**Activities:**
- Enable auto-approval state when `-y` flag is detected [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]
- Intercept tool approval requests and auto-respond with approval [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]
- Suppress all confirmation prompts (file edits, command execution, etc.) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]
- Stream execution output in real-time during automated execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]
- Continue task execution until natural completion or error [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]
- Ensure clean exit with appropriate exit code on task completion [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L809]

**Outputs:**
- Streamed execution results showing all tool executions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L810]
- Final task status (success/failure) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L810]
- Exit code 0 for successful completion, non-zero for failures [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L810]
- JSON-formatted output when combined with `--json` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L810]

**Outcomes:**
- Users can execute tasks without any user intervention from start to finish [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L811]
- Tasks complete automatically with appropriate exit codes for scripting integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L811]
- CI/CD pipelines can rely on Cline for automated code review, testing, and fixes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L811]

**Impacts:**
- Enables CI/CD integration for automated AI-assisted workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L812]
- Supports batch processing of multiple tasks without manual oversight [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L812]
- Reduces manual oversight required for routine automation tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L812]
- Accelerates DevOps workflows through reliable automation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L812]

## Technical Requirements

**Architecture Layer:** Application Layer (CLI command handling and task execution orchestration)

**Integration Points:**
- **Proposed:** Yolo Mode Controller - Go component to manage auto-approval state
  - Tracks yolo mode activation via `-y` flag
  - Intercepts approval requests and auto-responds
  - Ensures clean exit on task completion
- **Integration with Tool Approval System:** Hooks into the tool approval workflow from EPIC-DEV-TASK-003 to bypass prompts when yolo mode is active [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- **Integration with Plain Mode:** Yolo mode typically activates when CLI detects non-interactive environment (from EPIC-AUTO-MODE-005) or can be explicitly forced with `-y` flag [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- **Integration with JSON Output:** Combines with `--json` flag for structured automated output [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Data Requirements:**
- **Existing:** Auto-approve settings stored in `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- **Runtime state:** In-memory flag to track yolo mode activation status

**Performance Requirements:**
- Startup time: <100ms (same as existing CLI) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L132-L140]
- No additional latency for tool approval in yolo mode (approval decision must be instantaneous)
- Streaming output must not be buffered or delayed by yolo mode

**Security Requirements:**
- Yolo mode should only be used in trusted environments [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- When combined with EPIC-AUTO-EXEC-006-PERMS-002 (Command Permission Validation), all commands must still be validated against `CLINE_COMMAND_PERMISSIONS` before execution [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Audit logging should capture all auto-approved actions [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Dangerous character detection (backticks, unquoted newlines) should run even in yolo mode [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

## User Experience

**User Personas:** DevOps/Automation User - engineers integrating Cline into CI/CD pipelines and automation workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L132-L140]

**User Actions:**
1. **Automated Pipeline Execution:** Configure permissions, run `cline -y --json 'review code and fix any issues'`, receive JSON output with results, pipeline continues based on exit code [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
2. **Batch Processing:** Create script looping through repositories, run `cline -y 'analyze dependencies and suggest updates'` for each repo, capture results without manual intervention [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**UI Components:**
- **Proposed:** No TUI rendering in yolo mode - output is plain text or JSON only
- **Proposed:** Progress indicators via stdout (streaming dots or JSON status messages)
- **Proposed:** Clear indication when yolo mode is active (stdout message at startup)

**Command Line Interface:**
```bash
# Basic yolo mode execution
cline -y 'run tests and fix failures'

# Yolo mode with JSON output for scripting
cline -y --json 'analyze codebase'

# Yolo mode with specific model
cline -y -m claude-sonnet-4 'refactor code'

# Yolo mode with timeout
cline -y --timeout 300 'long running task'

# Yolo mode with task resumption
cline -y -T abc123 'continue working'
```

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L845 - BDD scenarios for this feature]

```gherkin
Feature: Yolo Mode Implementation (FEAT-AUTO-EXEC-006-YOLO-001)

Scenario: Execute task with yolo mode
  Given the user wants automated execution
  When the user runs "cline -y 'run tests and fix failures'"
  Then all tool approvals should be auto-approved
  And the task should run to completion without prompts
  And the exit code should indicate success or failure

Scenario: Yolo mode with JSON output
  Given the user wants automated execution with structured output
  When the user runs "cline -y --json 'analyze codebase'"
  Then the output should be JSON formatted
  And all approvals should be automatic

Scenario: Yolo mode exits on completion
  Given yolo mode is active
  When the task completes
  Then the process should exit automatically
  And return appropriate exit code
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1342-L1370 - Success criteria from PRD]

**Functional:**
- All BDD scenarios pass in both GoLang CLI and existing TypeScript CLI (dual testing mandate) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1342-L1345]
- Tasks complete without user prompts when `-y` flag is provided [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1346]
- All tool types (file edits, command execution, browser actions, MCP tools) are auto-approved in yolo mode [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Task continues to completion or error without requiring user intervention [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Performance:**
- Performance is equal to or better than existing CLI implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350]
- No measurable overhead from yolo mode auto-approval logic [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Quality:**
- Exit codes are reliable and consistent with existing CLI behavior (0 = success, non-zero = failure) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1347]
- JSON output is valid and parseable when combined with `-y --json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1348]
- Error handling follows same patterns as existing CLI (error messages, exit codes) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Integration:**
- Works seamlessly with command permission validation (EPIC-AUTO-EXEC-006-PERMS-002) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Integrates correctly with JSON output mode (EPIC-AUTO-OUT-007) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Compatible with task resumption (`-T` flag) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- No dependencies on existing TypeScript CLI code (independence verification) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1349]

**Business Value:**
- Enables CI/CD pipeline integration with reliable automation [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]
- Supports batch processing workflows without manual oversight [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

## Testing Strategy

**Unit Testing:**
- Yolo mode state management (enable/disable logic)
- Auto-approval decision logic for different tool types
- Exit code determination logic
- Integration with flag parsing (`-y` detection)

**Integration Testing:**
- End-to-end task execution with yolo mode enabled
- Integration with tool approval workflow from EPIC-DEV-TASK-003
- Integration with JSON output mode
- Integration with command permission validation (when EPIC-AUTO-EXEC-006-PERMS-002 is complete)

**Dual Testing (GoLang vs TypeScript CLI):**
- Side-by-side execution of identical commands with `-y` flag
- Output format comparison (plain text and JSON)
- Exit code verification matching
- Timing/performance comparison
- All BDD scenarios executed in both implementations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1342-L1370]

**User Acceptance:**
- CI/CD pipeline integration test (simulate pipeline execution)
- Batch processing workflow test (multiple sequential tasks)
- Error condition handling (task failures in yolo mode)

**Performance Testing:**
- Benchmark startup time with and without yolo mode
- Measure tool approval latency in yolo mode vs manual approval

## Tasks Overview
1. **Task 1:** Implement yolo mode flag parsing (`-y`, `--yolo`) in Cobra CLI framework
2. **Task 2:** Create Yolo Mode Controller component to manage auto-approval state
3. **Task 3:** Integrate yolo mode with tool approval workflow (hook into EPIC-DEV-TASK-003)
4. **Task 4:** Implement auto-approval logic for all tool types (file edits, commands, browser, MCP)
5. **Task 5:** Ensure proper exit code handling (0 for success, non-zero for failure)
6. **Task 6:** Integrate with JSON output mode for structured automation output
7. **Task 7:** Write comprehensive unit and integration tests
8. **Task 8:** Execute dual testing against existing TypeScript CLI

## Implementation Notes

**Phase 5 Implementation:** This feature is scheduled for Phase 5 in the AI Execution Plan, following the core CLI foundation, TUI development, and task management implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1544-L1560]

**Integration with EPIC-AUTO-MODE-005:** Yolo mode typically activates when the CLI detects a non-interactive environment (piped input or redirected output). The mode switching logic from EPIC-AUTO-MODE-005 should automatically enable yolo behavior when appropriate, or it can be explicitly forced with the `-y` flag [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Integration with EPIC-AUTO-OUT-007:** Yolo mode is commonly combined with `--json` output for automation. The integration should ensure that:
- JSON output continues streaming in real-time during yolo execution
- Final JSON includes task completion status and exit code information
- Error conditions are properly serialized as JSON [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**Security Integration:** When EPIC-ENT-SEC-008 (Security & Permissions) is implemented, yolo mode must still validate commands against `CLINE_COMMAND_PERMISSIONS` before execution. Denied commands should be logged and reported without breaking the task, and permission violations should result in appropriate error messages and exit codes [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-EXEC-006/epic.md:L##]

**GoLang Implementation Considerations:**
- Implement as part of the Cobra CLI command handling in `cmd/` package
- Yolo Mode Controller should be in `internal/yolo/` or similar package
- Use Go channels for coordinating auto-approval responses with the tool execution flow
- Ensure thread-safe state management for concurrent tool requests

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808-L845]
- [x] Existing code references cite actual file paths and lines (where applicable to existing TypeScript CLI)
- [x] New functionality clearly marked as "Proposed:" (Yolo Mode Controller, UI components)
- [x] Integration points cite existing interfaces or mark as new
- [x] BDD scenarios extracted verbatim from PRD
- [x] IAOOI framework complete and sourced from PRD