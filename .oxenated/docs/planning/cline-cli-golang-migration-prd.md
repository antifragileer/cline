# Cline CLI GoLang Migration - Product Requirements Document

## Scope

This PRD defines the complete migration of the Cline CLI from TypeScript/React Ink to GoLang, achieving full feature parity while leveraging Go's advantages: single binary distribution, faster startup, smaller footprint, and better cross-platform support. The migrated CLI will maintain full compatibility with existing state storage (~/.cline/data/) and the existing Cline core extension via gRPC/protobuf.

---

## System Overview

### System IAOOI Framework

**System Inputs:**
1. User commands and prompts (task mode, interactive mode)
2. Configuration files (`~/.cline/data/globalState.json`, `secrets.json`)
3. API credentials and provider settings
4. Task history and conversation data
5. Image files for multi-modal tasks
6. Environment variables (`CLINE_DIR`, `CLINE_COMMAND_PERMISSIONS`)
7. Stdin piped input for script workflows

**System Activities:**
1. Command parsing and validation (Cobra CLI framework)
2. Configuration loading and persistence
3. Authentication and API provider management
4. Terminal UI rendering (Bubble Tea TUI framework)
5. Task initialization and execution
6. Message streaming and display
7. Tool execution with user approval
8. State management and synchronization
9. Task history tracking
10. Update checking

**System Outputs:**
1. Interactive terminal UI with chat interface
2. Plain text output for scripting
3. JSON formatted output (`--json` flag)
4. Task execution results and diffs
5. Configuration files
6. Log files
7. Exit codes (0 success, non-zero error)

**System Outcomes:**
1. Users can execute AI coding tasks from terminal
2. Scriptable automation via piped input and JSON output
3. Consistent experience across VSCode/CLI/JetBrains
4. Portable binary distribution (no Node.js dependency)
5. Improved performance and startup time

**System Impacts:**
1. Broader adoption due to easier installation
2. Better CI/CD integration for enterprises
3. Reduced resource overhead
4. Cross-platform consistency
5. Foundation for future native integrations

---

## Personas

### Persona 1: Developer User (Individual Developer)

**Role:** Software developer using Cline for coding assistance

**Pain Points:**
- Slow CLI startup with Node.js
- Node.js version compatibility issues
- Inconsistent behavior across platforms
- Large installation size

**Needs:**
- Fast interactive mode (<100ms startup)
- File editing and code generation
- Debugging assistance
- Plan/Act mode for complex tasks
- Task resumption

**Usage Pattern:** Interactive tasks, plan/act mode, occasional scripting

---

### Persona 2: DevOps/Automation User (CI/CD, Scripting)

**Role:** DevOps engineer, automation specialist, CI/CD pipeline operator

**Pain Points:**
- Need for reliable exit codes
- JSON output for parsing
- No interactive prompts breaking scripts
- Timeout controls for long-running tasks

**Needs:**
- Yolo mode (auto-approve all actions)
- JSON output (`--json`)
- Piped input support
- Timeout controls (`--timeout`)
- Non-interactive mode detection

**Usage Pattern:** Automated pipelines, batch processing, integration with other tools

---

### Persona 3: Enterprise User (Corporate Environment)

**Role:** Developer in regulated/enterprise environment

**Pain Points:**
- Complex authentication requirements
- Policy compliance needs
- Audit trail requirements
- Team configuration management

**Needs:**
- SSO support (SAML/OIDC)
- Command permissions (`CLINE_COMMAND_PERMISSIONS`)
- Audit logging
- Configuration management
- Policy enforcement

**Usage Pattern:** Policy-compliant automation, team-shared configurations

---

## Epics and Features

### Epic 1: Command Line Interface Foundation
**ID:** EPIC-DEV-CLI-001  
**Persona:** Developer User

**Epic IAOOI:**
- **Inputs:** CLI arguments, environment variables, command definitions, help text requirements
- **Activities:** Command parsing with Cobra, flag validation, subcommand routing, help generation, version display
- **Outputs:** Parsed commands, validated inputs, help documentation, error messages
- **Outcomes:** Users can invoke CLI with proper arguments, discover available commands, understand usage
- **Impacts:** Reduced learning curve, fewer support requests, consistent CLI experience

**Requirements Coverage:** REQ-003

#### Feature 1: Root Command with Default Task Mode
**ID:** EPIC-DEV-CLI-001-CMD-001

**Feature IAOOI:**
- **Inputs:** `os.Args`, default command configuration, prompt argument
- **Activities:** Parse arguments with Cobra, detect if prompt provided, route to interactive or task mode
- **Outputs:** Command routing decision, parsed flags, help text if needed
- **Outcomes:** Users can run `cline` for interactive or `cline "prompt"` for direct task
- **Impacts:** Intuitive CLI experience, reduced friction

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Launch interactive mode without arguments
  Given the user has installed the Cline CLI
  When the user runs "cline" without any arguments
  Then the interactive welcome screen should display
  And the user should be able to type a prompt

Scenario: Execute task with direct prompt
  Given the user has configured an API provider
  When the user runs "cline 'create a hello world function'"
  Then a new task should start immediately
  And the AI should begin processing the request

Scenario: Show help with --help flag
  Given the user wants to see available options
  When the user runs "cline --help"
  Then the help documentation should display
  And all available commands should be listed
```

#### Feature 2: Task Subcommand with Flags
**ID:** EPIC-DEV-CLI-001-CMD-002

**Feature IAOOI:**
- **Inputs:** Task prompt string, mode flags (`-a`, `-p`), yolo flag (`-y`), timeout flag (`-t`), model flag (`-m`), image paths (`-i`), verbose flag (`-v`), cwd flag (`-c`), config flag (`--config`), thinking flag (`--thinking`), json flag (`--json`), taskId flag (`-T`)
- **Activities:** Parse all task-related flags, validate combinations, construct task request
- **Outputs:** Validated task configuration, initialized task context
- **Outcomes:** Users can customize task execution with all supported options
- **Impacts:** Flexible task execution, script compatibility

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Start task in act mode
  Given the user has a valid API configuration
  When the user runs "cline task -a 'fix the bug in main.go'"
  Then the task should start in act mode
  And Cline should begin executing tools

Scenario: Start task in plan mode
  Given the user wants to plan before executing
  When the user runs "cline task -p 'design a new API'"
  Then the task should start in plan mode
  And Cline should gather information and present a plan

Scenario: Start task with yolo mode
  Given the user wants automated execution
  When the user runs "cline task -y 'run tests'"
  Then all tool approvals should be auto-approved
  And the task should complete without prompts

Scenario: Start task with specific model
  Given the user wants to use a specific model
  When the user runs "cline task -m claude-sonnet-4 'refactor this code'"
  Then the task should use the specified model
  And the model selection should be confirmed

Scenario: Resume task by ID
  Given an existing task with ID "abc123"
  When the user runs "cline task -T abc123 'continue working'"
  Then the existing task should resume
  And the follow-up message should be added
```

