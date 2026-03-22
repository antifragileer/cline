# Task Resumption by ID

## Feature ID
FEAT-DEV-TASK-003-RESUME-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L405-L408]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Enable users to start, continue, and manage AI coding tasks seamlessly with full task lifecycle from initialization through resumption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L352-L365]
**Business Impact:** Task continuity and workflow efficiency for long-running development workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L384-L386]

## Feature Overview
**Purpose:** Enable users to resume existing tasks by ID with full conversation history restoration and optional follow-up messages, ensuring task continuity across CLI sessions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L405-L408]
**Scope:** Task ID-based resumption, conversation history loading, state restoration, optional follow-up message handling, and `--continue` flag for most recent task
**PRD References:** REQ-012 (Support task resumption by ID) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L888]
**PRD Feature ID:** EPIC-DEV-TASK-003-RESUME-002
**Dependencies:**
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for task persistence and history retrieval [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L686-L695]
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication with core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L686-L695]
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - Required for `-T` flag parsing and `--continue` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L686-L695]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L405-L408 - Feature 2: Task Resumption by ID]

**Inputs:**
1. Task ID from history (via `-T` flag or `--continue`)
2. Optional follow-up message text
3. Existing conversation history from storage
4. Task state metadata from persistence layer

**Activities:**
1. Load conversation history for the specified task ID
2. Restore task state when resuming
3. Append follow-up message if provided
4. Resume streaming conversation with core extension
5. Validate task ID exists before resumption attempt
6. Handle most recent task lookup when using `--continue` flag
7. Filter tasks by current working directory for `--continue` (if applicable)

**Outputs:**
1. Resumed task context with full conversation history
2. Continued conversation stream from AI
3. Updated task state with new messages
4. Error messages if task ID not found

**Outcomes:**
1. Users can seamlessly continue previous tasks without losing context
2. Task history is available for reference and resumption
3. Long-running workflows can be resumed across CLI sessions
4. Optional follow-up messages allow task progression

**Impacts:**
1. Improved productivity through task continuity
2. Reduced friction in long-running development workflows
3. Enhanced debugging support via conversation history
4. Workflow efficiency through reliable task management

## Technical Requirements
**Architecture Layer:** Application Layer (Task Management Service)
**Integration Points:**
- Proposed: Task Repository (`internal/task/repository.go`) - Data access for task persistence and history retrieval
- Proposed: Task Service (`internal/task/service.go`) - Core task management logic for resumption
- Proposed: gRPC Task Client (`internal/grpc/task_client.go`) - gRPC communication for resuming tasks with core extension
- Existing: File-based JSON storage at `~/.cline/data/` [Source: .clinerules/storage.md:L1-L10]
- Existing: StateManager for state access [Source: .clinerules/storage.md:L34-L41]

**Data Requirements:**
- Existing: Task history storage format in `~/.cline/data/taskHistory.json` [Source: .clinerules/storage.md:L47-L53]
- Existing: Conversation messages stored in task-specific directories
- Existing: Task metadata including ID, timestamp, working directory, and summary

**Performance Requirements:**
- Task resumption should complete in < 500ms (including history loading)
- Conversation history loading should support pagination for large conversations
- State restoration should be atomic to prevent data corruption

**Security Requirements:**
- Validate task ID format to prevent path traversal attacks
- Ensure users can only resume their own tasks (based on file permissions)
- Maintain encryption for any sensitive data in conversation history

## User Experience
**User Personas:** Developer User - software developers using Cline for coding assistance who need to continue previous work [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L63-L72]

**User Actions:**
1. Resume specific task by ID: `cline -T abc123`
2. Resume with follow-up message: `cline -T abc123 'add unit tests'`
3. Resume most recent task: `cline --continue`
4. View conversation history after resumption
5. Continue the conversation seamlessly

**UI Components:**
- Proposed: Resume confirmation message in TUI showing task ID and summary
- Proposed: Loading indicator while conversation history loads
- Proposed: Chat interface displaying loaded conversation history
- Existing: Message rendering components from FEAT-DEV-UI-002-CHAT-002

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L409-L428 - Feature: Task Resumption by ID]

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
[Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L80-L86 - Success Metrics]

