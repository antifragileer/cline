# Structured Output (JSON)

## Epic ID
EPIC-AUTO-OUT-007

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1078 - Epic 7 Section]

## Target Persona
DevOps/Automation User

## Epic Overview
This epic delivers machine-readable JSON output capabilities for the GoLang Cline CLI, enabling seamless integration with automation tools, CI/CD pipelines, and scripting workflows. By providing structured output alongside the existing interactive TUI mode, this epic addresses the critical need for programmatic access to Cline's functionality, allowing DevOps engineers and automation specialists to parse task results, integrate with other tools, and build sophisticated automation workflows around Cline's AI capabilities.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1027]

## Vision & Objectives
Enable DevOps and automation users to integrate Cline CLI into their toolchains by providing structured, machine-readable output that can be parsed, logged, and processed by other automation tools. This eliminates the need for fragile screen-scraping of TUI output and enables reliable automation of AI-assisted coding tasks.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1027]

## IAOOI System Components

### Inputs
- Message objects (type, text, timestamp, reasoning, say, ask, partial flags, images, files)
- Output format flag (`--json`)
- JSON schema requirements
- Streaming message chunks from AI responses
- Error conditions requiring structured error reporting

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]

### Activities
- Serialize message objects to JSON format with proper field mapping
- Ensure all required fields are included in output
- Include optional fields (reasoning, partial flags) when present
- Stream JSON lines for long-running tasks with real-time updates
- Handle partial streaming messages with appropriate partial flags
- Format error responses in JSON structure with type and message fields
- Marshal data structures using Go's JSON encoding
- Flush output buffers for real-time streaming

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]

### Outputs
- JSON-formatted message lines (newline-delimited JSON/NDJSON)
- Structured error responses with consistent schema
- Streaming JSON output for long-running tasks
- Partial message chunks with `partial: true` flag during streaming
- Final complete messages with `partial: false` flag
- Machine-readable output suitable for piping to other tools

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]

### Outcomes
- DevOps engineers can integrate Cline CLI into CI/CD pipelines with reliable output parsing
- Automation scripts can process Cline responses without fragile text parsing
- Tool ecosystem integration becomes possible through standardized JSON interface
- Programmatic access to AI coding assistance enables new automation workflows
- Output can be logged, stored, and analyzed by external monitoring systems

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]

### Impacts
- Broader adoption of Cline in enterprise automation environments
- Foundation for building higher-level automation tools on top of Cline CLI
- Improved CI/CD integration for AI-assisted code review and generation
- Enhanced observability through structured logging capabilities
- Reduced manual intervention in automated coding workflows
- Cross-platform consistency in automation toolchains

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]

## Key Features
- FEAT-AUTO-OUT-007-JSON-001: JSON Output Formatting
- FEAT-AUTO-OUT-007-STREAM-002: JSON Streaming for Long Tasks

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1031-L1078]

## Business Value & Requirements
This epic directly addresses **REQ-007: Support JSON output for scripting**.

Requirements mapping:
- REQ-007: JSON output for scripting - **FULLY ADDRESSED** by both features in this epic

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1019, L1294-L1296]

## User Journeys & Scenarios

### User Journey: CI/CD Pipeline Integration
A DevOps engineer wants to integrate Cline into their deployment pipeline to automatically review code changes. They configure their CI/CD system to pipe git diffs to Cline with the `--json` flag, then parse the structured output to determine if the changes meet quality standards. The JSON output allows their automation scripts to extract specific findings, categorize issues, and generate reports without parsing human-readable text.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L90-L98 (Persona 2: DevOps/Automation User)]

### User Journey: Batch Processing with Output Capture
An automation specialist needs to process hundreds of files with AI assistance. They create a script that invokes Cline in yolo mode with JSON output, processes the structured results to extract generated code, and writes it to appropriate output files. The JSON format ensures their script can reliably extract data even when the AI response contains complex formatting or multiple code blocks.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L90-L98 (Persona 2: DevOps/Automation User)]

## BDD Scenarios

### Feature: JSON Output Formatting (FEAT-AUTO-OUT-007-JSON-001)

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1033-L1054]

