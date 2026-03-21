---
description: Creates thoroughly researched and validated Cline workflow files grounded in actual project structure, following official Cline best practices
applies_to: [".clinerules/workflows/**/*.md"]
priority: high
---

# Create Comprehensive Cline Workflow

## Usage

This workflow creates a thoroughly researched and validated Cline workflow file for any specified topic, ensuring it's grounded in actual project structure, code, documentation, and existing implementations rather than assumptions.

**When to use this workflow:**
- When creating automation for project-specific processes
- When standardizing complex multi-step development tasks
- When documenting workflows that interact with existing code and features
- When you need workflows that accurately reflect current project architecture
- When converting ad-hoc processes into repeatable, validated workflows
- When converting a completed task into reusable automation (official Cline approach)

**Alternative Approach - Create from Completed Task:**
After completing any task you'll need to repeat, you can tell Cline:
```
Create a workflow for the process I just completed
```
Cline analyzes the conversation, identifies the steps, and generates the workflow file automatically. This approach produces workflows grounded in real project execution.

**Example Usage:**
```
Create a comprehensive Cline workflow for: [topic/process description]
```

or invoke directly:
```
/cline-workflow-create.md
```

## Overview

This workflow follows official Cline best practices:
- **Be clear and concise** - Use simple language and avoid ambiguity
- **Focus on outcomes** - Describe desired results rather than implementation details
- **Be modular** - Break complex tasks into smaller, reusable workflows
- **Comment why, not what** - Explain the reasoning behind steps

The workflow uses a systematic approach combining:
1. **Sequential Thinking MCP** - For structured analysis and validation
2. **Context7 MCP** - For Cline workflow best practices documentation
3. **Deep Code Analysis** - Reading actual implementation files
4. **Documentation Review** - Checking specs, features, and planning docs
5. **Pattern Recognition** - Analyzing existing workflows and rules
6. **Validation** - Ensuring accuracy against real project state

The result is a production-ready workflow file that accurately reflects the project's current state and follows official Cline best practices.

## Parameters

When invoking this workflow, provide:

- **topic** (required): Clear description of what the workflow should accomplish
- **scope** (optional): Specific areas of codebase or features to focus on
- **related_features** (optional): Feature IDs or documentation to reference

**Format for invocation:**
```
Create a comprehensive Cline workflow for: [detailed topic description]
Focus on: [specific scope or areas]
Related to: [features, epics, or documentation]
```

## Detailed Sequence of Steps

### Step 1: Initialize Sequential Thinking Analysis

Begin systematic analysis of the workflow requirements and project context.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Starting comprehensive workflow creation for specified topic. Analysis plan: 1) Understand the workflow objective and scope, 2) Identify relevant codebase areas, 3) Review existing documentation and features, 4) Analyze current implementations, 5) Identify required tools and steps, 6) Structure workflow following best practices, 7) Validate against actual project state. This ensures the workflow is grounded in reality, not assumptions.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 2: Query Context7 for Cline Workflow Best Practices

Gather authoritative guidance on workflow structure and conventions.

```xml
<use_mcp_tool>
<server_name>github.com/upstash/context7-mcp</server_name>
<tool_name>get-library-docs</tool_name>
<arguments>
{
  "context7CompatibleLibraryID": "/cline/prompts",
  "topic": "workflow creation best practices structure conventions",
  "page": 1
}
</arguments>
</use_mcp_tool>
```

### Step 3: Review Workflow Creation Rules

Read the project's own rules for creating workflows.

```xml
<read_file>
<path>.clinerules/cline_workflow_creation_best_practices.md</path>
</read_file>
```

### Step 4: Analyze Existing Workflows for Patterns

Study existing workflows to understand established patterns and structure.

```xml
<list_files>
<path>.clinerules/workflows</path>
<recursive>false</recursive>
</list_files>
```

### Step 5: Read Representative Workflow Examples

Examine 2-3 existing workflows to extract structural patterns.

```xml
<read_file>
<path>.clinerules/workflows/{relevant-example-workflow}.md</path>
</read_file>
```

### Step 6: Identify Relevant Codebase Areas

Use sequential thinking to map the workflow topic to specific code locations.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Mapping workflow topic to codebase structure. Identifying: 1) Primary directories that will be affected (src/app/, src/components/, src/lib/, etc.), 2) Relevant file patterns and extensions, 3) Key modules or services involved, 4) Configuration files that may be relevant, 5) Test directories for validation patterns. This determines which files to analyze in depth.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 2,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 7: Search for Related Code Patterns

