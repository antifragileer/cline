# New Task Initialization

## Feature ID
FEAT-DEV-TASK-003-INIT-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389-L407 - Feature 1: New Task Initialization]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Enable users to start, continue, and manage AI coding tasks seamlessly with full task lifecycle management from initialization through resumption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L352-L365]
**Business Impact:** Core functionality that all users depend on for executing AI coding tasks, ensuring task continuity and workflow efficiency [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L13-L18]

## Feature Overview
**Purpose:** Initialize new coding tasks with full configuration options including user prompts, image attachments, mode selection (plan/act), model preferences, and working directory specification. This feature delivers the core entry point for all AI coding tasks in the CLI. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389-L392]
**Scope:** 
- Included: Task ID generation, conversation initialization, gRPC communication with core extension, streaming response handling, image attachment processing, mode selection, model override, working directory configuration
- Excluded: Task resumption (handled by FEAT-DEV-TASK-003-RESUME-002), conversation history management (handled by FEAT-DEV-TASK-003-CONV-003), tool approval workflows (handled by FEAT-DEV-TASK-003-TOOL-004)
**PRD References:** REQ-001, REQ-004, REQ-013 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L467-L469]
**PRD Feature ID:** EPIC-DEV-TASK-003-INIT-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389]
**Dependencies:** 
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - Required for command parsing and flag handling [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L124-L126]
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L124-L126]
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for task persistence [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L124-L126]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389-L392]

**Inputs:**
1. User prompts (text instructions for the AI) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L367]
2. Image files for multi-modal tasks (screenshots, diagrams) via `-i` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L367-L368]
3. Task mode preferences (plan mode `-p` or act mode `-a`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L368]
4. Model selection preferences via `-m` flag (override default model) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L369]
5. Working directory specification via `-c` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L369]
6. Optional thinking mode via `--thinking` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L312]

**Activities:**
1. Generate unique task IDs for new tasks (URL-safe format) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L373]
2. Initialize conversation context and state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L374]
3. Send task initialization requests to core extension via gRPC [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L375]
4. Render streaming responses from AI in real-time [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L376]
5. Process and encode image attachments for gRPC transmission (base64 encoding) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L116]
6. Validate mode flag combinations (mutually exclusive -a and -p) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L312]

**Outputs:**
1. Unique task IDs for each new task [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L378]
2. Conversation messages (ask/say pairs) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L379]
3. Persisted task state to storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L381]
4. Task metadata for history tracking [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L382]
5. Streaming message updates in real-time [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L383]

**Outcomes:**
1. Users can start new coding tasks with all configuration options [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L381]
2. Task initialization completes in < 500ms [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L134]
3. Images are properly attached and transmitted to AI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L393-L407]
4. Mode selection correctly activates plan or act behavior [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L400]

**Impacts:**
1. Improved productivity through seamless task initialization [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L384]
2. Reduced friction in starting development workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L385]
3. Foundation for complex multi-session development tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L386]

## Technical Requirements
**Architecture Layer:** Application Layer (Task Service) with Infrastructure Layer (gRPC Client, Storage)
**Integration Points:**
- **Proposed:** New Task Service (`internal/task/service.go`) - Core task management logic [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L103]
- **Proposed:** Task Repository (`internal/task/repository.go`) - Data access for task persistence [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L104]
- **Proposed:** Task Model (`internal/task/model.go`) - Task domain models [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L105]
- **Proposed:** gRPC Task Client (`internal/grpc/task_client.go`) - gRPC communication for tasks [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L108]
- **Existing:** Core Extension gRPC Service - Protobuf definitions in `proto/` directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L671]

**Data Requirements:**
- **Existing:** File-backed JSON storage at `~/.cline/data/` [Source: .clinerules/storage.md:L1-L10]
- **Existing:** StateManager for state access - `StateManager.get().getGlobalStateKey()`, `StateManager.get().setGlobalState()` [Source: .clinerules/storage.md:L34-L41]
- **Existing:** Task history storage format compatible with existing Cline storage [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L116]
- **Proposed:** Task ID generation scheme (unique, URL-safe for command-line usage)

**Performance Requirements:**
- Task initialization completes in < 500ms [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L134]
- Image encoding and transmission must not block UI [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L116]

**Security Requirements:**
- Image file validation (size limits, format restrictions)
- Working directory validation (prevent directory traversal)
- Secure task ID generation (non-guessable)

## User Experience
**User Personas:** Developer User (primary), DevOps/Automation User (scripting mode)
**User Actions:**
1. Run `cline "prompt"` for quick task initialization with defaults
2. Run `cline -m claude-sonnet -c /project -p 'design API' -i screenshot.png` for full configuration
3. Run `cline task -a 'fix bug'` for explicit act mode
4. Run `cline task -p -i diagram.jpg 'plan architecture'` for plan mode with images

**UI Components:**
- **Proposed:** Task initialization spinner/progress indicator [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L376]
- **Existing:** Bubble Tea TUI framework for interactive mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L34]
- **Existing:** Chat message rendering components [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L285-L298]

**Scripting Mode Considerations:**
- Plain text output when stdout is not TTY [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L320-L346]
- JSON output support with `--json` flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L312]
- Exit codes for success/failure [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L35]

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L393-L407]

