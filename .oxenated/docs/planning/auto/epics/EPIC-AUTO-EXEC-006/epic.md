# Automated Execution & Yolo Mode

## Epic ID
EPIC-AUTO-EXEC-006

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L308-L375 - Epic 6: Automated Execution & Yolo Mode]

## Target Persona
DevOps/Automation User (Automation Persona)

## Epic Overview
This epic implements automated execution capabilities for the Cline CLI GoLang migration, enabling CI/CD integration and batch processing without requiring user intervention. The epic provides a `--yolo` flag for fully automated task execution, command permission validation for security compliance, and integration with the command permissions framework (`CLINE_COMMAND_PERMISSIONS`).

The epic addresses the critical need for automation users to run Cline in pipelines and scripted environments where interactive prompts would break the workflow. It provides both the convenience of auto-approval and the security of policy-based command restrictions.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L308-L320]

## Vision & Objectives
Enable fully automated task execution for DevOps workflows and CI/CD pipelines while maintaining security through policy-based command permissions.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L321-L329]

## IAOOI System Components

### Inputs
1. `--yolo` flag - Explicit flag to enable automated execution mode [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L321]
2. Command permission rules - Parsed from `CLINE_COMMAND_PERMISSIONS` environment variable [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L321]
3. Auto-approve settings - Configuration for which tools/actions should be auto-approved [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L322]
4. Task context - Current task state and execution environment [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L322]

### Activities
1. Parse permission configuration from environment variables [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L323]
2. Validate commands against allow/deny lists using glob pattern matching [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L323]
3. Enable auto-approval for all tools when yolo mode is active [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L323]
4. Suppress confirmation prompts during automated execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L323]
5. Stream output continuously without waiting for user input [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L323]
6. Detect dangerous characters and subshells in commands [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L373]
7. Validate compound commands by checking each segment separately [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L365-L368]

### Outputs
1. Command execution results - Success/failure status of executed commands [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L324]
2. Permission decisions - Allow/deny determinations with reasons [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L324]
3. Execution logs - Complete audit trail of automated actions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L324]
4. Exit codes - Standard Unix exit codes (0 for success, non-zero for failure) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L348]

### Outcomes
1. Fully automated task execution without user intervention [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L325]
2. CI/CD pipeline integration capability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L325]
3. Batch processing support for multiple tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L325]
4. Reduced manual oversight requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L325]

### Impacts
1. **CI/CD Integration**: Enables Cline to be used in automated build and deployment pipelines [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L326]
2. **Batch Processing**: Supports automated processing of multiple tasks or repositories [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L326]
3. **Reduced Manual Oversight**: Eliminates need for human monitoring during task execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L326]

## Key Features

### FEAT-AUTO-EXEC-006-YOLO-001: Yolo Mode Implementation
Enables fully automated task execution with auto-approval of all tools [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L331-L348]

### FEAT-AUTO-EXEC-006-PERMS-002: Command Permission Validation
Provides security through policy-based command restrictions using `CLINE_COMMAND_PERMISSIONS` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L349-L375]

## Business Value & Requirements

### Requirements Coverage
This epic addresses the following requirements:
- **REQ-006**: Implement yolo mode for automated execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L597]
- **REQ-008**: Implement command permission validation (CLINE_COMMAND_PERMISSIONS) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L598]

### Persona Alignment
**DevOps/Automation User** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L86-L98]
- **Role**: DevOps engineer, automation specialist, CI/CD pipeline operator
- **Pain Points Addressed**:
  - Need for reliable exit codes
  - No interactive prompts breaking scripts
  - Timeout controls for long-running tasks
- **Needs Met**:
  - Yolo mode (auto-approve all actions)
  - Non-interactive mode detection
  - Automation workflows without human intervention

## BDD Scenarios

### Yolo Mode Implementation (FEAT-AUTO-EXEC-006-YOLO-001)

```gherkin
Scenario: Execute task with yolo mode
  Given the user wants automated execution
  When the user runs "cline -y 'run tests and fix failures'"
  Then all tool approvals should be auto-approved
  And the task should run to completion without prompts
  And the exit code should indicate success or failure
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L335-L340]

```gherkin
Scenario: Yolo mode with JSON output
  Given the user wants automated execution with structured output
  When the user runs "cline -y --json 'analyze codebase'"
  Then the output should be JSON formatted
  And all approvals should be automatic
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L341-L346]

```gherkin
Scenario: Yolo mode exits on completion
  Given yolo mode is active
  When the task completes
  Then the process should exit automatically
  And return appropriate exit code
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L347-L348]

### Command Permission Validation (FEAT-AUTO-EXEC-006-PERMS-002)

```gherkin
Scenario: Validate command against permissions
  Given CLINE_COMMAND_PERMISSIONS allows "npm *" and denies "rm -rf *"
  When Cline attempts "npm install"
  Then the command should be allowed
  When Cline attempts "rm -rf /"
  Then the command should be denied
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L358-L364]