Find actual implementations related to the workflow topic.

```xml
<search_files>
<path>src/</path>
<regex>{relevant-pattern-based-on-topic}</regex>
<file_pattern>*.{ts,tsx}</file_pattern>
</search_files>
```

### Step 8: List Relevant Directory Structure

Understand the organization of code related to the workflow.

```xml
<list_files>
<path>{identified-relevant-directory}</path>
<recursive>true</recursive>
</list_files>
```

### Step 9: Read Key Implementation Files

Examine actual code to understand current patterns and practices.

```xml
<read_file>
<path>{key-implementation-file-path}</path>
</read_file>
```

### Step 10: Analyze Implementation Patterns

Use sequential thinking to extract patterns from actual code.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Analyzing actual implementations found in the codebase. Extracting: 1) Common patterns and conventions used, 2) File structures and naming conventions, 3) Import patterns and dependencies, 4) Error handling approaches, 5) Testing patterns, 6) API integration methods. These patterns must be reflected in the workflow to ensure accuracy.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 3,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 11: Search Documentation for Context

Find relevant documentation, PRDs, or feature specifications.

```xml
<search_files>
<path>docs/</path>
<regex>{topic-related-keywords}</regex>
<file_pattern>*.md</file_pattern>
</search_files>
```

### Step 12: Review Feature Documentation

Read feature specifications that relate to the workflow topic.

```xml
<read_file>
<path>{relevant-feature-doc-path}</path>
</read_file>
```

### Step 13: Check Existing Rules and Guidelines

Identify any existing rules that apply to the workflow topic.

```xml
<search_files>
<path>.clinerules/</path>
<regex>{topic-related-keywords}</regex>
<file_pattern>*.md</file_pattern>
</search_files>
```

### Step 14: Read Relevant Rule Files

Understand constraints and requirements from existing rules.

```xml
<read_file>
<path>.clinerules/{relevant-rules-file}.md</path>
</read_file>
```

### Step 15: Analyze Test Patterns

Examine how testing is currently implemented for related features.

```xml
<list_files>
<path>__tests__/</path>
<recursive>false</recursive>
</list_files>
```

### Step 16: Read Representative Test Files

Understand testing patterns to incorporate into workflow validation.

```xml
<read_file>
<path>__tests__/{relevant-test-area}/{example-test}.test.ts</path>
</read_file>
```

### Step 17: Synthesize Findings

Use sequential thinking to combine all gathered information.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Synthesizing all findings: 1) Actual code patterns discovered, 2) Documentation and feature requirements, 3) Existing rules and constraints, 4) Testing approaches used, 5) File naming and structure conventions, 6) Tool usage patterns from existing workflows. Creating comprehensive picture of how the workflow should operate within actual project context, not theoretical assumptions.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 4,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 18: Design Workflow Structure

Plan the complete workflow based on real project structure and patterns.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Designing workflow structure grounded in actual project state: 1) Usage section with realistic examples, 2) Overview explaining approach and tools (referencing actual project tools), 3) Parameters matching actual data structures found, 4) Sequential steps that operate on real file paths and patterns, 5) Tool usage examples with actual project paths, 6) Validation steps checking against real implementations, 7) Success criteria reflecting actual project structure.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 5,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 19: Draft Usage Section

Create clear, realistic usage documentation with actual project examples.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Writing Usage section with: 1) Context about when to use this workflow in current project, 2) Specific use cases based on actual features/modules found, 3) Prerequisites referencing real dependencies and configurations, 4) Example invocations using actual project terminology and paths. Making it immediately actionable for developers familiar with the project.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 6,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 20: Draft Overview and Parameters

Document workflow methodology and inputs based on project realities.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Creating Overview with: 1) Explanation of workflow's approach using actual project architecture, 2) List of MCP tools and Cline tools that will be used, 3) How workflow integrates with existing project structure. Writing Parameters section with: 1) Required inputs matching actual data formats found, 2) Optional parameters for configuration options that exist, 3) Format examples using real project conventions like file naming, IDs, etc.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 7,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 21: Create Detailed Step Sequence

