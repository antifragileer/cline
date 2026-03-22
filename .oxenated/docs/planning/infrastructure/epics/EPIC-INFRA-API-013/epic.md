# API Provider Integrations

## Epic ID
EPIC-INFRA-API-013

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L827 - Epic 13: API Provider Integrations Section]

## Target Persona
Infrastructure (Internal)

## Epic Overview
This epic encompasses the integration of multiple AI API providers into the GoLang Cline CLI. The goal is to provide a unified interface that allows users to connect to various AI providers (OpenAI, Anthropic, OpenRouter, Google Gemini, AWS Bedrock, etc.) with consistent request/response handling, streaming support, and error management. This enables provider flexibility, cost optimization, and ensures feature availability across different AI backends.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]

## Vision & Objectives
Create a provider-agnostic API layer that abstracts the differences between various AI service providers while maintaining full compatibility with the existing TypeScript CLI's provider configurations. The unified interface should handle provider-specific request formatting, response parsing, streaming responses, and rate limiting transparently.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]

## IAOOI System Components

### Inputs
- API credentials (API keys, OAuth tokens)
- Model configurations (model IDs, parameters)
- Streaming preferences (enable/disable streaming)
- Message history and conversation context
- Request parameters (temperature, max tokens, etc.)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L754-L755]

### Activities
- Connect to OpenAI API and handle GPT model requests
- Connect to Anthropic API and handle Claude model requests
- Connect to OpenRouter for multi-provider routing
- Connect to Google Gemini API
- Connect to AWS Bedrock service
- Handle streaming responses from all providers
- Manage rate limiting and retry logic
- Normalize responses to common format
- Handle authentication and credential validation

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L755-L756]

### Outputs
- AI-generated text responses
- Token usage information
- Streaming response chunks
- Error messages and status codes
- Normalized response format across providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L757]

### Outcomes
- Seamless access to multiple AI providers through unified interface
- Consistent behavior regardless of backend provider
- Ability to switch providers without changing CLI usage
- Reliable streaming responses across all supported providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L757-L758]

### Impacts
- **Provider Flexibility:** Users can choose the best provider for their needs (cost, performance, features)
- **Cost Optimization:** Ability to route to most cost-effective provider
- **Feature Availability:** Access to provider-specific capabilities while maintaining consistent interface
- **Reduced Vendor Lock-in:** Easy migration between providers
- **Enterprise Adoption:** Support for enterprise-preferred providers (AWS Bedrock, etc.)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758-L761]

## Key Features
- **EPIC-INFRA-API-013-OPENAI-001:** OpenAI Provider - GPT model integration with streaming support
- **EPIC-INFRA-API-013-ANTHROPIC-002:** Anthropic Provider - Claude model integration with streaming support
- **EPIC-INFRA-API-013-ROUTER-003:** OpenRouter Provider - Multi-provider routing through unified API
- **EPIC-INFRA-API-013-OTHER-004:** Other Providers (Gemini, Bedrock, etc.) - Additional provider support

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L763-L827]

## Business Value & Requirements
This epic addresses the following original requirements:
- **REQ-009:** Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L762-L763]
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L874-L884]

## User Journeys & Scenarios
**Developer User Journey:**
1. User configures preferred AI provider via `cline auth` command
2. User initiates task with `cline "prompt"`
3. CLI loads appropriate provider client based on configuration
4. Provider client formats request according to provider-specific API
5. Response streams back in real-time with provider-specific handling
6. Token usage and response metadata are captured for display

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L89-L120]

## BDD Scenarios

### OpenAI Provider Scenarios
```gherkin
Scenario: Send request to OpenAI
  Given valid OpenAI API key
  When sending chat completion request
  Then response should stream back
  And content should extract correctly
```

### Anthropic Provider Scenarios
```gherkin
Scenario: Send request to Anthropic
  Given valid Anthropic API key
  When sending messages to Claude
  Then response should stream back
  With Claude's characteristic formatting
```

### OpenRouter Provider Scenarios
```gherkin
Scenario: Route through OpenRouter
  Given valid OpenRouter key
  When requesting model "anthropic/claude-sonnet"
  Then request should route through OpenRouter
  And return Claude response
```

