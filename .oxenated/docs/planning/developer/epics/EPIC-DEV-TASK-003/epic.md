# Task Management

## Epic ID
EPIC-DEV-TASK-003

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L352-L434 - Epic 3: Task Management Section]

## Target Persona
Developer User

## Epic Overview
Task Management is a core epic for the Cline CLI GoLang migration that enables users to start, continue, and manage AI coding tasks seamlessly. This epic encompasses the full task lifecycle from initialization through resumption, including conversation history management and tool approval workflows. It delivers the core functionality that all users depend on for executing AI coding tasks, ensuring task continuity and workflow efficiency.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L352-L365]

## Vision & Objectives
The Task Management epic delivers a complete task lifecycle system that enables users to:
- Initialize new coding tasks with full configuration options
- Resume existing tasks seamlessly with conversation history intact
- Manage conversation history for reference and debugging
- Maintain human-in-the-loop control through tool approval workflows

This epic directly supports the Developer User persona's need for task resumption, plan/act mode workflows, and reliable task execution with appropriate safety controls.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L352-L365]

## IAOOI System Components

### Inputs
1. User prompts (text instructions for the AI)
2. Image files for multi-modal tasks (screenshots, diagrams)
3. Task mode preferences (plan mode `-p` or act mode `-a`)
4. Model selection preferences (override default model)
5. Working directory specification (`-c` flag)
6. Existing task IDs for resumption (`-T` flag)
7. Optional follow-up messages when resuming tasks
8. Tool use requests from the AI agent
9. User approval/rejection decisions for tools

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L367-L372]

### Activities
1. Generate unique task IDs for new tasks
2. Initialize conversation context and state
3. Send task initialization requests to core extension via gRPC
4. Render streaming responses from AI
5. Load conversation history for task resumption
6. Restore task state when resuming
7. Append follow-up messages to existing conversations
8. Store conversation messages persistently
9. Format conversation history for display
10. Handle large conversation pagination
11. Display tool use requests to users
12. Capture user approval or rejection decisions
13. Execute approved tools and return results
14. Return rejection messages for denied tools
15. Handle auto-approval in yolo mode

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L373-L377]

### Outputs
1. Unique task IDs for each new task
2. Conversation messages (ask/say pairs)
3. Tool use requests for user approval
4. Tool execution results
5. Error messages for rejected tools
6. Persisted task state to storage
7. Task metadata for history tracking
8. Streaming message updates in real-time

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L378-L380]

### Outcomes
1. Users can start new coding tasks with all configuration options
2. Users can seamlessly continue previous tasks without losing context
3. Task history is available for reference and resumption
4. Users maintain control over file changes and command execution
5. Long-running workflows can be resumed across CLI sessions
6. Tool execution is safe with human-in-the-loop approval

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L381-L383]

### Impacts
1. Improved productivity through task continuity
2. Reduced friction in long-running development workflows
3. Enhanced debugging support via conversation history
4. Increased user trust through safety controls
5. Workflow efficiency through reliable task management
6. Foundation for complex multi-session development tasks

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L384-L386]

## Key Features
1. **EPIC-DEV-TASK-003-INIT-001**: New Task Initialization - Initialize new tasks with user prompts, images, mode selection, model preferences, and working directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389-L392]
2. **EPIC-DEV-TASK-003-RESUME-002**: Task Resumption by ID - Resume existing tasks with full conversation history restoration and optional follow-up messages [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L405-L408]
3. **EPIC-DEV-TASK-003-CONV-003**: Conversation History Management - Persist, load, and display conversation messages with pagination support for large histories [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L421-L424]
4. **EPIC-DEV-TASK-003-TOOL-004**: Tool Approval Workflows - Display tool requests, capture user decisions, execute approved tools, and handle auto-approval in yolo mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L434]

## Business Value & Requirements
This epic addresses the following original requirements:
- **REQ-001**: Execute AI coding tasks from terminal with interactive UI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L880]
- **REQ-004**: Support plan and act modes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L884]
- **REQ-012**: Support task resumption by ID [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L888]
- **REQ-013**: Support image attachments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L889]
- **REQ-018**: Task history with pagination [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L891]

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L354-L355]

## User Journeys & Scenarios

### Developer Task Initialization Journey
A developer wants to start a new coding task. They run `cline -m claude-sonnet -c /project -p 'design API' -i screenshot.png -i diagram.jpg`. The CLI initializes a new task, attaches the images, activates plan mode, and uses the specified model. A unique task ID is generated and persisted to history for future resumption.

### Task Resumption Journey
A developer previously worked on a task with ID "abc123". They run `cline -T abc123 'add unit tests'`. The CLI loads the conversation history, restores the task state, appends the follow-up message, and resumes the AI conversation seamlessly.

### Tool Approval Journey
During task execution, the AI requests to edit a file. The CLI displays the tool request with the proposed changes. The developer reviews and approves the edit. The CLI executes the edit and returns the result to the AI. If the developer rejects, the rejection is returned to the AI with appropriate context.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L393-L438]

