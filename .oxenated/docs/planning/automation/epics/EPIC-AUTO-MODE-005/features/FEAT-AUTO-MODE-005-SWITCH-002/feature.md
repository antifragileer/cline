# Mode Switching Logic

## Feature ID
FEAT-AUTO-MODE-005-SWITCH-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L341]

## Epic Context
**Parent Epic:** EPIC-AUTO-MODE-005 - Plain Text & Scripting Modes [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/epic.md:L1]
**Target Persona:** DevOps/Automation User
**Epic Objective:** Enable the Cline CLI to operate in non-interactive environments such as CI/CD pipelines, shell scripts, and automated workflows with automatic TTY/redirect detection and mode switching [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L290-L308]
**Business Impact:** Creates a seamless dual-mode CLI that adapts automatically to its execution context, enabling broader adoption in automation and DevOps workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L308]

## Feature Overview
**Purpose:** Implement the logic that switches between interactive TUI mode and plain text mode based on TTY detection results and explicit user flags. This feature configures the appropriate output format, initializes the correct handlers, and ensures correct behavior in all execution contexts (interactive terminal, piped input, redirected output, and explicit flag overrides). [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L341]

**Scope:** 
- Mode decision logic based on TTY detection and explicit flags
- Output handler configuration for interactive vs plain modes
- Plain text output formatting when in non-interactive mode
- TUI suppression in scripted environments
- Script-friendly response handling and exit codes
- JSON-formatted output when `--json` flag is specified

**Excluded:** 
- TTY/redirect detection itself (handled by FEAT-AUTO-MODE-005-DETECT-001)
- Bubble Tea TUI implementation (handled by EPIC-DEV-UI-002)
- Yolo mode auto-approval logic (handled by EPIC-AUTO-EXEC-006)

**PRD References:** REQ-002 (Support both interactive and plain text modes) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L660-L668]
**Dependencies:** 
- FEAT-AUTO-MODE-005-DETECT-001 (TTY/Redirect Detection) - provides detection results
- EPIC-DEV-UI-002 (Interactive Terminal UI) - provides TUI implementation to conditionally initialize
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - provides flag parsing infrastructure

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L311-L341 - Feature 2: Mode Switching Logic]

**Inputs:**
1. TTY detection results (stdin is TTY, stdout is TTY, pipe/redirect detected) from FEAT-AUTO-MODE-005-DETECT-001
2. Explicit mode flags: `--json` for JSON output format
3. Yolo flag: `-y` or `--yolo` for automated execution mode
4. Mode override flags (e.g., `--interactive` for debugging)
5. Environment variables indicating non-interactive shells

**Activities:**
1. Evaluate TTY detection results to determine default mode (interactive vs plain)
2. Process explicit flags (`--json`, `-y`) that force plain mode regardless of TTY state
3. Switch between interactive and plain output modes
4. Configure output format (plain text vs TUI rendering)
5. Initialize appropriate output handlers for selected mode
6. Set up JSON serialization when `--json` flag is present
7. Enable auto-approval pipeline when `-y` flag is present
8. Handle mode transition edge cases (e.g., mid-task mode switches)

**Outputs:**
1. Mode decision (interactive vs plain text) finalized
2. Configured output handlers appropriate for the selected mode
3. Plain text output configuration when in non-interactive mode
4. Suppressed TUI rendering configuration for scripted environments
5. Script-friendly response formatting and exit codes configured
6. JSON-formatted output stream when `--json` flag is specified
7. Auto-approval enabled state when `-y` flag is present

**Outcomes:**
1. Correct behavior in all execution contexts (TTY, pipes, redirects, flags)
2. Seamless mode switching without user intervention in most cases
3. Consistent output formatting expectations across environments
4. Reliable exit codes for script integration
5. Proper handling of explicit flag overrides for debugging and automation

**Impacts:**
1. Better compatibility with Unix philosophy (pipes, composition)
2. Reduced friction for scripting and batch processing use cases
3. Foundation for programmatic tool integration and orchestration
4. Enhanced CI/CD integration capabilities for enterprises

## Technical Requirements
**Architecture Layer:** Application Layer (CLI initialization and configuration)

**Integration Points:**
- **Proposed:** Mode Manager component to centralize mode state and switching logic
- **Proposed:** Output Handler Factory for creating appropriate output handlers based on mode
- **Proposed:** Configuration bridge to pass mode settings to gRPC client initialization
- **Depends on:** TTY Detection module (FEAT-AUTO-MODE-005-DETECT-001) for input signals
- **Integrates with:** Bubble Tea TUI framework (EPIC-DEV-UI-002) - conditionally initializes
- **Integrates with:** JSON Output Formatter (EPIC-AUTO-OUT-007) - delegates JSON formatting

**Data Requirements:**
- Mode state storage (in-memory during CLI lifecycle)
- Flag state from command parsing
- Environment variable cache

**Performance Requirements:**
- Mode switching decision must complete in <1ms
- Handler initialization must not block CLI startup
- Mode detection and switching combined must complete in <10ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312]

**Security Requirements:**
- No special security considerations for mode switching itself
- Output handlers must not expose sensitive data in plain mode logs

## User Experience
**User Personas:** DevOps/Automation User, Developer User (when debugging)