### Feature: JSON Streaming for Long Tasks (FEAT-AUTO-OUT-007-STREAM-002)

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078]

## Technical Considerations

### Existing Code References (if any):
- **Message Types**: Cline message types are defined in the core extension and communicated via gRPC/protobuf. The GoLang CLI must map these to JSON output fields.
- **State Storage**: JSON output must be compatible with existing `~/.cline/data/` state storage format [Source: .clinerules/storage.md - File-backed JSON storage specification]
- **Plain Mode Integration**: JSON output is activated as part of plain mode when `--json` flag is provided or when output is redirected [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L947-L1015 - Epic 6 related plain mode behavior]

### Proposed New Components:
- **Proposed**: JSON marshaler for message types (Go stdlib `encoding/json`)
- **Proposed**: Streaming JSON encoder with `json.Encoder` for newline-delimited output
- **Proposed**: Message type definitions matching protobuf schema for consistent field naming
- **Proposed**: Error response structure with standardized fields (`type`, `message`, `code`)

### Implementation Notes:
- JSON output should follow newline-delimited JSON (NDJSON) format for streaming compatibility
- All timestamp fields should use Unix epoch integers for consistency
- Boolean flags like `partial` should be omitted when false to reduce output size
- Reasoning fields should be included only when present and non-empty
- Image and file attachments should be represented as arrays of objects with metadata
- Error responses must maintain the same top-level structure as success messages with an added `error` field

## Implementation Priority
**Phase 5: Automation & Scripting (Plain Mode)**

This epic is part of Phase 5 in the AI Execution Plan, following the completion of:
1. Phase 0: Dual Testing Framework
2. Phase 1: Foundational Setup
3. Phase 2: Core CLI Foundation
4. Phase 3: Interactive UI Development
5. Phase 4: Task Management

JSON output depends on the plain mode detection and switching logic implemented in Epic 6 (EPIC-AUTO-MODE-005), making it a natural follow-on to the automation mode infrastructure.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1554-L1571 - Phase 5 in AI Execution Plan]

## Success Metrics
- JSON output is parseable by standard JSON parsers (jq, Python json module, etc.)
- All message types (say, ask, tool, error) can be serialized to JSON
- Streaming output maintains valid JSON lines throughout long-running tasks
- Exit codes are consistent between JSON mode and interactive mode
- Performance overhead of JSON serialization is <5% compared to plain text output

## Dependencies
- **EPIC-AUTO-MODE-005 (Plain Text & Scripting Modes)**: JSON output is a sub-mode of plain text output, requiring the TTY/redirect detection and mode switching infrastructure
- **EPIC-DEV-TASK-003 (Task Management)**: Task execution and message generation must be complete before output formatting can be implemented
- **EPIC-INFRA-CORE-011 (Core Extension Integration)**: gRPC message streaming must be functional to provide messages for JSON serialization

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L947-L1015, L789-L891, L1135-L1177]

## Integration Points
- **Input**: Receives message objects from Task Management layer after gRPC streaming from core extension
- **Output**: Writes JSON lines to stdout (or file if redirected)
- **Integration with Yolo Mode**: JSON output is commonly used with `-y/--yolo` flag for fully automated execution
- **Integration with Timeout**: JSON output should respect `--timeout` flag for long-running tasks
- **State Storage Compatibility**: Task history JSON format should be consistent with output JSON format where applicable

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L906-L946 (Yolo mode integration), L1017-L1078]

## Dual Testing Requirements
Per the **Dual Testing Mandate** in the PRD, this epic's functionality must pass identical test scenarios in both the existing TypeScript CLI and the new GoLang CLI:

### Functional Parity Tests:
- Execute `cline --json 'test prompt'` in both CLIs and compare output structure
- Verify identical field names, types, and ordering in JSON output
- Compare streaming behavior byte-for-byte (excluding timestamps)
- Validate error response format matching

### Output Format Verification:
- Parse JSON output from both CLIs using identical jq queries
- Verify ANSI color codes are suppressed in JSON mode for both implementations
- Confirm identical progress indicator behavior (none in JSON mode)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1246-L1320 - Dual Testing Strategy]