#### Feature 3: History Subcommand with Pagination
**ID:** EPIC-DEV-CLI-001-CMD-003

**Feature IAOOI:**
- **Inputs:** Limit flag (`-n`), page flag (`-p`), config path (`--config`)
- **Activities:** Read task history from storage, paginate results, format output
- **Outputs:** Paginated task list with IDs, timestamps, and summaries
- **Outcomes:** Users can browse and reference previous tasks
- **Impacts:** Task continuity, referenceability

**Gherkin BDD Scenarios:**

```gherkin
Scenario: List recent tasks
  Given the user has completed tasks in history
  When the user runs "cline history"
  Then the last 10 tasks should display
  And each task should show ID, timestamp, and summary

Scenario: Paginate through history
  Given the user has many tasks in history
  When the user runs "cline history -n 20 -p 2"
  Then tasks 21-40 should display
  And pagination info should show

Scenario: Limit history results
  Given the user wants to see specific count
  When the user runs "cline history -n 5"
  Then only 5 most recent tasks should display
```

#### Feature 4: Config, Auth, Update, Version, Dev Subcommands
**ID:** EPIC-DEV-CLI-001-CMD-004

**Feature IAOOI:**
- **Inputs:** Subcommand specific flags and arguments
- **Activities:** Route to appropriate handler, execute subcommand logic
- **Outputs:** Subcommand results (config display, auth tokens, update status, version info, logs)
- **Outcomes:** Users can manage configuration, authenticate, check updates, view version, debug
- **Impacts:** Complete CLI functionality, self-service management

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Display configuration
  Given the user wants to see current config
  When the user runs "cline config"
  Then current global and workspace state should display

Scenario: Interactive authentication
  Given the user needs to authenticate
  When the user runs "cline auth"
  Then an interactive wizard should guide through provider selection and API key input

Scenario: Quick authentication with flags
  Given the user has an API key
  When the user runs "cline auth -p anthropic -k sk-ant-xxxxx"
  Then the provider should be configured immediately

Scenario: Check for updates
  Given the user wants latest version
  When the user runs "cline update"
  Then it should check npm for newer versions
  And offer to install if available

Scenario: Show version
  Given the user wants version info
  When the user runs "cline version"
  Then the current CLI version should display

Scenario: Open dev logs
  Given the user needs to debug
  When the user runs "cline dev log"
  Then the log file should open in default editor
```

---

### Epic 2: Interactive Terminal UI
**ID:** EPIC-DEV-UI-002  
**Persona:** Developer User

**Epic IAOOI:**
- **Inputs:** Terminal capabilities (TTY detection), theme preferences, message streams, user input
- **Activities:** Render Bubble Tea TUI, handle keyboard events, display streaming messages, manage focus states, render markdown
- **Outputs:** Interactive terminal interface, styled messages, progress indicators, input prompts
- **Outcomes:** Rich interactive experience comparable to VSCode extension, real-time feedback
- **Impacts:** Higher user satisfaction, increased engagement, feature parity across platforms

**Requirements Coverage:** REQ-001, REQ-002

#### Feature 1: Bubble Tea TUI Framework Setup
**ID:** EPIC-DEV-UI-002-BUBBLE-001

**Feature IAOOI:**
- **Inputs:** Terminal dimensions, theme configuration, initial state
- **Activities:** Initialize Bubble Tea program, setup update loop, handle window resize
- **Outputs:** Running TUI application, event handlers, rendered frames
- **Outcomes:** Responsive terminal UI that handles keyboard and resize events
- **Impacts:** Professional interactive experience

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Initialize TUI
  Given the terminal supports TTY
  When the CLI starts in interactive mode
  Then the Bubble Tea program should initialize
  And the initial view should render

Scenario: Handle window resize
  Given the TUI is running
  When the user resizes the terminal window
  Then the UI should adapt to new dimensions
  And content should reflow appropriately

Scenario: Handle keyboard events
  Given the TUI is running
  When the user presses keyboard keys
  Then the appropriate actions should trigger
  And focus should move correctly
```

#### Feature 2: Chat Message Rendering Components
**ID:** EPIC-DEV-UI-002-CHAT-002

**Feature IAOOI:**
- **Inputs:** Message objects (ask/say types), streaming chunks, message metadata
- **Activities:** Render message bubbles, apply syntax highlighting, handle streaming updates
- **Outputs:** Rendered chat interface with styled messages
- **Outcomes:** Users can read conversation history clearly
- **Impacts:** Readable, professional chat experience

**Gherkin BDD Scenarios:**

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

#### Feature 3: User Input Handling and Prompts
**ID:** EPIC-DEV-UI-002-INPUT-003

**Feature IAOOI:**
- **Inputs:** Keyboard input, prompt requests from agent
- **Activities:** Capture user input, render input fields, handle submission
- **Outputs:** User responses, input validation results
- **Outcomes:** Users can respond to agent prompts and questions
- **Impacts:** Interactive conversation flow

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Respond to approval prompt
  Given Cline asks for tool approval
  When the prompt displays
  Then the user should be able to approve or reject
  And the response should send to agent

Scenario: Type follow-up message
  Given a task is active
  When the user types a message and submits
  Then the message should send to Cline
  And display in the chat history
```

#### Feature 4: Welcome Screen and Onboarding
**ID:** EPIC-DEV-UI-002-WELCOME-004

**Feature IAOOI:**
- **Inputs:** First-time user detection, configuration status
- **Activities:** Render welcome UI, show quick start hints, guide to authentication
- **Outputs:** Welcome screen, onboarding guidance
- **Outcomes:** New users understand how to use the CLI
- **Impacts:** Reduced onboarding friction

**Gherkin BDD Scenarios:**

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

---

### Epic 3: Task Management
**ID:** EPIC-DEV-TASK-003  
**Persona:** Developer User

**Epic IAOOI:**
- **Inputs:** User prompts, task history, conversation state, file attachments, images
- **Activities:** Initialize new tasks, resume existing tasks, manage conversation flow, handle tool approvals
- **Outputs:** Task IDs, conversation messages, tool requests, execution results
- **Outcomes:** Users can start, continue, and manage coding tasks seamlessly
- **Impacts:** Improved productivity, task continuity, workflow efficiency

**Requirements Coverage:** REQ-001, REQ-004, REQ-012, REQ-013, REQ-018

#### Feature 1: New Task Initialization
**ID:** EPIC-DEV-TASK-003-INIT-001

**Feature IAOOI:**
- **Inputs:** User prompt, images, mode (plan/act), model preference, working directory
- **Activities:** Generate task ID, initialize conversation, send to core extension, render streaming response
- **Outputs:** Task ID, conversation messages, tool requests
- **Outcomes:** User can start new coding task with all options
- **Impacts:** Core functionality for all users

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Initialize new task with all options
  Given the user has images "screenshot.png" and "diagram.jpg"
  When the user runs "cline -m claude-sonnet -c /project -p 'design API' -i screenshot.png -i diagram.jpg"
  Then a new task should initialize
  And images should be attached
  And plan mode should activate
  And specified model should be used

Scenario: Task initialization generates unique ID
  Given a new task starts
  When initialization completes
  Then a unique task ID should generate
  And the task should be persisted to history
```

