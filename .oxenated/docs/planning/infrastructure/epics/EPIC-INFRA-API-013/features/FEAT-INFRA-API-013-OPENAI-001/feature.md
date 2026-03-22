# OpenAI Provider - GPT Model Integration with Streaming Support

## Feature ID
FEAT-INFRA-API-013-OPENAI-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L780-L787]

## Epic Context
**Parent Epic:** EPIC-INFRA-API-013 - API Provider Integrations [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Create a provider-agnostic API layer that abstracts differences between AI service providers while maintaining full compatibility with existing TypeScript CLI's provider configurations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]
**Business Impact:** Enables provider flexibility, cost optimization, and ensures feature availability across different AI backends [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758-L761]

## Feature Overview
**Purpose:** Implement OpenAI API provider integration for the GoLang Cline CLI, enabling access to GPT models (GPT-4, GPT-4o, GPT-3.5-Turbo, etc.) with full streaming support and response normalization [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L780-L787]

**Scope:** 
- OpenAI API client implementation in Go
- Streaming chat completion support via Server-Sent Events (SSE)
- Request/response normalization to unified provider interface
- API key authentication and validation
- Token usage tracking and metadata extraction
- Error handling and retry logic for OpenAI-specific errors

**PRD References:** 
- REQ-009: Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L874-L884]
- EPIC-INFRA-API-013-OPENAI-001: OpenAI Provider - GPT model integration with streaming support [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L780-L787]

**PRD Feature ID:** EPIC-INFRA-API-013-OPENAI-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L780]

**Dependencies:**
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for loading API credentials
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - Required for credential management

## IAOOI Components
**Source:** .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L780-L787 - OpenAI Provider Feature IAOOI

**Inputs:**
- OpenAI API key (from secure storage or environment)
- Model selection (GPT-4, GPT-4o, GPT-3.5-Turbo, etc.)
- Message history (conversation context in OpenAI format)
- Request parameters (temperature, max_tokens, top_p, etc.)
- Streaming preference (enable/disable streaming)
- Base URL (for OpenAI-compatible endpoints like Azure)

**Activities:**
- Format requests for OpenAI API chat completions endpoint
- Handle Server-Sent Events (SSE) streaming responses
- Parse completion chunks and extract content/delta
- Normalize OpenAI responses to unified provider format
- Extract token usage information from response headers
- Handle OpenAI-specific error codes (rate limits, context length, etc.)
- Implement retry logic with exponential backoff
- Validate API credentials on initialization

**Outputs:**
- AI-generated text responses (streaming and non-streaming)
- Token usage metrics (prompt_tokens, completion_tokens, total_tokens)
- Response metadata (model ID, finish reason, timestamp)
- Normalized response chunks for UI consumption
- Error information in standardized format

**Outcomes:**
- Seamless access to OpenAI GPT models through unified interface
- Real-time streaming responses matching existing TypeScript CLI behavior
- Accurate token tracking for cost monitoring
- Reliable error handling with actionable messages

**Impacts:**
- **Provider Flexibility:** Users can choose OpenAI models when optimal for their use case
- **Cost Optimization:** OpenAI pricing options available alongside other providers
- **Feature Availability:** Access to GPT-specific capabilities (function calling, JSON mode)
- **Consistency:** Same interface as other providers reduces cognitive load

## Technical Requirements

### Architecture Layer
Infrastructure Layer - API Provider Implementation

### Integration Points
- **Proposed:** New Go package `internal/providers/openai/` implementing Provider interface
- **Proposed:** Integration with provider registry for dynamic provider selection
- **Proposed:** Configuration loading from storage layer for API keys and settings
- **Integration:** Provider interface defined in EPIC-INFRA-API-013 [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L95-L110]

### Data Requirements
**Configuration Schema (Proposed):**
```go
type OpenAIConfig struct {
    APIKey       string
    Model        string
    BaseURL      string // Optional, defaults to https://api.openai.com/v1
    Temperature  float32
    MaxTokens    int
    TopP         float32
    FrequencyPenalty float32
    PresencePenalty  float32
}
```

**Provider Interface Implementation (Proposed):**
```go
// Provider defines the interface for all AI API providers
type Provider interface {
    SendMessage(ctx context.Context, req *MessageRequest) (<-chan ResponseChunk, error)
    ValidateCredentials(ctx context.Context) error
    GetAvailableModels() []ModelInfo
    GetDefaultModel() string
}
```

**Existing Code References:**
- TypeScript CLI provider configurations: `cli/src/agent/ClineAgent.ts:L20-L35` [Source: cli/src/agent/ClineAgent.ts - provider models mapping]
- Authentication handling: `cli/src/index.ts:L85-L120` [Source: cli/src/index.ts - auth command with provider options]

### Performance Requirements
- Streaming latency: First chunk within 500ms of request
- Throughput: Support 100+ concurrent streaming connections
- Retry logic: Exponential backoff starting at 1s, max 5 retries
- Timeout: Request timeout of 60s, streaming chunk timeout of 30s

### Security Requirements
- API keys stored encrypted in secrets storage (mode 0o600)
- Keys loaded from `~/.cline/data/secrets.json` via StateManager
- No API keys logged or exposed in error messages
- Support for environment variable fallback (OPENAI_API_KEY)

