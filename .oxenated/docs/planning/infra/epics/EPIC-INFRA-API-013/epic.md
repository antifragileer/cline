# API Provider Integrations

## Epic ID
EPIC-INFRA-API-013

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13: API Provider Integrations section]

## Target Persona
Infrastructure (Internal)

## Epic Overview
This epic covers the integration of multiple AI API providers into the GoLang CLI, enabling access to OpenAI, Anthropic, OpenRouter, Google Gemini, AWS Bedrock, and other providers through a unified interface. The implementation handles streaming responses, rate limiting, and error handling while maintaining a consistent response format across all providers. This is a foundational infrastructure component that enables the core AI functionality of the CLI.

The API provider integrations are critical for achieving feature parity with the existing TypeScript CLI, as they enable users to connect to their preferred AI models. The implementation must support the same providers with identical configuration and behavior.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Overview]

## Vision & Objectives
Provide broad provider support allowing users flexibility and cost optimization while delivering a unified interface that abstracts provider-specific differences. The implementation ensures that users can seamlessly switch between providers without changing their workflow.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Outcomes & Impacts]

## IAOOI System Components

### Inputs
- API credentials (API keys, OAuth tokens) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Inputs]
- Model configurations and selections [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Inputs]
- Streaming preferences [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Inputs]
- Message history and conversation context [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Feature IAOOI sections]
- Provider-specific endpoint configurations

### Activities
- Connect to OpenAI API and handle streaming responses [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OPENAI-001]
- Connect to Anthropic API (Claude models) with streaming [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-ANTHROPIC-002]
- Route requests through OpenRouter for multi-provider access [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-ROUTER-003]
- Implement Google Gemini provider support [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OTHER-004]
- Implement AWS Bedrock provider support [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OTHER-004]
- Manage rate limits and handle provider-specific error conditions
- Normalize responses to a unified format across all providers

### Outputs
- AI-generated responses from all supported providers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Outputs]
- Token usage information for billing/tracking [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OPENAI-001]
- Streaming response chunks for real-time display [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Activities]
- Error messages with provider-specific context
- Unified response format regardless of provider

### Outcomes
- Users have access to multiple AI providers through a single interface [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Outcomes]
- Provider flexibility enables cost optimization and feature availability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Impacts]
- Seamless provider switching without workflow changes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Outcomes]
- Consistent behavior across all providers

### Impacts
- Broad provider support increases user choice [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Impacts]
- Cost optimization through provider selection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Impacts]
- Failover capability through OpenRouter [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-ROUTER-003]
- Foundation for future provider additions

## Key Features
- **FEAT-INFRA-API-013-OPENAI-001**: OpenAI Provider - GPT model support with streaming
- **FEAT-INFRA-API-013-ANTHROPIC-002**: Anthropic Provider - Claude model support with streaming
- **FEAT-INFRA-API-013-ROUTER-003**: OpenRouter Provider - Multi-provider routing through single API
- **FEAT-INFRA-API-013-OTHER-004**: Other Providers - Google Gemini, AWS Bedrock, and additional providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Epic 13 Features]

## Business Value & Requirements
This epic addresses the following requirements:
- **REQ-009**: Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Traceability Matrix]

The API provider integrations are essential for the CLI to function as users expect, enabling access to the AI models that power the core Cline experience.

## Gherkin BDD Scenarios

### OpenAI Provider Scenarios
```gherkin
Scenario: Send request to OpenAI
  Given valid OpenAI API key
  When sending chat completion request
  Then response should stream back
  And content should extract correctly
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OPENAI-001]

### Anthropic Provider Scenarios
```gherkin
Scenario: Send request to Anthropic
  Given valid Anthropic API key
  When sending messages to Claude
  Then response should stream back
  With Claude's characteristic formatting
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-ANTHROPIC-002]

