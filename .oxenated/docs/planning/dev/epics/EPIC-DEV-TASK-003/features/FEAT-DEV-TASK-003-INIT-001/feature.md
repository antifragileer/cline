# New Task Initialization

## Feature ID
FEAT-DEV-TASK-003-INIT-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L394]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L1]
**Target Persona:** Developer User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L136]
**Epic Objective:** Enable users to start, continue, and manage coding tasks seamlessly through the CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L380]
**Business Impact:** Improved productivity, task continuity, and workflow efficiency for developers using Cline CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L387]

## Feature Overview
**Purpose:** Enable users to initialize new coding tasks with all configuration options including images, mode selection, model preference, and working directory. This is the core entry point for all AI-assisted coding work in the CLI. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L394-L395]
**Scope:** Initial task creation with prompt, image attachments, plan/act mode selection, model override, and working directory context.
**PRD References:** REQ-001, REQ-004, REQ-013 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L590]
**PRD Feature ID:** EPIC-DEV-TASK-003-INIT-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L394]
**Dependencies:** 
- EPIC-DEV-CLI-001: Command Line Interface Foundation (for command parsing) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L42]
- EPIC-DEV-UI-002: Interactive Terminal UI (for displaying streaming responses) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L43]
- EPIC-INFRA-CORE-011: Core Extension Integration (for gRPC communication) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L44]
- EPIC-INFRA-STORAGE-012: State & Storage Layer (for task persistence) [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md:L45]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L394-L397]

**Inputs:**
- User prompt (text string describing the coding task)
- Image file paths (optional, for multi-modal tasks) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L396]
- Mode selection (plan mode `-p` or act mode `-a`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L396]
- Model preference (override default model with `-m` flag) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L396]
- Working directory (override with `-c` flag) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L396]

**Activities:**
- Generate unique task ID (UUID or similar unique identifier)
- Initialize conversation state with user prompt and context
- Load and attach image files (convert to base64 if needed)
- Set mode context (plan vs act) for AI behavior
- Send initialization request to core extension via gRPC
- Render streaming response from AI in TUI or plain text mode
- Persist task metadata to history storage

**Outputs:**
- Unique task ID for later resumption
- Conversation messages (user prompt, AI response)
- Tool requests (if AI requires file edits, commands, etc.)
- Task metadata persisted to storage

**Outcomes:**
- User can start new coding task with all available options
- Task is immediately active with streaming AI response
- Task history is established for future resumption

**Impacts:**
- Core functionality used by all CLI users
- Foundation for all AI-assisted coding workflows
- Enables multi-modal tasks with image inputs
- Supports different AI behaviors via plan/act modes

## Technical Requirements
**Architecture Layer:** Application/Domain Layer
**Integration Points:** 
- Proposed: gRPC client to core extension (TaskService/NewTask RPC) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-L471]
- Proposed: FileStorage for task persistence (~/.cline/data/taskHistory.json) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-L471]
- Existing: StateManager for configuration loading [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-L471]
**Data Requirements:** 
- Proposed: Task record with ID, timestamp, prompt summary, mode, model, working directory
- Existing: Integration with ~/.cline/data/ storage format for compatibility
**Performance Requirements:** 
- Task initialization < 500ms before streaming begins
- Image loading and encoding < 2 seconds for typical files (< 5MB)
- Support for images up to 20MB each
**Security Requirements:** 
- Validate image file paths (prevent directory traversal)
- Sanitize user prompt before transmission
- Secure storage of task metadata

## User Experience
**User Personas:** Developer User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L136-L144]
**User Actions:**
1. Run `cline "prompt text"` for quick task with defaults
2. Run `cline -p "design API"` for plan mode
3. Run `cline -a "fix bug" -m claude-sonnet` for act mode with specific model
4. Run `cline -i screenshot.png "explain this error"` with image attachment
5. Run `cline -c /project/path "analyze codebase"` with custom working directory

**UI Components:** 
- Proposed: Chat message rendering component for streaming AI response
- Proposed: Progress indicator during initialization
- Proposed: Task ID display on successful initialization
- Existing: Bubble Tea TUI framework for interactive mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248-L250]

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L399-L408]

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

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L56-L62]

**Functional:**
- Users can initialize new tasks with prompts, images, mode selection, model override, and working directory
- Task ID is generated and persisted for later resumption
- Streaming response begins within 500ms of initialization
- Images are correctly loaded and transmitted to AI

**Performance:**
- Initialization completes in < 1 second (excluding AI response time)
- Image encoding handles files up to 20MB efficiently
- No blocking operations during streaming

**Quality:**
- Task history entry created atomically with initialization
- Graceful error handling for invalid image paths, network failures
- Clear error messages for configuration issues

**Integration:**
- Works seamlessly with gRPC core extension
- State compatible with existing ~/.cline/data/ storage
- Tasks can be resumed by ID (FEAT-DEV-TASK-003-RESUME-002)

**Business Value:**
- Core functionality enabling all AI-assisted coding workflows
- Foundation for task continuity and history features

## Testing Strategy
**Unit Testing:**
- Task ID generation (uniqueness, format validation)
- Image file loading and base64 encoding
- Command flag parsing and validation
- gRPC request construction

**Integration Testing:**
- End-to-end task initialization with mock core extension
- Image attachment workflow with various file types (PNG, JPG, GIF)
- State persistence verification
- Cross-platform path handling (Windows, macOS, Linux)

**User Acceptance:**
- Developer can start task with single command
- Image attachments display correctly in AI context
- Plan mode produces planning response, act mode produces execution
- Task appears in history immediately after start

**Performance Testing:**
- Initialization latency under various conditions
- Image encoding performance for large files
- Concurrent task initialization handling

## Tasks Overview
1. **Task 1:** Implement task ID generation and validation
2. **Task 2:** Implement image file loading and encoding
3. **Task 3:** Implement gRPC NewTask RPC client call
4. **Task 4:** Implement streaming response handling for new tasks
5. **Task 5:** Implement task history persistence on initialization
6. **Task 6:** Integrate with command flag parsing (mode, model, directory)
7. **Task 7:** Add error handling and user feedback

## Implementation Notes
**Critical Independence Requirement:** This feature MUST be implemented in pure Go without any dependencies on the existing TypeScript CLI code. The gRPC communication with the core extension is the ONLY permitted connection to existing Cline code. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L13-L18]

**Dual Testing Mandate:** All task initialization functionality MUST be tested in BOTH the existing Cline CLI AND the new GoLang CLI to ensure exact functional parity before the GoLang CLI can be considered production-ready. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L19-L20]

**State Compatibility:** Task initialization must write to the same ~/.cline/data/ storage format as the existing CLI to ensure tasks can be resumed across CLI implementations. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L11]

**Image Handling:** Images should be loaded and encoded to base64 for transmission via gRPC. Consider lazy loading for large files and size limits to prevent memory issues.

**Mode Context:** The plan/act mode affects the AI's system prompt and available tools. Ensure mode is correctly transmitted to core extension via gRPC.

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or marked as new