# TTY/Redirect Detection

## Feature ID
FEAT-AUTO-MODE-005-DETECT-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L334]

## Epic Context
**Parent Epic:** EPIC-AUTO-MODE-005 - Plain Text & Scripting Modes [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L1]
**Target Persona:** DevOps/Automation User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L66-L78]
**Epic Objective:** Enable the Cline CLI to operate in non-interactive environments such as CI/CD pipelines, shell scripts, and automated workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L308]
**Business Impact:** Enables broader adoption in automation and DevOps workflows, better compatibility with Unix philosophy (pipes, composition), reduced friction for scripting and batch processing use cases [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L308]

## Feature Overview
**Purpose:** Automatically detect when the GoLang CLI is running in a non-TTY environment (piped input or redirected output) and determine whether to operate in interactive TUI mode or plain text mode without requiring explicit user intervention or flags [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L320]

**Scope:** 
- **Included:** TTY detection on stdin and stdout, pipe/redirect detection, automatic mode selection, cross-platform support (Linux, macOS, Windows)
- **Excluded:** Explicit flag handling (handled by FEAT-AUTO-MODE-005-SWITCH-002), mode switching implementation details, output formatting

**PRD References:** REQ-002, REQ-005 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L660-L668]
**PRD Feature ID:** EPIC-AUTO-MODE-005-DETECT-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317]

**Dependencies:** 
- Proposed: New TTY detection utility using `golang.org/x/term` or similar library
- Proposed: Stdin/stdout stream analyzers for pipe detection
- Proposed: Mode configuration state management
- Required By: EPIC-DEV-UI-002 (decides whether to initialize Bubble Tea TUI), EPIC-AUTO-EXEC-006 (provides mode detection for yolo mode)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L320 - Feature IAOOI section]

**Inputs:**
1. Stdin file descriptor state (TTY vs pipe/redirect)
2. Stdout file descriptor state (TTY vs redirect)
3. Environment variable detection for non-interactive shells
4. Piped input content for script workflows
5. Output redirection targets (files, other commands)

**Activities:**
1. TTY detection using `isatty()` checks on stdin and stdout
2. Pipe/redirect detection for input and output streams
3. Automatic mode selection (interactive vs plain) based on detection results
4. Input stream reading for piped content injection into tasks
5. Cross-platform terminal capability detection

**Outputs:**
1. Mode decision (interactive vs plain text)
2. Detection results for stdin TTY status
3. Detection results for stdout TTY status
4. Piped input content capture (if applicable)
5. Mode configuration for downstream components

**Outcomes:**
1. CLI works seamlessly in pipes, scripts, and CI/CD pipelines without user intervention
2. Automatic mode selection eliminates need for explicit mode flags in most cases
3. Consistent behavior across interactive and automated execution contexts
4. Output redirection produces clean, parseable results
5. Piped input can be used as context for AI tasks

**Impacts:**
1. Broader adoption in automation and DevOps workflows
2. Enhanced CI/CD integration capabilities for enterprises
3. Better compatibility with Unix philosophy (pipes, composition)
4. Reduced friction for scripting and batch processing use cases
5. Foundation for programmatic tool integration and orchestration

## Technical Requirements
**Architecture Layer:** Application/CLI Infrastructure Layer

**Integration Points:**
- Proposed: New TTY detection module (to be created in `golang-cli/internal/terminal/` or similar)
- Proposed: Mode detection results passed to TUI initialization layer
- Proposed: Piped input reader for task context injection
- Integration with: EPIC-DEV-UI-002 (Bubble Tea TUI initialization decision)
- Integration with: EPIC-AUTO-MODE-005-SWITCH-002 (mode switching logic consumes detection results)

**Data Requirements:**
- Proposed: Mode state (interactive/plain/unknown) stored in application context
- Proposed: TTY status flags for stdin/stdout
- Proposed: Piped input buffer for task context

**Performance Requirements:**
- Mode detection must complete in <10ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L341]
- Detection must occur before any UI initialization to avoid rendering issues in pipes [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L341]

**Security Requirements:**
- No sensitive data exposure in detection process
- Proper handling of piped input without logging sensitive content

## User Experience
**User Personas:** DevOps/Automation User - engineers integrating Cline into CI/CD pipelines, shell scripts, and automated workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L66-L78]

**User Actions:**
1. Run Cline in terminal (interactive mode) - `cline`
2. Pipe input to Cline - `cat file.txt | cline 'summarize this'`
3. Redirect output from Cline - `cline 'list files' > output.txt`
4. Use Cline in shell scripts with piped input
5. Use Cline in CI/CD pipelines with redirected output

**UI Components:** 
- Proposed: No UI components (this is a detection layer feature)
- Proposed: Debug/logging output for detection results when verbose mode enabled

