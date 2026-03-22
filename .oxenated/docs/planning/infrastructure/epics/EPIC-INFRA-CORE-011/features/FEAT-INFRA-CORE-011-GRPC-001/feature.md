# gRPC Client Implementation

## Feature ID
FEAT-INFRA-CORE-011-GRPC-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1100-L1125]

## Epic Context
**Parent Epic:** EPIC-INFRA-CORE-011 - Core Extension Integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Establish seamless integration between the GoLang CLI and the existing Cline core extension via gRPC/protobuf, enabling feature reuse and consistent behavior across platforms. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1098]
**Business Impact:** Enables the GoLang CLI to leverage all existing Cline core functionality without reimplementation, ensuring feature parity and faster development. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1098-L1100]

## Feature Overview
**Purpose:** Implement a GoLang gRPC client that communicates with the Cline core extension, enabling the CLI to create tasks, stream messages, and synchronize state with the core extension via protobuf-defined RPC methods. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1100-L1108]
**Scope:** Includes protobuf code generation, gRPC client implementation, connection management, and all RPC method implementations required for task execution.
**PRD References:** REQ-011 (Integrate with existing Cline core via gRPC/protobuf) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1108]
**PRD Feature ID:** EPIC-INFRA-CORE-011-GRPC-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1100]
**Dependencies:** 
- Protobuf definitions from `proto/` directory [Source: proto/cline/]
- Core extension running and exposing gRPC endpoint
- Generated Go protobuf code (to be created)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1100-L1108 - IAOOI section for this feature]

**Inputs:**
- Protobuf definitions from `proto/` directory
- Core extension endpoint configuration
- Connection settings (timeout, retry policy)
- Outgoing RPC request messages (NewTask, ResumeTask, SendMessage, etc.)
- Authentication context for secure communication

**Activities:**
- Generate Go code from existing protobuf definitions using protoc-gen-go
- Implement gRPC client with connection pooling and lifecycle management
- Establish secure connection to Cline core extension gRPC endpoint
- Implement all RPC methods defined in proto files (NewTask, ResumeTask, StreamMessages, etc.)
- Handle connection failures with automatic retry and reconnection logic
- Manage request/response serialization and deserialization
- Implement interceptors for logging, metrics, and authentication

**Outputs:**
- Connected gRPC client ready for RPC calls
- Method implementations for all core extension RPCs
- Bidirectional streaming message handlers
- Connection state monitoring and health checks
- Error responses with proper gRPC status codes

**Outcomes:**
- GoLang CLI can communicate with Cline core extension via gRPC
- All task operations (create, resume, message) work through RPC calls
- Real-time message streaming between CLI and core extension
- Reliable connection with automatic reconnection on failure

**Impacts:**
- Feature parity with existing TypeScript CLI via shared core extension
- Reduced code duplication by reusing core extension logic
- Consistent behavior across VSCode/CLI/JetBrains through shared core
- Foundation for future native integrations and platform expansion

## Technical Requirements
**Architecture Layer:** Infrastructure/Integration Layer

