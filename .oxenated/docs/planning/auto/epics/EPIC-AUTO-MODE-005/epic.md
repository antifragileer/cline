# Plain Text & Scripting Modes

## Epic ID
EPIC-AUTO-MODE-005

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L441-475 - Epic 5: Plain Text & Scripting Modes]

## Target Persona
DevOps/Automation User

## Epic Overview
This epic enables the GoLang CLI to automatically detect non-interactive execution environments (pipes, redirects, CI/CD pipelines) and switch to plain text output mode. This functionality is critical for automation users who need to integrate Cline into scripts, CI/CD pipelines, and batch processing workflows. The epic ensures seamless operation across both interactive TUI mode and non-interactive scripting mode without requiring explicit user intervention.

The automatic mode detection eliminates friction for DevOps engineers by intelligently switching between rich interactive UI and script-friendly plain text output based on the execution context. This enables reliable automation workflows while maintaining the full power of Cline's AI coding assistance.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L441-475]

## Vision & Objectives
Enable fully automated, scriptable execution of Cline CLI tasks without sacrificing the rich interactive experience for manual usage. The CLI should automatically adapt to its execution environment, providing appropriate output formats for both human interaction and machine consumption.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L441-475]

## IAOOI System Components

### Inputs
1. Stdin file descriptor state (TTY vs pipe/redirect)
2. Stdout file descriptor state (TTY vs redirect to file)
3. `isatty()` system call results for stdin and stdout
4. Explicit mode flags from command line (`--json`, `--yolo`)
5. Environment context (CI/CD detection, terminal capabilities)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L452-456]

### Activities
1. **TTY Detection**: Check if stdin is connected to a terminal using `isatty()` equivalent
2. **Output Redirection Detection**: Check if stdout is being redirected to a file or pipe
3. **Mode Decision Logic**: Determine appropriate execution mode based on detection results
4. **Mode Switching**: Configure output handlers and UI components for selected mode
5. **Flag Override Processing**: Apply explicit mode flags that force specific behavior
6. **Environment Adaptation**: Adjust behavior for CI/CD and automation contexts

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L452-456]

### Outputs
1. **Mode Decision**: Interactive TUI mode or plain text mode selection
2. **Configured Output Handlers**: Appropriate renderers for selected mode
3. **Exit Codes**: Script-friendly exit codes (0 for success, non-zero for errors)
4. **Plain Text Output**: Human-readable text without TUI formatting when in plain mode
5. **JSON Formatted Output**: Machine-parseable output when `--json` flag is used

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L457-460]

### Outcomes
1. Users can pipe input to Cline without interactive prompts breaking the pipeline
2. Output redirection to files produces clean text without TUI escape sequences
3. CI/CD pipelines can execute Cline tasks reliably with predictable exit codes
4. Scripts can parse Cline output when using JSON mode
5. No manual mode selection required - automatic adaptation to context

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L461-464]

### Impacts
1. **Automation Adoption**: Broader use in DevOps workflows and CI/CD pipelines
2. **DevOps Integration**: Seamless integration with existing automation toolchains
3. **Batch Processing Capability**: Enable processing of multiple tasks in automated batches
4. **Cross-Platform Consistency**: Reliable behavior across different execution environments
5. **Reduced Learning Curve**: No need to learn different flags for different contexts

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L465-468]

## Key Features
- **EPIC-AUTO-MODE-005-DETECT-001**: TTY/Redirect Detection - Automatic detection of interactive vs non-interactive environments
- **EPIC-AUTO-MODE-005-SWITCH-002**: Mode Switching Logic - Intelligent switching between TUI and plain text modes

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-475]

## Business Value & Requirements
This epic addresses the following original requirements:
- **REQ-002**: Support both interactive and plain text modes
- **REQ-005**: Auto-detect TTY/redirect for automatic mode switching

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-471]

## User Journeys & Scenarios

### Primary User Journey: CI/CD Pipeline Integration
A DevOps engineer wants to integrate Cline into their deployment pipeline to automatically review code changes before deployment. They configure their pipeline to pipe git diffs into Cline and expect clean output that can be parsed or logged without interactive prompts blocking the pipeline.

### Secondary User Journey: Batch Processing
An automation specialist needs to process multiple code review tasks across several repositories. They create a script that invokes Cline for each repository and expects consistent, parseable output that can be aggregated into a report.

### Tertiary User Journey: Output Logging
A developer wants to capture Cline's output to a file for later review. When redirecting output to a file, the CLI automatically switches to plain text mode, producing a clean log without terminal escape sequences.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L476-506 - BDD Scenarios section]

## BDD Scenarios

### Feature: EPIC-AUTO-MODE-005-DETECT-001 - TTY/Redirect Detection

```gherkin
Scenario: Auto-switch to plain mode when piped
  Given the user pipes input to Cline
  When the user runs "cat file.txt | cline 'summarize this'"
  Then the CLI should detect the pipe
  And switch to plain text output mode
  And not render TUI

Scenario: Auto-switch to plain mode when output redirected
  Given the user redirects output to a file
  When the user runs "cline 'list files' > output.txt"
  Then the CLI should detect the redirection
  And use plain text format without TUI

Scenario: Force interactive mode with TTY
  Given the user runs in terminal
  When the user runs "cline"
  Then the CLI should detect TTY
  And launch interactive TUI mode

Scenario: Piped input with prompt
  Given the user pipes content
  When the user runs "git diff | cline 'review these changes'"
  Then the diff should be included as context
  And the task should process with piped input
```

