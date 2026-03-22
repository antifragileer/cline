# Other API Providers (Gemini, Bedrock, etc.)

## Feature ID
FEAT-INFRA-API-013-OTHER-004

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L827]

## Epic Context
**Parent Epic:** EPIC-INFRA-API-013 - API Provider Integrations [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Create a provider-agnostic API layer that abstracts the differences between various AI service providers while maintaining full compatibility with the existing TypeScript CLI's provider configurations. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L750-L761]
**Business Impact:** Provides provider flexibility, cost optimization, and ensures feature availability across different AI backends including Google Gemini, AWS Bedrock, and other providers beyond OpenAI, Anthropic, and OpenRouter. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L758-L761]

## Feature Overview
**Purpose:** Implement support for additional AI API providers beyond the primary three (OpenAI, Anthropic, OpenRouter), specifically Google Gemini and AWS Bedrock, ensuring they integrate seamlessly with the unified provider interface and deliver responses in the standardized format. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L820]

**Scope:** 
- **Included:** Google Gemini API integration, AWS Bedrock service integration, provider-specific request/response handling, streaming response support, credential management for these providers
- **Excluded:** OpenAI, Anthropic, and OpenRouter providers (covered in separate features FEAT-INFRA-API-013-OPENAI-001, FEAT-INFRA-API-013-ANTHROPIC-002, FEAT-INFRA-API-013-ROUTER-003)

**PRD References:** REQ-009 (Support all existing API providers) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L874-L884]
**PRD Feature ID:** EPIC-INFRA-API-013-OTHER-004 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L820]
**Dependencies:** 
- EPIC-INFRA-CORE-011 (Core Extension Integration) - Required for gRPC communication
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for loading provider configurations
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - Required for credential management
- FEAT-INFRA-API-013-OPENAI-001, FEAT-INFRA-API-013-ANTHROPIC-002, FEAT-INFRA-API-013-ROUTER-003 - Provider interface pattern established by primary providers

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L820 - Feature 4: Other Providers Section]

**Inputs:** 
- Provider-specific credentials (Google API keys for Gemini, AWS IAM credentials for Bedrock)
- Model configurations (model IDs: gemini-1.5-pro, gemini-1.5-flash, amazon.nova-pro-v1:0, etc.)
- Streaming preferences (enable/disable streaming)
- Message history and conversation context in provider-specific formats
- Request parameters (temperature, max tokens, top-p, etc.)
- Regional/endpoint configurations for AWS Bedrock

**Activities:** 
- Implement provider-specific request formatting for Google Gemini API (Google AI SDK format)
- Implement provider-specific request formatting for AWS Bedrock (InvokeModel/InvokeModelWithResponseStream)
- Handle Google Gemini streaming responses (Server-Sent Events)
- Handle AWS Bedrock streaming responses (EventStream format)
- Normalize Gemini responses to unified response format
- Normalize Bedrock responses to unified response format
- Manage rate limiting and retry logic specific to each provider
- Handle authentication and credential validation for Google and AWS
- Support AWS-specific features (cross-region inference, provisioned throughput)
- Support Gemini-specific features (safety settings, grounding)

**Outputs:** 
- AI-generated text responses in normalized format
- Token usage information (input/output tokens)
- Streaming response chunks normalized across providers
- Error messages and status codes translated to unified error types
- Normalized response format consistent with OpenAI/Anthropic/OpenRouter providers

**Outcomes:** 
- Users can access Google Gemini models through the unified CLI interface
- Users can access AWS Bedrock models (Claude, Nova, Llama, etc.) through the unified CLI interface
- Consistent behavior regardless of whether using Gemini, Bedrock, or other providers
- Ability to switch between all providers without changing CLI usage patterns
- Reliable streaming responses across Gemini and Bedrock providers

**Impacts:** 
- **Provider Flexibility:** Users can choose from an expanded set of providers including Google's AI models and AWS's enterprise-grade infrastructure
- **Cost Optimization:** Ability to route to most cost-effective provider including AWS Bedrock's competitive pricing for enterprise workloads
- **Feature Availability:** Access to provider-specific capabilities (Gemini's long context, Bedrock's enterprise integration) while maintaining consistent interface
- **Reduced Vendor Lock-in:** Easy migration between all supported providers including cloud-native options
- **Enterprise Adoption:** Support for enterprise-preferred providers (AWS Bedrock for AWS-native organizations, Google Cloud for GCP organizations)

