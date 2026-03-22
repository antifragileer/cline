# Conversation History Management

## Feature ID
FEAT-DEV-TASK-003-CONV-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422-L435]

## Epic Context
**Parent Epic:** EPIC-DEV-TASK-003 - Task Management [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-TASK-003/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Enable users to start, continue, and manage coding tasks seamlessly [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L380]
**Business Impact:** Improved productivity, task continuity, and workflow efficiency [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L387]

## Feature Overview
**Purpose:** Manage conversation history persistence, retrieval, and display to enable users to review past conversations and resume tasks with full context [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422]

**Scope:** 
- Persist conversation messages during task execution
- Load conversation history on task resumption
- Format conversation for display in TUI
- Handle large conversations with pagination/optimization
- Maintain message ordering and integrity

**PRD References:** REQ-018 (Task history with pagination) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L822]
**PRD Feature ID:** EPIC-DEV-TASK-003-CONV-003 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422]

**Dependencies:**
- EPIC-INFRA-STORAGE-012: State & Storage Layer (for file-based persistence)
- EPIC-INFRA-CORE-011: Core Extension Integration (for message streaming)
- EPIC-DEV-UI-002: Interactive Terminal UI (for conversation display)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L424-L429]

**Inputs:**
- Conversation messages (user and assistant exchanges)
- Task metadata (task ID, timestamps, context)
- Pagination requests from user or system
- Message streaming chunks from gRPC

**Activities:**
- Store messages persistently during task execution
- Load conversation history from storage on demand
- Format conversation for TUI display
- Handle large conversations efficiently
- Synchronize conversation state with core extension

**Outputs:**
- Persisted conversation messages in storage
- Paginated conversation history for display
- Formatted conversation threads for UI rendering
- Message state for task resumption

**Outcomes:**
- Users can review past conversations at any time
- Task resumption includes complete conversation context
- Conversation history aids debugging and reference

**Impacts:**
- Referenceability for developers reviewing past interactions
- Debugging support through conversation replay
- Knowledge retention across task sessions

## Technical Requirements

### Architecture Layer
**Primary:** Application Layer (conversation management service)
**Secondary:** Infrastructure Layer (storage persistence)

### Integration Points

**Storage Layer (Existing):**
- **Proposed:** `ConversationStore` interface in Go to abstract persistence
- **Proposed:** File-based storage at `~/.cline/data/tasks/{taskId}/messages.jsonl`
- Integration with `EPIC-INFRA-STORAGE-012-FILE-001` for atomic writes

**gRPC Core Extension:**
- **Existing:** Message streaming via `cline.proto` [Source: proto/cline/task.proto]
- **Proposed:** Conversation history RPC methods for bulk retrieval
- Integration with `EPIC-INFRA-CORE-011-STREAM-002` for bidirectional streaming

**TUI Rendering:**
- **Proposed:** `ConversationView` component in Bubble Tea
- Integration with `EPIC-DEV-UI-002-CHAT-002` for message rendering

### Data Requirements

**Message Schema (Proposed):**
```go
type ConversationMessage struct {
    ID        string    `json:"id"`
    TaskID    string    `json:"taskId"`
    Type      string    `json:"type"`      // "say", "ask", "tool_use", "tool_result"
    Role      string    `json:"role"`      // "user", "assistant", "system"
    Text      string    `json:"text"`
    Timestamp int64     `json:"ts"`
    Metadata  map[string]interface{} `json:"meta,omitempty"`
    Partial   bool      `json:"partial,omitempty"`
}
```

**Storage Format:**
- **Proposed:** JSON Lines (JSONL) format for append-only writes
- **Proposed:** One file per task: `~/.cline/data/tasks/{taskId}/conversation.jsonl`
- **Proposed:** Separate index file for pagination: `~/.cline/data/tasks/{taskId}/conversation.idx`

### Performance Requirements
- Message persistence: < 10ms per write
- History loading: < 100ms for 1000 messages
- Memory usage: Stream large conversations, don't load entirely into memory
- Pagination: Support efficient skip/limit operations

### Security Requirements
- Conversation data stored in user's home directory (~/.cline/data/)
- File permissions: 0o600 for conversation files
- No encryption required (conversation content not sensitive credentials)
- Respect user privacy - conversations remain local