### Feature: EPIC-AUTO-MODE-005-SWITCH-002 - Mode Switching Logic

```gherkin
Scenario: Explicit JSON flag forces plain mode
  Given the user runs "cline --json 'list files'"
  Then plain text mode should activate
  And output should be JSON formatted

Scenario: Yolo flag forces plain mode
  Given the user runs "cline -y 'automated task'"
  Then plain text mode should activate
  And auto-approval should enable
```

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L476-506]

## Technical Considerations

### Existing Code References
- TTY detection patterns: Referenced in existing CLI implementation [UNVERIFIED - requires confirmation]
- Output formatting: Referenced in existing CLI streaming handlers [UNVERIFIED - requires confirmation]

### Proposed New Components

#### Go Implementation Requirements
1. **TTY Detection Module**: Go implementation using `term.IsTerminal()` from `golang.org/x/term` package
2. **Mode Controller**: Central mode management that configures output handlers based on detection results
3. **Plain Text Renderer**: Alternative output renderer for non-interactive mode
4. **Flag Processor**: Command-line flag processing that can override automatic detection

#### Key Technical Decisions
- Use Go's `os.Stdin.Stat()` and `os.Stdout.Stat()` to check file descriptor modes
- Implement `isatty()` equivalent using `term.IsTerminal(int(os.Stdin.Fd()))`
- Support explicit `--json` flag to force JSON output format
- Support explicit `--yolo` flag to force plain mode with auto-approval
- Ensure exit codes are consistent (0 for success, non-zero for errors) in both modes

#### Platform Considerations
- TTY detection must work consistently across Linux, macOS, and Windows
- Windows console detection requires special handling (use `isatty` equivalent for Windows)
- Handle edge cases like pseudo-TTYs in CI/CD environments (GitHub Actions, Jenkins, etc.)

### Integration Points
- **Core Extension Integration**: Plain mode still communicates with core extension via gRPC, but suppresses TUI rendering
- **Output Formatting**: JSON output mode integrates with message serialization from EPIC-AUTO-OUT-007
- **Yolo Mode**: Auto-approval mode from EPIC-AUTO-EXEC-006 requires plain mode for non-interactive execution

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L507-518]

## Implementation Priority
**Priority:** High - Required for MVP

This epic is foundational for the DevOps/Automation persona and must be implemented early to support automated testing and CI/CD integration. The automatic mode detection is essential for the CLI to function correctly in both interactive and non-interactive contexts.

**Sequencing:**
- Should be implemented in Phase 5: Automation & Scripting
- Depends on: Core CLI foundation (Phase 2)
- Enables: Yolo mode (EPIC-AUTO-EXEC-006), JSON output (EPIC-AUTO-OUT-007)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1478-1514 - AI Execution Plan Phase 5]

## Success Metrics
1. **Detection Accuracy**: 100% accurate detection of TTY vs non-TTY environments
2. **Exit Code Consistency**: Identical exit codes (0 success, non-zero failure) across both modes
3. **Output Compatibility**: Plain text output is parseable and free of terminal escape sequences
4. **CI/CD Reliability**: Zero interactive prompts in non-interactive environments
5. **Flag Override Reliability**: Explicit flags (`--json`, `--yolo`) correctly force plain mode

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L441-475]

## Dependencies
- **EPIC-DEV-CLI-001** (Command Line Interface Foundation): Requires command parsing and flag handling
- **EPIC-INFRA-CORE-011** (Core Extension Integration): gRPC communication must work in both modes

## Dependents
- **EPIC-AUTO-EXEC-006** (Automated Execution & Yolo Mode): Depends on plain mode for non-interactive execution
- **EPIC-AUTO-OUT-007** (Structured Output): JSON output requires plain mode

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L470-475]

## Integration Points

### With Other Epics
- **EPIC-DEV-UI-002** (Interactive Terminal UI): TTY mode triggers TUI rendering; non-TTY suppresses it
- **EPIC-AUTO-EXEC-006** (Automated Execution & Yolo Mode): Yolo mode forces plain text mode
- **EPIC-AUTO-OUT-007** (Structured Output): JSON output flag forces plain mode
- **EPIC-INFRA-CORE-011** (Core Extension Integration): Message streaming works in both modes

### Cross-Persona Impact
- **Developer User (Persona 1)**: Benefits from automatic mode switching when logging output to files
- **DevOps/Automation User (Persona 2)**: Primary beneficiary - enables CI/CD integration
- **Enterprise User (Persona 3)**: Enables automated audit and compliance workflows

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L441-475, L470-475]

## Dual Testing Requirements
Per the dual testing mandate, this epic must pass:
1. **Functional Parity Tests**: TTY detection behavior must match existing TypeScript CLI exactly
2. **Side-by-Side Integration Tests**: Both CLIs must handle identical pipe/redirect scenarios identically
3. **Cross-Platform Consistency**: TTY detection must work identically on Linux, macOS, and Windows in both CLIs
4. **Output Format Verification**: Plain text output format must be byte-for-byte identical (except timestamps)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1029-1115 - Dual Testing Strategy]

## Notes
- This epic is critical for the "Automation User" persona and enables the CLI to be used in CI/CD pipelines
- The automatic detection eliminates the need for users to learn different command patterns for different contexts
- Implementation should follow Go best practices for TTY detection using standard library and `golang.org/x/term`
- Consider edge cases like GitHub Actions (which may report as TTY in some configurations)