## Technical Requirements
**Architecture Layer:** Infrastructure Layer (Provider implementations)

**Integration Points:** 
- **Proposed:** New provider implementations in `golang-cli/internal/providers/` following the Provider interface pattern
- **Existing Interface:** Provider interface defined in EPIC-INFRA-API-013 [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-API-013/epic.md:L89-L108]
- **Existing gRPC:** Core extension integration via EPIC-INFRA-CORE-011 for task execution
- **Existing Storage:** Provider configurations stored via EPIC-INFRA-STORAGE-012

**Data Requirements:** 
- **Proposed:** Provider-specific configuration structs for Gemini and Bedrock
- **Existing Pattern:** API credentials stored in `~/.cline/data/secrets.json` (mode 0o600) [Source: .clinerules/storage.md:L1-L30]
- **Existing Pattern:** Provider settings in `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1010]

**Performance Requirements:** 
- Streaming response latency comparable to primary providers (<500ms first chunk)
- Support for Gemini's 1M+ token context window
- Support for Bedrock's InvokeModelWithResponseStream for real-time responses
- Efficient handling of large context windows without memory issues

**Security Requirements:** 
- AWS credentials must support IAM roles, STS tokens, and profile-based authentication
- Google API keys must be stored encrypted in secrets storage
- Support for AWS credential chain (environment variables, ~/.aws/credentials, IAM role)
- No plaintext credential logging or exposure in error messages

## User Experience
**User Personas:** 
- Developer User: Wants access to Gemini's capabilities or has existing AWS infrastructure
- Enterprise User: Requires AWS Bedrock for policy/compliance reasons or has Google Cloud commitments

**User Actions:** 
1. Configure Gemini provider: `cline auth -p gemini -k <google-api-key>`
2. Configure Bedrock provider: `cline auth -p bedrock` (uses AWS credential chain)
3. Select Gemini model: `cline task -m gemini-1.5-pro 'analyze this code'`
4. Select Bedrock model: `cline task -m amazon.nova-pro-v1:0 'refactor this function'`
5. Switch between providers seamlessly without reconfiguration

**UI Components:** 
- **Proposed:** Provider selection in authentication wizard showing Gemini and Bedrock options
- **Proposed:** Model picker dropdown including Gemini and Bedrock model IDs
- **Existing:** Chat interface (EPIC-DEV-UI-002) displays responses uniformly regardless of provider

**Mobile Considerations:** N/A - CLI tool, no mobile interface

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L821-L827 - Other Providers Scenarios]

```gherkin
Scenario: Support Google Gemini
  Given Gemini API configuration with valid API key
  When sending chat completion request to Gemini provider
  Then response should return in standard unified format
  And streaming should work via Server-Sent Events
  And token usage should be reported accurately

Scenario: Support AWS Bedrock
  Given AWS credentials and Bedrock access permissions
  And Bedrock model "amazon.nova-pro-v1:0" is available in the region
  When sending request to Bedrock provider
  Then response should return in standard unified format
  And streaming should work via EventStream
  And AWS-specific error handling should translate to unified errors

Scenario: Switch from OpenAI to Gemini
  Given the user is currently using OpenAI provider
  When the user runs "cline auth -p gemini -k <key>"
  And runs "cline 'hello'"
  Then the request should be sent to Gemini API
  And the response format should match OpenAI response format

Scenario: Gemini with large context
  Given a conversation with 500K tokens of context
  When sending request to Gemini provider
  Then Gemini's extended context window should be utilized
  And the response should stream back successfully

Scenario: Bedrock with Claude model
  Given AWS Bedrock access to Claude 3.5 Sonnet
  When requesting model "anthropic.claude-3-5-sonnet-20240620-v1:0"
  Then request should route through AWS Bedrock
  And Claude response should return in standard format

Scenario: Handle Gemini rate limiting
  Given Gemini API rate limit is reached
  When sending request to Gemini provider
  Then the provider should implement exponential backoff
  And return appropriate error message in unified format

Scenario: Handle Bedrock credential expiration
  Given AWS credentials are expired or invalid
  When sending request to Bedrock provider
  Then appropriate authentication error should return
  With guidance to refresh AWS credentials
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1040-L1045 - Epic Success Metrics adapted for this feature]

