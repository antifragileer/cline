# Plain Text & Scripting Modes

## Epic ID
EPIC-AUTO-MODE-005

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L341 - Epic 5: Plain Text & Scripting Modes]

## Target Persona
DevOps/Automation User

## Epic Overview
This epic enables the Cline CLI to operate in non-interactive environments such as CI/CD pipelines, shell scripts, and automated workflows. The GoLang CLI must automatically detect when it's running in a non-TTY environment (piped input or redirected output) and switch from the interactive Bubble Tea TUI to plain text output mode. This ensures seamless operation across all execution contexts without requiring explicit user intervention or flags.

The epic covers automatic TTY/redirect detection, mode switching logic, and the foundation for scriptable automation. This functionality is critical for DevOps engineers who need to integrate Cline into automated pipelines, batch processing workflows, and tool chains that require machine-parseable output.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L308]

## Vision & Objectives
Enable fully automated, scriptable execution of Cline CLI in non-interactive environments while maintaining the ability to provide rich interactive experiences when running in terminal environments. This creates a seamless dual-mode CLI that adapts automatically to its execution context.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L308]

## IAOOI System Components

### Inputs
1. Stdin file descriptor state (TTY vs pipe/redirect)
2. Stdout file descriptor state (TTY vs redirect)
3. Mode flags (`--json`, `--yolo`) for explicit mode control
4. Environment variable detection for non-interactive shells
5. Piped input content for script workflows
6. Output redirection targets (files, other commands)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L302]

### Activities
1. TTY detection using `isatty()` checks on stdin and stdout
2. Pipe/redirect detection for input and output streams
3. Automatic mode selection (interactive vs plain) based on detection results
4. Mode switching logic to configure appropriate output format and handlers
5. Explicit flag processing to override automatic detection when needed
6. Input stream reading for piped content injection into tasks
7. Output format configuration (plain text vs TUI rendering)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L302]

### Outputs
1. Mode decision (interactive vs plain text)
2. Configured output handlers appropriate for the mode
3. Plain text output when in non-interactive mode
4. Suppressed TUI rendering in scripted environments
5. Script-friendly responses and exit codes
6. JSON-formatted output when `--json` flag is specified

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L302]

### Outcomes
1. CLI works seamlessly in pipes, scripts, and CI/CD pipelines without user intervention
2. Automatic mode selection eliminates need for explicit mode flags in most cases
3. Consistent behavior across interactive and automated execution contexts
4. Output redirection produces clean, parseable results
5. Piped input can be used as context for AI tasks

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L303-L306]

### Impacts
1. Broader adoption in automation and DevOps workflows
2. Enhanced CI/CD integration capabilities for enterprises
3. Better compatibility with Unix philosophy (pipes, composition)
4. Reduced friction for scripting and batch processing use cases
5. Foundation for programmatic tool integration and orchestration

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L308]

## Key Features
- FEAT-AUTO-MODE-005-DETECT-001: TTY/Redirect Detection
- FEAT-AUTO-MODE-005-SWITCH-002: Mode Switching Logic

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L341]

## Business Value & Requirements
This epic addresses the following requirements:
- REQ-002: Support both interactive and plain text modes
- REQ-005: Auto-detect TTY/redirect for automatic mode switching

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L291]

## User Journeys & Scenarios

### DevOps Automation User Journey
The DevOps engineer needs to integrate Cline into CI/CD pipelines for automated code reviews, documentation generation, and deployment automation. They require:
- Reliable non-interactive execution
- JSON output for parsing by other tools
- Proper exit codes for pipeline decision-making
- No breaking prompts in automated environments
- Ability to pipe input from other tools (e.g., `git diff | cline 'review these changes'`)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L66-L78]

## BDD Scenarios

### Feature: TTY/Redirect Detection (FEAT-AUTO-MODE-005-DETECT-001)

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L334]

### Feature: Mode Switching Logic (FEAT-AUTO-MODE-005-SWITCH-002)

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L337-L341]

## Technical Considerations

### Existing Code References
This epic involves new GoLang CLI functionality with no direct dependencies on existing TypeScript CLI code per the independence requirements.

