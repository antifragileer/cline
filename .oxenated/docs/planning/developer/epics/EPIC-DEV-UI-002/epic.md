# Interactive Terminal UI

## Epic ID
EPIC-DEV-UI-002

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L272-L285 - Epic 2: Interactive Terminal UI]

## Target Persona
Developer User

## Epic Overview
The Interactive Terminal UI epic delivers a rich, interactive terminal experience for the GoLang Cline CLI using the Bubble Tea TUI framework. This epic achieves feature parity with the existing TypeScript/React Ink CLI, providing users with a responsive terminal interface that handles real-time message streaming, keyboard interactions, and professional chat rendering. The TUI will render markdown-formatted messages, handle window resizing, manage focus states, and provide an onboarding experience for new users.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L272-L285]

## Vision & Objectives
The Interactive Terminal UI enables developers to engage with Cline's AI coding assistant through a rich, terminal-native interface that rivals the VSCode extension experience. By leveraging Go's Bubble Tea framework, we achieve better performance and responsiveness while maintaining the familiar interactive chat experience users expect.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L272-L285]

## IAOOI System Components

### Inputs
- Terminal capabilities (TTY detection) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Theme preferences stored in global state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Message streams from gRPC core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- User keyboard input events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]
- Terminal dimension changes (resize events) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L278]

### Activities
- Render Bubble Tea TUI components and frames [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Handle keyboard events for navigation and input [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Display streaming messages in real-time [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Manage focus states (chat view, input field, prompts) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Render markdown content with syntax highlighting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]
- Handle window resize events and reflow content [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279]

### Outputs
- Interactive terminal interface with chat view [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Styled message bubbles with proper formatting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Progress indicators for streaming content [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Input prompts for user responses [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Welcome screen and onboarding guidance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]
- Syntax-highlighted code blocks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L280]

### Outcomes
- Rich interactive experience comparable to VSCode extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]
- Real-time feedback during AI response streaming [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]
- Professional chat interface in the terminal [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]
- Reduced latency compared to Node.js/React Ink implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]
- Consistent cross-platform terminal behavior [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L281]

### Impacts
- Higher user satisfaction due to improved responsiveness [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]
- Increased user engagement with the CLI tool [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]
- Feature parity across VSCode, CLI, and JetBrains platforms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]
- Foundation for future TUI enhancements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]
- Competitive advantage through superior terminal experience [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L282]

## Key Features
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L368]

1. **FEAT-DEV-UI-002-BUBBLE-001**: Bubble Tea TUI Framework Setup [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L318]
2. **FEAT-DEV-UI-002-CHAT-002**: Chat Message Rendering Components [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L319-L349]
3. **FEAT-DEV-UI-002-INPUT-003**: User Input Handling and Prompts [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L350-L375]
4. **FEAT-DEV-UI-002-WELCOME-004**: Welcome Screen and Onboarding [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L376-L401]

## Business Value & Requirements
This epic directly addresses the following requirements:
- **REQ-001**: Execute AI coding tasks from terminal with interactive UI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L274]
- **REQ-002**: Support both interactive and plain text modes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L274]

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1228-L1229 - Traceability Matrix]

## User Journeys & Scenarios

### Primary User Journey: Interactive Coding Session
The Developer User launches Cline in interactive mode, initiates a coding task, observes streaming AI responses with syntax highlighting, provides follow-up input, and approves tool suggestions—all within the terminal TUI.

### BDD Scenarios

#### Feature 1: Bubble Tea TUI Framework Setup (FEAT-DEV-UI-002-BUBBLE-001)

```gherkin
Scenario: Initialize TUI
  Given the terminal supports TTY
  When the CLI starts in interactive mode
  Then the Bubble Tea program should initialize
  And the initial view should render
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302-L306]

Scenario: Handle window resize
  Given the TUI is running
  When the user resizes the terminal window
  Then the UI should adapt to new dimensions
  And content should reflow appropriately
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L307-L311]

Scenario: Handle keyboard events
  Given the TUI is running
  When the user presses keyboard keys
  Then the appropriate actions should trigger
  And focus should move correctly
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L312-L317]
```