#### Feature 2: Task Resumption by ID
**ID:** EPIC-DEV-TASK-003-RESUME-002

**Feature IAOOI:**
- **Inputs:** Task ID from history, optional follow-up message
- **Activities:** Load conversation history, restore state, append follow-up if provided, resume streaming
- **Outputs:** Resumed task context, continued conversation
- **Outcomes:** Users can continue previous tasks seamlessly
- **Impacts:** Task continuity, long-running workflow support

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Resume task without follow-up
  Given an existing task with ID "abc123"
  When the user runs "cline -T abc123"
  Then the task should resume from last state
  And conversation history should load

Scenario: Resume task with follow-up message
  Given an existing task with ID "abc123"
  When the user runs "cline -T abc123 'add unit tests'"
  Then the task should resume
  And the follow-up message should be added to conversation

Scenario: Resume most recent task
  Given there are completed tasks
  When the user runs "cline --continue"
  Then the most recent task from current directory should resume
```

#### Feature 3: Conversation History Management
**ID:** EPIC-DEV-TASK-003-CONV-003

**Feature IAOOI:**
- **Inputs:** Conversation messages, task metadata, pagination requests
- **Activities:** Store messages, load history, format for display, handle large conversations
- **Outputs:** Paginated conversation history
- **Outcomes:** Users can review past conversations
- **Impacts:** Referenceability, debugging support

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Persist conversation messages
  Given a task is active
  When messages are exchanged
  Then each message should persist to storage
  And be available for later retrieval

Scenario: Load conversation on resume
  Given a task with existing conversation
  When the task resumes
  Then the full conversation should load
  And display in correct order
```

#### Feature 4: Tool Approval Workflows
**ID:** EPIC-DEV-TASK-003-TOOL-004

**Feature IAOOI:**
- **Inputs:** Tool use requests from agent, user approval/rejection
- **Activities:** Display tool request, capture user decision, execute or reject tool, return result
- **Outputs:** Tool execution results, error messages if rejected
- **Outcomes:** Users maintain control over file changes and command execution
- **Impacts:** Safety, trust, human-in-the-loop control

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Approve file edit
  Given Cline requests to edit "main.go"
  When the approval prompt displays
  And the user approves
  Then the edit should execute
  And result should return to Cline

Scenario: Reject command execution
  Given Cline requests to run "rm -rf /"
  When the approval prompt displays
  And the user rejects
  Then the command should not execute
  And rejection should return to Cline

Scenario: Auto-approve in yolo mode
  Given yolo mode is enabled
  When Cline requests any tool
  Then it should execute without prompt
  And result should return automatically
```

---

### Epic 4: Authentication & Provider Configuration
**ID:** EPIC-DEV-AUTH-004  
**Persona:** Developer User

**Epic IAOOI:**
- **Inputs:** API keys, OAuth tokens, provider settings, model preferences
- **Activities:** Secure credential storage, provider validation, OAuth flows, configuration persistence
- **Outputs:** Authenticated sessions, provider configurations, available models list
- **Outcomes:** Secure access to AI providers, easy provider switching
- **Impacts:** Enterprise adoption, security compliance, user trust

**Requirements Coverage:** REQ-009, REQ-016, REQ-017

#### Feature 1: OAuth Authentication Flows
**ID:** EPIC-DEV-AUTH-004-OAUTH-001

**Feature IAOOI:**
- **Inputs:** Provider selection, OAuth endpoints, callback URLs
- **Activities:** Initiate OAuth flow, start local callback server, capture token, store securely
- **Outputs:** OAuth tokens, authenticated session
- **Outcomes:** Users can authenticate with providers supporting OAuth
- **Impacts:** Enhanced security, no API key exposure

**Gherkin BDD Scenarios:**

```gherkin
Scenario: OAuth authentication flow
  Given the user selects OAuth provider
  When authentication starts
  Then a browser should open for authorization
  And local callback server should start
  And token should capture and store securely

Scenario: Handle OAuth callback
  Given OAuth flow is in progress
  When the callback receives authorization code
  Then token should exchange
  And configuration should update
```

#### Feature 2: API Key Management
**ID:** EPIC-DEV-AUTH-004-KEY-002

**Feature IAOOI:**
- **Inputs:** API keys via flags or interactive input, provider identification
- **Activities:** Validate key format, test authentication, encrypt and store, update config
- **Outputs:** Stored credentials, validation status
- **Outcomes:** Users can configure API key-based providers
- **Impacts:** Broad provider support

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Configure API key via flag
  Given the user runs "cline auth -p openai -k sk-xxxxx"
  When the command executes
  Then the key should validate
  And store encrypted in secrets.json
  And configuration should update

Scenario: Interactive API key input
  Given the user runs "cline auth" interactively
  When they select API key provider
  Then secure input prompt should display
  And key should store encrypted
```

#### Feature 3: Provider Configuration Wizard
**ID:** EPIC-DEV-AUTH-004-PROV-003

**Feature IAOOI:**
- **Inputs:** User selections, provider options, model lists
- **Activities:** Guide user through provider selection, model selection, configuration validation
- **Outputs:** Complete provider configuration
- **Outcomes:** Easy provider setup for new users
- **Impacts:** Reduced onboarding friction

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Interactive provider wizard
  Given the user runs "cline auth"
  When the wizard starts
  Then available providers should list
  And model selection should follow
  And configuration should save

Scenario: Quick setup with all flags
  Given the user runs "cline auth -p anthropic -k key -m claude-sonnet"
  When the command executes
  Then provider should configure immediately
  Without interactive prompts
