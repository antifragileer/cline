# Bidirectional Streaming

## Feature ID
FEAT-INFRA-CORE-011-STREAM-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]

## Epic Context
**Parent Epic:** EPIC-INFRA-CORE-011 - Core Extension Integration [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Enable the GoLang CLI to communicate seamlessly with the existing Cline core extension through gRPC/protobuf, achieving full bidirectional message streaming for real-time task execution [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L859-L875]
**Business Impact:** This feature is foundational to the entire GoLang CLI migration, enabling real-time communication that supports responsive terminal UI and consistent behavior across VSCode extension, JetBrains, and CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L876-L879]

## Feature Overview
**Purpose:** Implement bidirectional streaming communication between the GoLang CLI and the TypeScript-based Cline core extension, enabling real-time message flow for task execution, tool approval workflows, and state synchronization [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]
**Scope:** 
- Outgoing message streaming from CLI to core (user inputs, approvals, commands)
- Incoming message streaming from core to CLI (AI responses, tool requests, state updates)
- Flow control and buffering management
- Stream error handling and reconnection logic
**PRD References:** REQ-011 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1045-L1055]
**PRD Feature ID:** FEAT-INFRA-CORE-011-STREAM-002
**Dependencies:** 
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation (must establish connection first) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L885-L900]
- Protobuf definitions from `proto/` directory (already exist) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1264]
- EPIC-INFRA-STORAGE-012: State & Storage Layer (for local state persistence during streaming) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916 - IAOOI section for this feature]

**Inputs:**
1. Outgoing messages from CLI (user prompts, tool approvals/rejections, commands)
2. Incoming streaming responses from core extension (AI-generated content, tool use requests)
3. Connection state and health metrics
4. Flow control signals (backpressure, buffer status)
5. Reconnection triggers and retry configuration

**Activities:**
1. Establish bidirectional gRPC streaming connection to core extension
2. Send user messages to core via outgoing stream
3. Receive AI responses via incoming stream in real-time
4. Handle streaming message chunks (partial content accumulation)
5. Implement flow control to prevent overwhelming either side
6. Buffer messages during temporary disconnections
7. Handle stream errors gracefully with automatic reconnection
8. Serialize/deserialize protobuf messages on the wire
9. Coordinate with TUI for real-time message display updates
10. Manage stream lifecycle (open, maintain, close, reconnect)

**Outputs:**
1. Bidirectional message stream established and maintained
2. User messages transmitted to core extension
3. AI responses streamed to CLI and rendered in TUI
4. Tool approval requests displayed to user in real-time
5. Stream health and connection status events
6. Reconnection events and recovery notifications
7. Error messages for unrecoverable stream failures

**Outcomes:**
1. Real-time communication enabling responsive terminal UI
2. Seamless user experience with immediate feedback
3. Support for long-running tasks with continuous streaming
4. Reliable message delivery despite network interruptions
5. Consistent behavior matching VSCode extension and JetBrains

**Impacts:**
1. Foundation for all interactive task execution features
2. Enables human-in-the-loop tool approval workflows
3. Supports complex multi-turn conversations
4. Reduces perceived latency through streaming chunks
5. Critical path for achieving feature parity with existing CLI

## Technical Requirements
**Architecture Layer:** Infrastructure (gRPC client, streaming handlers, message router)
**Integration Points:** 
- Proposed: gRPC streaming methods defined in `proto/` directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1264]
- Proposed: Message Router (`golang-cli/internal/messages/`) - Routes incoming gRPC messages to appropriate handlers [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1]
- Integrates with: EPIC-DEV-UI-002 (Interactive Terminal UI) for message display [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]
- Integrates with: EPIC-DEV-TASK-003 (Task Management) for task execution coordination [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]

**Data Requirements:** 
- Proposed: Stream state tracking (connection status, buffer state, retry count)
- Proposed: Message queue for buffering during reconnection
- Proposed: Flow control windows and backpressure thresholds

