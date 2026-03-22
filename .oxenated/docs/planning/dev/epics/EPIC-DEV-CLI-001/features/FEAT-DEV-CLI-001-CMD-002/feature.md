# Task Subcommand with Flags

## Feature ID
FEAT-DEV-CLI-001-CMD-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L341]

## Epic Context
**Parent Epic:** EPIC-DEV-CLI-001 - Command Line Interface Foundation [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-CLI-001/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Establish the core CLI infrastructure for the GoLang Cline CLI migration, providing foundational command parsing, routing, and subcommand structure using the Cobra CLI framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L230]
**Business Impact:** This feature provides the primary task execution interface for Cline CLI, enabling users to start AI coding tasks with full customization through command-line flags [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L341]

## Feature Overview
**Purpose:** Implement the `task` subcommand with comprehensive flag support, allowing users to execute AI coding tasks with precise control over execution mode, model selection, timeout, auto-approval, image attachments, and other configuration options [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L241-L252]
**Scope:** Full implementation of `cline task` command with all flags (-a, -p, -y, -t, -m, -i, -v, -c, --config, --thinking, --json, -T) matching existing TypeScript CLI functionality [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L291]
**PRD References:** REQ-003, REQ-004 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1253-L1255, L1258]
**PRD Feature ID:** EPIC-DEV-CLI-001-CMD-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L241-L252]
**Dependencies:** 
- EPIC-DEV-CLI-001-CMD-001 (Root Command) - for base command infrastructure [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1310-L1313]
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - for reading configuration and persisting task state [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-CLI-001/epic.md:L445-L448]
- EPIC-INFRA-CORE-011 (Core Extension Integration) - for gRPC communication to execute tasks [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-CLI-001/epic.md:L439-L443]

## IAOOI Components
**Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248-L252]

**Inputs:**
- Task prompt string (positional argument)
- Mode flags: `--act` (-a) for act mode, `--plan` (-p) for plan mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L273-L277]
- Auto-approval flag: `--yolo` (-y) for automated execution without prompts [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L278]
- Timeout flag: `--timeout` (-t) in seconds for task execution limit [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L279]
- Model flag: `--model` (-m) to specify model ID (e.g., claude-sonnet-4) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L280]
- Image attachment flag: `--image` (-i) for attaching image files to tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L281]
- Verbose flag: `--verbose` (-v) for detailed output [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L282]
- Working directory flag: `--cwd` (-c) to set task working directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L283]
- Config flag: `--config` for custom config file path [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L284]
- Thinking flag: `--thinking` to enable thinking mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L285]
- JSON output flag: `--json` for structured output format [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L286]
- Task ID flag: `--taskId` (-T) to resume existing task by ID [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L287]
- Environment variables: CLINE_DIR, CLINE_COMMAND_PERMISSIONS [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L235-L236]

**Activities:**
- Parse and validate all task-related flags using Cobra framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249, L236-L238]
- Validate flag combinations (e.g., -a and -p are mutually exclusive) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249]
- Construct task request object with validated configuration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249]
- Load configuration from state storage (~/.cline/data/globalState.json, secrets.json) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72, L425-L428]
- Initialize gRPC connection to core extension for task execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249, L422-L425]
- Route to appropriate execution mode (interactive TUI or plain text) based on flags and TTY detection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249, L591-L661]
- Handle task resumption when -T flag provided [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249, L469-L483]
- Process image attachments and include in task context [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L249, L467-L468]

**Outputs:**
- Validated task configuration object with all flag values resolved [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L250]
- Initialized task context with unique task ID [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L250, L463-L466]
- Help documentation for task command (auto-generated by Cobra) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L250, L236-L238]
- Error messages for invalid flag combinations or missing required inputs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L250]
- Exit code 0 on success, non-zero on failure [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L219-L221]

**Outcomes:**
- Users can customize task execution with all supported options matching existing CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251]
- Users can choose between plan mode (gather information first) and act mode (immediate execution) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251, L404-L411]
- Users can automate task execution with yolo mode for CI/CD integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251, L693-L705]
- Users can resume previous tasks by ID for continuity [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251, L469-L483]
- Users can attach images for multi-modal tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251, L467-L468]
- Script users can get structured JSON output for parsing [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251, L713-L740]

**Impacts:**
- Flexible task execution supports diverse use cases from interactive development to automated pipelines [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L252]
- Script compatibility enables CI/CD integration and batch processing workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L252]
- Full flag parity ensures zero learning curve for existing CLI users [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L251-L252]
- Foundation for all task-based workflows in the Cline CLI ecosystem [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239]

## Technical Requirements
**Architecture Layer:** Application Layer (CLI Command Handler)

**Integration Points:**
- **Existing Storage:** File-based JSON state at ~/.cline/data/globalState.json, secrets.json, workspace state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72, L933-L1011]
  - [Proposed: Go implementation of ClineFileStorage in `internal/storage/storage.go`]
  - [Proposed: StateManager equivalent in `internal/storage/state_manager.go`]
