# Chat Message Rendering Components

## Feature ID
FEAT-DEV-UI-002-CHAT-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L319-L349 - Feature 2: Chat Message Rendering Components]

## Epic Context
**Parent Epic:** EPIC-DEV-UI-002 - Interactive Terminal UI [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-UI-002/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Create a responsive, interactive terminal UI using Bubble Tea framework that delivers a rich chat experience with real-time message streaming, proper markdown formatting, and keyboard navigation comparable to the existing VSCode extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L276-L282]
**Business Impact:** Enables rich interactive experience comparable to VSCode extension, with real-time feedback and higher user satisfaction, ensuring feature parity across platforms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L284-L286]

## Feature Overview
**Purpose:** Implement chat message rendering components that display AI and user messages with proper styling, markdown formatting, syntax highlighting, and real-time streaming updates within the Bubble Tea TUI framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L325-L326]
**Scope:**
- IN SCOPE: Message bubble rendering, markdown formatting, syntax highlighting, streaming message display, styled message components
- OUT OF SCOPE: User input handling (covered by FEAT-DEV-UI-002-INPUT-003), window resize handling (covered by FEAT-DEV-UI-002-BUBBLE-001)
**PRD References:** REQ-001 (Execute AI coding tasks), REQ-002 (Interactive mode) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1754-L1760 - Traceability Matrix]
**PRD Feature ID:** EPIC-DEV-UI-002-CHAT-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L320]
**Dependencies:**
- FEAT-DEV-UI-002-BUBBLE-001: Bubble Tea TUI Framework Setup (required for TUI foundation) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L318]
- EPIC-INFRA-CORE-011: Core Extension Integration (for receiving message streams) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L327-L332 - Feature IAOOI section]

**Inputs:**
- Message objects (ask/say types) received from gRPC stream from core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L327]
- Streaming chunks arriving during AI response generation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L328]
- Message metadata including timestamps, message types, and sender information [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L328]
- Theme configuration from globalState for styling preferences [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]

**Activities:**
- Render message bubbles with appropriate styling based on sender (user vs AI) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L329]
- Apply markdown formatting to message content (headers, lists, emphasis) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L329]
- Apply syntax highlighting to code blocks using terminal-aware color schemes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L329]
- Handle streaming message updates in real-time as chunks arrive from core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L330]
- Manage message viewport scrolling and overflow for large conversations [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L7]
- Handle partial message states during streaming with visual indicators [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L330]

**Outputs:**
- Rendered chat interface displaying styled messages in terminal [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L331]
- Properly formatted markdown content with visual hierarchy [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L331]
- Syntax-highlighted code blocks for enhanced readability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L331]
- Real-time streaming updates visible to user as AI generates content [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]

**Outcomes:**
- Users can read conversation history clearly with professional formatting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]
- Code blocks are easily readable with syntax highlighting appropriate to language [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]
- Streaming responses provide immediate feedback during AI generation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]
- Message distinction between user and AI is visually clear [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]

**Impacts:**
- Readable, professional chat experience comparable to web-based interfaces [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]
- Improved code comprehension through syntax highlighting in terminal [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L332]
- Real-time feedback increases user engagement and perceived responsiveness [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L284-L286]
- Foundation for consistent chat experience across VSCode/CLI/JetBrains platforms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]

## Technical Requirements
**Architecture Layer:** UI Layer (Bubble Tea TUI components)

**Integration Points:**
- Proposed: Bubble Tea Model components in `golang-cli/internal/ui/components/` for message rendering [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L988-L997]
- Proposed: Markdown parser integration (e.g., `github.com/charmbracelet/glamour` or similar Go-native library) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55 - Independence requirements]
- Proposed: Syntax highlighter for code blocks (terminal-compatible, e.g., Chroma via Glamour or custom implementation) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55]
- Existing: Message types from gRPC proto definitions in `proto/cline/` directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1264]
- Existing: Core extension gRPC streaming interface via EPIC-INFRA-CORE-011 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]