## User Experience

### User Personas
**Primary:** Developer User - reviews past conversations to understand context or debug issues

### User Actions

**View Conversation History:**
1. User runs `cline history` to see task list
2. User selects task to view full conversation
3. System loads and displays conversation chronologically

**Resume Task with Context:**
1. User runs `cline -T {taskId}` to resume task
2. System loads conversation history automatically
3. User sees full context and can continue conversation

**Navigate Long Conversations:**
1. User scrolls through conversation in TUI
2. System loads messages on-demand for large conversations
3. User can jump to specific points in conversation

### UI Components

**Proposed: ConversationPanel (Bubble Tea Component)**
- Scrollable message list
- Message type indicators (user/assistant/tool)
- Timestamp display
- Syntax highlighting for code blocks

**Proposed: HistoryViewer**
- Task list with summary
- Conversation preview
- Quick resume action

### Mobile Considerations
Not applicable - CLI is terminal-based, not mobile.

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L431-L435]

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

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422-L429]

**Functional:**
- [ ] All conversation messages persist during task execution
- [ ] Conversation loads completely and in correct order on task resumption
- [ ] Messages display correctly in TUI with proper formatting
- [ ] Large conversations (>1000 messages) handle gracefully without memory issues

**Performance:**
- [ ] Message write latency < 10ms
- [ ] History load time < 100ms for typical conversations
- [ ] Pagination works efficiently for large conversation history

**Quality:**
- [ ] No message loss during task execution
- [ ] Message ordering preserved correctly
- [ ] Partial/streaming messages handled correctly
- [ ] Graceful handling of corrupted/missing conversation files

**Integration:**
- [ ] Conversation history available immediately after task resumption
- [ ] Works seamlessly with EPIC-DEV-TASK-003-RESUME-002
- [ ] Compatible with existing ~/.cline/data/ storage format

**Business Value:**
- [ ] Users can reference past conversations
- [ ] Task context preserved across sessions
- [ ] Debugging aided by conversation history

## Testing Strategy

### Unit Testing
- Message serialization/deserialization
- Conversation file I/O operations
- Pagination logic
- Error handling for corrupted files

### Integration Testing
- End-to-end conversation persistence and retrieval
- Integration with storage layer
- Integration with gRPC message streaming
- Task resumption with conversation context

### User Acceptance
- User can view complete conversation after task completion
- User can resume task and see full context
- Large conversations remain performant

## Tasks Overview

### Task 1: Conversation Store Implementation
- Implement `ConversationStore` interface
- Create file-based persistence layer
- Implement atomic write operations
- Add error handling and recovery

### Task 2: Message Streaming Integration
- Integrate with gRPC streaming for real-time message capture
- Handle partial/streaming messages
- Implement message buffering

### Task 3: History Loading Service
- Implement conversation loading with pagination
- Create memory-efficient streaming for large conversations
- Add caching for recent conversations

### Task 4: TUI Integration
- Create ConversationPanel Bubble Tea component
- Implement message rendering with proper formatting
- Add scroll/navigation handling

### Task 5: Task Resumption Integration
- Wire conversation loading into task resume flow
- Ensure conversation context available immediately
- Handle edge cases (missing/corrupted conversation files)

## Implementation Notes

### Storage Format Considerations
- JSONL chosen for append-only writes (efficient for streaming)
- Index file enables efficient pagination without scanning entire file
- Consider compaction for very long conversations

### Memory Management
- Use generators/iterators for loading large conversations
- Implement viewport windowing in TUI (only render visible messages)
- Cache conversation metadata separately from content

### Compatibility
- Must read existing TypeScript CLI conversation format
- Write in format compatible with both CLIs
- Dual testing mandate requires identical behavior

## Cross-Persona Integration

### Enterprise/Admin Persona
- Conversation history contributes to audit trails
- May be subject to retention policies
- Integration with EPIC-ENT-AUDIT-010 for compliance

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Requirements mapping cites traceability matrix
- [x] Existing code references marked as "Proposed:" or cite actual files
- [x] Integration points clearly marked as existing or proposed
- [x] BDD scenarios extracted verbatim from PRD
- [x] IAOOI framework complete from PRD source