## User Experience
**User Personas:** Developer User (via CLI authentication), Infrastructure (implementation)

**User Actions:**
1. Configure OpenAI provider: `cline auth -p openai-native -k sk-...`
2. Use OpenAI model for task: `cline -m gpt-4o "write a function"`
3. Switch between providers without changing usage patterns

**UI Components:**
- **Proposed:** Provider selection in authentication wizard
- **Proposed:** Model picker showing available OpenAI models
- **Existing:** Auth command supports `-p` flag for provider selection [Source: cli/src/index.ts:L98]

**Mobile Considerations:** N/A - Infrastructure feature

## BDD Scenarios
**Source:** .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L773-L787 - OpenAI Provider BDD Scenarios

```gherkin
Scenario: Send request to OpenAI
  Given valid OpenAI API key
  When sending chat completion request
  Then response should stream back
  And content should extract correctly

Scenario: Stream chat completion from OpenAI
  Given valid OpenAI API key configured
  And streaming is enabled
  When sending a chat completion request
  Then chunks should arrive as Server-Sent Events
  And each chunk should be parsed and normalized
  And final response should include token usage

Scenario: Handle OpenAI rate limit error
  Given rate limit exceeded on OpenAI API
  When sending a request
  Then appropriate error should return
  And retry logic should attempt with exponential backoff
  And user should receive clear error message if retries exhausted

Scenario: Validate OpenAI credentials
  Given an invalid API key
  When validating credentials
  Then validation should fail
  And appropriate authentication error should return
```

## Success Criteria
**Source:** .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L773-L787

**Functional:**
- OpenAI API requests complete successfully with valid credentials
- Streaming responses work consistently with real-time chunk delivery
- Response format is normalized to match other providers' output
- All GPT models (GPT-4, GPT-4o, GPT-3.5-Turbo) are supported
- Token usage is accurately tracked and reported

**Performance:**
- First response chunk received within 500ms
- Streaming maintains consistent throughput without drops
- Retry logic handles transient failures automatically

**Quality:**
- Error messages are clear and actionable
- API key validation provides immediate feedback
- No memory leaks during long streaming sessions

**Integration:**
- Works seamlessly with provider switching mechanism
- Compatible with task execution layer via gRPC
- State storage integration for credential persistence

**Business Value:**
- Full OpenAI provider parity with existing TypeScript CLI
- Enables users to leverage OpenAI models when preferred

## Testing Strategy
**Unit Testing:**
- OpenAI client initialization with various configurations
- Request formatting for chat completions API
- Response parsing and normalization
- Error handling for different HTTP status codes
- Token usage extraction from responses

**Integration Testing:**
- Live API calls to OpenAI (with test API key)
- Streaming response handling
- Provider registry integration
- Configuration loading from storage

**User Acceptance:**
- Authentication flow with OpenAI provider
- Task execution using OpenAI models
- Comparison testing with TypeScript CLI output

**Performance Testing:**
- Streaming latency under various network conditions
- Concurrent request handling
- Memory usage during long conversations

**Dual Testing Requirements (Critical):**
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]:
- Functional parity tests: Same API calls produce identical results in both CLIs
- Output comparison: Streaming chunks match in content
- Error handling: Error messages equivalent between implementations
- Performance: GoLang implementation matches or exceeds TypeScript

## Tasks Overview
1. Create OpenAI provider package structure (`internal/providers/openai/`)
2. Implement HTTP client for OpenAI API with proper authentication
3. Implement chat completions request/response types
4. Implement Server-Sent Events (SSE) streaming parser
5. Implement Provider interface methods (SendMessage, ValidateCredentials, etc.)
6. Implement token usage extraction and tracking
7. Implement error handling and retry logic
8. Add OpenAI provider to provider registry
9. Write unit tests with mocked API responses
10. Write integration tests with real API calls
11. Perform dual testing against TypeScript CLI
12. Document usage and configuration

## Implementation Notes

### Provider-Specific Considerations
Per the epic documentation [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L118-L125]:

| Aspect | OpenAI Implementation |
|--------|----------------------|
| Authentication | API Key (Bearer token) |
| Streaming | Server-Sent Events (SSE) |
| Special Features | Function calling, JSON mode, tools |

### Go Dependencies (Proposed)
- Standard library: `net/http`, `encoding/json`, `bufio` for SSE parsing
- External: May use `github.com/sashabaranov/go-openai` or implement custom client

### Independence Verification
Per critical requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L28]:
- **No TypeScript imports:** Implementation must be pure Go
- **No npm dependencies:** Use Go HTTP clients only
- **Single binary:** All provider code embedded in Go binary

### Streaming Implementation Details
OpenAI uses Server-Sent Events format:
```
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"Hello"}}]}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":" world"}}]}

data: [DONE]
```

Parser must:
1. Handle line-by-line reading
2. Parse JSON from `data:` lines
3. Extract content from `choices[0].delta.content`
4. Handle `[DONE]` signal for stream completion
5. Aggregate usage from final response

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths (where applicable)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or marked as new
- [x] All BDD scenarios from PRD included verbatim

## References
1. Epic Documentation: `.oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md`
2. PRD Source: `.oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L827`
3. Provider Interface: `.oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L95-L110`
4. CLI Auth Implementation: `cli/src/index.ts` (provider flag handling)