## BDD Scenarios

### Feature: New Task Initialization (EPIC-DEV-TASK-003-INIT-001)

```gherkin
Scenario: Initialize new task with all options
  Given the user has images "screenshot.png" and "diagram.jpg"
  When the user runs "cline -m claude-sonnet -c /project -p 'design API' -i screenshot.png -i diagram.jpg"
  Then a new task should initialize
  And images should be attached
  And plan mode should activate
  And specified model should be used

Scenario: Task initialization generates unique ID
  Given a new task starts
  When initialization completes
  Then a unique task ID should generate
  And the task should be persisted to history
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L393-L407]

### Feature: Task Resumption by ID (EPIC-DEV-TASK-003-RESUME-002)

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L409-L428]

### Feature: Conversation History Management (EPIC-DEV-TASK-003-CONV-003)

```gherkin
Scenario: Persist conversation messages
  Given a task is active
  When messages are exchanged
  Then each message should persist to storage
  And be available for later retrieval

Scenario: Load conversation on resume
  Given a task with existing conversation
  When the task resumes
  Then the full conversation should load
  And display in correct order
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L429-L438]

### Feature: Tool Approval Workflows (EPIC-DEV-TASK-003-TOOL-004)

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L439-L458]

## Technical Considerations

### Existing Code References

The GoLang CLI implementation must integrate with the existing Cline infrastructure:

1. **State Storage Compatibility**: The CLI must read from and write to the existing file-backed JSON storage at `~/.cline/data/` [Source: .clinerules/storage.md:L1-L10]

2. **StateManager Integration**: Use StateManager for state access - `StateManager.get().getGlobalStateKey()`, `StateManager.get().setGlobalState()` [Source: .clinerules/storage.md:L34-L41]

3. **gRPC Communication**: The ONLY permitted connection to existing Cline code is via gRPC/protobuf communication with the core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L12-L18]

4. **Protobuf Definitions**: Generate Go code from existing proto definitions in `proto/` directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L671]

### Proposed New Components

1. **Task Service** (`internal/task/service.go`): Core task management logic
2. **Task Repository** (`internal/task/repository.go`): Data access for task persistence
3. **Task Model** (`internal/task/model.go`): Task domain models
4. **Conversation Manager** (`internal/task/conversation.go`): Conversation history handling
5. **Tool Approver** (`internal/task/approver.go`): Tool approval workflow logic
6. **gRPC Task Client** (`internal/grpc/task_client.go`): gRPC communication for tasks

### Implementation Notes

- Task IDs should be unique and URL-safe for command-line usage
- Conversation history must be stored in a format compatible with existing Cline storage
- Tool approval UI must integrate with Bubble Tea TUI framework
- Image attachments require base64 encoding for transmission via gRPC
- Task resumption requires loading both conversation history and task state

## Implementation Priority
**HIGH** - This epic is foundational for the CLI. Task Management is core functionality that must be implemented in Phase 4 (Task Management) of the AI Execution Plan, following CLI Foundation (Phase 2) and Interactive UI (Phase 3).

Dependencies:
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - Required for command parsing
- EPIC-DEV-UI-002 (Interactive Terminal UI) - Required for tool approval prompts
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for task persistence

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L686-L695]

## Success Metrics
- Task initialization completes in < 500ms
- Task resumption loads full conversation history correctly
- Tool approval workflow has < 100ms response time
- Conversation history persists across CLI invocations
- Tasks can be resumed between TypeScript and GoLang CLI implementations (dual testing)
- All BDD scenarios pass in both CLI implementations

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L354-L355]

## Dependencies
- **EPIC-DEV-CLI-001**: Command parsing and flag handling for task subcommand
- **EPIC-DEV-UI-002**: TUI components for tool approval prompts
- **EPIC-INFRA-CORE-011**: gRPC client for core extension communication
- **EPIC-INFRA-STORAGE-012**: File-based storage for task persistence
- **EPIC-DEV-AUTH-004**: Authentication for API provider access

## Integration Points

### With EPIC-DEV-CLI-001 (Command Line Interface Foundation)
- Task subcommand parsing and flag validation
- Command routing to task handlers

### With EPIC-DEV-UI-002 (Interactive Terminal UI)
- Tool approval prompt rendering via Bubble Tea
- Message streaming display
- User input capture for approvals

### With EPIC-AUTO-EXEC-006 (Automated Execution & Yolo Mode)
- Auto-approval logic for yolo mode bypasses tool approval UI
- Yolo flag propagation to task execution

### With EPIC-INFRA-CORE-011 (Core Extension Integration)
- gRPC calls for task initialization and streaming
- Bidirectional message streaming for conversations

### With EPIC-INFRA-STORAGE-012 (State & Storage Layer)
- Task history persistence to `~/.cline/data/`
- Conversation message storage and retrieval