**Integration Points:**
- **Existing Protobuf Definitions:** [Source: proto/cline/*.proto]
  - `proto/cline/task.proto` - Task-related RPCs (NewTask, ResumeTask)
  - `proto/cline/ui.proto` - UI-related RPCs (scrollToSettings, etc.)
  - `proto/cline/common.proto` - Shared message types (StringRequest, KeyValuePair, Empty)
  - `proto/cline/state.proto` - State management RPCs
- **Proposed: New Go gRPC Client** - To be created in `golang-cli/internal/grpc/` or similar
- **Proposed: Connection Manager** - To be created for lifecycle management

**Data Requirements:**
- **Existing:** Protobuf message definitions in `proto/` directory
- **Proposed: Go protobuf generated code** - Generated via `protoc` with `--go_out` and `--go-grpc_out` plugins
- **Proposed: Client configuration** - Endpoint URL, TLS settings, authentication tokens

**Performance Requirements:**
- Connection establishment: <100ms on localhost
- RPC latency: <10ms for unary calls on localhost
- Streaming latency: <50ms for message delivery
- Automatic reconnection within 5 seconds of connection loss
- Support for concurrent RPC calls (at least 10 concurrent streams)

**Security Requirements:**
- TLS encryption for all gRPC connections
- Authentication via API keys or OAuth tokens
- Certificate validation for production endpoints
- Secure handling of authentication context in interceptors

## User Experience
**User Personas:** Infrastructure Team, CLI Developers (indirect - this is an internal infrastructure feature)

**User Actions:** (System-level interactions)
- CLI initializes and establishes gRPC connection on startup
- CLI creates new tasks via RPC calls
- CLI resumes existing tasks via RPC calls
- CLI sends user messages to core extension
- CLI receives streaming AI responses from core extension
- CLI handles connection failures gracefully with retry

**UI Components:** 
- **Existing:** Core extension handles all UI logic - CLI receives streaming messages and renders them via Bubble Tea TUI [Source: EPIC-DEV-UI-002]
- **Proposed:** Connection status indicator in TUI (optional) showing gRPC connection state

**Mobile Considerations:** N/A - CLI is desktop terminal application

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1110-L1125 - BDD scenarios for this feature]

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

Scenario: Handle connection failure
  Given an active gRPC connection
  When the core extension becomes unavailable
  Then the client should detect the failure
  And attempt automatic reconnection
  And resume operations when core is back

Scenario: Stream messages bidirectionally
  Given an active task
  When user sends message
  Then it should transmit to core via RPC
  And AI responses should stream back
  In real-time via gRPC streaming

Scenario: Handle streaming errors
  Given an active stream
  When network error occurs
  Then stream should handle gracefully
  And attempt reconnection without data loss
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1100-L1125 - Success criteria from PRD]

**Functional:**
- Go code generation from all proto files succeeds without errors
- gRPC client connects successfully to running core extension
- All unary RPC methods (NewTask, ResumeTask, etc.) execute successfully
- Bidirectional streaming RPCs handle message flow correctly
- Connection failures trigger automatic retry with exponential backoff
- All protobuf message types serialize/deserialize correctly

**Performance:**
- Connection establishment completes within 100ms on localhost
- Unary RPC calls complete within 10ms on localhost
- Streaming messages deliver with <50ms latency
- Reconnection occurs within 5 seconds of connection loss

**Quality:**
- Unit test coverage >80% for gRPC client code
- Integration tests pass against mock core extension
- Error handling covers all gRPC status codes appropriately
- No memory leaks in long-running streaming connections

**Integration:**
- Works seamlessly with Bubble Tea TUI for message display [Source: EPIC-DEV-UI-002]
- Integrates correctly with task management layer [Source: EPIC-DEV-TASK-003]
- State synchronization works bidirectionally [Source: EPIC-INFRA-CORE-011-STATE-003]

**Business Value:**
- Enables all other CLI features that depend on core extension
- Reduces implementation time by reusing existing core logic
- Ensures consistent behavior with VSCode extension

## Testing Strategy
**Unit Testing:**
- Test protobuf message serialization/deserialization for all message types
- Test gRPC client initialization with various configuration options
- Test connection manager lifecycle (connect, disconnect, reconnect)
- Test interceptor chain execution (logging, auth, metrics)
- Mock gRPC server for testing RPC method handlers

**Integration Testing:**
- Test against running core extension in test environment
- Test bidirectional streaming with simulated message flow
- Test connection failure and reconnection scenarios
- Test concurrent RPC calls and streaming
- Test TLS/authentication integration

**User Acceptance:** (Infrastructure team validation)
- Verify connection to core extension succeeds in development environment
- Verify all task operations work end-to-end through gRPC
- Verify streaming performance meets latency requirements
- Verify reconnection works during core extension restart

**Performance Testing:**
- Benchmark connection establishment time
- Benchmark RPC call latency under load
- Benchmark streaming throughput (messages/second)
- Benchmark memory usage during long-running streams

## Tasks Overview
1. **Generate Go Protobuf Code** - Run protoc to generate Go types from proto definitions
2. **Implement gRPC Client Structure** - Create client struct with connection management
3. **Implement Connection Manager** - Handle connection lifecycle, retry logic, health checks
4. **Implement RPC Methods** - NewTask, ResumeTask, SendMessage, and all other RPCs
5. **Implement Streaming Support** - Bidirectional streaming for real-time message flow
6. **Implement Interceptors** - Logging, authentication, metrics collection
7. **Write Unit Tests** - Comprehensive test coverage for all components
8. **Write Integration Tests** - Test against mock and real core extension

## Implementation Notes

### Critical Independence Requirement
**The gRPC client MUST NOT depend on any TypeScript/JavaScript code from the existing CLI.** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L30]

### Protobuf Generation
Use the existing proto definitions in `proto/` directory:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/cline/*.proto
```

### Project Structure (Proposed)
```
golang-cli/
  internal/
    grpc/
      client.go          # Main gRPC client implementation
      connection.go      # Connection manager with retry logic
      interceptors.go    # Logging, auth interceptors
      generated/         # Generated protobuf Go code
        cline/
          task.pb.go
          ui.pb.go
          common.pb.go
          state.pb.go
```

### Key Dependencies (Go)
- `google.golang.org/grpc` - Core gRPC library
- `google.golang.org/protobuf` - Protobuf runtime
- `google.golang.org/grpc/credentials` - TLS support

### Connection Configuration
- Default endpoint: `localhost:50051` (configurable)
- Default timeout: 30 seconds
- Retry policy: Exponential backoff (initial 1s, max 30s, 2x multiplier)
- Keepalive: Ping every 10 seconds, timeout 5 seconds

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new

---

**Related Features:**
- FEAT-INFRA-CORE-011-STREAM-002: Bidirectional Streaming (complementary streaming implementation)
- FEAT-INFRA-CORE-011-STATE-003: State Synchronization (uses gRPC client for state sync)
- EPIC-DEV-TASK-003: Task Management (depends on this gRPC client for task operations)