**User Actions:**
1. Run CLI in terminal → automatically uses interactive TUI mode
2. Pipe input to CLI → automatically switches to plain text mode
3. Redirect output to file → automatically switches to plain text mode
4. Run with `--json` flag → forces plain mode with JSON output
5. Run with `-y` flag → forces plain mode with auto-approval
6. Run with `--interactive` flag → forces TUI mode even when redirected (debugging)

**UI Components:**
- **Proposed:** Mode indicator in debug/verbose output showing current mode
- **Proposed:** Mode transition messages when verbose logging enabled

**Scripting Considerations:**
- Exit codes must be reliable and consistent in plain mode
- Output should be deterministic (no progress spinners, no ANSI codes unless TTY)
- JSON output must be valid JSON lines (newline-delimited JSON)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L337-L341 - Mode Switching Logic scenarios]

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

**Additional Derived Scenarios:**

```gherkin
Scenario: TTY detection selects interactive mode
  Given the CLI detects stdin and stdout are both TTYs
  And no explicit mode flags are provided
  When the CLI initializes
  Then interactive TUI mode should activate
  And Bubble Tea TUI should initialize

Scenario: Pipe detection selects plain mode
  Given the CLI detects stdin is a pipe
  And no explicit mode flags are provided
  When the CLI initializes
  Then plain text mode should activate
  And TUI should not initialize

Scenario: Output redirect selects plain mode
  Given the CLI detects stdout is redirected to a file
  And no explicit mode flags are provided
  When the CLI initializes
  Then plain text mode should activate
  And output should not contain ANSI escape codes

Scenario: Explicit interactive flag overrides redirect
  Given the user runs "cline --interactive 'task' > output.txt"
  When the CLI initializes
  Then interactive TUI mode should activate
  And the TUI should render (useful for debugging)

Scenario: Combined yolo and JSON flags
  Given the user runs "cline -y --json 'automated analysis'"
  When the CLI initializes
  Then plain text mode should activate
  And JSON output should be enabled
  And auto-approval should be enabled

Scenario: Mode selection priority
  Given TTY is detected but --json flag is provided
  When the CLI initializes
  Then explicit flag should take precedence
  And plain text mode with JSON should activate
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L303-L312]

**Functional:**
- Mode switching correctly responds to TTY detection results [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L302]
- Explicit flags (`--json`, `-y`) correctly force plain mode
- Interactive mode only activates when both stdin and stdout are TTYs (unless overridden)
- Plain mode activates when either stdin or stdout is not a TTY
- Mode decision is made before any UI initialization to avoid rendering issues

**Performance:**
- Mode decision completes in <1ms
- Total mode detection and switching completes in <10ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312]

**Quality:**
- Zero false positives (TUI rendered in pipes/redirections)
- Zero false negatives (plain mode in interactive terminals)
- Consistent behavior across Linux, macOS, and Windows

**Integration:**
- Works seamlessly with TTY detection module
- Integrates correctly with TUI framework initialization
- Integrates correctly with JSON output formatter
- Integrates correctly with yolo mode auto-approval

**Business Value:**
- Enables reliable CI/CD pipeline integration
- Supports batch processing workflows
- Maintains interactive experience when appropriate

## Testing Strategy
**Unit Testing:**
- Mode decision logic with mocked TTY detection results
- Flag parsing and priority ordering
- Handler factory creates correct handler types

**Integration Testing:**
- End-to-end mode switching with actual TTY/pipe scenarios
- Integration with TTY detection module
- Integration with TUI framework (verify conditional initialization)
- Integration with JSON output formatter

**User Acceptance:**
- Manual verification in terminal, pipe, and redirect scenarios
- Cross-platform testing (Linux, macOS, Windows)
- Exit code verification in plain mode

**Dual Testing Requirements:**
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L419-L449]:
- Verify identical mode switching behavior when using explicit flags in both CLIs
- Confirm both CLIs force plain mode with `--json` and `-y` flags identically
- Test mode priority rules match between implementations

## Tasks Overview
1. **Design Mode Manager Component** - Define interfaces and state management for mode switching
2. **Implement Mode Decision Logic** - Build logic combining TTY detection and explicit flags
3. **Create Output Handler Factory** - Implement factory pattern for handler initialization
4. **Integrate with CLI Initialization** - Wire mode switching into CLI startup sequence
5. **Add Flag Processing** - Implement `--json`, `-y`, and `--interactive` flag handlers
6. **Implement Mode Indicators** - Add debug/verbose mode display

## Implementation Notes

### Mode Decision Priority (Highest to Lowest)
1. Explicit `--interactive` flag → Force interactive mode
2. Explicit `--json` or `-y` flag → Force plain mode
3. TTY detection results → Auto-select based on stdin/stdout state
   - Both TTY → Interactive mode
   - Either non-TTY → Plain mode

### Early Detection Requirement
Mode detection and switching MUST happen as early as possible in CLI initialization to avoid:
- Initializing Bubble Tea TUI unnecessarily in pipes
- Rendering TUI elements that won't be visible
- Performance overhead in scripted environments

### Cross-Platform Considerations
- Windows consoles may require special handling for TTY detection
- SSH sessions should be treated as TTY when appropriate
- Terminal multiplexers (tmux, screen) should preserve TTY detection

### Integration with Existing TypeScript CLI
The GoLang CLI implementation must achieve exact functional parity with the existing TypeScript CLI's mode switching behavior. This includes:
- Identical flag behavior
- Identical TTY detection integration
- Identical output handler selection logic

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines (none applicable - new feature)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or marked as new
- [x] Citation Verification checklist completed