**Performance Requirements:**
- Message latency: <100ms for delivery between CLI and core [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Streaming throughput: Support high-frequency message exchange during active tasks
- Reconnection time: <500ms to re-establish stream after interruption [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Buffer capacity: Sufficient to queue messages during brief disconnections

**Security Requirements:**
- All streaming communication over gRPC (inherently uses HTTP/2 with TLS)
- No sensitive data logging in stream buffers
- Secure handling of tool approval responses

## User Experience
**User Personas:** Infrastructure (Internal - enables Developer User experience)
**User Actions:**
1. User submits prompt → CLI streams to core → AI responses stream back immediately
2. AI requests tool approval → Request displays instantly via incoming stream
3. User approves/rejects → Response streams back to core via outgoing stream
4. Network interruption → Stream reconnects automatically → User sees brief "reconnecting" indicator
5. Long-running task → Continuous streaming updates keep user informed

**UI Components:**
- Proposed: Connection status indicator (connected, reconnecting, error)
- Proposed: Streaming message renderer (handles partial content updates)
- Proposed: Tool approval prompt (triggered by incoming stream messages)

**Mobile Considerations:** N/A (Infrastructure feature)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916 - BDD scenarios for this feature]

### Feature: Bidirectional Streaming

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

### Additional Derived Scenarios

```gherkin
Scenario: Receive tool approval request via stream
  Given an active bidirectional stream
  When core sends tool use request
  Then the request should display immediately in TUI
  And user should be able to respond

Scenario: Send approval response via stream
  Given a tool approval prompt is displayed
  When user approves the tool
  Then approval should transmit to core via outgoing stream
  And task execution should continue

Scenario: Handle stream interruption during task
  Given a long-running task with active stream
  When network connection drops
  Then stream should detect disconnection
  And begin reconnection attempts with exponential backoff
  And user should see "reconnecting" status

Scenario: Recover from stream interruption
  Given stream was interrupted during active task
  When connection is restored
  Then stream should re-establish
  And pending messages should transmit
  And task should resume from last state

Scenario: Buffer messages during brief disconnection
  Given stream experiences brief network glitch
  When connection drops for <5 seconds
  Then outgoing messages should buffer locally
  And incoming messages should queue on core side
  And no messages should be lost after reconnection

Scenario: Handle backpressure from core
  Given core is processing messages slowly
  When CLI sends multiple rapid messages
  Then flow control should throttle outgoing stream
  And prevent overwhelming the core extension
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1190-L1205 - Success criteria from PRD]

**Functional:**
- Bidirectional streaming establishes and maintains connection successfully
- User messages transmit to core within 100ms latency target
- AI responses stream back in real-time with partial content updates
- Tool approval requests display immediately upon receipt
- Stream handles network interruptions with automatic reconnection
- Zero message loss during brief disconnections (<5 seconds)

**Performance:**
- Message delivery latency: <100ms [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Reconnection success rate: >99% after network interruption [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1190-L1205]
- Stream establishment time: <500ms from connection to first message

**Quality:**
- Graceful handling of all error conditions
- No memory leaks during long-running streams
- Thread-safe message queuing and delivery
- Proper resource cleanup on stream close

**Integration:**
- Works seamlessly with Bubble Tea TUI for message display
- Integrates correctly with task management for state coordination
- Compatible with existing TypeScript core extension streaming

**Business Value:**
- Enables real-time interactive task execution
- Supports human-in-the-loop tool approval workflows
- Foundation for all user-facing interactive features

## Testing Strategy
**Unit Testing:**
- Stream connection establishment and teardown
- Message serialization/deserialization
- Flow control and backpressure handling
- Reconnection logic with mocked network failures
- Buffer management and overflow handling

**Integration Testing:**
- Bidirectional streaming with mock gRPC server
- End-to-end message flow with actual core extension
- Concurrent message handling (multiple rapid sends/receives)
- State synchronization during stream interruptions

**User Acceptance:**
- Real-time feel verified through latency measurements
- Reconnection behavior tested with actual network interruptions
- Side-by-side comparison with existing TypeScript CLI streaming
- Tool approval workflow timing verification

**Performance Testing:**
- Message throughput under load
- Latency benchmarks against existing CLI
- Memory usage during extended streaming sessions
- Reconnection timing verification

**Dual Testing Requirements:**
- All streaming tests must pass in BOTH existing TypeScript CLI and new GoLang CLI
- Side-by-side task resumption verification (start in one CLI, resume in other)
- Concurrent execution scenarios with both CLIs streaming to same core
- Output format and timing comparison between implementations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]

## Tasks Overview
1. **TASK-STREAM-001:** Implement bidirectional gRPC streaming client in Go
2. **TASK-STREAM-002:** Create message router for incoming stream handling
3. **TASK-STREAM-003:** Implement outgoing message queue and sender
4. **TASK-STREAM-004:** Build reconnection logic with exponential backoff
5. **TASK-STREAM-005:** Integrate streaming with Bubble Tea TUI update loop
6. **TASK-STREAM-006:** Add flow control and backpressure management
7. **TASK-STREAM-007:** Write comprehensive unit and integration tests

## Implementation Notes

### Critical Architecture Decisions

**Stream Type Selection:**
- Use gRPC bidirectional streaming (not unary or server-streaming only)
- Single long-lived stream per task session
- Separate goroutines for send and receive to prevent blocking

**Reconnection Strategy:**
- Implement exponential backoff: 100ms, 200ms, 400ms, 800ms, max 5 seconds
- Maximum retry attempts: 10 before failing task
- Buffer outgoing messages during reconnection (max 100 messages)
- On successful reconnection, sync state and resume from last checkpoint

**Flow Control:**
- Implement gRPC flow control (HTTP/2 windowing)
- Add application-level backpressure: pause outgoing stream if core signals busy
- Buffer size limits: 1000 incoming, 100 outgoing messages

**Thread Safety:**
- All stream operations must be thread-safe for concurrent access
- Use channels for message passing between stream handlers and TUI
- Protect shared state with mutexes or atomic operations

### Independence Requirements

**This feature MUST maintain strict independence from existing TypeScript CLI:**
- No code import from `cli/src/` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]
- No npm package dependencies [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]
- Pure Go implementation using standard gRPC Go libraries
- Communication ONLY through protobuf/gRPC interface [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]

### Proposed New Components

**Proposed: Stream Manager** (`golang-cli/internal/grpc/stream_manager.go`)
- Manages bidirectional stream lifecycle
- Handles reconnection logic
- Coordinates send/receive goroutines

**Proposed: Message Router** (`golang-cli/internal/messages/router.go`)
- Routes incoming gRPC messages to appropriate handlers
- Dispatches user actions to core via gRPC
- Manages message queuing during reconnection

**Proposed: Flow Controller** (`golang-cli/internal/grpc/flow_control.go`)
- Implements backpressure mechanisms
- Monitors buffer states
- Throttles message flow when needed

## Dependencies

**Prerequisites:**
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation (connection foundation)
- Protobuf definitions compiled to Go (`proto/` → generated Go code)
- EPIC-INFRA-STORAGE-012: State storage for buffering during reconnection

**Blocks:**
- EPIC-DEV-TASK-003: Task Management (requires streaming for execution)
- EPIC-DEV-UI-002: Interactive Terminal UI (requires streaming for real-time display)
- FEAT-INFRA-CORE-011-STATE-003: State Synchronization (uses same stream for state updates)

**Coordination Points:**
- After Phase 1: Align on proto definitions with all agents
- After Phase 4: Core functionality works end-to-end; full dual test suite runs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1292]

## Traceability
| Feature ID | Requirement IDs |
|------------|-----------------|
| FEAT-INFRA-CORE-011-STREAM-002 | REQ-011 |
| TASK-STREAM-001 | REQ-011 |
| TASK-STREAM-002 | REQ-011 |
| TASK-STREAM-003 | REQ-011 |
| TASK-STREAM-004 | REQ-011 |
| TASK-STREAM-005 | REQ-011 |
| TASK-STREAM-006 | REQ-011 |
| TASK-STREAM-007 | REQ-011 |

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L##]
- [x] Existing code references cite actual file paths (proto/ directory)
- [x] New functionality clearly marked as "Proposed:" when it doesn't exist yet
- [x] Integration points cite existing interfaces or mark as new
- [x] Epic reference cites .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md
- [x] BDD scenarios extracted verbatim from PRD
- [x] IAOOI components extracted from PRD epic section
- [x] Success metrics extracted from PRD