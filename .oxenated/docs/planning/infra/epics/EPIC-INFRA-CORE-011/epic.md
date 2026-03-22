# Core Extension Integration

## Epic ID
EPIC-INFRA-CORE-011

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Epic Section]

## Target Persona
Infrastructure (Internal)

## Epic Overview
This epic covers the integration between the GoLang CLI and the existing Cline core extension via gRPC/protobuf communication. The GoLang CLI must establish a seamless connection to the TypeScript-based core extension, enabling bidirectional message streaming, state synchronization, and task execution while maintaining the core extension as the single source of truth for AI processing and business logic.

The Core Extension Integration is foundational to the entire GoLang CLI migration, as it enables the CLI to leverage all existing Cline functionality (AI providers, tool execution, state management) without reimplementing the core extension logic in Go. This integration ensures feature parity while keeping the core extension as the authoritative backend.
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L859-L875]

## Vision & Objectives
Enable the GoLang CLI to communicate seamlessly with the existing Cline core extension through gRPC/protobuf, achieving:
- Full bidirectional message streaming for real-time task execution
- State synchronization between CLI and core extension
- Reliable connection management with reconnection handling
- Zero duplication of core extension logic in Go
- Complete feature parity with existing TypeScript CLI

## IAOOI System Components

### Inputs
1. Protobuf messages (defined in `proto/` directory)
2. gRPC/streaming connections to core extension
3. Task state from local storage (`~/.cline/data/`)
4. User commands and prompts from CLI interface
5. Connection settings and endpoint configuration
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L875]

### Activities
1. Generate Go code from existing protobuf definitions in `proto/`
2. Establish gRPC connection to core extension endpoint
3. Implement bidirectional streaming for real-time message flow
4. Handle connection lifecycle (connect, disconnect, reconnect)
5. Synchronize state bidirectionally between CLI and core
6. Resolve state conflicts when they occur
7. Serialize/deserialize protobuf messages correctly
8. Manage flow control for streaming responses
9. Handle network errors and retry logic
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L875]

### Outputs
1. Connected gRPC client with working method implementations
2. Bidirectional message stream between CLI and core
3. Synchronized state across CLI and core extension
4. Task execution results streamed from core
5. Error messages and connection status
6. Reconnection events and recovery
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L876-L879]

### Outcomes
1. Seamless integration with existing Cline core (no Go reimplementation needed)
2. Real-time communication enabling responsive terminal UI
3. Consistent behavior across VSCode extension, JetBrains, and CLI
4. Reliable state management across components
5. Feature reuse from existing core extension
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L876-L879]

### Impacts
1. Faster development by leveraging existing core extension
2. Consistent AI behavior across all Cline interfaces
3. Reduced maintenance burden (single core, multiple frontends)
4. Foundation for future native integrations
5. Architectural pattern for other potential frontends
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L880-L883]

## Key Features
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation
- FEAT-INFRA-CORE-011-STREAM-002: Bidirectional Streaming
- FEAT-INFRA-CORE-011-STATE-003: State Synchronization
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L885-L925]

## Business Value & Requirements
This epic addresses the following requirements:
- **REQ-011**: Integrate with existing Cline core via gRPC/protobuf
  - The GoLang CLI MUST communicate with the existing TypeScript core extension via gRPC
  - All AI processing, tool execution, and business logic remains in the core
  - CLI acts as a frontend/client to the core extension server

**Critical Independence Note**: While the GoLang CLI integrates with the core extension via gRPC, it MUST NOT:
- Import, transpile, or execute any TypeScript/JavaScript code from `cli/src/`
- Depend on `cli/package.json` or npm packages
- Use the existing CLI's build system or esbuild configuration
- Require Node.js runtime
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36, L859-L883]

## User Journeys & Scenarios
**Infrastructure Developer Journey:**
1. GoLang CLI starts and initializes
2. CLI establishes gRPC connection to core extension
3. User submits task via CLI interface
4. CLI sends task initialization to core via gRPC
5. Core processes AI request and streams responses back
6. CLI renders streaming responses in TUI
7. Core requests tool approval via stream
8. CLI displays approval prompt and captures user response
9. CLI sends approval/rejection back to core
10. Core continues task execution
11. State changes sync bidirectionally throughout
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L879]

## BDD Scenarios

### Feature: gRPC Client Implementation (FEAT-INFRA-CORE-011-GRPC-001)

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L889-L900]

### Feature: Bidirectional Streaming (FEAT-INFRA-CORE-011-STREAM-002)

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]