- **Existing gRPC/Protobuf:** Communication with core extension via proto definitions in `proto/` directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422-L517]
  - [Proposed: Go gRPC client in `internal/grpc/client.go`]
- **TUI Integration:** Routes to Bubble Tea TUI when in interactive mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L343-L420]
  - [Proposed: TUI initialization in `internal/tui/app.go`]
- **Plain Mode Output:** JSON or plain text output for scripting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L591-L661, L713-L740]
  - [Proposed: Output handlers in `internal/output/`]

**Data Requirements:**
- Task configuration structure with all flag fields [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L248, L273-L287]
  - [Proposed: TaskConfig struct in `internal/task/config.go`]
- API provider configuration (provider ID, model, API keys) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L519-L589]
  - [Existing: ~/.cline/data/secrets.json - encrypted API keys]
  - [Existing: ~/.cline/data/globalState.json - provider settings]
- Task history for resumption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L469-L483]
  - [Existing: ~/.cline/data/tasks/ directory structure]

**Performance Requirements:**
- Command parsing and validation: < 50ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239, L1419]
- gRPC connection establishment: < 100ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1419]
- Task initialization: < 200ms total before streaming begins [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1419]

**Security Requirements:**
- API keys must be read from encrypted secrets storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1025]
- Command permissions (CLINE_COMMAND_PERMISSIONS) must be respected when executing commands [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L745-L811]
- Image file paths must be validated before attachment [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L467-L468]

## Flag Reference Table
**Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L291]

| Flag | Short | Description | Type | Default |
|------|-------|-------------|------|---------|
| --act | -a | Start in act mode (immediate execution) | boolean | false |
| --plan | -p | Start in plan mode (gather information first) | boolean | false |
| --yolo | -y | Auto-approve all tools without prompts | boolean | false |
| --timeout | -t | Timeout in seconds for task execution | integer | 0 (no timeout) |
| --model | -m | Specify model ID to use | string | from config |
| --image | -i | Attach image file (can be specified multiple times) | string array | [] |
| --verbose | -v | Enable verbose output | boolean | false |
| --cwd | -c | Set working directory for task | string | current dir |
| --config | | Custom config file path | string | ~/.cline/data/ |
| --thinking | | Enable thinking mode | boolean | false |
| --json | | Output in JSON format (implies plain mode) | boolean | false |
| --taskId | -T | Resume existing task by ID | string | "" |

## User Experience
**User Personas:** 
- Developer User (primary) - interactive task execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L137-L145]
- DevOps/Automation User - scripted task execution with yolo and JSON flags [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L147-L154]
- Enterprise User - policy-compliant execution with permission validation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L156-L163]

**User Actions:**
1. Run `cline task --help` to see all available options
2. Run `cline task -a "fix the bug"` to start in act mode
3. Run `cline task -p "design an API"` to start in plan mode
4. Run `cline task -y "run tests"` for automated execution
5. Run `cline task -m claude-sonnet-4 "refactor code"` with specific model
6. Run `cline task -T abc123 "continue"` to resume task
7. Run `cline task -i screenshot.png "analyze this"` with image attachment
8. Run `cline task --json "analyze" > output.json` for scripted workflows

**UI Components:**
- Help text display (auto-generated by Cobra) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L236-L238]
  - [Proposed: Cobra-generated help in `internal/cli/task.go`]
- Error messages for invalid usage
  - [Proposed: Error formatting in `internal/cli/errors.go`]
- Progress indication in TUI mode (delegated to Bubble Tea components)
  - [Proposed: TUI integration in `internal/tui/task_view.go`]
- JSON output stream in plain mode
  - [Proposed: JSON encoder in `internal/output/json.go`]

## BDD Scenarios
**Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L252-L270]

### Scenario 1: Start task in act mode
```gherkin
Scenario: Start task in act mode
  Given the user has a valid API configuration
  When the user runs "cline task -a 'fix the bug in main.go'"
  Then the task should start in act mode
  And Cline should begin executing tools
```

### Scenario 2: Start task in plan mode
```gherkin
Scenario: Start task in plan mode
  Given the user wants to plan before executing
  When the user runs "cline task -p 'design a new API'"
  Then the task should start in plan mode
  And Cline should gather information and present a plan
```

### Scenario 3: Start task with yolo mode
```gherkin
Scenario: Start task with yolo mode
  Given the user wants automated execution
  When the user runs "cline task -y 'run tests'"
  Then all tool approvals should be auto-approved
  And the task should complete without prompts
```

### Scenario 4: Start task with specific model
```gherkin
Scenario: Start task with specific model
  Given the user wants to use a specific model
  When the user runs "cline task -m claude-sonnet-4 'refactor this code'"
  Then the task should use the specified model
  And the model selection should be confirmed
```