#### Feature 2: Chat Message Rendering Components (FEAT-DEV-UI-002-CHAT-002)

```gherkin
Scenario: Render text message
  Given a text message from Cline
  When it displays in the UI
  Then it should render with proper styling
  And markdown should be formatted
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L333-L338]

Scenario: Render code block
  Given a message containing code
  When it displays in the UI
  Then syntax highlighting should apply
  And the code should be readable
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L339-L343]

Scenario: Stream message updates
  Given a streaming response from AI
  When chunks arrive
  Then the message should update in real-time
  And partial content should display
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L344-L348]
```

#### Feature 3: User Input Handling and Prompts (FEAT-DEV-UI-002-INPUT-003)

```gherkin
Scenario: Respond to approval prompt
  Given Cline asks for tool approval
  When the prompt displays
  Then the user should be able to approve or reject
  And the response should send to agent
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L364-L369]

Scenario: Type follow-up message
  Given a task is active
  When the user types a message and submits
  Then the message should send to Cline
  And display in the chat history
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L370-L374]
```

#### Feature 4: Welcome Screen and Onboarding (FEAT-DEV-UI-002-WELCOME-004)

```gherkin
Scenario: Show welcome screen for new user
  Given the user runs Cline for first time
  When the interactive mode starts
  Then the welcome screen should display
  And quick start hints should show
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L390-L394]

Scenario: Show recent tasks on welcome
  Given the user has previous tasks
  When the interactive mode starts
  Then recent tasks should display for quick resumption
  [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L399]
```

## Technical Considerations

### Existing Code References
This epic implements the GoLang equivalent of the existing TypeScript CLI's React Ink-based UI. The existing implementation resides in:
- `cli/src/components/` - React Ink components [Source: cli/src/components/]
- `cli/src/constants/colors.ts` - Terminal color definitions [Source: cli/src/constants/colors.ts]

**Important**: The GoLang implementation MUST NOT import or transpile any TypeScript/JavaScript code from the existing CLI per the Critical Independence Requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L15-L27].

### Proposed New Components

**GoLang Implementation Structure:**
```
golang-cli/
├── cmd/
│   └── cline/
│       └── main.go              # Entry point
├── internal/
│   ├── tui/
│   │   ├── model.go             # Bubble Tea Model
│   │   ├── update.go            # Message handling
│   │   ├── view.go              # Render functions
│   │   ├── components/
│   │   │   ├── chat.go          # Chat message component
│   │   │   ├── input.go         # User input component
│   │   │   ├── welcome.go       # Welcome screen
│   │   │   └── markdown.go      # Markdown renderer
│   │   └── styles/
│   │       └── theme.go         # Color and style definitions
│   └── grpc/
│       └── client.go            # Core extension communication
└── pkg/
    └── messages/
        └── types.go             # Message type definitions
```

**Key Go Dependencies (Proposed):**
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Pre-built components
- `github.com/yuin/goldmark` - Markdown parsing
- `github.com/alecthomas/chroma` - Syntax highlighting

### Integration Points

**Integration with EPIC-DEV-CLI-001 (Command Line Interface Foundation):**
- The TUI initializes when the root command runs without arguments or when explicitly requested
- Subcommands (task, history, config) may launch TUI components for interactive modes

**Integration with EPIC-DEV-TASK-003 (Task Management):**
- TUI receives message streams from task execution via gRPC
- TUI renders tool approval prompts and captures user responses
- TUI displays conversation history during task resumption

**Integration with EPIC-INFRA-CORE-011 (Core Extension Integration):**
- Bidirectional gRPC streaming provides message data to the TUI
- TUI sends user responses back to the core extension

## Implementation Priority
**Phase 3: Interactive UI Development** in the AI Execution Plan [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1281]

This epic is scheduled for Phase 3, following the foundational CLI setup (Phase 2) and preceding task management implementation (Phase 4). The Bubble Tea framework setup must be completed before task management can integrate with the UI.

**Dependencies:**
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - Required for command routing
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for message streaming
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for theme preferences