Write comprehensive steps with actual file paths and realistic tool usage.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Writing detailed steps that: 1) Reference actual project file paths and patterns, 2) Use realistic search patterns based on observed code, 3) Include XML tool examples with project-specific parameters, 4) Incorporate validation against real implementations, 5) Follow patterns observed in existing workflows, 6) Check actual test patterns, 7) Validate against real documentation structure. Each step grounded in project reality.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 8,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 22: Add Context Analysis Steps

Include steps that review actual implementations before proceeding.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Adding critical context gathering steps to the workflow: 1) Reading existing implementations to understand current patterns, 2) Checking documentation to verify requirements, 3) Reviewing test files to understand validation approaches, 4) Analyzing related features for consistency, 5) Verifying against established rules. These steps ensure the workflow users don't make assumptions but operate on verified project state.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 9,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 23: Add Validation and Verification Steps

Include steps that confirm actions against actual project state.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Adding validation steps throughout the workflow: 1) After creating files, verify they match project conventions by comparing with existing files, 2) After modifications, check against actual test patterns, 3) Before completion, validate against documentation requirements, 4) Cross-reference with related implementations for consistency, 5) Confirm no conflicts with existing rules. This prevents theoretical workflows that don't work in practice.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 10,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 24: Create Important Notes Section

Document critical guidelines based on actual project findings.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Writing Important Notes with project-specific guidance: DON'Ts: 1) Common mistakes found in codebase analysis, 2) Anti-patterns discovered in code review, 3) Things that break existing implementations. DOs: 1) Patterns observed in successful implementations, 2) Established project conventions from code analysis, 3) Required validation steps against actual code, 4) How to verify against real documentation. Making notes actionable and specific to actual project.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 11,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 25: Create Realistic Examples

Provide examples using actual project terminology and structures.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Creating example invocations based on: 1) Actual features found in documentation, 2) Real modules discovered in codebase, 3) Existing workflows for related tasks, 4) Common patterns observed in code. Making examples concrete with real feature IDs, actual file paths, authentic project terminology. Users can copy-paste and adapt examples that reference real project elements.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 12,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 26: Define Success Criteria

Create measurable criteria based on actual project validation patterns.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Defining success criteria that: 1) Reference actual test commands from project (npm run test, etc.), 2) Specify real validation checks from codebase (linting, type-checking), 3) Include verification against actual documentation structure, 4) Confirm consistency with existing implementations found, 5) Ensure compliance with discovered rules and conventions. Criteria are verifiable against actual project state.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 13,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 27: Add User Confirmation Checkpoints

Include interaction points for user validation and direction.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Adding user confirmation checkpoints using ask_followup_question tool: 1) After context gathering - confirm understanding of scope, 2) After analyzing implementations - verify correct files identified, 3) Before major changes - get approval for approach, 4) After critical steps - confirm expected results achieved, 5) Before completion - verify all requirements met. Providing clear option arrays for user choices at each checkpoint.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 14,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 28: Validate Workflow Against Rules

Check the drafted workflow against established rules and conventions.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating workflow draft against workflow creation rules: 1) Has clear H1 title, 2) Includes Usage section with examples, 3) Has Overview explaining approach, 4) Contains Parameters section, 5) Includes numbered sequential steps, 6) Each step has H3 heading, 7) Steps include XML tool examples, 8) Has user confirmation checkpoints, 9) Includes Important Notes with DOs/DON'Ts, 10) Has realistic examples, 11) Defines clear success criteria, 12) No code implementations included.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 15,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 29: Verify Against Project Structure

Ensure workflow accurately reflects actual project organization.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Verifying workflow accuracy: 1) All referenced file paths exist in actual project structure, 2) File patterns match real file extensions and naming conventions, 3) Search patterns will find actual implementations, 4) Tool parameters use correct project-specific values, 5) Examples reference real features/modules, 6) Validation steps check against actual test patterns, 7) Success criteria align with real project validation. No assumptions about project state.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 16,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 30: Refine and Polish

Improve clarity, completeness, and accuracy of the workflow.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Final refinement: 1) Ensuring all steps are clear and actionable, 2) Verifying XML examples are properly formatted and complete, 3) Confirming parameter descriptions are unambiguous, 4) Checking that validation steps are thorough, 5) Ensuring terminology is consistent throughout, 6) Verifying all cross-references to files/docs are accurate, 7) Confirming workflow is self-contained yet properly integrated with project. Ready to generate final file.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 17,
  "totalThoughts": 20
}
</arguments>
</use_mcp_tool>
```

### Step 31: Generate Complete Workflow File

Create the final workflow markdown file with all sections.

```xml
<write_to_file>
<path>.clinerules/workflows/{topic-name-kebab-case}.md</path>
<content># {Workflow Title}