```

---

### Epic 5: Plain Text & Scripting Modes
**ID:** EPIC-AUTO-MODE-005  
**Persona:** DevOps/Automation User

**Epic IAOOI:**
- **Inputs:** TTY detection, output redirection detection, mode flags (`--json`, `--yolo`)
- **Activities:** Detect non-interactive environment, switch output mode, suppress TUI rendering
- **Outputs:** Plain text output, exit codes, script-friendly responses
- **Outcomes:** CLI works seamlessly in pipes, scripts, and CI/CD pipelines
- **Impacts:** Automation adoption, DevOps integration, batch processing capability

**Requirements Coverage:** REQ-002, REQ-005

#### Feature 1: TTY/Redirect Detection
**ID:** EPIC-AUTO-MODE-005-DETECT-001

**Feature IAOOI:**
- **Inputs:** Stdin file descriptor, stdout file descriptor, `isatty()` checks
- **Activities:** Check if stdin is TTY, check if stdout is TTY, detect pipe/redirect
- **Outputs:** Mode decision (interactive vs plain)
- **Outcomes:** Automatic mode selection without user intervention
- **Impacts:** Seamless scripting experience

**Gherkin BDD Scenarios:**

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

#### Feature 2: Mode Switching Logic
**ID:** EPIC-AUTO-MODE-005-SWITCH-002

**Feature IAOOI:**
- **Inputs:** Detection results, explicit flags (`--json`, `--yolo`)
- **Activities:** Switch between interactive/plain modes, configure output format, setup appropriate handlers
- **Outputs:** Configured output mode, initialized handlers
- **Outcomes:** Correct behavior in all execution contexts
- **Impacts:** Reliable operation across environments

**Gherkin BDD Scenarios:**

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

---

### Epic 6: Automated Execution & Yolo Mode
**ID:** EPIC-AUTO-EXEC-006  
**Persona:** DevOps/Automation User

**Epic IAOOI:**
- **Inputs:** `--yolo` flag, command permission rules, auto-approve settings
- **Activities:** Parse permission configuration, validate commands against allow/deny lists, execute without prompts, stream output
- **Outputs:** Command execution results, permission decisions, execution logs
- **Outcomes:** Fully automated task execution without user intervention
- **Impacts:** CI/CD integration, batch processing, reduced manual oversight

**Requirements Coverage:** REQ-006, REQ-008

#### Feature 1: Yolo Mode Implementation
**ID:** EPIC-AUTO-EXEC-006-YOLO-001

**Feature IAOOI:**
- **Inputs:** `--yolo` flag, auto-approve configuration, task context
- **Activities:** Enable auto-approval for all tools, suppress confirmation prompts, continue until completion
- **Outputs:** Streamed execution results, final status
- **Outcomes:** Fully automated task execution
- **Impacts:** CI/CD integration capability

**Gherkin BDD Scenarios:**

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

#### Feature 2: Command Permission Validation
**ID:** EPIC-AUTO-EXEC-006-PERMS-002

**Feature IAOOI:**
- **Inputs:** Command strings, permission rules from environment
- **Activities:** Parse commands, validate against allow/deny patterns, check redirects
- **Outputs:** Permission decisions, validation errors
- **Outcomes:** Controlled command execution based on policy
- **Impacts:** Security compliance, safe automation

**Gherkin BDD Scenarios:**

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

---

### Epic 7: Structured Output (JSON)
**ID:** EPIC-AUTO-OUT-007  
**Persona:** DevOps/Automation User

**Epic IAOOI:**
- **Inputs:** Message objects, output format flag, JSON schema requirements
- **Activities:** Serialize messages to JSON, stream JSON lines, handle errors in JSON format
- **Outputs:** JSON-formatted messages, structured error responses
- **Outcomes:** Machine-readable output for integration with other tools
- **Impacts:** Tool ecosystem integration, programmatic access, automation workflows

**Requirements Coverage:** REQ-007

#### Feature 1: JSON Output Formatting
**ID:** EPIC-AUTO-OUT-007-JSON-001

**Feature IAOOI:**
- **Inputs:** Message objects (type, text, ts, reasoning, say, ask, partial, images, files)
- **Activities:** Marshal to JSON, ensure required fields, include optional fields when present
- **Outputs:** JSON lines output
- **Outcomes:** Parseable output for scripts and tools
- **Impacts:** Automation integration

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Output message as JSON
  Given a message with type "say", text "Hello", ts "1234567890"
  When JSON output is requested
  Then output should be: {"type":"say","text":"Hello","ts":1234567890}

Scenario: Include optional fields in JSON
  Given a message with reasoning and partial flag
  When JSON output is requested
  Then reasoning and partial should be included
  And output should be valid JSON

Scenario: Error output in JSON format
  Given an error occurs
  When JSON output is requested
  Then error should be in JSON format
  With error type and message fields
```

#### Feature 2: JSON Streaming for Long Tasks
**ID:** EPIC-AUTO-OUT-007-STREAM-002

**Feature IAOOI:**
- **Inputs:** Streaming message chunks
- **Activities:** Stream JSON lines as they arrive, handle partial messages, flush output
- **Outputs:** Streaming JSON lines
- **Outcomes:** Real-time JSON output for long-running tasks
- **Impacts:** Live monitoring of automated tasks

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Stream JSON in real-time
  Given a long-running task
  When messages stream from AI
  Then each message should output as JSON line immediately
  Without waiting for task completion

Scenario: Handle partial streaming messages
  Given a streaming text response
  When partial chunks arrive
  Then each chunk should output with partial flag
  And final message should have partial=false
```

---

### Epic 8: Security & Permissions
**ID:** EPIC-ENT-SEC-008  
**Persona:** Enterprise User

**Epic IAOOI:**
- **Inputs:** `CLINE_COMMAND_PERMISSIONS` env var, allow/deny patterns, redirect settings
- **Activities:** Parse permission rules, validate commands against patterns, detect dangerous characters, validate subshells
- **Outputs:** Permission decisions, security warnings, blocked command notifications
- **Outcomes:** Controlled command execution in enterprise environments
- **Impacts:** Security compliance, reduced risk, enterprise adoption

**Requirements Coverage:** REQ-008

#### Feature 1: Permission Rule Parsing
**ID:** EPIC-ENT-SEC-008-PARSE-001

**Feature IAOOI:**
- **Inputs:** `CLINE_COMMAND_PERMISSIONS` env var, JSON schema
- **Activities:** Parse JSON, validate schema, compile glob patterns
- **Outputs:** Parsed permission rules (allow, deny, allowRedirects)
- **Outcomes:** Ready-to-use permission configuration
- **Impacts:** Security policy enforcement

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Parse permission configuration
  Given CLINE_COMMAND_PERMISSIONS='{"allow":["npm *","git *"],"deny":["rm -rf *"],"allowRedirects":false}'
  When the CLI initializes
  Then allow list should contain "npm *" and "git *"
  And deny list should contain "rm -rf *"
  And allowRedirects should be false

Scenario: Handle missing permissions
  Given CLINE_COMMAND_PERMISSIONS is not set
  When the CLI runs
  Then all commands should be allowed
  And no restrictions should apply

Scenario: Parse complex patterns
  Given patterns with wildcards "npm run *" and "git push origin *"
  When permissions parse
  Then glob patterns should compile correctly
  For matching during validation
```