### Feature: State Synchronization (FEAT-INFRA-CORE-011-STATE-003)

```gherkin
Scenario: Sync task state
  Given a task is active
  When core updates task state
  Then CLI should receive update
  And local state should synchronize
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L917-L925]

## Technical Considerations

### Existing Code References

**Protobuf Definitions:**
- Proto files located in `proto/` directory at project root
- Existing proto definitions used by VSCode extension and JetBrains
- Generate Go code from these existing definitions
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1264]

**State Storage Compatibility:**
- Uses `~/.cline/data/globalState.json` for configuration
- Uses `~/.cline/data/secrets.json` for encrypted credentials
- Uses `~/.cline/data/workspaces/` for workspace-specific state
- Go implementation must maintain 100% compatibility with existing TypeScript storage format
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1420-L1425]

### Proposed New Components

**Proposed: gRPC Client Package** (`golang-cli/internal/grpc/`)
- Go-generated protobuf code from `proto/` definitions
- gRPC client implementation with connection management
- Streaming message handlers
- Reconnection logic with exponential backoff

**Proposed: Message Router** (`golang-cli/internal/messages/`)
- Routes incoming gRPC messages to appropriate handlers
- Dispatches user actions to core via gRPC
- Manages message queuing during reconnection

**Proposed: State Sync Manager** (`golang-cli/internal/state/sync.go`)
- Monitors local state changes and pushes to core
- Receives state updates from core
- Resolves conflicts using timestamp/version vectors

### Critical Independence Requirements

**The gRPC integration is the ONLY permitted connection to existing Cline code:**
1. GoLang CLI MUST NOT import, transpile, or execute TypeScript/JavaScript from `cli/src/`
2. GoLang CLI MUST NOT depend on `cli/package.json` or npm packages
3. All communication MUST go through gRPC/protobuf only
4. Core extension remains TypeScript; CLI is pure Go
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]

### Dual Testing Requirements

All gRPC functionality MUST be tested in BOTH existing CLI and GoLang CLI:
- Side-by-side integration tests running both CLIs against same core
- Verify they can resume each other's tasks
- Confirm state file compatibility
- Test concurrent execution scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]

## Implementation Priority
**Phase 1: Foundational Setup** - Critical path item
- Protobuf integration must be completed early (AI Agent 2)
- gRPC client implementation blocks all task-related features
- Required before Phase 4 (Task Management) can begin

**Dependency Order:**
1. Protobuf code generation (prerequisite)
2. gRPC client implementation (this epic)
3. Task initialization and streaming (depends on this)
4. Tool approval workflows (depends on bidirectional streaming)
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1292]

## Success Metrics
1. gRPC connection establishes successfully within 500ms
2. Bidirectional streaming maintains <100ms latency for message delivery
3. State synchronization accuracy: 100% (no data loss or corruption)
4. Reconnection success rate: >99% after network interruption
5. Message serialization/deserialization: Zero errors in production
6. Dual testing parity: 100% of tests pass in both CLI implementations
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1190-L1205]

## Dependencies
**Prerequisites:**
- EPIC-INFRA-STORAGE-012: State & Storage Layer (must be implemented first for state sync)
- Protobuf definitions from `proto/` directory (already exist)

**Blocks:**
- EPIC-DEV-TASK-003: Task Management (depends on gRPC for task execution)
- EPIC-DEV-UI-002: Interactive Terminal UI (depends on streaming for real-time display)

**Coordination Points:**
- After Phase 1: All agents align on storage interfaces, proto definitions
- After Phase 4: Core functionality works end-to-end; full dual test suite runs
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]

## Integration Points
**Integrates with:**
- VSCode Extension: Shares core extension via gRPC
- JetBrains: Shares core extension via gRPC (same integration pattern)
- Existing TypeScript CLI: Both use same gRPC interface to core

**State Sharing:**
- GoLang CLI reads/writes same `~/.cline/data/` files as existing CLI
- Tasks created in one CLI can be resumed in the other
- Configuration changes sync across all interfaces via shared storage
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1095-L1105, L1420-L1425]

## Traceability Matrix
| Epic/Feature ID | Requirement IDs |
|-----------------|-----------------|
| EPIC-INFRA-CORE-011 | REQ-011 |
| FEAT-INFRA-CORE-011-GRPC-001 | REQ-011 |
| FEAT-INFRA-CORE-011-STREAM-002 | REQ-011 |
| FEAT-INFRA-CORE-011-STATE-003 | REQ-011 |
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1045-L1055]