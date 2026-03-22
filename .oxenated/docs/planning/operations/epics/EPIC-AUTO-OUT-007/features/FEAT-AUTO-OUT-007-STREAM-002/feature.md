# JSON Streaming for Long Tasks

## Feature ID
FEAT-AUTO-OUT-007-STREAM-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078]

## Epic Context
**Parent Epic:** EPIC-AUTO-OUT-007 - Structured Output (JSON) [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L1]
**Target Persona:** DevOps/Automation User
**Epic Objective:** Deliver machine-readable JSON output capabilities for the GoLang Cline CLI, enabling seamless integration with automation tools, CI/CD pipelines, and scripting workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1027]
**Business Impact:** Enables programmatic access to Cline's functionality, allowing DevOps engineers and automation specialists to parse task results, integrate with other tools, and build sophisticated automation workflows around Cline's AI capabilities [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1017-L1027]

## Feature Overview
**Purpose:** Stream JSON lines in real-time as messages arrive with partial message handling, enabling live monitoring of automated tasks and providing immediate feedback for long-running AI operations. This feature allows automation scripts and CI/CD pipelines to process Cline's output incrementally rather than waiting for task completion. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1058]
**Scope:** Real-time JSON line streaming for long-running tasks, partial message chunk handling with appropriate flags, and newline-delimited JSON (NDJSON) output format for streaming compatibility.
**PRD References:** REQ-007 (JSON output for scripting) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1294-L1296]
**PRD Feature ID:** EPIC-AUTO-OUT-007-STREAM-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056]
**Dependencies:**
- FEAT-AUTO-OUT-007-JSON-001: JSON Output Formatting (base JSON serialization) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1031-L1054]
- EPIC-AUTO-MODE-005: Plain Text & Scripting Modes (plain mode foundation) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L947-L1015]
- EPIC-INFRA-CORE-011: Core Extension Integration (gRPC message streaming) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1135-L1177]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078 - Feature 2: JSON Streaming for Long Tasks]

**Inputs:**
- Streaming message chunks from AI responses via gRPC
- Partial message flags indicating incomplete content
- Message metadata (type, timestamp, reasoning, say/ask flags)
- Output format flag (`--json`)
- Flush timing requirements for real-time output

**Activities:**
- Stream JSON lines as messages arrive from gRPC connection
- Handle partial streaming messages with appropriate `partial` flags
- Marshal each message chunk to JSON immediately upon arrival
- Flush output buffers for real-time streaming without waiting for task completion
- Track partial message state and mark final messages with `partial: false`
- Coordinate with plain mode output handlers to suppress TUI rendering during JSON streaming
- Ensure thread-safe JSON encoding for concurrent message streams

**Outputs:**
- Streaming JSON lines (newline-delimited JSON/NDJSON) output immediately to stdout
- Partial message chunks with `partial: true` flag during streaming
- Final complete messages with `partial: false` flag
- Machine-readable real-time output suitable for piping to other tools and monitoring systems
- Consistent JSON schema across all streamed messages

**Outcomes:**
- DevOps engineers can monitor long-running tasks in real-time through JSON streams
- Automation scripts can process partial results incrementally without waiting for completion
- CI/CD pipelines can track task progress through streamed JSON events
- External monitoring systems can consume live task updates via structured output
- Tool ecosystem integration enables real-time automation workflows

**Impacts:**
- Enhanced observability for automated workflows through live JSON streaming
- Reduced latency in automation pipelines by enabling incremental result processing
- Foundation for building real-time monitoring dashboards on top of Cline CLI
- Improved debugging capabilities for long-running AI tasks
- Better integration with enterprise logging and monitoring systems

## Technical Requirements
**Architecture Layer:** Application/Infrastructure
**Integration Points:**
- **Existing gRPC:** Bidirectional streaming from EPIC-INFRA-CORE-011-STREAM-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1135-L1177]
- **Existing:** Plain mode detection and switching from EPIC-AUTO-MODE-005 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L947-L1015]
- **Proposed:** JSON streaming encoder component using Go's `json.Encoder` with newline-delimited output
- **Proposed:** Partial message state tracker to manage `partial` flag lifecycle

**Data Requirements:**
- **Existing:** Message type definitions matching protobuf schema from core extension [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1021-L1023]
- **Existing:** Task state and conversation history from `~/.cline/data/` [Source: .clinerules/storage.md]
- **Proposed:** Streaming buffer management for efficient JSON line output

**Performance Requirements:**
- JSON streaming must output messages immediately upon arrival from gRPC (<10ms latency) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1031-L1054]
- Streaming overhead must be <5% compared to plain text output [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L1]
- Buffer flushing must not block gRPC message reception
- Memory usage must remain constant during long streaming sessions (no unbounded buffer growth)

**Security Requirements:**
- JSON output must not include sensitive data (API keys, tokens) unless explicitly marked as log-safe
- Partial message content must follow same security rules as complete messages
- Streaming output to redirected files must respect file permissions

## User Experience
**User Personas:** DevOps/Automation User - engineers integrating Cline into CI/CD pipelines, shell scripts, and automated workflows requiring real-time monitoring [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L90-L98]

**User Actions:**
- Run long-running tasks with `--json` flag to enable streaming JSON output
- Pipe JSON stream to monitoring tools or log aggregation systems
- Process partial results incrementally using line-by-line JSON parsers (jq, Python json module)
- Monitor task progress in real-time through streamed events
- Parse final results from complete messages with `partial: false`