#### Feature 2: Command Validation Engine
**ID:** EPIC-ENT-SEC-008-VALID-002

**Feature IAOOI:**
- **Inputs:** Command strings, parsed permission rules
- **Activities:** Check against deny list first, then allow list, validate all segments
- **Outputs:** Allow/deny decision with reason
- **Outcomes:** Commands only execute if permitted
- **Impacts:** Security enforcement

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Deny takes precedence over allow
  Given allow=["*"] and deny=["rm *"]
  When validating "rm file.txt"
  Then command should be denied
  Because deny rules take precedence

Scenario: Allow list restricts to specific commands
  Given allow=["npm *","git *"] and no deny
  When validating "npm install"
  Then command should be allowed
  When validating "python script.py"
  Then command should be denied
```

#### Feature 3: Dangerous Character Detection
**ID:** EPIC-ENT-SEC-008-DANGER-003

**Feature IAOOI:**
- **Inputs:** Command strings
- **Activities:** Detect backticks outside quotes, unquoted newlines, subshells
- **Outputs:** Dangerous character detection results
- **Outcomes:** Prevention of command injection
- **Impacts:** Security hardening

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Detect backticks outside single quotes
  Given command "echo `whoami`"
  When dangerous character check runs
  Then it should detect backticks
  And command should be flagged as dangerous

Scenario: Allow backticks inside single quotes
  Given command "echo '`whoami`'"
  When dangerous character check runs
  Then it should not flag as dangerous
  Because backticks are inside single quotes

Scenario: Detect unquoted newlines
  Given command with embedded newline
  When validation runs
  Then it should detect unquoted newline
  And flag as dangerous
```

---

### Epic 9: Enterprise Configuration Management
**ID:** EPIC-ENT-CONFIG-009  
**Persona:** Enterprise User

**Epic IAOOI:**
- **Inputs:** Global configuration files, workspace settings, policy definitions
- **Activities:** Load tiered configuration, apply policy overrides, manage shared settings
- **Outputs:** Effective configuration, policy compliance status
- **Outcomes:** Consistent configuration across teams, policy enforcement
- **Impacts:** Enterprise governance, team consistency, reduced configuration drift

**Requirements Coverage:** REQ-016

#### Feature 1: Configuration Layering
**ID:** EPIC-ENT-CONFIG-009-LAYER-001

**Feature IAOOI:**
- **Inputs:** ~/.cline/data/globalState.json, workspace state, environment variables
- **Activities:** Load global config, overlay workspace config, apply env var overrides
- **Outputs:** Merged effective configuration
- **Outcomes:** Hierarchical configuration with proper precedence
- **Impacts:** Flexibility with consistency

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Load tiered configuration
  Given global config has setting A=1
  And workspace config has setting A=2
  When configuration loads
  Then effective A should be 2
  Because workspace overrides global

Scenario: Environment variable override
  Given CLINE_DIR is set to "/custom/path"
  When configuration loads
  Then config directory should be "/custom/path"
  Overriding any file-based setting
```

#### Feature 2: Policy Enforcement
**ID:** EPIC-ENT-CONFIG-009-POLICY-002

**Feature IAOOI:**
- **Inputs:** Policy definitions, user actions
- **Activities:** Check actions against policies, enforce restrictions, log violations
- **Outputs:** Policy decisions, enforcement actions
- **Outcomes:** Enterprise policies are respected
- **Impacts:** Compliance, governance

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Enforce allowed models policy
  Given policy restricts models to ["gpt-4","claude-sonnet"]
  When user tries to use "gpt-3.5"
  Then the request should be blocked
  And policy violation should log

Scenario: Enforce API key rotation policy
  Given policy requires key rotation every 90 days
  When checking key age
  Then warning should display if key is older than 90 days
```

---

### Epic 10: Audit & Compliance
**ID:** EPIC-ENT-AUDIT-010  
**Persona:** Enterprise User

**Epic IAOOI:**
- **Inputs:** Command executions, user actions, timestamps, task metadata
- **Activities:** Log audit events, generate audit trails, export compliance reports
- **Outputs:** Audit logs, compliance reports, activity history
- **Outcomes:** Complete audit trail for compliance requirements
- **Impacts:** Regulatory compliance, security auditing, incident investigation

**Requirements Coverage:** REQ-016 (audit aspect)

#### Feature 1: Audit Logging
**ID:** EPIC-ENT-AUDIT-010-LOG-001

**Feature IAOOI:**
- **Inputs:** Action events (command execution, file edit, API call), user context, timestamps
- **Activities:** Write audit events to log file, include all relevant metadata
- **Outputs:** Audit log entries
- **Outcomes:** Complete record of all actions
- **Impacts:** Accountability, forensics

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Log command execution
  Given Cline executes "npm install"
  When the command runs
  Then audit log should contain the command
  And timestamp, user, and result should be recorded

Scenario: Log file edits
  Given Cline edits file "main.go"
  When the edit completes
  Then audit log should contain file path
  And diff summary should be recorded
```

#### Feature 2: Compliance Report Generation
**ID:** EPIC-ENT-AUDIT-010-EXPORT-002

**Feature IAOOI:**
- **Inputs:** Audit logs, date ranges, export format
- **Activities:** Filter logs by range, format for export, generate report
- **Outputs:** Compliance report files
- **Outcomes:** Audit-ready documentation
- **Impacts:** Regulatory compliance efficiency

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Generate compliance report
  Given audit logs exist for the past month
  When admin requests compliance report
  Then report should generate in requested format
  And include all relevant audit events
```

---

### Epic 11: Core Extension Integration
**ID:** EPIC-INFRA-CORE-011  
**Persona:** Infrastructure (Internal)

**Epic IAOOI:**
- **Inputs:** Protobuf messages, gRPC/streaming connections, task state
- **Activities:** Establish connection to core, stream messages bidirectionally, handle reconnections
- **Outputs:** Synchronized state, streamed messages, task execution results
- **Outcomes:** Seamless integration with existing Cline core
- **Impacts:** Feature reuse, consistent behavior, faster development

**Requirements Coverage:** REQ-011

#### Feature 1: gRPC Client Implementation
**ID:** EPIC-INFRA-CORE-011-GRPC-001

**Feature IAOOI:**
- **Inputs:** Protobuf definitions, core extension endpoint, connection settings
- **Activities:** Generate Go code from proto, implement gRPC client, handle connection management
- **Outputs:** Connected gRPC client, method implementations
- **Outcomes:** Communication with Cline core extension
- **Impacts:** Feature parity with existing implementation

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Connect to core extension
  Given the Cline core extension is running
  When the CLI initializes
  Then it should establish a gRPC connection
  And be able to send and receive messages

