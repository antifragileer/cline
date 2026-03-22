# Anthropic Provider Integration

## Feature ID
FEAT-INFRA-API-013-ANTHROPIC-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L784-L791]

## Epic Context
**Parent Epic:** EPIC-INFRA-API-013 - API Provider Integrations [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Provide a unified interface that allows users to connect to various AI providers (OpenAI, Anthropic, OpenRouter, Google Gemini, AWS Bedrock, etc.) with consistent request/response handling, streaming support, and error management. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]
**Business Impact:** Enables provider flexibility, cost optimization, and ensures feature availability across different AI backends. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758-L761]

## Feature Overview
**Purpose:** Implement Anthropic API provider integration for Claude model support with streaming responses in the GoLang Cline CLI, enabling users to access Claude models through a unified provider interface. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L784-L791]
**Scope:** Includes Anthropic API client implementation, request/response formatting for Claude models, SSE streaming support, authentication handling, and integration with the unified provider interface.
**PRD References:** REQ-009 (Support all existing API providers) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L762-L763]
**PRD Feature ID:** EPIC-INFRA-API-013-ANTHROPIC-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L790]
**Dependencies:** 
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for loading provider configurations
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - Required for credential management

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L784-L791 - Feature 2: Anthropic Provider]

**Inputs:**
- Anthropic API key (stored in secrets.json) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L785]
- Model selection (Claude variants: claude-sonnet-4, claude-opus-4, etc.)
- Message format (Anthropic messages API format)
- Streaming preferences (enable/disable streaming)
- Request parameters (temperature, max tokens, etc.) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L754-L755]

**Activities:**
- Format requests for Anthropic Messages API [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L785]
- Handle Server-Sent Events (SSE) streaming responses [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L785]
- Parse Anthropic-specific response format (content blocks, stop reasons)
- Transform internal message format to Anthropic format
- Normalize Anthropic responses to unified format
- Handle Anthropic-specific error codes and rate limits
- Support Claude's extended thinking and computer use features [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L859-L867]

**Outputs:**
- Claude-generated text responses [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L786]
- Token usage information (input/output tokens)
- Streaming response chunks with content blocks
- Error messages and status codes
- Normalized response format compatible with unified provider interface

**Outcomes:**
- Seamless access to Claude models through unified interface [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L787]
- Consistent behavior regardless of using Claude vs other providers
- Reliable streaming responses from Anthropic API
- Support for Claude-specific capabilities (thinking, tool use)

**Impacts:**
- **Provider Flexibility:** Users can choose Claude models for their specific needs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758]
- **Cost Optimization:** Ability to route to Claude when most cost-effective
- **Feature Availability:** Access to Claude's extended thinking and capabilities
- **Reduced Vendor Lock-in:** Easy migration between Anthropic and other providers
- **Enterprise Adoption:** Support for enterprise-preferred Claude models

## Technical Requirements

**Architecture Layer:** Infrastructure Layer (API Provider)

**Integration Points:**
- **Proposed:** New Anthropic provider implementation following the Provider interface pattern [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L97-L110]
- **Proposed:** Provider Registry integration to map "anthropic" provider ID to implementation
- **Existing:** Credentials stored via StateManager in secrets.json [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]

**Data Requirements:**
- **Proposed:** Anthropic API request/response structs matching Messages API v1
- **Existing:** API key storage in ~/.cline/data/secrets.json

**Performance Requirements:**
- Streaming response latency < 500ms first chunk
- Support for concurrent requests to Anthropic API
- Handle rate limiting with exponential backoff

**Security Requirements:**
- API keys must be encrypted at rest in secrets.json
- API keys must never be logged or displayed in plain text
- HTTPS-only communication with api.anthropic.com

## User Experience

**User Personas:** 
- Developer User - Uses Claude for coding assistance
- Enterprise User - May prefer Claude for specific compliance/requirements

**User Actions:**
1. Configure Anthropic provider via `cline auth -p anthropic -k <api_key>`
2. Select Claude model via configuration or `-m` flag
3. Execute tasks that automatically route to Anthropic API
4. Receive streaming responses in real-time

**Integration Points with Existing TypeScript Implementation:**
- Must achieve functional parity with existing TypeScript Anthropic provider [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L134-L142]
- TypeScript provider implementations exist in `src/api/providers/` directory [UNVERIFIED]
- Must support same model IDs and configuration options
- Dual testing mandate requires identical behavior [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]

**Mobile Considerations:** N/A (Infrastructure feature, not user-facing mobile UI)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L773-L827 - Anthropic Provider Scenarios]

```gherkin
Scenario: Send request to Anthropic
  Given valid Anthropic API key
  When sending messages to Claude
  Then response should stream back
  With Claude's characteristic formatting
```

Additional scenarios implied from epic requirements:

