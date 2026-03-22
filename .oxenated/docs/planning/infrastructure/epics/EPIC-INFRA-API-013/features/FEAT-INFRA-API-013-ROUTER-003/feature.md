# OpenRouter Provider - Multi-provider Routing

## Feature ID
FEAT-INFRA-API-013-ROUTER-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797-L801 - Feature 3: OpenRouter Provider Section]

## Epic Context
**Parent Epic:** EPIC-INFRA-API-013 - API Provider Integrations [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Create a provider-agnostic API layer that abstracts the differences between various AI service providers while maintaining full compatibility with the existing TypeScript CLI's provider configurations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]
**Business Impact:** Provider flexibility, cost optimization, and reduced vendor lock-in through multi-provider routing capability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758-L761]

## Feature Overview
**Purpose:** Implement the OpenRouter provider client that routes requests to multiple AI providers (OpenAI, Anthropic, Google, Meta, and others) through a unified API endpoint, providing users with access to hundreds of models through a single API key [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797-L801]

**Scope:** 
- OpenRouter API client implementation in Go
- Request formatting for OpenRouter's unified API
- Response parsing and normalization
- Streaming SSE response handling
- Model routing through OpenRouter's provider selection
- Error handling and rate limit management
- Authentication with OpenRouter API keys

**Exclusions:**
- Provider-specific features not exposed through OpenRouter API
- Custom provider routing logic beyond OpenRouter's built-in routing
- Billing or cost management features (handled by OpenRouter platform)

**PRD References:** REQ-009 - Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L874-L884]
**PRD Feature ID:** EPIC-INFRA-API-013-ROUTER-003
**Dependencies:** 
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for loading provider configurations
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - Required for credential management

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797-L801 - Feature 3: OpenRouter Provider Section]

**Inputs:**
- OpenRouter API key
- Model selection in format "provider/model" (e.g., "anthropic/claude-sonnet", "openai/gpt-4")
- Message history and conversation context
- Streaming preferences (enable/disable streaming)
- Request parameters (temperature, max tokens, top_p, etc.)
- Optional provider routing preferences
- Optional fallback provider configuration

**Activities:**
- Format requests according to OpenRouter's unified API specification
- Send requests to OpenRouter API endpoint (https://openrouter.ai/api/v1)
- Handle SSE streaming responses from OpenRouter
- Parse OpenRouter-specific response format (including routing metadata)
- Extract content from various provider responses (Claude, GPT, etc.)
- Normalize responses to common format for core extension
- Handle rate limiting and retry logic specific to OpenRouter
- Manage token usage tracking across routed providers
- Handle authentication with OpenRouter API key
- Support OpenRouter's model selection and fallback mechanisms

**Outputs:**
- AI-generated text responses from the routed provider
- Token usage information (input/output tokens)
- Streaming response chunks with content
- Provider routing metadata (which backend served the request)
- Error messages and status codes from OpenRouter or downstream providers
- Normalized response format consistent with other providers

**Outcomes:**
- Seamless access to multiple AI providers through OpenRouter's unified interface
- Consistent behavior regardless of which backend provider serves the request
- Ability to switch between hundreds of models without changing CLI usage
- Reliable streaming responses across all OpenRouter-supported providers
- Automatic failover to alternative providers when primary is unavailable

**Impacts:**
- **Provider Flexibility:** Users can access 100+ models from 20+ providers through single API key
- **Cost Optimization:** Access to OpenRouter's competitive pricing and provider selection
- **Feature Availability:** Access to latest models from all major providers (Claude, GPT, Gemini, Llama, etc.)
- **Reduced Vendor Lock-in:** Easy migration between providers without configuration changes
- **Simplified Management:** Single API key and endpoint for all AI provider access
- **Enterprise Adoption:** Support for organizations preferring OpenRouter's unified billing and management

## Technical Requirements

**Architecture Layer:** Infrastructure Layer (API Provider Integration)

**Integration Points:**
- **Proposed:** New OpenRouter provider implementation in `golang-cli/internal/providers/openrouter/`
- **Existing Pattern:** Follow provider interface pattern from TypeScript `src/api/providers/` [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L186-L196 - Provider Interface]
- **gRPC Integration:** Provider responses normalized and sent to core extension via gRPC [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L265-L270]

**Data Requirements:**
- **Proposed:** OpenRouter configuration schema including API key, default model, routing preferences
- **Existing Storage:** Provider configurations stored in `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L186]
- **Secrets:** API key stored encrypted in `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L186]

**Performance Requirements:**
- Response latency: <500ms for first token (streaming start)
- Support for concurrent streaming connections
- Efficient handling of large context windows (up to 200K tokens for supported models)
- Graceful handling of provider-specific rate limits

**Security Requirements:**
- API key encryption at rest using OS keyring
- No logging of sensitive request/response content
- Secure HTTPS communication with OpenRouter API
- Input validation for model identifiers and parameters

## User Experience

**User Personas:** 
- Developer User - Access to wide variety of models for different use cases
- DevOps/Automation User - Consistent API interface for scripting across providers
- Enterprise User - Simplified vendor management and unified billing

**User Actions:**
1. Configure OpenRouter provider: `cline auth -p openrouter -k sk-or-...`
2. Select specific model: `cline -m anthropic/claude-sonnet "prompt"`
3. Use OpenRouter routing: `cline -m openai/gpt-4 "prompt"`
4. Switch models without re-authentication