### Scenario 5: Resume task by ID
```gherkin
Scenario: Resume task by ID
  Given an existing task with ID "abc123"
  When the user runs "cline task -T abc123 'continue working'"
  Then the existing task should resume
  And the follow-up message should be added
```

## Success Criteria
**Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L252, L1413-L1422]

**Functional:**
- All flags (-a, -p, -y, -t, -m, -i, -v, -c, --config, --thinking, --json, -T) are implemented and functional
- Flag parsing produces identical behavior to existing TypeScript CLI
- Task command routes to appropriate execution mode (TUI or plain)
- Task resumption by ID loads conversation history correctly
- Image attachments are processed and included in task context

**Performance:**
- Command parsing and validation completes in < 50ms
- Task initialization completes in < 200ms before streaming begins
- Binary startup time remains < 100ms for help/version

**Quality:**
- Help text is auto-generated and matches existing CLI coverage
- Error messages are helpful and actionable
- Flag validation prevents invalid combinations
- All exit codes match existing CLI behavior

**Integration:**
- Works seamlessly with gRPC core extension integration
- Reads configuration correctly from state storage
- Integrates with TUI for interactive mode
- Outputs correct JSON format for scripting

**Business Value:**
- Enables full task execution capability for Cline CLI
- Supports all use cases from interactive development to CI/CD automation
- Maintains zero learning curve for existing users through exact flag parity

## Testing Strategy
**Unit Testing:**
- Flag parsing validation for all flag combinations
- Task configuration construction logic
- Flag mutual exclusion validation (-a vs -p)

**Integration Testing:**
- gRPC task initialization with mock core
- State storage reading for configuration
- Image file loading and validation

**Functional Parity Tests (Dual Testing):**
- Execute identical task commands in both TypeScript and Go CLIs
- Compare help output byte-for-byte
- Verify flag behavior equivalence for all flags
- Match exit codes for success and error cases
- Compare error message formatting

**E2E Tests:**
- Full task execution with various flag combinations
- Task resumption flow
- Image attachment processing
- JSON output verification

## Tasks Overview
1. **TASK-001**: Implement Cobra task subcommand structure with flag definitions
2. **TASK-002**: Implement flag validation logic (mutual exclusion, type checking)
3. **TASK-003**: Implement task configuration construction from flags
4. **TASK-004**: Integrate with storage layer to load API configuration
5. **TASK-005**: Implement task initialization and gRPC routing
6. **TASK-006**: Implement task resumption by ID
7. **TASK-007**: Implement image attachment processing
8. **TASK-008**: Add dual testing verification for all flag combinations

## Implementation Notes
- This feature MUST achieve byte-for-byte output parity with existing TypeScript CLI (except timestamps/IDs) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L813-L822]
- The GoLang CLI MUST NOT import or depend on any TypeScript CLI code [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L34-L53]
- All flag names, descriptions, and behaviors must match existing CLI exactly [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L34-L53, L813-L822]
- Use Cobra's built-in help generation for consistency with Go CLI conventions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239]
- Environment variable support (CLINE_DIR, CLINE_COMMAND_PERMISSIONS) must be preserved [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L235-L236, L239]
- Flag -T for task ID uses capital T to avoid conflict with -t for timeout [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L287]

## Proposed File Structure
```
golang-cli/
├── cmd/
│   └── cline/
│       └── main.go                    # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                    # Root command setup
│   │   ├── task.go                    # Task subcommand (THIS FEATURE)
│   │   └── errors.go                  # Error formatting
│   ├── task/
│   │   ├── config.go                  # TaskConfig struct
│   │   ├── init.go                    # Task initialization logic
│   │   └── resume.go                  # Task resumption logic
│   ├── storage/
│   │   ├── storage.go                 # ClineFileStorage
│   │   └── state_manager.go           # StateManager
│   ├── grpc/
│   │   └── client.go                  # gRPC client for core
│   ├── tui/
│   │   └── app.go                     # Bubble Tea TUI integration
│   └── output/
│       ├── plain.go                   # Plain text output
│       └── json.go                    # JSON output handler
```

## Dependencies
- **Cobra**: CLI framework for command parsing and help generation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L236-L238]
- **Viper**: Optional for configuration management [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311]
- **Go standard library**: flag, os, fmt, encoding/json

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L341, L248-L252, L273-L291]
- [x] Existing code references cite actual file paths and lines (storage paths, proto directory)
- [x] New functionality clearly marked as "Proposed:" for Go implementation
- [x] Integration points cite existing interfaces or marked as new
- [x] Feature ID matches exact PRD feature ID (EPIC-DEV-CLI-001-CMD-002)
- [x] IAOOI framework extracted verbatim from PRD
- [x] BDD scenarios included exactly as written in PRD
- [x] Flag reference table complete from PRD