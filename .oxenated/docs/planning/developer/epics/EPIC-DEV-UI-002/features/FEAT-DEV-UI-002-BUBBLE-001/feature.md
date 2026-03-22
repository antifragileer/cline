# Bubble Tea TUI Framework Setup

## Feature ID
FEAT-DEV-UI-002-BUBBLE-001

## Source
**PRD Source:** [.oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L318](.oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L318)

## Epic Context
**Parent Epic:** EPIC-DEV-UI-002 - Interactive Terminal UI [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Deliver a rich, interactive terminal experience for the GoLang Cline CLI using the Bubble Tea TUI framework, achieving feature parity with the existing TypeScript/React Ink CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L272-L285]
**Business Impact:** Provides a responsive terminal interface that handles real-time message streaming, keyboard interactions, and professional chat rendering; foundation for all subsequent TUI features [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281-L282]

## Feature Overview
**Purpose:** Initialize and configure the Bubble Tea TUI framework to create a responsive, interactive terminal user interface that serves as the foundation for all CLI interactive mode functionality [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L293]
**Scope:** 
- Included: Bubble Tea program initialization, terminal dimension detection, keyboard event handling, window resize handling, theme integration, basic frame rendering
- Excluded: Chat message rendering (FEAT-DEV-UI-002-CHAT-002), user input handling (FEAT-DEV-UI-002-INPUT-003), welcome screen (FEAT-DEV-UI-002-WELCOME-004) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L401]
**PRD References:** REQ-001 (Execute AI coding tasks from terminal with interactive UI) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1228]
**PRD Feature ID:** EPIC-DEV-UI-002-BUBBLE-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L318]
**Dependencies:** 
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - Required for command routing and TTY detection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L247-L271]
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for message streaming integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L905-L948]
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for theme preferences access [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L949-L1004]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L296-L302]

**Inputs:**
- Terminal capabilities (TTY detection) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Theme preferences stored in global state (~/.cline/data/globalState.json) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Terminal dimensions (width, height) at startup and on resize events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L296-L299]
- Initial state configuration from command invocation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L296-L299]
- Keyboard input events from terminal [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]

**Activities:**
- Initialize Bubble Tea program with Model-Update-View architecture [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L300]
- Setup update loop for handling messages and events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L300]
- Handle window resize events and reflow content appropriately [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L300]
- Process keyboard events for navigation and shortcuts [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Load and apply theme configuration from storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Initialize event handlers for TUI interactions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L300]

**Outputs:**
- Running TUI application with active event loop [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L300-L301]
- Event handlers registered for keyboard and resize events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L300-L301]
- Rendered frames updated at 60fps target [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Terminal state management (alternate screen buffer if supported) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Clean terminal restoration on exit [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]

**Outcomes:**
- Responsive terminal UI that handles keyboard navigation and resize events gracefully [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L301-L302]
- Foundation for all subsequent TUI components (chat, input, welcome screen) [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]
- Consistent theming across sessions based on user preferences [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L301-L302]
- TUI initializes in <100ms from command invocation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]

**Impacts:**
- Professional interactive experience that rivals the VSCode extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302]
- Reduced latency compared to Node.js/React Ink implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]
- Foundation for future TUI enhancements and features [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]
- Higher user satisfaction due to improved responsiveness [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]

## Technical Requirements

**Architecture Layer:** UI/Presentation Layer (Bubble Tea Model-Update-View pattern)

**Integration Points:**
- Proposed: New Bubble Tea Model implementation in `golang-cli/internal/tui/model.go`
- Proposed: Update handler in `golang-cli/internal/tui/update.go` for message processing
- Proposed: View renderer in `golang-cli/internal/tui/view.go` for frame generation
- Proposed: Theme/styles module in `golang-cli/internal/tui/styles/theme.go`
- Integration with EPIC-INFRA-CORE-011: gRPC client provides message stream to TUI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L905-L948]
- Integration with EPIC-INFRA-STORAGE-012: Reads theme from ~/.cline/data/globalState.json [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L949-L1004]