**Data Requirements:**
- Existing message schema from proto definitions (type, text, ts, reasoning, say, ask, partial fields) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L985-L987]
- Existing theme/styling configuration from `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]
- Proposed: Go structs for message bubbles with Bubble Tea lipgloss styling [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L988-L997]

**Performance Requirements:**
- Message rendering latency: <50ms for initial render of message [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Streaming chunk display: Immediate (<16ms) update per chunk for smooth experience
- Memory usage: Efficient handling of large conversations (100+ messages) without significant degradation
- Terminal reflow: <100ms response to window resize events

**Security Requirements:**
- Sanitize markdown content to prevent terminal escape sequence injection attacks [Source: .clinerules/bash_console_ui_best_practices.md:L97-L102]
- Validate message content length to prevent memory exhaustion [Source: .clinerules/bash_console_ui_best_practices.md:L97-L102]
- Handle untrusted AI-generated content safely without arbitrary code execution [Source: .clinerules/bash_console_ui_best_practices.md:L97-L102]

## User Experience
**User Personas:** Developer User - Individual developer using Cline for coding assistance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L185-L205]

**User Actions:**
- View conversation history with clear visual distinction between user and AI messages
- Read code blocks with syntax highlighting appropriate to the programming language
- Observe AI responses appearing in real-time as they are generated
- Scroll through long conversations to review previous messages
- Identify different message types (questions, answers, tool requests, system messages)

**UI Components:**
- Proposed: MessageBubble component - Renders individual messages with sender styling, timestamps, and content [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L8]
- Proposed: CodeBlock component - Renders syntax-highlighted code with language detection and proper formatting [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L9]
- Proposed: ChatViewport component - Manages scrollable message list with efficient rendering [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L8]
- Proposed: StreamingIndicator component - Visual feedback for messages being generated [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L330]
- Proposed: MarkdownRenderer component - Converts markdown to styled terminal output [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L329]
- Proposed: MessageList component - Container for ordered message display with proper spacing [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L8]

**Go-Native Implementation Requirements:**
Per the independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55], the chat rendering MUST:
- Use pure Go libraries for markdown parsing and terminal rendering
- Use Charm Bracelet ecosystem (Bubble Tea, Lipgloss, Bubbles) for TUI components
- Avoid any dependencies on React, Ink, or Node.js-based rendering
- Implement custom message components using Bubble Tea's Model-Update-View pattern

**Recommended Go Libraries:**
- `github.com/charmbracelet/bubbletea` - Core TUI framework [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L33]
- `github.com/charmbracelet/lipgloss` - Terminal styling and layout [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L34]
- `github.com/charmbracelet/glamour` - Markdown rendering for terminal (if compatible with requirements)
- `github.com/charmbracelet/bubbles` - Reusable TUI components (viewport, spinner)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L333-L349 - Feature BDD Scenarios]

```gherkin
Scenario: Render text message
  Given a text message from Cline
  When it displays in the UI
  Then it should render with proper styling
  And markdown should be formatted

Scenario: Render code block
  Given a message containing code
  When it displays in the UI
  Then syntax highlighting should apply
  And the code should be readable

Scenario: Stream message updates
  Given a streaming response from AI
  When chunks arrive
  Then the message should update in real-time
  And partial content should display
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1190-L1205 - Success metrics adapted for UI features]

**Functional:**
- Text messages render with proper markdown formatting (headers, lists, bold, italic, links) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L333-L336]
- Code blocks display with syntax highlighting appropriate to the language [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L337-L340]
- User and AI messages are visually distinct with different styling [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L333-L336]
- Streaming messages update in real-time as content arrives from core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L341-L344]
- Messages display correctly at various terminal widths (80 columns minimum) [Source: .clinerules/bash_console_ui_best_practices.md:L25-L30]

**Performance:**
- Initial message render completes within 50ms [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Streaming chunks display with <16ms latency for smooth visual updates
- Large conversations (100+ messages) render without significant performance degradation
- Terminal resize triggers content reflow within 100ms

**Quality:**
- Markdown rendering handles edge cases (nested lists, code in lists, blockquotes)
- Syntax highlighting supports major languages (Go, TypeScript, JavaScript, Python, JSON, YAML, Bash)
- Graceful degradation when terminal doesn't support colors or Unicode [Source: .clinerules/bash_console_ui_best_practices.md:L36-L42]
- No terminal escape sequence injection vulnerabilities [Source: .clinerules/bash_console_ui_best_practices.md:L97-L102]

**Integration:**
- Messages integrate seamlessly with Bubble Tea TUI framework [Source: .oxenated/docs/planning/developer/epics/EPIC-DEV-UI-002/epic.md:L33-L37]
- Compatible with gRPC message streaming from core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]
- Styling respects user theme preferences from global configuration