**Functional:** 
- Google Gemini API integration works with all Gemini models (1.5 Pro, 1.5 Flash, etc.)
- AWS Bedrock integration works with all Bedrock models (Claude, Nova, Llama, etc.)
- Response format is normalized to match other providers exactly
- Streaming responses work consistently with Gemini and Bedrock

**Performance:** 
- First chunk latency <500ms for both providers under normal conditions
- Streaming throughput matches provider capabilities
- Large context handling (500K+ tokens) works without memory issues

**Quality:** 
- Error messages are clear and actionable
- Authentication errors provide helpful guidance
- Rate limiting is handled gracefully with retry logic

**Integration:** 
- Works seamlessly with task management system (EPIC-DEV-TASK-003)
- Works seamlessly with authentication system (EPIC-DEV-AUTH-004)
- Works seamlessly with configuration system (EPIC-ENT-CONFIG-009)

**Business Value:** 
- Users can access Gemini and Bedrock models through the same interface as other providers
- Enterprise users with AWS infrastructure can use Bedrock without additional configuration
- No vendor lock-in - users can switch between all providers freely

## Testing Strategy
**Unit Testing:** 
- Test provider-specific request formatting for Gemini and Bedrock
- Test response normalization logic
- Test credential validation for both providers
- Test error handling and retry logic
- Test streaming response parsing

**Integration Testing:** 
- Test with mocked Gemini API responses
- Test with mocked Bedrock API responses
- Test provider switching between Gemini/Bedrock and other providers
- Test credential refresh scenarios for AWS

**User Acceptance:** 
- Verify Gemini models produce expected output quality
- Verify Bedrock models produce expected output quality
- Verify streaming feels responsive for both providers
- Verify error messages are helpful

**Performance Testing:** 
- Benchmark first-chunk latency against primary providers
- Test with large context windows (100K, 500K, 1M tokens)
- Test concurrent streaming performance

**Dual Testing Requirements:** 
Per EPIC-INFRA-API-013 requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L827-L856]:
- Functional parity tests: Same API calls produce identical results in TypeScript and GoLang CLIs
- Output comparison: Streaming chunks should match in content structure
- Error handling: Error messages and codes should be equivalent

## Tasks Overview
1. **Implement Gemini Provider Client** - Create Go client for Google Gemini API with request/response handling
2. **Implement Bedrock Provider Client** - Create Go client for AWS Bedrock with InvokeModel and streaming support
3. **Implement Response Normalization** - Convert Gemini and Bedrock responses to unified format
4. **Implement Authentication Handlers** - Handle Google API keys and AWS credential chain
5. **Add Streaming Support** - Implement SSE parsing for Gemini and EventStream for Bedrock
6. **Add Error Handling** - Translate provider-specific errors to unified error types
7. **Write Unit Tests** - Comprehensive tests for all provider functionality
8. **Write Integration Tests** - Tests with mocked provider APIs

## Implementation Notes

### Provider-Specific Considerations
| Provider | Authentication | Streaming | Special Features |
|----------|---------------|-----------|------------------|
| Gemini | API Key | SSE | 1M+ token context, safety settings, grounding |
| Bedrock | AWS IAM/STS | EventStream | Cross-region inference, provisioned throughput |

### Key Implementation Components
1. **Proposed:** `GeminiProvider` struct implementing the Provider interface
2. **Proposed:** `BedrockProvider` struct implementing the Provider interface
3. **Proposed:** Gemini request formatter (Google AI SDK format)
4. **Proposed:** Bedrock request formatter (AWS SDK format)
5. **Proposed:** Streaming parsers for SSE (Gemini) and EventStream (Bedrock)

### Dependencies to Use
- **AWS SDK for Go v2:** For Bedrock integration (`github.com/aws/aws-sdk-go-v2/service/bedrockruntime`)
- **Google Generative AI Go SDK:** For Gemini integration (`github.com/google/generative-ai-go`)
- **Standard Go HTTP:** For SSE streaming handling

### Independence Verification
Per critical independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L28]:
- Provider implementations must be pure Go with no TypeScript/JavaScript dependencies
- No imports from existing `src/api/providers/` TypeScript code
- Use Go-native AWS and Google SDKs, not Node.js equivalents

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L815-L827]
- [x] Existing code references cite actual file paths and lines where applicable
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Storage layer references cite .clinerules/storage.md
- [x] Epic context references epic.md file
- [x] Independence requirements cited from PRD