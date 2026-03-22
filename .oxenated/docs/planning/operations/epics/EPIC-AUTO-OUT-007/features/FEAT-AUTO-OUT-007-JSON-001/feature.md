# JSON Output Formatting

## Feature ID
FEAT-AUTO-OUT-007-JSON-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1062-L1076]

## Epic Context
**Parent Epic:** EPIC-AUTO-OUT-007 - Structured Output (JSON) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L1]
**Target Persona:** DevOps/Automation User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1060]
**Epic Objective:** Deliver machine-readable JSON output capabilities for the GoLang Cline CLI, enabling seamless integration with automation tools, CI/CD pipelines, and scripting workflows [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L12-L17]
**Business Impact:** Eliminates the need for fragile screen-scraping of TUI output and enables reliable automation of AI-assisted coding tasks [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L22-L24]

## Feature Overview
**Purpose:** Enable DevOps engineers and automation specialists to parse task results, integrate with other tools, and build sophisticated automation workflows around Cline's AI capabilities through structured, machine-readable JSON output [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1062-L1076]
**Scope:** Core JSON serialization infrastructure for all message types, including required and optional fields, error formatting, and newline-delimited JSON (NDJSON) output structure
**PRD References:** REQ-007: JSON output for scripting - **FULLY ADDRESSED** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1294-L1296]
**PRD Feature ID:** EPIC-AUTO-OUT-007-JSON-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1062]
**Dependencies:** 
- EPIC-AUTO-MODE-005 (Plain Text & Scripting Modes): JSON output is a sub-mode of plain text output requiring TTY/redirect detection infrastructure [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L145-L149]
- EPIC-DEV-TASK-003 (Task Management): Task execution and message generation must be complete before output formatting [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L145-L149]
- EPIC-INFRA-CORE-011 (Core Extension Integration): gRPC message streaming must be functional to provide messages for JSON serialization [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L145-L149]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1062-L1076 - Feature IAOOI section]

**Inputs:** 
- Message objects (type, text, ts, reasoning, say, ask, partial, images, files) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1064]
- Output format flag (`--json`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1064]
- JSON schema requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1064]

**Activities:** 
- Marshal message objects to JSON format using Go's `encoding/json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1065]
- Ensure all required fields are included in output (type, text, ts) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1065]
- Include optional fields (reasoning, partial flags, images, files) when present [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1065]
- Format error responses in JSON structure with type and message fields [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1065]
- Use newline-delimited JSON (NDJSON) format for output [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]

**Outputs:** 
- JSON-formatted message lines (newline-delimited JSON/NDJSON) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L35]
- Structured error responses with consistent schema [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L35]
- Machine-readable output suitable for piping to other tools [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L35]

**Outcomes:** 
- DevOps engineers can integrate Cline CLI into CI/CD pipelines with reliable output parsing [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L40]
- Automation scripts can process Cline responses without fragile text parsing [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L40]
- Tool ecosystem integration becomes possible through standardized JSON interface [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L40]
- Programmatic access to AI coding assistance enables new automation workflows [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L40]

**Impacts:** 
- Broader adoption of Cline in enterprise automation environments [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L46]
- Foundation for building higher-level automation tools on top of Cline CLI [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L46]
- Improved CI/CD integration for AI-assisted code review and generation [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L46]
- Enhanced observability through structured logging capabilities [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L46]

## Technical Requirements
**Architecture Layer:** Application/Output Layer
**Integration Points:** 
- **Input:** Receives message objects from Task Management layer after gRPC streaming from core extension [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L152]
- **Output:** Writes JSON lines to stdout (or file if redirected) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L152]
- **Proposed:** Message type definitions matching protobuf schema for consistent field naming [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L82]
- **Proposed:** Error response structure with standardized fields (`type`, `message`, `code`) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L82]