## Usage

This workflow {clear description of what it accomplishes in the context of the actual project}.

**When to use this workflow:**
- {Specific use case based on actual features found}
- {Use case referencing real project modules}
- {Use case related to actual documentation structure}

**Example Usage:**
```
{Realistic example using actual project terminology}
```

## Overview

This workflow {comprehensive explanation of approach, mentioning actual project structure, real tools, and existing patterns found in codebase}. 

The workflow uses:
- **Sequential Thinking MCP** - For systematic analysis
- **Context7 MCP** - For best practices (where applicable)
- **Deep Analysis** - Reading actual {specific files/patterns} found in the project
- **Validation** - Checking against {real test patterns/documentation}

## Parameters

When invoking this workflow, provide:

- **{parameter_name}** (required): {Description based on actual data structures found}
- **{optional_param}** (optional): {Description of optional config that exists in project}

**Format for invocation:**
```
{Format example using real project conventions}
```

## Detailed Sequence of Steps

### Step 1: {First Step - Context Gathering}

{Description explaining why this step is needed and what it discovers}

```xml
<{tool_name}>
<{parameter}>{actual-project-path-or-pattern}</{parameter}>
</{tool_name}>
```

### Step 2: {Analysis Step}

{Description of analysis based on real implementations}

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "{Specific analysis relevant to actual project state}",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": {appropriate-number}
}
</arguments>
</use_mcp_tool>
```

### Step 3: {Implementation Review Step}

{Description of reading actual code}

```xml
<read_file>
<path>{actual-project-file-path}</path>
</read_file>
```

### Step 4: {Validation Step}

{Description of checking against real implementations}

```xml
<search_files>
<path>{actual-directory}</path>
<regex>{realistic-pattern-for-project}</regex>
<file_pattern>{actual-extensions}</file_pattern>
</search_files>
```

### Step 5: {User Confirmation Checkpoint}

Confirm the analysis findings with the user before proceeding.

```xml
<ask_followup_question>
<question>Based on the analysis of {actual findings}, I found {specific results}. How would you like to proceed?</question>
<options>["Continue with implementation", "Review findings first", "Adjust scope", "Cancel workflow"]</options>
</ask_followup_question>
```

{Continue with additional steps following actual project patterns...}

### Step N: {Final Validation Step}

{Description of comprehensive validation against project state}

```xml
<{validation_tool}>
{Parameters checking actual implementations}
</{validation_tool}>
```

### Step N+1: {Completion}

{Description of completion criteria}

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Workflow complete. Verified: {List of actual validations against project}. {Summary of accomplishments}. Ready to complete.",
  "nextThoughtNeeded": false,
  "thoughtNumber": {final-number},
  "totalThoughts": {total-number}
}
</arguments>
</use_mcp_tool>
```

## Important Notes

**Do NOT:**
- {Project-specific anti-pattern found in code analysis}
- {Common mistake observed in related implementations}
- {Pattern that breaks existing conventions discovered}
- Make assumptions - always verify against actual implementations
- Skip context-gathering steps - review existing code first
- Ignore existing patterns - follow conventions already established
- Proceed without validation - check against real project state

**DO:**
- {Best practice observed in successful implementations}
- {Pattern consistently used across actual project}
- {Validation approach found in existing tests}
- Always read existing implementations before making changes
- Verify file paths and patterns match actual project structure
- Check documentation for requirements before proceeding
- Validate results against real test patterns
- Confirm consistency with established project conventions

## Example Invocations

### Example 1: {Realistic Scenario Using Actual Features}
```
{Example referencing real feature IDs or modules found in project}
```

### Example 2: {Another Realistic Scenario}
```
{Example using actual terminology and paths from project}
```

### Example 3: {Edge Case from Project}
```
{Example addressing real scenario discovered in codebase}
```

## Success Criteria

The workflow is complete when:
- [ ] {Criterion based on actual project validation - e.g., "Tests pass using npm run test:unit"}
- [ ] {Criterion checking real implementations - e.g., "Consistent with patterns in src/lib/"}
- [ ] {Criterion verifying documentation - e.g., "Matches requirements in docs/features/"}
- [ ] {Criterion confirming conventions - e.g., "Follows .clinerules/{relevant-rules}.md"}
- [ ] All file paths referenced exist in actual project structure
- [ ] Patterns match existing implementations discovered in codebase
- [ ] Validation checks pass against real test patterns
- [ ] No conflicts with established project rules and conventions