**Functional:**
- Task resumption loads full conversation history correctly
- Task IDs are validated before resumption attempt
- Follow-up messages are appended correctly when provided
- `--continue` flag resolves to most recent task
- Tasks can be resumed between TypeScript and GoLang CLI implementations (dual testing)

**Performance:**
- Task resumption completes in < 500ms
- Conversation history loads without blocking the UI
- Large conversation histories are handled with pagination

**Quality:**
- No data loss during task resumption
- Error messages are clear when task ID not found
- State is consistent after resumption

**Integration:**
- Works seamlessly with EPIC-DEV-TASK-003-CONV-003 (Conversation History Management)
- Integrates with EPIC-INFRA-CORE-011 (Core Extension Integration) for gRPC streaming
- Uses EPIC-INFRA-STORAGE-012 (State & Storage Layer) for persistence

**Business Value:**
- Enables long-running development workflows
- Reduces friction in task continuity
- Supports debugging via conversation history reference

## Testing Strategy
**Unit Testing:**
- Task ID validation logic
- Task repository history loading methods
- Task service resumption orchestration
- Error handling for missing task IDs

**Integration Testing:**
- End-to-end task resumption with mock storage
- gRPC communication with mock core extension
- State compatibility between CLI implementations (dual testing)
- Conversation history loading and display

**User Acceptance:**
- All BDD scenarios pass
- Manual verification of conversation continuity
- Cross-CLI resumption (TypeScript to GoLang and vice versa)

**Performance Testing:**
- Load testing with large conversation histories
- Benchmark resumption time against < 500ms requirement

## Tasks Overview
1. **Task 1:** Implement task repository methods for loading task history by ID
2. **Task 2:** Implement task service resumption logic with state restoration
3. **Task 3:** Implement `-T` flag parsing and validation in CLI commands
4. **Task 4:** Implement `--continue` flag for most recent task resolution
5. **Task 5:** Integrate with gRPC client for task resumption streaming
6. **Task 6:** Add error handling for invalid/missing task IDs
7. **Task 7:** Write unit tests for resumption components
8. **Task 8:** Write integration tests including dual-CLI compatibility tests

## Implementation Notes
- Task resumption requires reading from the existing file-based storage at `~/.cline/data/` which is shared between TypeScript and GoLang CLI implementations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L12-L18]
- The ONLY permitted connection to existing Cline code is via gRPC/protobuf communication with the core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L12-L18]
- Conversation history format must remain compatible with existing Cline storage to enable dual testing and cross-CLI resumption
- Task IDs should be unique and URL-safe for command-line usage [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L117-L121]
- The `--continue` flag should resolve to the most recent task from the current working directory to provide context-aware resumption
- Follow-up messages should be appended to the conversation before resuming the AI stream

## Cross-Persona Integration
This feature is specific to the Developer User persona and does not have cross-persona integration points. However, it enables workflows that may feed into:
- DevOps/Automation persona through scriptable task resumption
- Enterprise persona through audit trails of task resumption events

## Implementation Priority
**HIGH** - This feature is foundational for the CLI and part of Phase 4 (Task Management) of the AI Execution Plan [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L123-L130]

## Dependencies Detail
- **EPIC-INFRA-STORAGE-012 (State & Storage Layer):** Required for reading task history and conversation data from `~/.cline/data/`
- **EPIC-INFRA-CORE-011 (Core Extension Integration):** Required for gRPC communication to resume task streaming with core extension
- **EPIC-DEV-CLI-001 (Command Line Interface Foundation):** Required for parsing `-T` and `--continue` flags
- **EPIC-DEV-UI-002 (Interactive Terminal UI):** Required for displaying resumed conversation history and handling user input

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L###]
- [x] Existing code references cite actual file paths and lines [Source: .clinerules/storage.md:L###]
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or marked as new
- [x] Citation Verification checklist completed