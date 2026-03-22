# Automated Execution & Yolo Mode

## Epic ID
EPIC-AUTO-EXEC-006

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L##-L## - Epic 6 Section]

## Target Persona
DevOps/Automation User

## Epic Overview
This epic delivers fully automated task execution capabilities for the GoLang Cline CLI, enabling CI/CD integration and batch processing workflows. The yolo mode feature suppresses all confirmation prompts and auto-approves tool executions, allowing the CLI to run unsupervised from start to finish. Combined with command permission validation through the `CLINE_COMMAND_PERMISSIONS` environment variable, this provides secure, controlled automation for enterprise environments. This functionality is essential for DevOps engineers and automation specialists who need reliable, scriptable AI-assisted operations without manual intervention.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L813]

## Vision & Objectives
Enable fully automated, unattended execution of Cline tasks for CI/CD pipelines, batch processing, and integration with other tools. Provide security controls through command permission validation to ensure automated execution remains safe and compliant with enterprise policies.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L813]

## IAOOI System Components

### Inputs
- `--yolo` flag from command line arguments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797]
- Command permission rules from `CLINE_COMMAND_PERMISSIONS` environment variable [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797]
- Auto-approve settings from configuration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797]
- Task context and prompt [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797]

### Activities
- Parse permission configuration from environment variables [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L798]
- Validate commands against allow/deny lists before execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L798]
- Execute tools without user prompts when yolo mode is enabled [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L798]
- Stream output in real-time during execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L798]
- Apply auto-approval settings to suppress confirmation dialogs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L798]

### Outputs
- Command execution results with exit codes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L799]
- Permission decisions (allow/deny) for each command [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L799]
- Execution logs for audit trails [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L799]
- Streamed output during task execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L799]

### Outcomes
- Users can execute tasks without any user intervention from start to finish [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L800]
- Tasks complete automatically with appropriate exit codes for scripting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L800]
- Command execution adheres to security policies through permission validation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L800]

### Impacts
- Enables CI/CD integration for automated AI-assisted workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L801]
- Supports batch processing of multiple tasks without manual oversight [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L801]
- Reduces manual oversight required for routine automation tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L801]
- Accelerates DevOps workflows through reliable automation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L801]

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L801]

## Key Features
- EPIC-AUTO-EXEC-006-YOLO-001: Yolo Mode Implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L808]
- EPIC-AUTO-EXEC-006-PERMS-002: Command Permission Validation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L856]

## Business Value & Requirements
This epic directly addresses the following requirements:

- **REQ-006**: Implement yolo mode for automated execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L803]
- **REQ-008**: Implement command permission validation (`CLINE_COMMAND_PERMISSIONS`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L803]

The DevOps/Automation User persona specifically requires:
- Yolo mode (auto-approve all actions) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L137]
- JSON output for parsing [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L136]
- Piped input support [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L137]
- Timeout controls (`--timeout`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L137]
- Non-interactive mode detection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L137]
- Reliable exit codes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L136]

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L132-L140]

## User Journeys & Scenarios

### Primary User Journey: Automated Pipeline Execution
A DevOps engineer wants to integrate Cline into their CI/CD pipeline to automatically review and fix code issues on every commit.

1. Configure `CLINE_COMMAND_PERMISSIONS` to allow safe commands (git, npm, go test)
2. Run `cline -y --json 'review code and fix any issues'` in pipeline
3. CLI auto-approves all tool executions within policy bounds
4. Task completes with JSON output containing results
5. Pipeline continues based on exit code (0 = success, non-zero = failure)

### Secondary User Journey: Batch Processing
An automation specialist needs to process multiple repositories with the same task.

1. Create script looping through repository list
2. For each repo, run `cline -y 'analyze dependencies and suggest updates'`
3. Each task runs to completion without prompts
4. Results captured for batch analysis
5. No manual intervention required for entire batch

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L132-L149]

## BDD Scenarios

### Feature: Yolo Mode Implementation (EPIC-AUTO-EXEC-006-YOLO-001)

```gherkin
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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L845]

### Feature: Command Permission Validation (EPIC-AUTO-EXEC-006-PERMS-002)

```gherkin
Scenario: Validate command against permissions
  Given CLINE_COMMAND_PERMISSIONS allows "npm *" and denies "rm -rf *"
  When Cline attempts "npm install"
  Then the command should be allowed
  When Cline attempts "rm -rf /"
  Then the command should be denied

Scenario: Validate command segments
  Given a compound command "npm install && npm test"
  When validation runs
  Then each segment should validate separately
  And all must pass for command to execute
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L856-L880]

## Technical Considerations

### Existing Code References
This epic builds upon existing infrastructure:
- Command parsing via Cobra CLI framework (established in EPIC-DEV-CLI-001) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L580-L600]
- TTY/redirect detection (implemented in EPIC-AUTO-MODE-005) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L737-L785]
- gRPC integration with core extension (EPIC-INFRA-CORE-011) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1140-L1180]
- JSON output formatting (EPIC-AUTO-OUT-007) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L888-L940]