**UI Components:**
- **Proposed:** Model selection dropdown showing OpenRouter models with provider prefixes
- **Existing Pattern:** Similar to TypeScript CLI model picker [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]

**Mobile Considerations:** N/A - Infrastructure feature

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797-L801 - OpenRouter Provider Section]
[Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L220-L227 - OpenRouter Provider Scenarios]

```gherkin
Scenario: Route through OpenRouter
  Given valid OpenRouter key
  When requesting model "anthropic/claude-sonnet"
  Then request should route through OpenRouter
  And return Claude response

Scenario: Send request to OpenRouter with streaming
  Given valid OpenRouter API key
  And streaming is enabled
  When sending chat completion request to model "openai/gpt-4"
  Then response should stream back via SSE
  And content should extract correctly from OpenRouter format

Scenario: Handle OpenRouter model routing
  Given valid OpenRouter key
  When requesting model "google/gemini-pro"
  Then request should route to Google provider
  And return Gemini response in normalized format

Scenario: Handle OpenRouter rate limiting
  Given rate limit is approaching
  When sending request to OpenRouter
  Then appropriate retry logic should apply
  And rate limit headers should be respected

Scenario: Handle OpenRouter errors
  Given an invalid model identifier
  When sending request to OpenRouter
  Then appropriate error should return
  With clear message about model availability

Scenario: Support OpenRouter fallback routing
  Given primary provider is unavailable
  And fallback is configured
  When sending request
  Then OpenRouter should route to fallback provider
  And response should return successfully
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856 - Dual Testing Requirements]
[Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L272-L279 - Success Metrics]

**Functional:**
- OpenRouter provider connects successfully with valid API key
- Requests route to specified models correctly
- Streaming responses work for all OpenRouter-supported models
- Response format is normalized regardless of backend provider
- Token usage is accurately tracked and reported
- Error handling is consistent with other providers

**Performance:**
- First token latency <500ms for typical requests
- Streaming chunks arrive without significant buffering
- Efficient handling of large context windows

**Quality:**
- All BDD scenarios pass
- Unit test coverage >80%
- No memory leaks during long-running streams
- Graceful degradation on network issues

**Integration:**
- Works seamlessly with core extension via gRPC
- Compatible with existing task management system
- Provider switching works without configuration changes
- State persistence works correctly

**Business Value:**
- Users can access 100+ models through single API key
- Consistent interface across all OpenRouter models
- Reduced vendor management complexity

## Testing Strategy

**Unit Testing:**
- HTTP client request formatting
- Response parsing and normalization
- Streaming chunk processing
- Error handling and retry logic
- Rate limit header parsing

**Integration Testing:**
- Mock OpenRouter API server for testing
- End-to-end request/response flow
- Streaming response handling
- Authentication and credential validation
- Cross-provider response normalization

**User Acceptance:**
- Manual testing with real OpenRouter API key
- Verification of model routing for major providers (Anthropic, OpenAI, Google)
- Confirmation of streaming behavior
- Validation of token usage reporting

**Dual Testing Requirements:**
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]:
- Functional parity tests with TypeScript CLI OpenRouter provider
- Output comparison for identical requests
- Error handling equivalence verification
- Performance benchmarking against TypeScript implementation

## Tasks Overview
1. **Task 1:** Implement OpenRouter HTTP client with authentication
2. **Task 2:** Implement request formatting for OpenRouter API
3. **Task 3:** Implement streaming response handling (SSE)
4. **Task 4:** Implement response parsing and normalization
5. **Task 5:** Implement error handling and rate limit management
6. **Task 6:** Implement token usage tracking
7. **Task 7:** Create unit tests for all components
8. **Task 8:** Create integration tests with mock server
9. **Task 9:** Perform dual testing against TypeScript CLI

## Implementation Notes

**OpenRouter API Specifics:**
- Base URL: https://openrouter.ai/api/v1
- Authentication: Bearer token (API key)
- Model format: "provider/model" (e.g., "anthropic/claude-3-sonnet-20240229")
- Supports OpenAI-compatible API format
- Provides additional headers for routing metadata

**Provider Interface Compliance:**
Must implement the Provider interface as defined in epic documentation [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L186-L196]:
```go
type Provider interface {
    SendMessage(ctx context.Context, req *MessageRequest) (<-chan ResponseChunk, error)
    ValidateCredentials(ctx context.Context) error
    GetAvailableModels() []ModelInfo
    GetDefaultModel() string
}
```

**Streaming Implementation:**
- Use Server-Sent Events (SSE) for streaming responses
- Handle OpenRouter's streaming format which wraps provider-specific chunks
- Normalize chunks to common ResponseChunk format for core extension

**Independence Verification:**
Per critical independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L28]:
- Implementation must be pure Go with no TypeScript/JavaScript dependencies
- No imports from existing `src/api/providers/` TypeScript code
- Use Go-native HTTP client (net/http) instead of axios
- Build must produce single static binary with provider code embedded

**Provider-Specific Considerations:**
- OpenRouter uses standard SSE streaming (similar to OpenAI/Anthropic)
- Response format includes routing metadata in headers
- Error responses include provider-specific details
- Rate limits are per-model and per-account

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L797-L801, L827-L856, L874-L884]
- [x] Existing code references cite actual file paths where applicable [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L186-L196]
- [x] New functionality clearly marked as "Proposed:" for GoLang implementation
- [x] Integration points cite existing interfaces or mark as new
- [x] Citation Verification checklist completed