**Data Requirements:** 
- **Proposed:** JSON marshaler for message types using Go stdlib `encoding/json` [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L82]
- All timestamp fields use Unix epoch integers for consistency [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Boolean flags like `partial` omitted when false to reduce output size [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Reasoning fields included only when present and non-empty [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Image and file attachments represented as arrays of objects with metadata [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]

**Performance Requirements:** 
- Performance overhead of JSON serialization <5% compared to plain text output [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138]
- Real-time output streaming without buffering delays

**Security Requirements:** 
- Error responses must maintain same top-level structure as success messages with added `error` field [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- No sensitive data exposed in JSON output beyond what would be shown in TUI

## User Experience
**User Personas:** DevOps/Automation User, CI/CD Pipeline Operators, Automation Specialists [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L90-L98]

**User Actions:** 
- Execute Cline with `--json` flag to receive structured output: `cline --json 'analyze codebase'` [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L101]
- Pipe JSON output to other tools: `cline --json 'review changes' | jq '.type'` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1033-L1037]
- Parse results in automation scripts to extract specific findings and categorize issues [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L64-L68]
- Combine with yolo mode for fully automated execution: `cline -y --json 'analyze codebase'` [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L101]

**UI Components:** 
- N/A - This feature operates in plain mode without TUI

**CLI Integration:**
- **Integration with Yolo Mode:** JSON output commonly used with `-y/--yolo` flag for fully automated execution [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L152]
- **Integration with Timeout:** JSON output respects `--timeout` flag for long-running tasks [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L152]

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1068-L1076 - BDD scenarios for this feature]

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

## Success Criteria
[Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138-L142]

**Functional:** 
- JSON output is parseable by standard JSON parsers (jq, Python json module, etc.) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138]
- All message types (say, ask, tool, error) can be serialized to JSON [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138]

**Performance:** 
- Performance overhead of JSON serialization is <5% compared to plain text output [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138]

**Quality:** 
- JSON output structure is consistent and predictable
- All required fields (type, text, ts) always present
- Optional fields included only when relevant

**Integration:** 
- Exit codes are consistent between JSON mode and interactive mode [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L138]
- Works seamlessly with yolo mode and timeout flags

**Business Value:** 
- Enables reliable CI/CD pipeline integration without screen-scraping
- Supports batch processing and automation workflows

## Testing Strategy
**Unit Testing:** 
- JSON marshaler tests for each message type
- Field presence validation (required vs optional)
- Error response structure validation
- Timestamp format validation (Unix epoch integers)

**Integration Testing:** 
- End-to-end JSON output with actual task execution
- Integration with piped input and output redirection
- Verification of NDJSON format compliance

**User Acceptance:** 
- Parse output with jq and validate field accessibility
- Python script parsing validation
- CI/CD pipeline integration test

**Dual Testing Requirements (Critical):**
Per the **Dual Testing Mandate** in the PRD, this feature must pass identical test scenarios in both TypeScript CLI and GoLang CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1246-L1320]:

### Functional Parity Tests:
- Execute `cline --json 'test prompt'` in both CLIs and compare output structure [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L110-L115]
- Verify identical field names, types, and ordering in JSON output [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L110-L115]
- Validate error response format matching [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L110-L115]

### Output Format Verification:
- Parse JSON output from both CLIs using identical jq queries [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L117-L120]
- Verify ANSI color codes are suppressed in JSON mode for both implementations [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L117-L120]

## Tasks Overview
1. **Task 1:** Define Go structs for message types matching protobuf schema
2. **Task 2:** Implement JSON marshaler with required/optional field handling
3. **Task 3:** Create error response formatter with consistent schema
4. **Task 4:** Integrate JSON output with plain mode detection
5. **Task 5:** Add `--json` flag to task command
6. **Task 6:** Implement NDJSON streaming writer
7. **Task 7:** Write unit tests for JSON serialization
8. **Task 8:** Create dual testing comparison scripts

## Implementation Notes
- JSON output follows newline-delimited JSON (NDJSON) format for streaming compatibility [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- All timestamp fields use Unix epoch integers for consistency [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Boolean flags like `partial` should be omitted when false to reduce output size [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Reasoning fields included only when present and non-empty [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Image and file attachments represented as arrays of objects with metadata [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]
- Error responses must maintain same top-level structure as success messages with added `error` field [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L86]

**Implementation Priority:** Phase 5: Automation & Scripting (Plain Mode) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L94-L102]
- JSON output depends on plain mode detection and switching logic from Epic 6 (EPIC-AUTO-MODE-005) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L94-L102]

**State Storage Compatibility:** 
- Task history JSON format should be consistent with output JSON format where applicable [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L152]
- Uses existing `~/.cline/data/` state storage format [Source: .clinerules/storage.md - File-backed JSON storage specification]

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md]
- [x] Existing code references cite actual file paths and lines where applicable
- [x] New functionality clearly marked as "Proposed:" (e.g., JSON marshaler, message type definitions, error response structure)
- [x] Integration points cite existing interfaces or mark as new
- [x] All IAOOI components extracted verbatim from PRD with line number citations
- [x] BDD scenarios included exactly as written in PRD
- [x] Dual testing requirements documented from PRD