For End Users: {Clear, non-technical summary of what users can now accomplish with this workflow, explaining the benefit and how it improves their work based on actual project context}
</content>
</write_to_file>
```

## Important Notes

**Official Cline Best Practices:**
- Be clear and concise - use simple language and avoid ambiguity
- Focus on desired outcomes rather than specific implementation steps
- Be modular - break complex tasks into smaller, reusable workflows
- Comment your workflow steps like code - explain WHY, not just WHAT
- Balance precision and flexibility - high-level instructions often suffice
- Consider creating workflows from completed tasks for maximum accuracy

**Do NOT:**
- Create massive monolithic workflows - break into smaller, reusable modules
- Use overly complex language when simple instructions would work
- Focus on implementation details when outcomes would be clearer
- Create workflows based on assumptions about project structure
- Skip Sequential Thinking steps for complex analysis
- Skip Context7 research for domain-specific topics
- Omit user confirmation checkpoints from generated workflows
- Use placeholder paths without verifying actual project paths exist
- Copy patterns from other projects without validating against this codebase
- Generate workflows with missing required sections (Usage, Overview, Parameters, Steps, Important Notes, Examples, Success Criteria)

**DO:**
- Always keep workflows simple and focused on outcomes
- Always explain WHY each step is needed, not just what it does
- Always read existing implementations before defining workflow steps
- Always use Sequential Thinking MCP for structured analysis
- Always query Context7 for best practices on domain-specific topics
- Always validate generated file paths against actual project structure
- Always include at least 3 example invocations with realistic scenarios
- Always define measurable success criteria with checklists
- Always include user confirmation checkpoints at key decision points
- Cross-reference .clinerules/cline_workflow_creation_best_practices.md during creation
- Place project-specific workflows in `.clinerules/workflows/`
- Place global workflows in `~/Documents/Cline/Workflows/` when applicable

## Example Invocations

### Example 1: API Endpoint Workflow
```
Create a comprehensive Cline workflow for: Creating new API endpoints following RESTful conventions
Focus on: src/app/api/, src/lib/api-helpers/
Related to: docs/api/, .clinerules/api_*.md
```

### Example 2: Component Development Workflow
```
Create a comprehensive Cline workflow for: Building admin persona components with proper state management
Focus on: src/components/admin/, src/hooks/
Related to: docs/features/, .clinerules/ui_*.md
```

### Example 3: Business Rules Workflow
```
Create a comprehensive Cline workflow for: Implementing business rules using json-rules-engine
Focus on: src/lib/rules-engine/, prisma/seed-data/
Related to: docs/features/rules-management/, .clinerules/business-logic_*.md
```

### Example 4: Integration Testing Workflow
```
Create a comprehensive Cline workflow for: Writing integration tests for API endpoints
Focus on: __tests__/integration/, __tests__/integration/helpers/
Related to: .clinerules/integration_*.md, docs/testing/
```

### Example 5: Create Workflow from Completed Task (Official Cline Approach)
After completing any repeatable task, use this approach:
```
Create a workflow for the process I just completed
```
Cline analyzes the conversation history, identifies the steps taken, and generates a workflow grounded in real execution.

## Success Criteria

The workflow is complete when:
- [ ] YAML frontmatter is present with description, applies_to, and priority fields
- [ ] Generated workflow file is placed in `.clinerules/workflows/` directory
- [ ] Generated workflow has all required sections (Usage, Overview, Parameters, Steps, Important Notes, Examples, Success Criteria)
- [ ] All steps include complete XML tool syntax examples
- [ ] At least one user confirmation checkpoint is included
- [ ] At least 3 example invocations are provided with H3 headings
- [ ] Success criteria use checklist format with measurable items
- [ ] Important Notes section has both DO and DO NOT subsections
- [ ] File paths referenced in workflow match actual project structure
- [ ] Workflow follows `.clinerules/cline_workflow_creation_best_practices.md` rules
- [ ] No code implementations are included in the workflow (only tool usage examples)

For End Users: This workflow helps you create reliable, project-aware automation workflows that accurately reflect your codebase's actual structure and patterns. Instead of generating theoretical workflows that might not work, this ensures every workflow you create is grounded in real code analysis and follows established best practices