Scenario: Call RPC methods
  Given a connected gRPC client
  When calling "NewTask" RPC
  Then the request should serialize correctly
  And response should deserialize correctly
```

#### Feature 2: Bidirectional Streaming
**ID:** EPIC-INFRA-CORE-011-STREAM-002

**Feature IAOOI:**
- **Inputs:** Outgoing messages, incoming message stream
- **Activities:** Send messages to core, receive streaming responses, handle flow control
- **Outputs:** Bidirectional message flow
- **Outcomes:** Real-time communication with core
- **Impacts:** Responsive UI, real-time updates

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Stream messages bidirectionally
  Given an active task
  When user sends message
  Then it should transmit to core
  And AI responses should stream back
  In real-time

Scenario: Handle streaming errors
  Given an active stream
  When network error occurs
  Then stream should handle gracefully
  And attempt reconnection
```

#### Feature 3: State Synchronization
**ID:** EPIC-INFRA-CORE-011-STATE-003

**Feature IAOOI:**
- **Inputs:** Local state changes, remote state updates
- **Activities:** Sync state bidirectionally, resolve conflicts, persist changes
- **Outputs:** Synchronized state across CLI and core
- **Outcomes:** Consistent state across components
- **Impacts:** Reliable operation

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Sync task state
  Given a task is active
  When core updates task state
  Then CLI should receive update
  And local state should synchronize
```

---

### Epic 12: State & Storage Layer
**ID:** EPIC-INFRA-STORAGE-012  
**Persona:** Infrastructure (Internal)

**Epic IAOOI:**
- **Inputs:** Global state JSON, workspace state, secrets, task history
- **Activities:** Read/write JSON files, manage file locking, handle migrations, encrypt secrets
- **Outputs:** Persisted state, loaded configuration, task history
- **Outcomes:** Reliable state management across CLI invocations
- **Impacts:** Data integrity, user experience, reliability

**Requirements Coverage:** REQ-010, REQ-016, REQ-017, REQ-018

#### Feature 1: File-based JSON Storage
**ID:** EPIC-INFRA-STORAGE-012-FILE-001

**Feature IAOOI:**
- **Inputs:** Storage paths, JSON data, file locking requirements
- **Activities:** Read JSON files, write with atomic rename, handle concurrent access
- **Outputs:** Loaded state, persisted changes
- **Outcomes:** Reliable state management
- **Impacts:** Data integrity across invocations

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Read global state
  Given ~/.cline/data/globalState.json exists
  When CLI loads configuration
  Then global state should read from file
  And parse into Go structs

Scenario: Atomic write operation
  Given updated state needs persistence
  When write operation executes
  Then it should write to temp file
  And atomically rename to target
  To prevent corruption

Scenario: Handle concurrent access
  Given multiple CLI instances
  When they access same file
  Then file locking should prevent corruption
```

#### Feature 2: Secrets Encryption
**ID:** EPIC-INFRA-STORAGE-012-SECRET-002

**Feature IAOOI:**
- **Inputs:** Plaintext secrets, encryption key from OS keyring
- **Activities:** Encrypt secrets before storage, decrypt on read, integrate with OS keyring
- **Outputs:** Encrypted secrets file
- **Outcomes:** Secure credential storage
- **Impacts:** Security compliance

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Encrypt API key
  Given API key "sk-xxxxx"
  When storing to secrets.json
  Then it should encrypt before writing
  And not be readable as plaintext

Scenario: Decrypt on read
  Given encrypted secrets file
  When CLI reads API key
  Then it should decrypt using OS keyring
  And return plaintext for use
```

#### Feature 3: State Migration
**ID:** EPIC-INFRA-STORAGE-012-MIGRATE-003

**Feature IAOOI:**
- **Inputs:** Legacy state format, migration version tracking
- **Activities:** Detect state version, apply migrations, update version marker
- **Outputs:** Migrated state in current format
- **Outcomes:** Backward compatibility
- **Impacts:** Smooth upgrades

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Migrate old state format
  Given state file from older version
  When CLI initializes
  Then migration should run automatically
  And state should be in current format

Scenario: Track migration version
  Given migration completes
  Then version marker should update
  And future runs should skip migration
```

---

### Epic 13: API Provider Integrations
**ID:** EPIC-INFRA-API-013  
**Persona:** Infrastructure (Internal)

**Epic IAOOI:**
- **Inputs:** API credentials, model configurations, streaming preferences
- **Activities:** Connect to OpenAI, Anthropic, OpenRouter, etc., handle streaming responses, manage rate limits
- **Outputs:** AI-generated responses, token usage, error handling
- **Outcomes:** Access to multiple AI providers through unified interface
- **Impacts:** Provider flexibility, cost optimization, feature availability

**Requirements Coverage:** REQ-009

#### Feature 1: OpenAI Provider
**ID:** EPIC-INFRA-API-013-OPENAI-001

**Feature IAOOI:**
- **Inputs:** OpenAI API key, model selection, message history
- **Activities:** Format requests for OpenAI API, handle streaming responses, parse completions
- **Outputs:** Generated responses, token usage info
- **Outcomes:** Access to GPT models
- **Impacts:** Core AI functionality

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Send request to OpenAI
  Given valid OpenAI API key
  When sending chat completion request
  Then response should stream back
  And content should extract correctly
```

#### Feature 2: Anthropic Provider
**ID:** EPIC-INFRA-API-013-ANTHROPIC-002

**Feature IAOOI:**
- **Inputs:** Anthropic API key, model (Claude), message format
- **Activities:** Format for Anthropic API, handle streaming, parse responses
- **Outputs:** Claude-generated responses
- **Outcomes:** Access to Claude models
- **Impacts:** Core AI functionality

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Send request to Anthropic
  Given valid Anthropic API key
  When sending messages to Claude
  Then response should stream back
  With Claude's characteristic formatting
```

#### Feature 3: OpenRouter Provider
**ID:** EPIC-INFRA-API-013-ROUTER-003

**Feature IAOOI:**
- **Inputs:** OpenRouter API key, model selection from multiple providers
- **Activities:** Route to appropriate backend model, handle unified response format
- **Outputs:** Responses from various providers
- **Outcomes:** Multi-provider access through single API
- **Impacts:** Flexibility, failover capability

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Route through OpenRouter
  Given valid OpenRouter key
  When requesting model "anthropic/claude-sonnet"
  Then request should route through OpenRouter
  And return Claude response
