# Tool Approval Workflows

## Feature ID
FEAT-DEV-TASK-003-TOOL-004

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L436]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Enable users to start, continue, and manage coding tasks seamlessly through the CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L380]
**Business Impact:** Improved productivity, task continuity, and workflow efficiency for developers using Cline CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L387]

## Feature Overview
**Purpose:** Provide human-in-the-loop control for file changes and command execution through interactive tool approval workflows in the terminal UI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L436-L441]
**Scope:** Interactive approval prompts for tool requests, rejection handling, auto-approve in yolo mode, and returning execution results to the core extension
**PRD References:** REQ-001, REQ-004, REQ-006, REQ-008
**PRD Feature ID:** EPIC-DEV-TASK-003-TOOL-004 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L436]
**Dependencies:** 
- EPIC-DEV-CLI-001: Command Line Interface Foundation (for flag parsing)
- EPIC-DEV-UI-002: Interactive Terminal UI (for rendering approval prompts)
- EPIC-INFRA-CORE-011: Core Extension Integration (for gRPC communication)
- FEAT-DEV-TASK-003-INIT-001: New Task Initialization (task context)
- FEAT-AUTO-EXEC-006-YOLO-001: Yolo Mode Implementation (for auto-approval bypass)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L436-L441 - IAOOI section for this feature]

**Inputs:**
- Tool use requests from agent via gRPC streaming
- Tool type (file edit, command execution, etc.)
- Tool parameters (file path, command string, proposed changes)
- User approval/rejection responses
- Auto-approve configuration from yolo mode

**Activities:**
- Display tool request in terminal UI with clear details
- Capture user decision (approve/reject) via keyboard input
- Execute approved tools through appropriate handlers
- Return tool execution results or rejection reasons to agent
- Bypass prompts when auto-approve is enabled

**Outputs:**
- Tool execution results (success/failure, output)
- Rejection messages with reasons
- Execution confirmation for user
- State updates to core extension via gRPC

**Outcomes:**
- Users maintain control over file changes and command execution
- Safe execution of AI-suggested operations
- Clear visibility into what Cline is doing

**Impacts:**
- Safety and trust in AI-assisted development
- Human-in-the-loop control for critical operations
- Reduced risk of unintended changes

## Technical Requirements
**Architecture Layer:** Application Layer (UI + Business Logic)
**Integration Points:**
- **Proposed:** New tool approval handler in `golang-cli/internal/tool/` package
- **Proposed:** Bubble Tea UI components for approval prompts in `golang-cli/internal/ui/components/`
- **Existing gRPC:** Core extension communication via TaskService [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L600-L650]
- **Proposed:** Tool execution interface with implementations for file operations and command execution

**Data Requirements:**
- **Proposed:** ToolRequest model with fields: type, id, parameters, timestamp
- **Proposed:** ToolResponse model with fields: requestId, status (approved/rejected), result, error
- **Existing:** Task state storage in `~/.cline/data/tasks/` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L610]

**Performance Requirements:**
- Approval prompts must render within 100ms of tool request
- Tool execution should not block UI updates
- Support for rapid consecutive approvals without UI lag

**Security Requirements:**
- Validate all tool parameters before execution
- Sanitize command strings to prevent injection
- Respect CLINE_COMMAND_PERMISSIONS for command execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L450-L500]
- Clear display of potentially dangerous operations with warnings

## User Experience
**User Personas:** Developer User (primary), DevOps/Automation User (via yolo mode)
**User Actions:**
1. Cline requests to edit a file - user sees diff preview and approves/rejects
2. Cline requests to execute a command - user sees command and approves/rejects
3. User enables yolo mode - all tools auto-approve without prompts
4. User reviews execution results in the conversation stream

**UI Components:**
- **Proposed:** ApprovalPrompt component with keyboard shortcuts (y/n)
- **Proposed:** DiffPreview component for file edit visualization
- **Proposed:** CommandDisplay component with syntax highlighting
- **Proposed:** ExecutionResult component showing success/failure
- **Existing:** Chat message components for tool feedback [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L350-L375]

