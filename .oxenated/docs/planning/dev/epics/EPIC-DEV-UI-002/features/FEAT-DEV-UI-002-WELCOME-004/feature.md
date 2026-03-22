# Welcome Screen and Onboarding

## Feature ID
FEAT-DEV-UI-002-WELCOME-004

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L193-L210]

## Epic Context
**Parent Epic:** EPIC-DEV-UI-002 - Interactive Terminal UI [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-UI-002/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Provide a rich interactive terminal experience comparable to the VSCode extension, with real-time feedback, streaming messages, and professional TUI components. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L169-L178]
**Business Impact:** Higher user satisfaction, increased engagement, and feature parity across platforms. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L177-L178]

## Feature Overview
**Purpose:** Provide new users with a welcoming onboarding experience that guides them through initial setup and helps them understand how to use the CLI effectively. For returning users, display recent tasks for quick resumption. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L193-L210]
**Scope:** First-time user detection, welcome screen rendering, quick start hints, recent tasks display, and onboarding guidance to authentication.
**PRD References:** REQ-001, REQ-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L700-L701]
**PRD Feature ID:** EPIC-DEV-UI-002-WELCOME-004 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L193]
**Dependencies:** 
- FEAT-DEV-UI-002-BUBBLE-001 (Bubble Tea TUI Framework Setup) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L180-L192]
- FEAT-INFRA-STORAGE-012-FILE-001 (File-based JSON Storage) - for reading globalState.json to detect first-time users [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L432-L443]
- FEAT-DEV-TASK-003-CONV-003 (Conversation History Management) - for displaying recent tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L299]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L193-L210]

**Inputs:**
- First-time user detection flag (based on configuration status in `~/.cline/data/globalState.json`)
- Configuration status (API provider configured or not)
- Recent task history from task storage
- Terminal dimensions for responsive layout
- Theme preferences from global state

**Activities:**
- Detect if user is first-time by checking for existing configuration in global state
- Render welcome screen with Cline branding and welcome message
- Display quick start hints and keyboard shortcuts
- Show recent tasks list for returning users (up to 5-10 recent tasks)
- Guide users toward authentication if not configured
- Handle user input for task selection or new task creation

**Outputs:**
- Rendered welcome screen in terminal UI
- Recent tasks list with task IDs and summaries
- Quick start guidance and keyboard shortcut hints
- Navigation options to authentication or new task

**Outcomes:**
- New users understand how to get started with Cline CLI
- Returning users can quickly resume previous work
- Users are guided to complete authentication setup if needed
- Reduced onboarding friction and time-to-first-task

**Impacts:**
- Improved user adoption and retention
- Reduced support requests from confused first-time users
- Faster time-to-value for new users
- Professional first impression of the CLI tool

## Technical Requirements
**Architecture Layer:** UI Layer (Bubble Tea TUI Framework)
**Integration Points:**
- **Proposed:** New welcome model/component in Bubble Tea TUI to be created at `golang-cli/internal/ui/welcome.go`
- **Existing API:** StateManager reads via file storage at `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L432-L443]
- **Existing API:** Task history reads via task storage interface (to be implemented per FEAT-INFRA-STORAGE-012-FILE-001)
- **Proposed:** Navigation to auth subcommand or task initialization

**Data Requirements:**
- **Existing model:** Global state structure containing `apiProvider` configuration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L15-L17]
- **Existing model:** Task history with fields: taskId, timestamp, summary, status [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L288-L299]
- **Proposed:** Welcome state model with fields: isFirstTime, recentTasks[], hasAuthConfig

**Performance Requirements:**
- Welcome screen should render within 100ms of TUI initialization
- Recent tasks list should load asynchronously without blocking UI
- Smooth transitions between welcome and chat/task views

**Security Requirements:**
- Do not display API keys or sensitive configuration in welcome screen
- Ensure task summaries don't expose sensitive file paths or code

## User Experience
**User Personas:** Developer User (primary), DevOps/Automation User (secondary for returning workflow)

**User Actions:**
1. **First-time user flow:**
   - User runs `cline` without arguments
   - Welcome screen displays with branding and quick start guide
   - User sees hint to run `cline auth` to configure API provider
   - User can press Enter or specific key to start interactive auth

2. **Returning user flow:**
   - User runs `cline` without arguments
   - Welcome screen displays with recent tasks list
   - User can navigate to and select a recent task to resume
   - User can press key to start new task
   - User can access settings/configuration

3. **Quick actions:**
   - Keyboard shortcut to dismiss welcome and go to new task input
   - Keyboard shortcut to go directly to auth
   - Arrow key navigation for recent tasks

**UI Components:**
- **Proposed:** WelcomeHeader component - displays Cline logo/branding
- **Proposed:** QuickStartHints component - shows keyboard shortcuts and tips
- **Proposed:** RecentTasksList component - scrollable list of recent tasks
- **Proposed:** AuthPrompt component - shown when no provider configured
- **Existing component pattern:** Bubble Tea list component for task selection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L180-L192]

**Mobile Considerations:** Not applicable - CLI is desktop terminal only

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L203-L210]

```gherkin
Scenario: Show welcome screen for new user
  Given the user runs Cline for first time
  When the interactive mode starts
  Then the welcome screen should display
  And quick start hints should show