**Data Requirements:**
- Existing: Theme preferences from globalState.json [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Proposed: TUI state model (screen dimensions, current view, focus state)
- Proposed: Message queue for handling async updates from gRPC stream

**Performance Requirements:**
- TUI initialization: <100ms from command invocation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Frame rendering latency: <16ms per frame (60fps target) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Resize handling: <50ms to reflow content [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Support terminal widths: 80-240 columns [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Zero crashes during 1000 consecutive message streaming sessions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]

**Security Requirements:**
- Terminal state cleanup on exit (restore cursor, clear alternate screen buffer) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Graceful handling of SIGINT/SIGTERM signals [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- No sensitive data logged to terminal title or history

## User Experience

**User Personas:** Developer User (primary), DevOps/Automation User (when interactive mode selected)

**User Actions:**
1. Launch CLI in interactive mode (`cline` without arguments) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302-L306]
2. Resize terminal window during active session [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L311]
3. Navigate using keyboard shortcuts (Tab, Arrow keys, Enter, Esc, Ctrl+C) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312-L317]
4. Exit TUI gracefully using 'q' or Ctrl+C [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]

**UI Components:**
- Proposed: Main TUI container with Bubble Tea program loop
- Proposed: Status bar showing connection state and current mode
- Proposed: Responsive layout manager handling terminal dimensions
- Proposed: Theme-aware color and styling system
- Reference: Existing TypeScript CLI colors in `cli/src/constants/colors.ts` [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L94]

**Mobile Considerations:** N/A - Terminal UI focused on desktop/laptop development environments

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302-L317]