```gherkin
Feature: New Task Initialization

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

Scenario: Initialize task with direct prompt (default mode)
  Given the user has configured an API provider
  When the user runs "cline 'create a hello world function'"
  Then a new task should start immediately
  And the AI should begin processing the request

Scenario: Initialize task in act mode
  Given the user has a valid API configuration
  When the user runs "cline task -a 'fix the bug in main.go'"
  Then the task should start in act mode
  And Cline should begin executing tools

Scenario: Initialize task in plan mode
  Given the user wants to plan before executing
  When the user runs "cline task -p 'design a new API'"
  Then the task should start in plan mode
  And Cline should gather information and present a plan

Scenario: Initialize task with specific model
  Given the user wants to use a specific model
  When the user runs "cline task -m claude-sonnet-4 'refactor this code'"
  Then the task should use the specified model
  And the model selection should be confirmed

Scenario: Initialize task with working directory
  Given the user wants to work in a specific directory
  When the user runs "cline -c /path/to/project 'analyze code'"
  Then the task should initialize with working directory set to "/path/to/project"
  And file operations should use that directory as base

Scenario: Initialize task with single image
  Given the user has an image "mockup.png"
  When the user runs "cline -i mockup.png 'implement this UI'"
  Then the task should initialize
  And the image should be attached to the task
  And the AI should receive the image for analysis

Scenario: Initialize task with multiple images
  Given the user has images "screen1.png" and "screen2.png"
  When the user runs "cline -i screen1.png -i screen2.png 'compare these screens'"
  Then the task should initialize
  And both images should be attached to the task

Scenario: Reject invalid mode combination
  Given the user provides both -a and -p flags
  When the user runs "cline -a -p 'do something'"
  Then the CLI should display an error
  And the task should not initialize
  And exit code should be non-zero

Scenario: Reject invalid image path
  Given the user provides a non-existent image path
  When the user runs "cline -i nonexistent.png 'analyze image'"
  Then the CLI should display an error about missing file
  And the task should not initialize
  And exit code should be non-zero

Scenario: Reject invalid working directory
  Given the user provides a non-existent working directory
  When the user runs "cline -c /nonexistent/path 'do something'"
  Then the CLI should display an error about invalid directory
  And the task should not initialize
  And exit code should be non-zero
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389-L407]

**Functional:**
- Task initializes with all supported flag combinations
- Unique task ID generates for each new task
- Images attach correctly and transmit to AI
- Mode selection (plan/act) works correctly
- Model override applies correctly
- Working directory configuration applies correctly
- Task persists to history immediately on initialization

**Performance:**
- Task initialization completes in < 500ms [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L134]
- Image encoding does not block initialization
- gRPC connection establishes within timeout

**Quality:**
- All BDD scenarios pass in both TypeScript and GoLang CLI implementations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L636]
- Identical behavior between existing and migrated CLI (dual testing) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L594-L636]
- Proper error handling with clear messages
- Exit codes match between implementations

**Integration:**
- Tasks created in GoLang CLI can be resumed in TypeScript CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L603-L605]
- State file format compatibility maintained [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L605]
- gRPC communication with core extension works correctly

**Business Value:**
- Users can start coding tasks seamlessly from terminal
- Configuration flexibility meets all use cases
- Foundation for task resumption and history features

## Testing Strategy
**Unit Testing:**
- Task ID generation (uniqueness, URL-safety)
- Flag validation logic (mutually exclusive flags)
- Image file validation and encoding
- Working directory validation

**Integration Testing:**
- gRPC communication with mock core extension
- State persistence to file storage
- Task history recording

**User Acceptance:**
- All BDD scenarios pass manually
- Interactive mode works in terminal
- Scripting mode works with pipes and redirects

**Dual Testing (Critical):**
- Execute identical initialization commands in both CLIs
- Compare generated task IDs (format, uniqueness)
- Compare state file output (byte-for-byte compatibility)
- Compare streaming output behavior
- Verify task resumption between implementations

## Tasks Overview
1. **Task 1:** Implement Task ID generation service with uniqueness and URL-safety guarantees
2. **Task 2:** Create Task initialization command handler with flag parsing and validation
3. **Task 3:** Implement Image attachment processing (validation, base64 encoding)
4. **Task 4:** Build gRPC task initialization client for core extension communication
5. **Task 5:** Implement Task state persistence to file storage
6. **Task 6:** Create Streaming response handler for real-time AI output
7. **Task 7:** Add Error handling and validation for all initialization scenarios

## Implementation Notes
- Task IDs should be unique and URL-safe for command-line usage (consider UUID v4 or similar) [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L116]
- Image attachments require base64 encoding for transmission via gRPC [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L116]
- Conversation history must be stored in a format compatible with existing Cline storage [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-TASK-003/epic.md:L116]
- Mode flags (-a and -p) are mutually exclusive and should be validated [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L312]
- Implementation should follow Phase 4 (Task Management) of AI Execution Plan [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L697-L712]
- Must pass dual testing with existing TypeScript CLI before completion [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L594-L636]

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:" or "To be created:"
- [x] Integration points cite existing interfaces or mark as new