### Other Providers Scenarios
```gherkin
Scenario: Support Google Gemini
  Given Gemini API configuration
  When sending request
  Then response should return in standard format

Scenario: Support AWS Bedrock
  Given AWS credentials and Bedrock access
  When sending request
  Then response should return in standard format
```

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L773-L827]

## Technical Considerations

### Existing Code References
The existing TypeScript CLI implements API providers in the following locations. The GoLang implementation must achieve functional parity without importing this code:
- TypeScript provider implementations exist in `src/api/providers/` directory [UNVERIFIED - requires confirmation]
- Provider configurations stored in `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]

### Proposed New Components

**Provider Interface (Proposed):**
```go
// Provider defines the interface for all AI API providers
type Provider interface {
    // SendMessage sends a message and returns a streaming response
    SendMessage(ctx context.Context, req *MessageRequest) (<-chan ResponseChunk, error)
    
    // ValidateCredentials checks if the configured credentials are valid
    ValidateCredentials(ctx context.Context) error
    
    // GetAvailableModels returns list of models supported by this provider
    GetAvailableModels() []ModelInfo
    
    // GetDefaultModel returns the default model for this provider
    GetDefaultModel() string
}
```

**Key Implementation Components (Proposed):**
1. **Provider Registry** - Maps provider IDs to provider implementations
2. **Request Normalizer** - Converts internal message format to provider-specific format
3. **Response Normalizer** - Converts provider responses to unified format
4. **Streaming Handler** - Manages SSE/streaming responses across providers
5. **Rate Limiter** - Handles provider rate limits with backoff/retry
6. **Error Translator** - Converts provider errors to unified error types

**Provider-Specific Considerations:**

| Provider | Authentication | Streaming | Special Features |
|----------|---------------|-----------|------------------|
| OpenAI | API Key | SSE | Function calling, JSON mode |
| Anthropic | API Key | SSE | Extended thinking, computer use |
| OpenRouter | API Key | SSE | Multi-provider routing |
| Gemini | API Key | SSE | Google AI features |
| Bedrock | AWS IAM | EventStream | AWS integration |

## Implementation Priority
**Phase:** 7 (API Providers Phase)
**Priority:** High - Required for core functionality (REQ-009)

This epic is scheduled for Phase 7 of the GoLang migration, after the foundational infrastructure (storage, gRPC, CLI framework) is complete. The provider integrations are critical for the CLI to function but depend on the core extension integration (EPIC-INFRA-CORE-011) and storage layer (EPIC-INFRA-STORAGE-012).

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1046-L1053]

## Success Metrics
- All API providers from existing CLI are supported
- Streaming responses work consistently across providers
- Response format is normalized regardless of provider
- Token usage is accurately tracked for all providers
- Error handling is consistent across providers
- Provider switching works without configuration changes

## Dependencies
- **EPIC-INFRA-CORE-011** (Core Extension Integration) - Required for gRPC communication with core extension
- **EPIC-INFRA-STORAGE-012** (State & Storage Layer) - Required for loading provider configurations
- **EPIC-DEV-AUTH-004** (Authentication & Provider Configuration) - Required for credential management

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1083-L1088]

## Integration Points
- **Core Extension (EPIC-INFRA-CORE-011):** Providers are called by the task execution layer through gRPC
- **Authentication (EPIC-DEV-AUTH-004):** Providers consume credentials stored by authentication system
- **Configuration (EPIC-ENT-CONFIG-009):** Provider selection and model preferences stored in configuration
- **Task Management (EPIC-DEV-TASK-003):** Tasks use providers to send requests and receive responses

## Dual Testing Requirements
Per the dual testing mandate, all provider functionality must be tested in BOTH the existing TypeScript CLI AND the new GoLang CLI:

1. **Functional Parity Tests:** Same API calls produce identical results in both CLIs
2. **Output Comparison:** Streaming chunks should match in content and timing
3. **Error Handling:** Error messages and codes should be equivalent
4. **Performance:** GoLang implementation should match or exceed TypeScript performance

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]

## Independence Verification
Per the critical independence requirements:
- Provider implementations must be pure Go with no TypeScript/JavaScript dependencies
- No imports from existing `src/api/providers/` TypeScript code
- No npm package dependencies (axios, etc.) - use Go HTTP clients
- Build must produce single static binary with all provider code embedded

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L28]