Scenario: Show recent tasks on welcome
  Given the user has previous tasks
  When the interactive mode starts
  Then recent tasks should display for quick resumption
```

**Additional BDD Scenarios (derived from feature requirements):**

```gherkin
Scenario: Guide new user to authentication
  Given the user has no API provider configured
  When the welcome screen displays
  Then a prompt to run "cline auth" should show
  And the user should be able to start auth with a keypress

Scenario: Select recent task from welcome
  Given the user has previous tasks
  And the welcome screen displays the recent tasks list
  When the user navigates to a task and presses Enter
  Then that task should resume
  And the conversation history should load

Scenario: Dismiss welcome and start new task
  Given the welcome screen is displayed
  When the user presses the new task shortcut key
  Then the welcome screen should dismiss
  And the task input prompt should appear

Scenario: Empty state for no tasks
  Given the user has no previous tasks
  And the user has configured authentication
  When the welcome screen displays
  Then a "No recent tasks" message should show
  And a hint to start a new task should display
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L193-L210]

**Functional:**
- Welcome screen displays for first-time users with onboarding guidance
- Recent tasks display correctly for returning users
- Task selection successfully resumes the selected task
- Authentication guidance appears when no provider is configured

**Performance:**
- Welcome screen renders within 100ms of TUI initialization
- Recent tasks load without blocking the UI
- Navigation between welcome elements is responsive (no perceptible lag)

**Quality:**
- Welcome screen renders correctly across terminal sizes (80x24 minimum)
- No visual glitches or flickering during welcome display
- Graceful handling when task history is corrupted or unreadable

**Integration:**
- Seamless transition from welcome to task execution
- Proper integration with authentication subcommand
- Consistent styling with rest of TUI

**Business Value:**
- New users can complete first task within 5 minutes of installation
- Returning users can resume work in under 10 seconds

## Testing Strategy
**Unit Testing:**
- Test first-time user detection logic based on global state
- Test recent tasks list formatting and pagination
- Test keyboard navigation and selection handlers
- Test theme application and styling

**Integration Testing:**
- Integration with storage layer for reading global state
- Integration with task history loading
- Integration with auth subcommand routing
- Integration with task resumption flow

**User Acceptance:**
- First-time user can complete onboarding without documentation
- Returning user can quickly find and resume recent task
- Visual appearance matches design expectations across terminals

**Performance Testing:**
- Measure welcome screen render time (<100ms target)
- Measure task list load time with 100+ tasks in history

## Tasks Overview
1. **Implement first-time user detection** - Check global state for existing configuration
2. **Create welcome screen Bubble Tea model** - Build TUI component with branding and layout
3. **Implement recent tasks list component** - Display and navigate recent tasks
4. **Create quick start hints component** - Show keyboard shortcuts and guidance
5. **Implement auth prompt for unconfigured users** - Guide to authentication
6. **Add keyboard navigation and shortcuts** - Handle user input for selection
7. **Integrate with task resumption flow** - Connect task selection to resume
8. **Add transitions to main chat UI** - Smooth handoff to task execution

## Implementation Notes
- Use Bubble Tea's `list` component for recent tasks display with custom styling
- Implement `Init()`, `Update()`, and `View()` methods following Bubble Tea patterns
- Consider using lipgloss for styling to match existing Cline branding colors
- First-time detection should check for presence of `apiProvider` in global state
- Recent tasks should be sorted by timestamp descending, showing most recent first
- Task summaries should be truncated if too long for terminal width
- Include keyboard shortcuts legend (e.g., "↑↓ navigate, Enter select, n new task, a auth")
- Handle edge case: no terminal (non-TTY) - skip welcome and go directly to task input
- Consider persisting "welcome dismissed" preference for users who prefer immediate task input

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new