```

#### Feature 4: Other Providers (Gemini, Bedrock, etc.)
**ID:** EPIC-INFRA-API-013-OTHER-004

**Feature IAOOI:**
- **Inputs:** Provider-specific credentials and configurations
- **Activities:** Implement provider-specific request/response handling
- **Outputs:** Unified response format across all providers
- **Outcomes:** Broad provider support
- **Impacts:** User choice, cost optimization

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Support Google Gemini
  Given Gemini API configuration
  When sending request
  Then response should return in standard format

Scenario: Support AWS Bedrock
  Given AWS credentials and Bedrock access
  When sending request
  Then response should return in standard format
```

---

### Epic 14: Distribution & Packaging
**ID:** EPIC-INFRA-DIST-014  
**Persona:** Infrastructure (Internal)

**Epic IAOOI:**
- **Inputs:** Go source code, build targets (Linux, macOS, Windows), package managers
- **Activities:** Cross-compile binaries, create packages (Homebrew, npm, scoop), sign binaries
- **Outputs:** Platform-specific binaries, installable packages
- **Outcomes:** Easy installation across all platforms
- **Impacts:** User adoption, distribution reach, professional appearance

**Requirements Coverage:** REQ-014, REQ-015

#### Feature 1: Cross-platform Build Scripts
**ID:** EPIC-INFRA-DIST-014-BUILD-001

**Feature IAOOI:**
- **Inputs:** Go source, target platforms, version tags
- **Activities:** Compile for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64), sign binaries
- **Outputs:** Platform-specific binaries
- **Outcomes:** Binaries for all supported platforms
- **Impacts:** Broad platform support

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Build for Linux AMD64
  Given Go source code
  When build script runs with GOOS=linux GOARCH=amd64
  Then binary should compile
  And be executable on Linux AMD64

Scenario: Build for macOS ARM64
  Given Go source code
  When build script runs with GOOS=darwin GOARCH=arm64
  Then binary should compile
  And be executable on Apple Silicon Macs

Scenario: Build for Windows
  Given Go source code
  When build script runs with GOOS=windows
  Then binary should compile with .exe extension
```

#### Feature 2: Homebrew Formula
**ID:** EPIC-INFRA-DIST-014-HOMEBREW-002

**Feature IAOOI:**
- **Inputs:** Release binaries, version info, checksums
- **Activities:** Generate/update Homebrew formula, publish to tap
- **Outputs:** Homebrew formula file
- **Outcomes:** macOS/Linux users can install via `brew install cline`
- **Impacts:** Easy macOS/Linux installation

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Generate Homebrew formula
  Given new release binaries
  When formula generation runs
  Then formula should include URLs for all platforms
  And checksums should be correct

Scenario: Install via Homebrew
  Given Homebrew tap is configured
  When user runs "brew install cline"
  Then cline should install correctly
  And be available in PATH
```

#### Feature 3: NPM Wrapper Package
**ID:** EPIC-INFRA-DIST-014-NPM-003

**Feature IAOOI:**
- **Inputs:** Platform binaries, package.json template
- **Activities:** Create npm package that downloads correct binary, setup postinstall script
- **Outputs:** Published npm package
- **Outcomes:** Users can install via `npm install -g cline`
- **Impacts:** Familiar installation for Node.js users

**Gherkin BDD Scenarios:**

```gherkin
Scenario: Install via npm
  Given npm package is published
  When user runs "npm install -g cline"
  Then correct binary for platform should download
  And cline should be available in PATH

Scenario: Platform detection in npm install
  Given npm postinstall script runs
  When detecting platform
  Then correct binary URL should be selected
  Based on process.platform and process.arch
```

---

## Traceability Matrix

| Epic/Feature ID | Requirement IDs |
|-----------------|-----------------|
| EPIC-DEV-CLI-001 | REQ-003 |
| EPIC-DEV-CLI-001-CMD-001 | REQ-001, REQ-002 |
| EPIC-DEV-CLI-001-CMD-002 | REQ-003, REQ-004 |
| EPIC-DEV-CLI-001-CMD-003 | REQ-018 |
| EPIC-DEV-UI-002 | REQ-001, REQ-002 |
| EPIC-DEV-UI-002-BUBBLE-001 | REQ-001 |
| EPIC-DEV-TASK-003 | REQ-001, REQ-004 |
| EPIC-DEV-TASK-003-INIT-001 | REQ-001, REQ-004, REQ-013 |
| EPIC-DEV-TASK-003-RESUME-002 | REQ-012 |
| EPIC-DEV-TASK-003-CONV-003 | REQ-018 |
| EPIC-DEV-AUTH-004 | REQ-009, REQ-016, REQ-017 |
| EPIC-DEV-AUTH-004-OAUTH-001 | REQ-009 |
| EPIC-AUTO-MODE-005 | REQ-002, REQ-005 |
| EPIC-AUTO-MODE-005-DETECT-001 | REQ-005 |
| EPIC-AUTO-EXEC-006 | REQ-006 |
| EPIC-AUTO-EXEC-006-YOLO-001 | REQ-006 |
| EPIC-AUTO-EXEC-006-PERMS-002 | REQ-008 |
| EPIC-AUTO-OUT-007 | REQ-007 |
| EPIC-AUTO-OUT-007-JSON-001 | REQ-007 |
| EPIC-ENT-SEC-008 | REQ-008 |
| EPIC-ENT-SEC-008-PARSE-001 | REQ-008 |
| EPIC-ENT-SEC-008-VALID-002 | REQ-008 |
| EPIC-INFRA-CORE-011 | REQ-011 |
| EPIC-INFRA-CORE-011-GRPC-001 | REQ-011 |
| EPIC-INFRA-STORAGE-012 | REQ-010, REQ-016, REQ-017, REQ-018 |
| EPIC-INFRA-STORAGE-012-FILE-001 | REQ-010, REQ-016, REQ-018 |
| EPIC-INFRA-STORAGE-012-SECRET-002 | REQ-017 |
| EPIC-INFRA-API-013 | REQ-009 |
| EPIC-INFRA-DIST-014 | REQ-014, REQ-015 |
| EPIC-INFRA-DIST-014-BUILD-001 | REQ-014 |
| EPIC-INFRA-DIST-014-HOMEBREW-002 | REQ-015 |
| EPIC-INFRA-DIST-014-NPM-003 | REQ-015 |

