# State Synchronization

## Feature ID
FEAT-INFRA-CORE-011-STATE-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L917-L925]

## Epic Context
**Parent Epic:** EPIC-INFRA-CORE-011 - Core Extension Integration [Source: .oxenated/docs/planning/infra/epics/EPIC-INFRA-CORE-011/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Enable the GoLang CLI to communicate seamlessly with the existing Cline core extension through gRPC/protobuf, achieving full bidirectional message streaming for real-time task execution, state synchronization between CLI and core extension, and reliable connection management. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L859-L875]
**Business Impact:** This feature is foundational to the entire GoLang CLI migration, ensuring consistent state across CLI and core extension without duplicating business logic in Go. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L883]

## Feature Overview
**Purpose:** Implement bidirectional state synchronization between the GoLang CLI and the Cline core extension, ensuring task state, conversation history, and configuration remain consistent across both components in real-time. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L917-L925]
**Scope:** Includes state change detection, bidirectional sync protocol, conflict resolution, and persistence coordination between CLI local storage and core extension state.
**PRD References:** REQ-011 (Integrate with existing Cline core via gRPC/protobuf) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1045-L1055]
**PRD Feature ID:** EPIC-INFRA-CORE-011-STATE-003 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L885-L925]
**Dependencies:** 
- EPIC-INFRA-STORAGE-012: State & Storage Layer (must be implemented first for state sync) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1350-L1376]
- FEAT-INFRA-CORE-011-GRPC-001: gRPC Client Implementation (for communication channel) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L885-L900]
- FEAT-INFRA-CORE-011-STREAM-002: Bidirectional Streaming (for real-time sync) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L901-L916]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L917-L925 - IAOOI section for State Synchronization feature]

**Inputs:**
1. Local state changes from CLI (user actions, configuration updates)
2. Remote state updates from core extension (task progress, AI responses)
3. State version vectors/timestamps for conflict detection
4. Task state from local storage (`~/.cline/data/`)
5. Synchronization events from gRPC stream [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L875]

**Activities:**
1. Monitor local state for changes and push to core
2. Receive state updates from core via gRPC stream
3. Resolve conflicts using timestamp/version vectors
4. Persist synchronized state to local storage
5. Handle network interruption and reconnection scenarios
6. Validate state consistency after sync operations
7. Queue state changes during disconnection for later sync
8. Notify UI components of state updates for re-rendering [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L872-L875]

**Outputs:**
1. Synchronized state across CLI and core extension
2. Conflict resolution decisions
3. State persistence confirmations
4. Sync status events (success, conflict, error)
5. Updated local state files (`~/.cline/data/globalState.json`, workspace state)
6. Reconnection recovery events [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L876-L879]

**Outcomes:**
1. Consistent state across CLI and core extension at all times
2. No data loss during network interruptions
3. Automatic recovery and sync after reconnection
4. Seamless user experience with real-time state updates
5. Reliable task resumption across sessions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L876-L879]

**Impacts:**
1. Single source of truth maintained (core extension)
2. Reduced state-related bugs and inconsistencies
3. Foundation for real-time collaborative features
4. Improved reliability for long-running tasks
5. Consistent behavior across VSCode extension, JetBrains, and CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L880-L883]

## Technical Requirements
**Architecture Layer:** Infrastructure/Integration
**Integration Points:** 
- Proposed: State Sync Manager (`golang-cli/internal/state/sync.go`) - Monitors local state changes and pushes to core, receives state updates from core, resolves conflicts using timestamp/version vectors [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L988-L997]
- Proposed: Message Router (`golang-cli/internal/messages/`) - Routes incoming gRPC messages to appropriate handlers, dispatches user actions to core via gRPC [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L988-L997]
- gRPC Client Package (`golang-cli/internal/grpc/`) - Go-generated protobuf code from `proto/` definitions, gRPC client implementation with connection management [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L988-L997]
**Data Requirements:** 
- Existing storage format: `~/.cline/data/globalState.json` for configuration, `~/.cline/data/secrets.json` for encrypted credentials, `~/.cline/data/workspaces/` for workspace-specific state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]
- Go implementation must maintain 100% compatibility with existing TypeScript storage format [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]
**Performance Requirements:** 
- State synchronization latency: <100ms for local state changes to reflect in UI
- Reconnection sync time: <500ms after network recovery
- Conflict resolution: Automatic, no user intervention required
**Security Requirements:** 
- State changes must be validated before application
- Sensitive state (tokens, keys) must remain encrypted in transit and at rest

## User Experience
**User Personas:** Infrastructure developers maintaining the CLI, end users who experience seamless state persistence
**User Actions:** 
- Start CLI and see previous state restored automatically
- Switch between devices (VSCode, CLI, JetBrains) with consistent state
- Experience automatic recovery after network interruptions
**UI Components:** 
- Proposed: State sync status indicator in TUI (connected, syncing, error)
- Proposed: Reconnection progress indicator
**Mobile Considerations:** N/A (Infrastructure feature)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L917-L925 - BDD scenarios for State Synchronization feature]