```gherkin
Scenario: Authenticate with Anthropic API key
  Given the user runs "cline auth -p anthropic -k sk-ant-xxxxx"
  When the command executes
  Then the API key should validate against Anthropic
  And store encrypted in secrets.json
  And configuration should update with provider "anthropic"

Scenario: Stream Claude response
  Given a configured Anthropic provider
  When a task sends a message to Claude
  Then the response should stream via SSE
  And each chunk should display in real-time
  And final response should be complete

Scenario: Handle Anthropic rate limiting
  Given many concurrent requests to Anthropic
  When rate limit is exceeded
  Then the provider should retry with backoff
  And return appropriate error if persistent

Scenario: Use Claude with plan mode
  Given the user runs "cline -p 'design an API'"
  And Anthropic is the configured provider
  Then Claude should respond with planning approach
  And the response should format correctly

Scenario: Switch from OpenAI to Anthropic
  Given the user has OpenAI configured
  When the user switches provider to Anthropic
  Then the same task format should work
  And responses should come from Claude instead
```

## Success Criteria
[Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L173-L179 - Epic Success Metrics]

**Functional:**
- Anthropic provider accepts API key and validates successfully
- Claude models (claude-sonnet-4, claude-opus-4, etc.) return responses
- Streaming responses work with SSE protocol
- Request/response format matches Anthropic Messages API specification

**Performance:**
- First response chunk received within 500ms
- Streaming is smooth without buffering delays
- Rate limiting handled gracefully with retries

**Quality:**
- Error handling provides clear messages for common failures (invalid key, rate limit, model unavailable)
- Response normalization produces consistent output format across providers
- No memory leaks during long streaming sessions

**Integration:**
- Works seamlessly with gRPC core extension integration
- Compatible with task management system
- Provider switching works without configuration changes

**Business Value:**
- Users can access Claude models through unified interface
- Functional parity with existing TypeScript CLI Anthropic provider
- Passes dual testing comparison with existing CLI

## Testing Strategy

**Unit Testing:**
- Test request formatting for Anthropic Messages API
- Test response parsing and normalization
- Test streaming chunk handling
- Test error code translation
- Test rate limit retry logic

**Integration Testing:**
- Test with mocked Anthropic API responses
- Test gRPC integration with core extension
- Test credential loading from secrets storage
- Test provider registry integration

**User Acceptance:**
- End-to-end task execution with Anthropic provider
- Compare output with existing TypeScript CLI
- Verify streaming behavior matches expectations
- Test all Claude model variants

**Dual Testing Requirements:**
- Same API calls must produce identical results in both GoLang and TypeScript CLIs [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]
- Streaming chunks should match in content and timing
- Error messages and codes should be equivalent
- GoLang implementation should match or exceed TypeScript performance

**Performance Testing:**
- Measure latency to first chunk vs TypeScript implementation
- Benchmark throughput under load
- Compare memory usage during streaming

## Tasks Overview
1. **Implement Anthropic Provider Struct** - Create Go struct implementing Provider interface with SendMessage, ValidateCredentials, GetAvailableModels, GetDefaultModel methods
2. **Implement Request Formatting** - Convert internal message format to Anthropic Messages API format
3. **Implement Response Parsing** - Parse Anthropic SSE streams and content blocks
4. **Implement Streaming Handler** - Manage SSE connection and chunk delivery
5. **Implement Error Handling** - Map Anthropic errors to unified error types
6. **Add Provider to Registry** - Register "anthropic" in provider registry
7. **Create Unit Tests** - Test all provider methods with mocked responses
8. **Create Integration Tests** - Test with actual Anthropic API (optional, manual)
9. **Dual Testing Verification** - Compare behavior with TypeScript CLI

## Implementation Notes

**Provider Interface Implementation (Proposed):**
```go
// AnthropicProvider implements the Provider interface for Anthropic API
type AnthropicProvider struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func (p *AnthropicProvider) SendMessage(ctx context.Context, req *MessageRequest) (<-chan ResponseChunk, error) {
    // Format request for Anthropic Messages API
    // Establish SSE connection
    // Stream response chunks
}

func (p *AnthropicProvider) ValidateCredentials(ctx context.Context) error {
    // Test API key with minimal request
}

func (p *AnthropicProvider) GetAvailableModels() []ModelInfo {
    // Return supported Claude models
}

func (p *AnthropicProvider) GetDefaultModel() string {
    // Return default Claude model (claude-sonnet-4)
}
```

**Anthropic-Specific Considerations:**
- Uses API Key authentication (not OAuth)
- Supports SSE streaming
- Special features: Extended thinking, computer use [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L104-L110]
- Message format differs from OpenAI (roles: user/assistant vs system/user/assistant)

**Independence Requirements:**
- Must be pure Go implementation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L28]
- No imports from existing `src/api/providers/` TypeScript code
- No npm package dependencies - use Go HTTP clients
- Build must produce single static binary

**Phase Assignment:**
- Phase 7 (API Providers Phase) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L142-L147]

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines (where verified)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Independence requirements cited from PRD
- [x] Dual testing requirements cited from PRD
- [x] BDD scenarios extracted from PRD