### Proposed New Components
1. **Yolo Mode Controller**: Go component to manage auto-approval state
   - Tracks yolo mode activation via `-y` flag
   - Intercepts approval requests and auto-responds
   - Ensures clean exit on task completion

2. **Permission Engine**: Command validation system
   - Parses `CLINE_COMMAND_PERMISSIONS` JSON environment variable
   - Implements glob pattern matching for allow/deny lists
   - Validates compound commands by segment
   - Integrates with tool execution flow

3. **Exit Code Manager**: Ensures proper exit codes for scripting
   - Returns 0 on successful task completion
   - Returns non-zero on failures or permission denials
   - Provides meaningful exit codes for different failure modes

### Security Considerations
- Yolo mode should only be used in trusted environments
- Permission rules must be validated before any command execution
- Deny rules take precedence over allow rules
- Dangerous character detection (backticks, unquoted newlines) should run even in yolo mode
- Audit logging should capture all auto-approved actions

### Dependencies
- **EPIC-DEV-CLI-001**: Command Line Interface Foundation (provides `-y` flag parsing)
- **EPIC-AUTO-MODE-005**: Plain Text & Scripting Modes (provides non-interactive mode foundation)
- **EPIC-DEV-TASK-003**: Task Management (provides tool approval workflow hooks)
- **EPIC-ENT-SEC-008**: Security & Permissions (provides permission validation engine)
- **EPIC-INFRA-CORE-011**: Core Extension Integration (provides gRPC communication)

### Integration Points
- **Tool Approval System**: Hooks into the tool approval workflow to bypass prompts when yolo mode is active
- **Command Execution Flow**: Integrates with command validation before execution
- **JSON Output Mode**: Combines with `--json` flag for structured automated output
- **Task Resumption**: Works with `-T` flag for automated task continuation
- **Timeout Controls**: Integrates with `--timeout` flag for automated timeout handling

## Implementation Priority
**Phase 5: Automation & Scripting** - This epic is scheduled for Phase 5 in the AI Execution Plan, following the core CLI foundation, TUI development, and task management implementation. It builds upon the plain mode detection from EPIC-AUTO-MODE-005 and integrates with the security features from EPIC-ENT-SEC-008.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1544-L1560]

## Success Metrics
- All BDD scenarios pass in both GoLang CLI and existing TypeScript CLI (dual testing mandate)
- Tasks complete without user prompts when `-y` flag is provided
- Exit codes are reliable and consistent with existing CLI behavior
- Permission validation correctly allows/denies commands per configuration
- JSON output is valid and parseable when combined with `-y --json`
- Performance is equal to or better than existing CLI implementation
- No dependencies on existing TypeScript CLI code (independence verification)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1342-L1370]

## Dependencies
### Required Epics (Must Complete First)
1. **EPIC-DEV-CLI-001**: Command Line Interface Foundation - Provides command parsing, `-y` flag support, and subcommand structure
2. **EPIC-DEV-TASK-003**: Task Management - Provides task initialization, execution, and tool approval workflows
3. **EPIC-AUTO-MODE-005**: Plain Text & Scripting Modes - Provides TTY detection and non-interactive mode foundation
4. **EPIC-INFRA-CORE-011**: Core Extension Integration - Provides gRPC communication with core extension

### Related Epics (Parallel Development)
- **EPIC-ENT-SEC-008**: Security & Permissions - Shares permission validation logic; coordinate implementation
- **EPIC-AUTO-OUT-007**: Structured Output (JSON) - Often used together; ensure compatibility

### Downstream Dependencies (Depend on This Epic)
- **EPIC-INFRA-DIST-014**: Distribution & Packaging - Yolo mode is a key feature for CI/CD adoption

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L803]

## Integration Points

### With EPIC-AUTO-MODE-005 (Plain Text & Scripting Modes)
Yolo mode typically activates when the CLI detects a non-interactive environment (piped input or redirected output). The mode switching logic from EPIC-AUTO-MODE-005 should automatically enable yolo behavior when appropriate, or it can be explicitly forced with the `-y` flag.

### With EPIC-AUTO-OUT-007 (Structured Output)
Yolo mode is commonly combined with `--json` output for automation. The integration should ensure that:
- JSON output continues streaming in real-time during yolo execution
- Final JSON includes task completion status and exit code information
- Error conditions are properly serialized as JSON

### With EPIC-ENT-SEC-008 (Security & Permissions)
Command permission validation is critical for safe yolo mode operation. The integration ensures:
- All commands are validated against `CLINE_COMMAND_PERMISSIONS` before execution
- Denied commands are logged and reported without breaking the task
- Permission violations result in appropriate error messages and exit codes

### With EPIC-DEV-TASK-003 (Task Management)
Tool approval workflows must recognize yolo mode:
- Approval prompts are suppressed and auto-approved
- Tool results are streamed normally
- Task continues to completion or error

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L792-L880]