```gherkin
Feature: State Synchronization
  As a CLI user
  I want my task state to remain synchronized with the core extension
  So that I have a consistent experience across all Cline interfaces

Scenario: Sync task state from core to CLI
  Given a task is active in the core extension
  When the core updates task state (e.g., new AI response)
  Then the CLI should receive the state update via gRPC stream
  And the local state should synchronize
  And the UI should reflect the updated state

Scenario: Sync local state changes to core
  Given the user modifies configuration in the CLI
  When the change is detected locally
  Then the CLI should push the state change to the core extension
  And the core should acknowledge the update
  And both states should remain consistent

Scenario: Handle state conflict resolution
  Given the CLI has pending local changes
  And the core sends conflicting state updates
  When the conflict is detected
  Then the State Sync Manager should resolve using timestamp/version vectors
  And the most recent change should prevail
  And the resolved state should synchronize to both sides

Scenario: Queue state changes during disconnection
  Given the gRPC connection is lost
  When the user makes local state changes
  Then the changes should be queued locally
  And a "sync pending" indicator should display
  When the connection restores
  Then all queued changes should sync to the core

Scenario: Full state sync on reconnection
  Given the CLI reconnects after network interruption
  When the gRPC connection is re-established
  Then the CLI should request full state from core
  And compare with local state
  And resolve any conflicts
  And the UI should show "synced" status
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1190-L1205 - Success metrics for Core Extension Integration]

**Functional:**
- State synchronization accuracy: 100% (no data loss or corruption)
- Bidirectional sync works for all state types (task state, configuration, conversation history)
- Conflict resolution handles all edge cases automatically

**Performance:**
- State synchronization latency: <100ms for updates to reflect
- Reconnection recovery time: <500ms after network restoration

**Quality:**
- Zero sync-related errors in production
- Automatic recovery from all network interruption scenarios
- No user intervention required for conflict resolution

**Integration:**
- Works seamlessly with gRPC client and streaming features
- Compatible with existing TypeScript storage format
- Tasks created in CLI can be resumed in VSCode/JetBrains and vice versa

**Business Value:**
- Consistent user experience across all Cline interfaces
- Reduced support tickets related to state inconsistencies
- Foundation for future real-time collaborative features

## Testing Strategy
**Unit Testing:**
- State change detection and event generation
- Conflict resolution algorithm with various timestamp/version scenarios
- Queue management during disconnection
- Storage format serialization/deserialization

**Integration Testing:**
- End-to-end sync flow with mock core extension
- Network interruption and reconnection scenarios
- Concurrent state changes from multiple sources
- Dual testing: Compare state sync behavior between GoLang CLI and existing TypeScript CLI

**User Acceptance:**
- State persistence across CLI restarts
- Seamless task resumption between VSCode and CLI
- Visual feedback for sync status in TUI

**Performance Testing:**
- Sync latency under various network conditions
- Large state payload handling (long conversations)
- Reconnection recovery time measurement

## Tasks Overview
1. Implement State Sync Manager core structure (`golang-cli/internal/state/sync.go`)
2. Implement state change detection and event publishing
3. Implement gRPC state sync protocol handlers
4. Implement conflict resolution using timestamp/version vectors
5. Implement disconnection queue and recovery logic
6. Integrate with Bubble Tea TUI for sync status indicators
7. Write comprehensive unit and integration tests
8. Perform dual testing against existing TypeScript CLI

## Implementation Notes
**Critical Independence Requirements:**
- The GoLang CLI MUST NOT import, transpile, or execute TypeScript/JavaScript from `cli/src/` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]
- All communication MUST go through gRPC/protobuf only [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L10-L36]
- State storage format MUST maintain 100% compatibility with existing TypeScript implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]

**State Storage Compatibility:**
- Uses `~/.cline/data/globalState.json` for configuration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]
- Uses `~/.cline/data/secrets.json` for encrypted credentials [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]
- Uses `~/.cline/data/workspaces/` for workspace-specific state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L982-L987]

**Dual Testing Requirements:**
- All state sync functionality MUST be tested in BOTH existing CLI and GoLang CLI [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]
- Verify they can resume each other's tasks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]
- Confirm state file compatibility [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]
- Test concurrent execution scenarios [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1090-L1125]

**Coordination Points:**
- After Phase 1: All agents align on storage interfaces, proto definitions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1292]
- After Phase 4: Core functionality works end-to-end; full dual test suite runs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1262-L1292]

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L##]
- [x] Existing code references cite actual file paths and lines (N/A - this is a new GoLang CLI feature)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new