# JSON Output Formatting

## Feature ID
FEAT-AUTO-OUT-007-JSON-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1263-L1283]

## Epic Context
**Parent Epic:** EPIC-AUTO-OUT-007 - Structured Output (JSON) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-OUT-007/epic.md]
**Target Persona:** DevOps/Automation User
**Epic Objective:** Implement structured JSON output formatting for the GoLang CLI to enable machine-readable output for integration with other tools, CI/CD pipelines, and automation workflows.
**Business Impact:** Enables programmatic access, tool ecosystem integration, and automation workflows that parse CLI output programmatically, directly addressing DevOps/Automation User pain points around JSON output for parsing.

## Feature Overview
**Purpose:** This feature enables the GoLang CLI to serialize all message objects to JSON format when the `--json` flag is provided, producing machine-readable output that can be parsed by scripts and automation tools. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1263-L1268]
**Scope:** Includes JSON marshaling of message objects with all required and optional fields, structured error output, and support for JSON output mode in both interactive and non-interactive contexts.
**PRD References:** REQ-007 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1354]
**Dependencies:**
- TTY/Redirect detection (FEAT-AUTO-MODE-005-DETECT-001) - for automatic mode switching context
- Message streaming from core extension (EPIC-INFRA-CORE-011-STREAM-002) - for message source
- Protobuf message definitions - for message structure [Source: proto/cline/ directory]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1263-L1273]

**Inputs:**
- Message objects containing: type, text, ts (timestamp), reasoning, say, ask, partial flag, images, files
- Output format flag (`--json`) from command line arguments
- JSON schema requirements for field naming and structure

**Activities:**
- Marshal message structs to JSON format using Go's encoding/json package
- Ensure required fields are always included in output
- Include optional fields only when present (non-empty/non-nil)
- Handle error serialization to JSON format with error type and message fields
- Output JSON as single lines (JSONL format) for streaming compatibility

**Outputs:**
- JSON-formatted message lines output to stdout
- Structured error responses in JSON format with `error` type field
- Consistent field ordering and naming matching existing TypeScript CLI

**Outcomes:**
- Scripts and automation tools can parse CLI output programmatically
- Integration with CI/CD pipelines through machine-readable output
- Tool ecosystem can consume Cline CLI output reliably
- Exit codes can be checked while output is parsed for results

**Impacts:**
- Broader adoption in automation and DevOps workflows
- Better CI/CD integration for enterprises
- Foundation for building tools that orchestrate Cline CLI
- Reduced need for screen scraping or text parsing heuristics

## Technical Requirements
**Architecture Layer:** Application/Output Layer
**Integration Points:**
- Proposed: New JSON output handler to be created in `golang-cli/internal/output/` OR `golang-cli/pkg/output/`
- Existing: Message stream from core extension via gRPC [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1311-L1326]
- Proposed: Integration with TTY detection for automatic mode switching
**Data Requirements:**
- Proposed: Message struct definition matching protobuf message structure
- Required fields: type (string), text (string), ts (int64/timestamp)
- Optional fields: reasoning (string), say (string), ask (string), partial (bool), images ([]string), files ([]string)
**Performance Requirements:**
- JSON marshaling must not block message streaming (<1ms per message)
- Output must be flushed immediately for real-time streaming
**Security Requirements:**
- No sensitive data (API keys, secrets) should appear in JSON output
- Error messages must not expose internal system details

## User Experience
**User Personas:** DevOps/Automation User - primarily uses CLI in scripts and pipelines
**User Actions:**
1. Run CLI with `--json` flag to get structured output
2. Pipe JSON output to `jq` or other parsing tools
3. Parse JSON output in scripts to extract task results
4. Monitor long-running tasks via streaming JSON lines
**UI Components:**
- Proposed: JSON output formatter component (no TUI in JSON mode)
- Text output when not in JSON mode (normal TUI behavior)
**Mobile Considerations:** Not applicable for CLI tool

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1275-L1283]

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1263-L1273]

**Functional:**
- All message types serialize correctly to JSON with exact field names matching existing TypeScript CLI
- Required fields (type, text, ts) are always present in output
- Optional fields (reasoning, say, ask, partial, images, files) only appear when non-empty/non-nil
- Error messages output as JSON with `{"type":"error","text":"..."}` format
- JSON output is valid parseable JSON (one object per line)

**Performance:**
- JSON marshaling latency <1ms per message
- Output flushes immediately without buffering delays
- No memory leaks during long-running streaming tasks

**Quality:**
- Passes dual testing comparison with existing TypeScript CLI JSON output
- Byte-for-byte identical output (except for timestamps and generated IDs)
- All edge cases handled (empty strings, null values, special characters)

**Integration:**
- Works seamlessly with `--yolo` flag for automated execution
- Compatible with piped input and output redirection
- Integrates with TTY detection for automatic mode switching

**Business Value:**
- DevOps/Automation Users can reliably parse CLI output in scripts
- Enables CI/CD pipeline integration without interactive prompts
- Allows building higher-level tools that orchestrate Cline CLI

## Testing Strategy
**Unit Testing:**
- Test JSON marshaling for all message field combinations
- Verify optional field omission when empty
- Test error message JSON formatting
- Validate JSON schema compliance

**Integration Testing:**
- Test JSON output with actual message stream from core extension
- Verify streaming behavior with multiple rapid messages
- Test combined `--json --yolo` flag behavior
- Validate exit codes are still returned correctly in JSON mode

**User Acceptance:**
- Script can parse JSON output using `jq`
- JSON output can be piped to other CLI tools
- Output matches existing TypeScript CLI format exactly
- Dual testing shows functional parity

**Performance Testing:**
- Benchmark JSON marshaling latency
- Test memory usage during long streaming sessions
- Verify no output buffering delays

## Tasks Overview
1. Define Message struct matching protobuf structure
2. Implement JSON marshaling with proper field tags
3. Create JSON output handler/writer
4. Add `--json` flag to CLI commands
5. Integrate JSON output with message stream
6. Implement error JSON formatting
7. Add unit tests for all message types
8. Implement dual testing comparison

## Implementation Notes
- Use Go struct tags for JSON field naming: `json:"type,omitempty"`
- Consider using `omitempty` for optional fields to match TypeScript behavior
- JSON output must be line-delimited (JSONL) for streaming compatibility
- Ensure proper escaping of special characters in text fields
- Maintain exact field name compatibility with existing TypeScript CLI JSON output
- Message type values must match existing CLI: "say", "ask", "error", etc.
- Timestamp field should be Unix timestamp in milliseconds or seconds (verify existing behavior)

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1263-L1283]
- [x] Existing code references cite actual file paths and lines (protobuf directory referenced)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new