**Original Requirements:**
1. REQ-001: Execute AI coding tasks from terminal with interactive UI
2. REQ-002: Support both interactive and plain text modes
3. REQ-003: Implement all existing CLI commands (task, history, config, auth, update, version, dev)
4. REQ-004: Support plan and act modes
5. REQ-005: Auto-detect TTY/redirect for automatic mode switching
6. REQ-006: Implement yolo mode for automated execution
7. REQ-007: Support JSON output for scripting
8. REQ-008: Implement command permission validation (CLINE_COMMAND_PERMISSIONS)
9. REQ-009: Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.)
10. REQ-010: Maintain state persistence (~/.cline/data/)
11. REQ-011: Integrate with existing Cline core via gRPC/protobuf
12. REQ-012: Support task resumption by ID
13. REQ-013: Support image attachments
14. REQ-014: Cross-platform distribution (Linux, macOS, Windows)
15. REQ-015: Homebrew and npm distribution
16. REQ-016: Configuration management (global and workspace)
17. REQ-017: Secure secrets storage
18. REQ-018: Task history with pagination

---

## AI Execution Plan

### Phase 1: Foundational Setup (Prerequisites and Core Infrastructure)

**1.1. Project Structure Setup**
- Initialize Go module with proper structure (`cmd/`, `internal/`, `pkg/`)
- Setup build scripts for cross-compilation
- Configure linting (golangci-lint), testing, CI/CD
- **Action:** AI Agent 1 creates project scaffolding

**1.2. Protobuf Integration**
- Generate Go code from existing proto definitions in `proto/`
- Setup proto compilation in build process
- **Action:** AI Agent 2 handles protobuf generation

**1.3. Storage Layer Implementation**
- Implement `ClineFileStorage` equivalent in Go
- Port StateManager logic to Go
- Handle JSON serialization/deserialization
- **Action:** AI Agent 3 builds storage layer

**Deliverable:** Foundation project with storage, proto support, and build system

---

### Phase 2: Core CLI Foundation (Command Infrastructure)

**2.1. Cobra CLI Framework**
- Setup Cobra with all subcommands (task, history, config, auth, update, version, dev)
- Implement flag parsing and validation matching existing CLI
- **Action:** AI Agent 4 implements CLI commands

**2.2. Configuration Management**
- Port configuration loading logic from TypeScript
- Implement environment variable handling (CLINE_DIR, CLINE_COMMAND_PERMISSIONS)
- **Action:** AI Agent 4 continues with config

**Deliverable:** Functional CLI with all commands and help text

---

### Phase 3: Interactive UI Development (Bubble Tea TUI)

**3.1. TUI Framework Setup**
- Initialize Bubble Tea program structure
- Implement basic Model-Update-View loop
- **Action:** AI Agent 5 builds TUI foundation

**3.2. Chat Components**
- Port chat message rendering components from React Ink to Bubble Tea
- Implement streaming message display
- **Action:** AI Agent 5 continues with chat UI

**3.3. Input Handling**
- Implement user input components (text input, approval prompts)
- Handle keyboard shortcuts (quit, scroll, approve, reject)
- **Action:** AI Agent 5 completes UI

**Deliverable:** Interactive terminal UI with chat and input handling

---

### Phase 4: Task Management (Core Business Logic)

**4.1. Task Initialization**
- Port task creation logic from TypeScript
- Implement mode switching (plan/act)
- **Action:** AI Agent 6 implements task logic

**4.2. gRPC Integration**
- Implement gRPC client for core extension communication
- Handle bidirectional streaming for messages
- **Action:** AI Agent 7 handles gRPC

**4.3. Tool Approval Workflows**
- Implement approval prompt handling in TUI
- Port tool execution logic
- **Action:** AI Agent 6 completes task management

**Deliverable:** Working task execution with gRPC integration

---

### Phase 5: Automation & Scripting (Plain Mode)

**5.1. Mode Detection**
- Implement TTY/redirect detection (stdin, stdout isatty checks)
- **Action:** AI Agent 8 implements mode switching

**5.2. Yolo Mode**
- Implement auto-approval logic for all tools
- **Action:** AI Agent 8 continues

**5.3. JSON Output**
- Implement JSON serialization for all message types
- Handle streaming JSON output
- **Action:** AI Agent 8 completes automation

**Deliverable:** Full scripting support with plain mode, yolo, and JSON

---

### Phase 6: Security & Enterprise (Permissions & Audit)

**6.1. Permission System**
- Implement CLINE_COMMAND_PERMISSIONS parsing
- Port command validation logic (glob matching, dangerous char detection)
- **Action:** AI Agent 9 implements security

**6.2. Audit Logging**
- Implement audit trail for command executions
- **Action:** AI Agent 9 completes security

**Deliverable:** Enterprise-ready security features

---

### Phase 7: API Providers (External Integrations)

**7.1. Provider Implementations**
- Port OpenAI, Anthropic, OpenRouter clients from TypeScript
- Implement streaming response handling
- Add support for remaining providers (Gemini, Bedrock, etc.)
- **Action:** AI Agent 10 implements providers

**Deliverable:** All API providers working with streaming

---

### Phase 8: Distribution (Packaging & Release)

**8.1. Build System**
- Cross-platform build scripts (Linux, macOS, Windows; AMD64, ARM64)
- Binary signing
- **Action:** AI Agent 11 handles distribution

**8.2. Package Managers**
- Homebrew formula generation
- npm wrapper package
- **Action:** AI Agent 11 completes distribution

**Deliverable:** Installable packages for all platforms

---

### Coordination Points

| Phase | Coordination Activity |
|-------|----------------------|
| After Phase 1 | All agents align on storage interfaces, proto definitions |
| After Phase 2 | CLI commands can be tested individually |
| After Phase 4 | Core functionality works end-to-end with core extension |
| After Phase 6 | Enterprise features ready for security review |
| After Phase 8 | Full release with distribution |

### Testing Strategy

**Unit Tests:**
- All packages with >80% coverage
- Storage operations, command parsing, validation logic

**Integration Tests:**
- gRPC communication with mock core
- Storage with temporary directories
- Provider API clients with mocked responses

**E2E Tests:**
- Full CLI commands with test core extension
- Cross-platform binary execution
- Installation verification

---

## End User Summary

This GoLang migration delivers a completely rewritten Cline CLI that maintains full feature parity with the existing TypeScript implementation while providing significant advantages. Users will experience faster startup times, smaller installation footprint, and true single-binary distribution without Node.js dependencies. The CLI supports all existing workflows: interactive chat with rich terminal UI, automated scripting with JSON output and yolo mode, and enterprise security with command permissions and audit logging. State storage remains compatible with existing `~/.cline/data/` directories, ensuring seamless migration. The gRPC integration with the existing Cline core extension means all AI capabilities work identically to the current implementation. Distribution through Homebrew and npm provides familiar installation paths, while the underlying Go binaries offer superior cross-platform compatibility. Whether you're a developer using interactive mode for coding assistance, a DevOps engineer automating CI/CD pipelines, or an enterprise user requiring policy compliance, this migrated CLI delivers the same powerful Cline experience with improved performance and reliability.