**UI Components:**
- **Proposed:** JSON streaming encoder component (`internal/output/json_streamer.go`)
- **Proposed:** Partial message tracker (`internal/output/partial_tracker.go`)
- **Existing:** Plain mode output handler (suppresses TUI during JSON streaming) [Source: .oxenated/docs/planning/automation/epics/EPIC-AUTO-MODE-005/features/FEAT-AUTO-MODE-005-SWITCH-002/feature.md:L1]

**Mobile Considerations:** Not applicable - CLI is desktop/server-focused

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078 - Gherkin scenarios for JSON Streaming]

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

Scenario: Complete message stream lifecycle
  Given a task generating multiple message types
  When the task starts and produces messages
  Then each message type (say, ask, tool) should stream as JSON
  And partial messages should have partial=true
  And completed messages should have partial=false
  And the stream should end with a completion indicator

Scenario: JSON stream with error handling
  Given a long-running task that encounters an error
  When an error occurs mid-stream
  Then the error should output as JSON line immediately
  With error type and message fields
  And the stream should terminate with appropriate exit code
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078 - Success Metrics for Epic 7]

**Functional:**
- Each message streams as valid JSON line immediately upon arrival from gRPC
- Partial messages correctly include `partial: true` flag
- Final messages correctly include `partial: false` flag
- All message types (say, ask, tool, error) can be streamed to JSON
- JSON streaming maintains valid NDJSON format throughout long-running tasks

**Performance:**
- Message-to-output latency is <10ms from gRPC receipt to stdout
- Streaming overhead is <5% compared to plain text output
- Memory usage remains constant during continuous streaming
- No message loss or corruption during high-frequency streaming

**Quality:**
- All streamed JSON is parseable by standard JSON parsers (jq, Python json module)
- Partial flag transitions are consistent and predictable
- Exit codes are consistent between JSON streaming mode and interactive mode
- Streaming continues correctly after temporary gRPC connection interruptions

**Integration:**
- JSON streaming works seamlessly with `--yolo` flag for automated execution
- JSON streaming respects `--timeout` flag for long-running tasks
- Output format is identical between streaming and non-streaming JSON modes
- Compatible with pipe operations: `cline --json 'task' | jq '.text'`

**Business Value:**
- DevOps engineers can integrate real-time Cline monitoring into existing dashboards
- CI/CD pipelines can process partial results without waiting for task completion
- Automation scripts can react to intermediate outputs during long tasks
- Enterprise monitoring systems can consume live Cline task events

## Testing Strategy
**Unit Testing:**
- Test JSON encoder produces valid NDJSON with proper newline delimiters
- Test partial message tracker correctly manages flag state transitions
- Test streaming buffer flush behavior under various timing conditions
- Test concurrent message handling for thread safety

**Integration Testing:**
- Test gRPC streaming integration with mock core extension
- Test JSON stream output matches expected schema for all message types
- Test partial message lifecycle from first chunk to completion
- Test interaction with plain mode switching logic
- Test buffer behavior under slow consumer conditions (backpressure)

**User Acceptance:**
- DevOps engineer can pipe JSON stream to jq and parse results in real-time
- CI/CD pipeline can process partial outputs incrementally
- Long-running task (5+ minutes) streams complete JSON output without corruption
- Partial messages display correctly in monitoring tools

**Performance Testing:**
- Measure message-to-output latency under various loads
- Verify memory usage remains constant during 1-hour continuous streaming
- Benchmark streaming overhead vs plain text mode
- Test with high-frequency message streams (100+ messages/second)

## Tasks Overview
1. **Task 1:** Implement JSON streaming encoder with NDJSON format support
2. **Task 2:** Create partial message state tracker for managing `partial` flag lifecycle
3. **Task 3:** Integrate streaming encoder with gRPC message reception
4. **Task 4:** Implement buffer management and flush strategies for real-time output
5. **Task 5:** Add thread-safe concurrent message handling
6. **Task 6:** Create comprehensive unit and integration tests for streaming functionality

## Implementation Notes
1. **NDJSON Format**: Use newline-delimited JSON format where each line is a complete JSON object, enabling easy line-by-line parsing by consumers [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md:L1]

2. **Go Implementation**: Use Go's standard `encoding/json` package with `json.Encoder` for efficient streaming. The encoder provides built-in newline handling and buffering suitable for NDJSON output.

3. **Partial Message Handling**: The core extension signals partial messages through gRPC streaming. The Go CLI must track these signals and set the `partial` flag accordingly in JSON output.

4. **Buffer Management**: Implement configurable buffer flushing strategies:
   - Immediate flush for real-time monitoring (default)
   - Buffered flush for high-throughput scenarios (optional optimization)

5. **Error Handling**: Errors during streaming must:
   - Output error as JSON line if possible
   - Set appropriate exit code
   - Not leave incomplete JSON lines in output

6. **Cross-Persona Integration**: This feature works closely with:
   - **EPIC-AUTO-EXEC-006-YOLO-001**: JSON streaming is commonly used with yolo mode for fully automated CI/CD pipelines
   - **EPIC-ENT-AUDIT-010-LOG-001**: JSON stream format should be compatible with audit logging requirements

7. **Dual Testing Mandate**: Per the PRD dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L650-L730], all JSON streaming functionality must be tested in BOTH the existing TypeScript CLI and the new GoLang CLI to ensure:
   - Identical streaming behavior byte-for-byte (excluding timestamps)
   - Identical partial flag handling
   - Identical NDJSON format compliance
   - Performance parity or improvement

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1056-L1078]
- [x] Existing code references cite actual file paths and lines where applicable
- [x] Integration points cite existing interfaces or marked as new
- [x] Epic source cited: [Source: .oxenated/docs/planning/operations/epics/EPIC-AUTO-OUT-007/epic.md]