### OpenRouter Provider Scenarios
```gherkin
Scenario: Route through OpenRouter
  Given valid OpenRouter key
  When requesting model "anthropic/claude-sonnet"
  Then request should route through OpenRouter
  And return Claude response
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-ROUTER-003]

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
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - EPIC-INFRA-API-013-OTHER-004]

## Technical Considerations

### Existing Code References
This is a new GoLang implementation with no dependencies on existing TypeScript provider code. The implementation must:
- Use Go-native HTTP clients for API communication
- Implement streaming response handling using Go channels
- Support the same provider configuration schema as the existing CLI

**Proposed New Components:**
- Proposed: `internal/providers/` - Provider implementation directory
- Proposed: `internal/providers/openai/` - OpenAI provider implementation
- Proposed: `internal/providers/anthropic/` - Anthropic provider implementation
- Proposed: `internal/providers/openrouter/` - OpenRouter provider implementation
- Proposed: `internal/providers/gemini/` - Google Gemini provider implementation
- Proposed: `internal/providers/bedrock/` - AWS Bedrock provider implementation
- Proposed: `internal/providers/types.go` - Shared provider interfaces and types
- Proposed: `internal/providers/factory.go` - Provider factory for instantiation

### Implementation Approach
1. Define a common provider interface that all providers implement
2. Create provider-specific implementations for each supported service
3. Implement streaming response handling with Go channels
4. Normalize responses to a common format
5. Handle rate limiting and retries at the provider level
6. Support both API key and OAuth authentication methods

### Key Technical Challenges
- **Streaming Implementation**: Must support Server-Sent Events (SSE) for streaming responses from all providers
- **Response Normalization**: Different providers return responses in different formats; must normalize to a common structure
- **Error Handling**: Provider-specific error codes and messages must be handled consistently
- **Rate Limiting**: Must respect rate limits and implement appropriate backoff strategies
- **Authentication**: Support multiple authentication methods (API keys, OAuth, AWS IAM)

## Implementation Priority
**Phase**: Phase 7 (after Storage, CLI Foundation, TUI, Task Management, gRPC Integration, and Security)

This epic is scheduled for Phase 7 because:
1. It depends on the core infrastructure being in place (storage, configuration)
2. It requires the task management system to be functional for end-to-end testing
3. It can be developed in parallel with distribution features but before final packaging

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - AI Execution Plan]

## Success Metrics
- All providers from existing CLI are supported (OpenAI, Anthropic, OpenRouter, Gemini, Bedrock)
- Streaming responses work for all providers
- Response format is consistent across all providers
- Token usage information is correctly reported
- Authentication works for all supported methods
- Rate limiting is handled gracefully
- Provider switching works seamlessly

## Dependencies
This epic depends on:
- **EPIC-INFRA-STORAGE-012**: State & Storage Layer - For reading API credentials and provider configuration
- **EPIC-INFRA-CORE-011**: Core Extension Integration - For task execution context (indirectly, tasks use providers)
- **EPIC-DEV-AUTH-004**: Authentication & Provider Configuration - For credential management

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Traceability Matrix and Epic Dependencies]

## Integration Points
- **Task Management (EPIC-DEV-TASK-003)**: Tasks call provider APIs through the unified interface
- **Authentication (EPIC-DEV-AUTH-004)**: Credentials are retrieved from secure storage
- **Configuration (EPIC-ENT-CONFIG-009)**: Provider settings are read from configuration files
- **Core Extension**: Provider responses are streamed to the core extension via gRPC

## Dual Testing Requirements
Per the Dual Testing Strategy, all provider functionality must be tested in BOTH the existing TypeScript CLI and the new GoLang CLI:
- Execute identical API requests in both CLIs
- Compare responses for format and content consistency
- Verify streaming behavior matches
- Test error handling and rate limiting
- Confirm token usage reporting

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Dual Testing Strategy]

## Independence Verification
This implementation must be pure Go with:
- No imports from `cli/src/` or TypeScript provider code
- No Node.js runtime dependencies
- No transpiled or bundled JavaScript
- All dependencies must be pure Go modules
- Build must produce a single static binary

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md - Critical Independence Requirements]