**Proposed New Components:**
- TTY detection utility using `golang.org/x/term` or similar
- Mode manager for switching between interactive and plain modes
- Stdin/stdout stream analyzers for pipe detection
- Environment detector for non-interactive shell detection
- Mode configuration state management

### Implementation Notes
1. **TTY Detection**: Use Go's `term.IsTerminal()` or `isatty` equivalent to check file descriptors
2. **Cross-Platform**: Ensure TTY detection works on Linux, macOS, and Windows
3. **Explicit Override**: Allow `--interactive` or similar flag to force TUI even when output is redirected (for debugging)
4. **Stdin Content**: Capture piped stdin for injection as context into AI tasks
5. **Early Detection**: Perform mode detection as early as possible in CLI initialization to avoid initializing Bubble Tea unnecessarily

### Dependencies
- Go terminal detection library (e.g., `golang.org/x/term`)
- File descriptor inspection utilities
- Environment variable access for shell detection

## Implementation Priority
**Phase 5** in the AI Execution Plan - Automation & Scripting (Plain Mode)

This epic should be implemented after core CLI foundation (Phase 2) and interactive UI (Phase 3), but before yolo mode (EPIC-AUTO-EXEC-006) as it provides the foundational mode detection that yolo mode builds upon.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L512-L526]

## Success Metrics
1. Automatic mode detection works correctly in 100% of TTY/pipe/redirect scenarios
2. Zero false positives (TUI in pipes) or false negatives (plain mode in interactive terminal)
3. Piped input successfully captured and passed to AI context
4. Exit codes properly returned in plain mode for script integration
5. Performance: Mode detection completes in <10ms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L303-L306]

## Dependencies
**Depends On:**
- EPIC-DEV-CLI-001: Command Line Interface Foundation (for command parsing and flag handling)
- EPIC-DEV-UI-002: Interactive Terminal UI (for TUI mode implementation)

**Required By:**
- EPIC-AUTO-EXEC-006: Automated Execution & Yolo Mode (builds on plain mode detection)
- EPIC-AUTO-OUT-007: Structured Output (JSON) (requires plain mode foundation)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L512-L540]

## Integration Points
**Integration with Other Epics:**
- **EPIC-DEV-CLI-001**: Uses command parsing and flag infrastructure to detect `--json` and `--yolo` flags that force plain mode
- **EPIC-DEV-UI-002**: Decides whether to initialize Bubble Tea TUI based on detection results
- **EPIC-AUTO-EXEC-006**: Provides mode detection foundation for yolo mode execution
- **EPIC-AUTO-OUT-007**: Enables JSON output by switching to plain mode

**Integration with Other Personas:**
- **Developer User**: Benefits from automatic mode when piping output to other dev tools
- **Enterprise User**: Relies on non-interactive mode for policy-compliant automation and audit trails

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L341]

## Related Requirements
| Requirement ID | Description |
|----------------|-------------|
| REQ-002 | Support both interactive and plain text modes |
| REQ-005 | Auto-detect TTY/redirect for automatic mode switching |

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L291]

## Traceability
| Epic/Feature ID | Requirement IDs |
|-----------------|-----------------|
| EPIC-AUTO-MODE-005 | REQ-002, REQ-005 |
| FEAT-AUTO-MODE-005-DETECT-001 | REQ-005 |
| FEAT-AUTO-MODE-005-SWITCH-002 | REQ-002 |

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L660-L668]

## Dual Testing Requirements
Per the dual testing mandate, all functionality in this epic must be tested in BOTH the existing TypeScript CLI AND the new GoLang CLI to ensure exact functional parity:

1. **TTY Detection Parity**: Verify identical behavior when piping input in both CLIs
2. **Redirect Detection Parity**: Confirm both CLIs switch to plain mode when output is redirected
3. **Flag Override Parity**: Ensure `--json` and `-y` flags force plain mode identically
4. **Exit Code Parity**: Verify exit codes are identical in plain mode between implementations

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L419-L449]

## Notes
- This epic is foundational for all automation use cases
- Mode detection must happen before any UI initialization to avoid rendering issues in pipes
- Consider edge cases like Windows consoles, SSH sessions, and terminal multiplexers (tmux, screen)