```gherkin
Scenario: Validate command segments
  Given a compound command "npm install && npm test"
  When validation runs
  Then each segment should validate separately
  And all must pass for command to execute
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L365-L368]

## Technical Considerations

### Existing Code References
- **State Storage**: Uses existing `~/.cline/data/` state storage pattern [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L19]
- **gRPC Integration**: Communicates with Cline core extension via protobuf [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L24]
- **Environment Variables**: Leverages `CLINE_COMMAND_PERMISSIONS` environment variable pattern [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L308]

### Proposed New Components
- **Proposed**: GoLang implementation of permission parser for `CLINE_COMMAND_PERMISSIONS` JSON schema
- **Proposed**: GoLang glob pattern matcher for command validation
- **Proposed**: GoLang dangerous character detector for command injection prevention
- **Proposed**: Integration with Bubble Tea TUI for yolo mode status indication (when interactive)
- **Proposed**: Exit code handler for automated script integration

### Integration Points

#### Related Epics
- **EPIC-AUTO-MODE-005** (Plain Text & Scripting Modes): Yolo mode builds on the plain mode foundation for non-interactive execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L297-L307]
- **EPIC-ENT-SEC-008** (Security & Permissions): Command permission validation is shared with enterprise security requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L376-L426]
- **EPIC-AUTO-OUT-007** (Structured Output JSON): Yolo mode frequently paired with `--json` flag for automation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L427-L460]

#### Persona Integration
- **DevOps/Automation User**: Primary target persona for this epic [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L86-L98]
- **Enterprise User**: Benefits from command permission validation for policy compliance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L99-L111]

## Implementation Priority
**Phase 5** of the AI Execution Plan [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L723-L738]

This epic is implemented during Phase 5 (Automation & Scripting) after the core CLI foundation, TUI, and task management are complete. This sequencing ensures:
1. Basic task execution works before adding automation
2. Tool approval system exists before implementing auto-approval
3. Plain mode detection works before yolo mode can function

## Success Metrics

### Functional Metrics
1. **Command Execution**: All commands execute without prompts in yolo mode
2. **Exit Code Accuracy**: Exit codes match task success/failure (0 = success, non-zero = failure)
3. **Permission Enforcement**: Denied commands are blocked; allowed commands execute

### Performance Metrics
1. **Startup Time**: No significant degradation when yolo mode enabled
2. **Permission Check Overhead**: <1ms per command validation

### Dual Testing Requirements
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L621-L696]:
- All yolo mode scenarios must pass in both TypeScript and GoLang CLI implementations
- Exit codes must match exactly between implementations
- Permission validation behavior must be identical
- JSON output format must match byte-for-byte

## Dependencies

### Hard Dependencies
1. **EPIC-DEV-CLI-001** (CLI Foundation): Command parsing and flag handling must be complete
2. **EPIC-DEV-TASK-003** (Task Management): Task initialization and tool approval system required
3. **EPIC-AUTO-MODE-005** (Plain Text & Scripting Modes): TTY/redirect detection prerequisite

### Soft Dependencies
1. **EPIC-AUTO-OUT-007** (Structured Output JSON): Often used together but not strictly required
2. **EPIC-ENT-SEC-008** (Security & Permissions): Permission validation overlaps significantly

## Security Considerations

### Command Injection Prevention
- Detect backticks outside single quotes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L417]
- Detect unquoted newlines in commands [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422-L426]
- Detect subshell attempts [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L409-L413]

### Permission Rule Precedence
- Deny rules take precedence over allow rules [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L393-L397]
- Empty/missing permissions default to allow all (backward compatibility) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L383-L386]

## Testing Strategy

### Unit Tests
- Permission rule parsing [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L378-L386]
- Glob pattern matching for commands
- Dangerous character detection
- Exit code handling

### Integration Tests
- End-to-end yolo mode execution with mock core
- Command permission validation with various rule configurations
- Integration with plain mode detection

### Dual Testing (Critical)
Per the dual testing framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L621-L696]:
- Execute identical yolo commands in both CLIs
- Compare exit codes
- Compare output formatting
- Verify identical permission validation behavior
- Test concurrent execution scenarios

## CLI Usage Examples

### Basic Yolo Mode
```bash
# Automated execution without prompts
cline -y 'run tests and fix any failures'

# Yolo mode with specific model
cline -y -m claude-sonnet 'refactor the codebase'
```

### Yolo with JSON Output (CI/CD)
```bash
# JSON output for parsing in pipelines
cline -y --json 'analyze code quality' > report.json

# Check exit code for pipeline control
cline -y 'run deployment' || echo "Deployment failed"
```

### With Command Permissions
```bash
# Set permissions before running
export CLINE_COMMAND_PERMISSIONS='{"allow":["npm *","git *"],"deny":["rm -rf *"],"allowRedirects":false}'

# Run with permissions enforced
cline -y 'npm install && npm test'
```

### Resume with Yolo
```bash
# Resume existing task in yolo mode
cline -y -T abc123 'continue with auto-approval'
```

## Notes for Implementation

### GoLang Specific Considerations
1. Use Go's `os/exec` package for command validation (not actual execution - that's handled by core)
2. Implement glob matching using `filepath.Match` or similar library
3. Use `syscall` package for proper exit code handling
4. Consider `regexp` for dangerous character detection

### Independence Requirements
Per the critical independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L9-L28]:
- Implement permission validation purely in Go
- Do NOT import TypeScript permission logic
- Do NOT depend on Node.js `minimatch` or similar libraries
- Use Go-native glob libraries only

### State Compatibility
- Store auto-approve preferences in existing state storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L510-L517]
- Maintain compatibility with TypeScript CLI's yolo mode settings