**Dual Testing Requirements:**
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L55-L58, L1667-L1720], all chat rendering must be tested in BOTH:
1. Existing TypeScript CLI (verify visual parity)
2. New GoLang CLI (verify new implementation)

**Test Scenarios:**
- Side-by-side visual comparison of identical conversations
- Markdown rendering output comparison
- Code block syntax highlighting comparison
- Streaming behavior timing comparison

## Testing Strategy
**Unit Testing:**
- Individual message bubble component rendering tests
- Markdown parser output verification tests
- Syntax highlighting accuracy tests for supported languages
- Message viewport scrolling behavior tests
- Streaming state management tests

**Integration Testing:**
- Integration with Bubble Tea framework update loop
- gRPC message stream to UI component integration
- Theme configuration loading and application
- Large conversation performance testing

**User Acceptance:**
- Visual comparison with existing CLI output for identical conversations
- Markdown rendering verification against reference examples
- Code syntax highlighting validation across supported languages
- Accessibility verification (monochrome terminal support, sufficient contrast)

**Performance Testing:**
- Message render timing benchmarks
- Memory usage profiling with large conversations
- Streaming chunk throughput testing
- Terminal resize responsiveness testing

## Tasks Overview
- **Task 1:** Create MessageBubble component with Bubble Tea Model pattern
- **Task 2:** Implement markdown parsing and terminal rendering
- **Task 3:** Add syntax highlighting for code blocks
- **Task 4:** Implement streaming message updates
- **Task 5:** Create ChatViewport for scrollable message list
- **Task 6:** Add visual styling and theme integration
- **Task 7:** Implement terminal resize handling for messages
- **Task 8:** Create comprehensive test suite with dual-CLI comparison

## Implementation Notes

### Go-Native Implementation Requirements
Per the independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55], this feature MUST:
- Use only Go standard library or pure Go modules (no CGO dependencies)
- Implement all rendering logic in Go without TypeScript/JavaScript dependencies
- Use Bubble Tea's Model-Update-View pattern for component architecture
- Maintain 100% independence from existing CLI's React/Ink implementation

### Critical Implementation Considerations
1. **Terminal Compatibility**: Support both modern terminals with full Unicode/color support and limited terminals with fallback to ASCII [Source: .clinerules/bash_console_ui_best_practices.md:L36-L42]
2. **Markdown Rendering**: Glamour library provides excellent terminal markdown rendering but evaluate performance for streaming updates; may need custom implementation for streaming chunks
3. **Syntax Highlighting**: Chroma library (via Glamour) supports many languages; ensure language detection works for common code block formats
4. **Memory Management**: Large conversations require efficient rendering; implement viewport virtualization if needed for very long conversations
5. **Streaming Updates**: Bubble Tea's update loop must handle high-frequency chunk updates without blocking; use batching if necessary

### Dependencies on Other Features
- **FEAT-DEV-UI-002-BUBBLE-001**: Must have Bubble Tea framework initialized before chat components can render
- **EPIC-INFRA-CORE-011**: gRPC streaming must be functional to receive messages for display
- **EPIC-INFRA-STORAGE-012**: Theme configuration loaded from storage affects styling

### Independence Verification
This feature MUST NOT:
- Import any code from `cli/src/` or reference React/Ink components
- Use npm packages or Node.js runtime
- Bundle TypeScript/JavaScript code in the binary
- Depend on any non-Go libraries

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L319-L349]
- [x] Existing code references cite actual file paths (proto/ directory, storage paths)
- [x] New functionality clearly marked as "Proposed:" for GoLang implementation
- [x] Integration points cite existing interfaces (gRPC, globalState)
- [x] Epic reference cites .oxenated/docs/planning/dev/epics/EPIC-DEV-UI-002/epic.md
- [x] BDD scenarios extracted verbatim from PRD
- [x] Success criteria mapped to PRD requirements
- [x] Independence requirements cited from PRD