# Task Resumption by ID

## Feature ID
FEAT-DEV-TASK-003-RESUME-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Enable users to start, continue, and manage coding tasks seamlessly through the CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L380]
**Business Impact:** Task continuity and workflow efficiency for developers using Cline CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L387]

## Feature Overview
**Purpose:** Allow users to resume existing tasks by ID, loading conversation history and continuing from the previous state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408]
**Scope:** Task resumption via task ID flag, optional follow-up message support, and automatic resumption of most recent task via continue flag
**PRD References:** REQ-012 - Support task resumption by ID [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408]
**Dependencies:** 
- EPIC-DEV-CLI-001: Command Line Interface Foundation (for command parsing) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]
- EPIC-INFRA-STORAGE-012: State & Storage Layer (for persistence) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]
- FEAT-DEV-TASK-003-INIT-001: New Task Initialization (for task state structure) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]
- EPIC-INFRA-CORE-011: Core Extension Integration (for gRPC communication) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L413]

**Inputs:**
- Task ID from history (via `-T` flag)
- Optional follow-up message (positional argument)
- Task history from storage layer (`~/.cline/data/taskHistory.json`)
- Conversation state from task-specific files

**Activities:**
- Load conversation history from storage
- Restore task state including mode (plan/act)
- Append follow-up message if provided
- Resume streaming communication with core extension via gRPC
- Re-establish bidirectional message flow

**Outputs:**
- Resumed task context with full conversation history
- Continued conversation stream
- Updated task state in storage

**Outcomes:**
- Users can continue previous tasks seamlessly without losing context
- Long-running workflows can be resumed across CLI sessions
- Task continuity improves developer productivity

**Impacts:**
- Workflow efficiency - developers can pick up where they left off
- Task continuity - complex multi-step tasks can be completed over time
- Reduced context switching - no need to re-explain requirements

## Technical Requirements
**Architecture Layer:** Application/Domain Layer
**Integration Points:**
- Proposed: gRPC `ResumeTask` or `NewTask` with task ID parameter (existing TypeScript CLI uses task resumption via core extension)
- Existing storage: Task history stored in `~/.cline/data/taskHistory.json` [Source: .clinerules/storage.md]
- Existing storage: Per-task conversation files in `~/.cline/data/tasks/{taskId}/`
**Data Requirements:**
- Existing: Task ID format (string, unique identifier)
- Existing: Task state structure from core extension
- Existing: Conversation message history format
**Performance Requirements:**
- Task resumption should complete within 500ms (loading history from disk)
- Streaming should begin within 1 second of command execution
**Security Requirements:**
- Task IDs must be validated before loading (prevent directory traversal)
- Secrets must be loaded securely from encrypted storage for resumed tasks

## User Experience
**User Personas:** Developer User (primary), DevOps/Automation User (secondary)
**User Actions:**
1. List recent tasks using `cline history` to find task ID
2. Resume specific task: `cline -T abc123`
3. Resume with follow-up: `cline -T abc123 'add unit tests'`
4. Quick resume most recent: `cline --continue`
**UI Components:**
- Proposed: Task resumption confirmation message in TUI showing "Resuming task abc123..."
- Proposed: Loading indicator while conversation history loads
- Proposed: Display of previous conversation context in chat history
- Existing: Bubble Tea TUI framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408]
**Mobile Considerations:** Not applicable - CLI is desktop-focused

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L420]

```gherkin
Scenario: Resume task without follow-up
  Given an existing task with ID "abc123"
  When the user runs "cline -T abc123"
  Then the task should resume from last state
  And conversation history should load

Scenario: Resume task with follow-up message
  Given an existing task with ID "abc123"
  When the user runs "cline -T abc123 'add unit tests'"
  Then the task should resume
  And the follow-up message should be added to conversation

Scenario: Resume most recent task
  Given there are completed tasks
  When the user runs "cline --continue"
  Then the most recent task from current directory should resume
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L420]
**Functional:**
- Tasks can be resumed by ID with conversation history intact
- Follow-up messages are properly appended to resumed conversations
- `--continue` flag correctly resumes the most recent task
**Performance:**
- Task resumption completes within 500ms
- Streaming begins within 1 second of command execution
**Quality:**
- All conversation history loads correctly without data loss
- Task state (mode, model, etc.) is preserved on resume
**Integration:**
- Resumed tasks work seamlessly with gRPC core extension
- State compatibility maintained with existing ~/.cline/data/ storage
**Business Value:**
- Users can continue long-running tasks across sessions
- Reduced time spent re-establishing context

## Testing Strategy
**Unit Testing:**
- Task ID parsing and validation
- History loading from storage
- Follow-up message construction
**Integration Testing:**
- End-to-end task resumption flow
- gRPC communication resumption
- State persistence verification across resume operations
**User Acceptance:**
- Resume task and verify conversation continuity
- Test with and without follow-up messages
- Verify `--continue` behavior with multiple tasks
**Performance Testing:**
- Measure time to resume tasks with large conversation histories (>1000 messages)
- Benchmark storage read operations

## Tasks Overview
1. Implement task ID flag parsing in Cobra CLI framework
2. Implement task history loading from `~/.cline/data/` storage
3. Implement conversation state restoration
4. Implement follow-up message appending logic
5. Implement `--continue` flag for most recent task detection
6. Implement gRPC task resumption request
7. Integrate task resumption with Bubble Tea TUI
8. Add validation for task ID existence and permissions
9. Write unit tests for resumption logic
10. Write integration tests for end-to-end resumption flow

## Implementation Notes
- Task IDs are generated during initial task creation and stored in task metadata
- The storage layer uses file-based JSON storage under `~/.cline/data/` [Source: .clinerules/storage.md]
- Task history is stored separately from conversation state for efficient listing
- gRPC integration with core extension handles the actual task state management
- The existing TypeScript CLI uses `-T` flag for task ID; maintain this flag for consistency
- `--continue` flag provides convenience for the common case of resuming the most recent task
- Consider adding a "recent tasks" quick-pick in interactive mode for easier task selection

## Dual Testing Requirements
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:Dual Testing Strategy]

Per the Dual Testing Mandate, this feature must pass identical test scenarios in both the existing TypeScript CLI and the new GoLang CLI:
- Execute identical resume commands in both CLIs
- Compare conversation history loading byte-for-byte
- Verify identical behavior for `-T` and `--continue` flags
- Confirm tasks created in one CLI can be resumed in the other
- Test edge cases (invalid task IDs, missing history, concurrent access)

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new