```gherkin
Scenario: Initialize TUI
  Given the terminal supports TTY
  When the CLI starts in interactive mode
  Then the Bubble Tea program should initialize
  And the initial view should render
  And the TUI should be ready for user input within 100ms
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302-L306]

Scenario: Handle window resize
  Given the TUI is running
  When the user resizes the terminal window
  Then the UI should adapt to new dimensions
  And content should reflow appropriately
  And the layout should remain usable at the new size
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L311]

Scenario: Handle keyboard events
  Given the TUI is running
  When the user presses keyboard keys
  Then the appropriate actions should trigger
  And focus should move correctly between components
  And the event loop should remain responsive
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312-L317]

Scenario: Terminal state cleanup on exit
  Given the TUI is running in alternate screen buffer
  When the user exits the application
  Then the terminal should restore to original state
  And the cursor should be visible and positioned correctly
  And the main screen buffer should be restored
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]

Scenario: Handle non-TTY environment gracefully
  Given the terminal does not support TTY
  When the CLI attempts to start interactive mode
  Then it should detect the non-interactive environment
  And fall back to plain text mode
  And display an informative message about the mode switch
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L425-L429]
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355-L1375]

**Functional:**
- Bubble Tea program initializes successfully in TTY environments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302-L306]
- All keyboard events are captured and routed correctly [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312-L317]
- Window resize events trigger appropriate content reflow [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L311]
- Terminal state is properly restored on exit [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Theme preferences are loaded and applied from global state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]

**Performance:**
- TUI initialization completes in <100ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Frame rendering maintains 60fps (16ms max per frame) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Resize handling completes in <50ms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Zero memory leaks during extended usage sessions

**Quality:**
- Zero crashes during 1000 consecutive initialization cycles [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]
- Graceful degradation when terminal features are limited
- Proper cleanup even on unexpected termination (SIGTERM, SIGINT)
- Support for terminal widths from 80 to 240 columns [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355]

**Integration:**
- TUI framework integrates seamlessly with gRPC message streaming [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L905-L948]
- Theme system reads correctly from EPIC-INFRA-STORAGE-012 state layer [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L949-L1004]
- Foundation supports all downstream TUI features (chat, input, welcome) [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]

**Business Value:**
- Achieves feature parity with existing React Ink TUI initialization [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355-L1375]
- Provides foundation for 100% of interactive mode functionality [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281-L282]
- Enables all subsequent UI epic features [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]

## Testing Strategy

**Unit Testing:**
- Bubble Tea Model initialization and state management
- Update handler message processing (keyboard, resize, custom messages)
- View rendering functions with various terminal dimensions
- Theme loading and color application
- Terminal capability detection (isatty mocks)

**Integration Testing:**
- Integration with mock gRPC client for message streaming
- Integration with storage layer for theme retrieval
- Keyboard event routing to appropriate handlers
- Resize event handling with content reflow verification

**User Acceptance:**
- Manual verification of TUI startup time (<100ms)
- Visual confirmation of proper theming
- Keyboard navigation testing across components
- Resize testing at various terminal dimensions (80x24, 120x40, 240x60)
- Exit behavior verification (terminal state restoration)

**Performance Testing:**
- Startup time benchmarking across platforms
- Frame rate measurement under load
- Memory profiling during extended sessions
- Resize handling latency measurement

**Dual Testing (Critical):**
- Side-by-side comparison with existing TypeScript CLI TUI initialization
- Verify identical behavior for keyboard shortcuts
- Compare startup performance (Go must be faster or equal)
- Test concurrent execution of both CLIs against same core extension

## Tasks Overview
1. **Setup Bubble Tea project structure** - Create initial Model, Update, View files and program initialization
2. **Implement terminal capability detection** - TTY detection, dimension querying, feature detection
3. **Build theme integration** - Load and apply themes from globalState.json, color definitions
4. **Create event handling system** - Keyboard input processing, resize event handling, message routing
5. **Implement frame rendering** - View layout system, responsive design, 60fps rendering target
6. **Add terminal state management** - Alternate screen buffer, cleanup handlers, signal handling
7. **Performance optimization** - Reduce initialization time, optimize render loop, memory efficiency

## Implementation Notes

**Key Go Dependencies:**
- `github.com/charmbracelet/bubbletea` - Core TUI framework [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L102]
- `github.com/charmbracelet/lipgloss` - Styling and layout [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L102]
- `github.com/charmbracelet/bubbles` - Pre-built TUI components [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L102]

**Proposed File Structure:**
```
golang-cli/
├── internal/
│   └── tui/
│       ├── model.go          # Bubble Tea Model definition
│       ├── update.go         # Update message handlers
│       ├── view.go           # View rendering functions
│       ├── init.go           # Program initialization
│       ├── styles/
│       │   └── theme.go      # Color and style definitions
│       └── messages/
│           └── types.go      # Custom Bubble Tea messages
```

**Critical Independence Requirement:**
This feature MUST NOT import or depend on any TypeScript/JavaScript code from `cli/src/` or the existing CLI. All implementation must be pure Go using Go-native libraries only [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L15-L27].

**Phase Alignment:**
This feature is part of Phase 3: Interactive UI Development in the AI Execution Plan [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1281].

**Downstream Dependencies:**
- FEAT-DEV-UI-002-CHAT-002 requires this framework for message rendering [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]
- FEAT-DEV-UI-002-INPUT-003 requires this framework for input handling [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]
- FEAT-DEV-UI-002-WELCOME-004 requires this framework for welcome screen [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L70-L73]
- EPIC-DEV-TASK-003 depends on this TUI for task execution display [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L402-L505]

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md]
- [x] Existing code references cite actual file paths (cli/src/constants/colors.ts) [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L94]
- [x] New functionality clearly marked as "Proposed:" (golang-cli internal structure)
- [x] Integration points cite existing interfaces or mark as new (EPIC references)
- [x] Epic dependencies properly cited with PRD line references
- [x] BDD scenarios extracted verbatim from PRD with citations
- [x] Success criteria aligned with epic success metrics
- [x] Critical independence requirements cited from PRD