**Mobile Considerations:** N/A - CLI is desktop-focused

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L443-L455 - BDD scenarios for this feature]

```gherkin
Scenario: Approve file edit
  Given Cline requests to edit "main.go"
  When the approval prompt displays
  And the user approves
  Then the edit should execute
  And result should return to Cline

Scenario: Reject command execution
  Given Cline requests to run "rm -rf /"
  When the approval prompt displays
  And the user rejects
  Then the command should not execute
  And rejection should return to Cline

Scenario: Auto-approve in yolo mode
  Given yolo mode is enabled
  When Cline requests any tool
  Then it should execute without prompt
  And result should return automatically
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L436-L441 - Success criteria implied by IAOOI]

**Functional:**
- All tool requests display clear approval prompts in the TUI
- Users can approve or reject with single keystroke (y/n)
- File edits show diff preview before approval
- Commands display with clear warnings for dangerous operations
- Yolo mode bypasses all prompts correctly
- Tool results return to core extension accurately

**Performance:**
- Approval prompts render within 100ms
- No UI blocking during tool execution
- Smooth handling of rapid consecutive approvals

**Quality:**
- Clear error messages on tool execution failures
- Proper handling of partial tool execution states
- Graceful recovery from interrupted tool operations

**Integration:**
- Works seamlessly with gRPC streaming from core extension
- Compatible with existing task state persistence
- Integrates with command permission validation (EPIC-ENT-SEC-008)

**Business Value:**
- Users trust Cline CLI for safe file modifications
- Human oversight prevents unintended changes
- Automation-friendly via yolo mode for CI/CD use cases

## Testing Strategy
**Unit Testing:**
- Tool approval handler logic
- Keyboard input handling for y/n responses
- Auto-approve configuration parsing
- Tool result formatting

**Integration Testing:**
- End-to-end tool approval flow with mock core extension
- gRPC message serialization/deserialization for tool requests
- Integration with Bubble Tea UI update loop

**User Acceptance:**
- Manual testing of file edit approvals with diff preview
- Manual testing of command execution with various command types
- Verification of yolo mode auto-approval behavior
- Cross-platform keyboard handling (Linux, macOS, Windows)

**Performance Testing:**
- UI rendering latency under rapid tool request sequences
- Memory usage during long-running tasks with many approvals

## Tasks Overview
1. **Task 1:** Implement ToolRequest and ToolResponse models for gRPC communication
2. **Task 2:** Create Bubble Tea approval prompt component with keyboard handling
3. **Task 3:** Implement file edit diff preview visualization
4. **Task 4:** Implement command execution display with danger warnings
5. **Task 5:** Integrate auto-approve logic with yolo mode configuration
6. **Task 6:** Implement tool execution handlers (file edit, command run)
7. **Task 7:** Wire up gRPC streaming for bidirectional tool communication
8. **Task 8:** Add integration tests with mock core extension

## Implementation Notes
**Critical Design Decisions:**
- Use Bubble Tea's text input and key handling for approval prompts
- Implement non-blocking tool execution to keep UI responsive
- Store approval decisions in local state for potential retry scenarios
- Leverage existing gRPC bidirectional streaming for real-time tool request/response

**Architectural Patterns:**
- Handler pattern for different tool types (FileEditHandler, CommandHandler, etc.)
- State machine for tool request lifecycle (pending → approved/rejected → executing → completed)
- Observer pattern for UI updates during tool execution

**GoLang-Specific Considerations:**
- Use goroutines for non-blocking tool execution
- Implement proper context cancellation for long-running tools
- Channel-based communication between UI and execution layers
- Leverage Go's type safety for tool request/response structures

**Dual Testing Requirement:**
- All scenarios must be tested in both existing TypeScript CLI and new GoLang CLI
- Output format for tool approval prompts must match exactly
- Keyboard shortcuts and interaction patterns must be identical

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Epic reference cites actual epic.md file
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing gRPC patterns
- [x] BDD scenarios extracted verbatim from PRD
- [x] IAOOI components extracted from PRD section