**Cross-Platform Considerations:**
- Must work on Linux (xterm, console), macOS (Terminal.app, iTerm2), Windows (Command Prompt, PowerShell, Windows Terminal)
- Must handle SSH sessions correctly
- Must work with terminal multiplexers (tmux, screen)
- Windows console detection requires platform-specific handling

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L334 - Gherkin scenarios for this feature]

```gherkin
Feature: TTY/Redirect Detection (FEAT-AUTO-MODE-005-DETECT-001)

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

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L341 - Success Metrics]

**Functional:**
- [ ] Automatic mode detection works correctly in 100% of TTY/pipe/redirect scenarios
- [ ] Zero false positives (TUI in pipes) or false negatives (plain mode in interactive terminal)
- [ ] Piped input successfully captured and passed to AI context
- [ ] Detection completes before any UI initialization

**Performance:**
- [ ] Mode detection completes in <10ms
- [ ] No perceptible delay in CLI startup

**Quality:**
- [ ] Works across all supported platforms (Linux, macOS, Windows)
- [ ] Handles edge cases (SSH, tmux, screen, Windows consoles)
- [ ] Proper error handling for detection failures

**Integration:**
- [ ] Detection results correctly consumed by mode switching logic (FEAT-AUTO-MODE-005-SWITCH-002)
- [ ] Detection results correctly consumed by TUI initialization (EPIC-DEV-UI-002)
- [ ] Works seamlessly with yolo mode (EPIC-AUTO-EXEC-006)

**Business Value:**
- [ ] Enables CI/CD pipeline integration
- [ ] Enables shell script automation
- [ ] Maintains seamless experience for interactive users

## Testing Strategy
**Unit Testing:**
- Test TTY detection with mocked file descriptors
- Test pipe detection scenarios
- Test cross-platform detection logic
- Test edge cases (SSH sessions, multiplexers)

**Integration Testing:**
- Test detection with actual piped input
- Test detection with actual output redirection
- Test detection in Docker containers (common CI/CD scenario)
- Test detection in GitHub Actions environment

**User Acceptance:**
- Manual verification in terminal, piped, and redirected scenarios
- Cross-platform manual testing
- CI/CD pipeline integration testing

**Dual Testing Requirements (Per Epic Mandate):**
- TTY Detection Parity: Verify identical behavior when piping input in both TypeScript and GoLang CLIs
- Redirect Detection Parity: Confirm both CLIs switch to plain mode when output is redirected
- Exit Code Parity: Verify exit codes are identical in plain mode between implementations
- Run test scenarios against both CLIs and compare outputs [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L419-L449]

## Tasks Overview
1. **Task 1:** Implement TTY detection utility using `golang.org/x/term` library
2. **Task 2:** Implement stdin pipe/redirect detection
3. **Task 3:** Implement stdout redirect detection
4. **Task 4:** Create mode decision logic based on detection results
5. **Task 5:** Implement piped input content capture
6. **Task 6:** Add cross-platform support (Windows console handling)
7. **Task 7:** Write unit tests for detection functions
8. **Task 8:** Write integration tests for pipe/redirect scenarios
9. **Task 9:** Perform dual testing against TypeScript CLI for parity

## Implementation Notes
1. **Early Detection**: Mode detection must happen before any UI initialization to avoid rendering issues in pipes [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L341]
2. **Library Choice**: Use Go's `term.IsTerminal()` from `golang.org/x/term` or `isatty` equivalent to check file descriptors [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L334]
3. **Cross-Platform**: Ensure TTY detection works on Linux, macOS, and Windows - Windows may need special handling [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L334]
4. **Stdin Content**: Capture piped stdin for injection as context into AI tasks [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L334]
5. **Edge Cases**: Consider SSH sessions, terminal multiplexers (tmux, screen), and Windows consoles [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L341]

## Dependencies
**Go Libraries:**
- `golang.org/x/term` - Terminal detection utilities
- Standard library `os` - File descriptor access

**Required By:**
- EPIC-DEV-UI-002 (Interactive Terminal UI) - needs detection result to decide TUI initialization
- EPIC-AUTO-MODE-005-SWITCH-002 (Mode Switching Logic) - consumes detection results
- EPIC-AUTO-EXEC-006 (Automated Execution & Yolo Mode) - builds on plain mode detection

## Traceability
| Feature ID | Requirement IDs | Epic ID |
|------------|-----------------|---------|
| FEAT-AUTO-MODE-005-DETECT-001 | REQ-005 | EPIC-AUTO-MODE-005 |

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L317-L341]
- [x] Existing code references cite actual file paths and lines (N/A - new feature with no existing code dependencies per independence requirements)
- [x] New functionality clearly marked as "Proposed:" when it doesn't exist yet
- [x] Integration points cite existing interfaces or marked as new
- [x] Epic source cited: [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md]