## Success Metrics
- TUI initializes in <100ms from command invocation
- Message rendering latency <16ms per chunk (60fps)
- 100% feature parity with existing React Ink UI (verified via dual testing)
- Zero crashes during 1000 consecutive message streaming sessions
- Support for terminal widths from 80 to 240 columns
- Accessibility: All features usable with keyboard-only navigation

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1355-L1375 - Dual Testing Strategy]

## Dependencies

**Upstream Dependencies (Must be completed first):**
1. **EPIC-DEV-CLI-001** - Command Line Interface Foundation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L247-L271]
   - Provides command routing and flag parsing
   - TUI launches from the root command handler

2. **EPIC-INFRA-CORE-011** - Core Extension Integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L905-L948]
   - Provides gRPC client for message streaming
   - Bidirectional streaming required for real-time chat

3. **EPIC-INFRA-STORAGE-012** - State & Storage Layer [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L949-L1004]
   - Provides access to theme preferences in globalState.json
   - Required for consistent theming across sessions

**Downstream Dependencies (Depends on this epic):**
1. **EPIC-DEV-TASK-003** - Task Management [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L402-L505]
   - Task execution requires TUI for message display and approval prompts

## Integration Points with Other Personas/Epics

**DevOps/Automation User (EPIC-AUTO-MODE-005):**
- The TUI automatically disables when TTY detection indicates non-interactive mode
- Provides foundation for plain text mode detection logic
- Shared components for output formatting between TUI and plain modes

**Enterprise User (EPIC-ENT-SEC-008):**
- TUI must respect command permission prompts in enterprise environments
- Audit logging integration for UI actions (approvals, rejections)

**State & Storage (EPIC-INFRA-STORAGE-012):**
- Reads theme preferences from `~/.cline/data/globalState.json`
- Reads workspace state for task resumption display on welcome screen

**Core Extension Integration (EPIC-INFRA-CORE-011):**
- Receives `ClineSay` and `ClineAsk` messages via gRPC streaming
- Sends user responses back through gRPC client

---

## Feature Details

### FEAT-DEV-UI-002-BUBBLE-001: Bubble Tea TUI Framework Setup
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L318]

**IAOOI Framework:**
- **Inputs:** Terminal dimensions, theme configuration, initial state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L296-L299]
- **Activities:** Initialize Bubble Tea program, setup update loop, handle window resize [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L299-L300]
- **Outputs:** Running TUI application, event handlers, rendered frames [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L300-L301]
- **Outcomes:** Responsive terminal UI that handles keyboard and resize events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L301-L302]
- **Impacts:** Professional interactive experience [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L302]

### FEAT-DEV-UI-002-CHAT-002: Chat Message Rendering Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L319-L349]

**IAOOI Framework:**
- **Inputs:** Message objects (ask/say types), streaming chunks, message metadata [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L327-L330]
- **Activities:** Render message bubbles, apply syntax highlighting, handle streaming updates [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L330-L331]
- **Outputs:** Rendered chat interface with styled messages [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L331-L332]
- **Outcomes:** Users can read conversation history clearly [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332-L333]
- **Impacts:** Readable, professional chat experience [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L333]

### FEAT-DEV-UI-002-INPUT-003: User Input Handling and Prompts
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L350-L375]

**IAOOI Framework:**
- **Inputs:** Keyboard input, prompt requests from agent [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L358-L360]
- **Activities:** Capture user input, render input fields, handle submission [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L360-L361]
- **Outputs:** User responses, input validation results [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L361-L362]
- **Outcomes:** Users can respond to agent prompts and questions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L362-L363]
- **Impacts:** Interactive conversation flow [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L363]

### FEAT-DEV-UI-002-WELCOME-004: Welcome Screen and Onboarding
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L376-L401]

**IAOOI Framework:**
- **Inputs:** First-time user detection, configuration status [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L384-L386]
- **Activities:** Render welcome UI, show quick start hints, guide to authentication [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L386-L387]
- **Outputs:** Welcome screen, onboarding guidance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L387-L388]
- **Outcomes:** New users understand how to use the CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L388-L389]
- **Impacts:** Reduced onboarding friction [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L389]