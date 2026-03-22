---
description: Creates properly formatted Cline rules files for specific topics or domains, following best practices and conventions
applies_to: [".clinerules/workflows/**/*.md"]
priority: high
---

# Generate Cline Rules File

## Usage

This workflow creates a properly formatted Cline rules file for a specific topic or domain, following best practices and conventions from the Cline prompts community.

**When to use this workflow:**
- When you need to document best practices for a specific technology or domain
- When establishing coding standards for a new area of the codebase
- When standardizing architectural patterns and conventions
- When creating enforceable guidelines for development practices

**Example Usage:**
```
Create a Cline rules file for GraphQL API development best practices
```

## Overview

This workflow uses Sequential Thinking MCP server to systematically research and organize best practices, Context7 MCP server to gather domain-specific knowledge, and generates a complete, well-structured Cline rules file with proper YAML frontmatter and organized rule sections.

## Parameters

When invoking this workflow, provide:

- **topic** (required): The domain, technology, or area for which rules should be created
- **file_path** (optional): Custom path for the rules file (defaults to `.oxenated.d/rules.d/{always-rules|opt-rules}/{category}_{topic-kebab-case}.md` based on category)
- **specific_aspects** (optional): Particular areas within the topic to focus on

**Format for invocation:**
```
Create a Cline rules file for [topic] with best practices on [specific aspects]
```

## Detailed Sequence of Steps

### Step 1: Initialize Sequential Thinking

Begin analysis of the topic with sequential thinking to structure the research and rule generation process.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Starting Cline rules file generation for [topic]. I will: 1) Research the topic thoroughly using Context7, 2) Identify key best practices and patterns, 3) Organize rules into logical categories, 4) Ensure rules are concise and actionable, 5) Generate proper YAML frontmatter with appropriate globs and tags, 6) Validate rules don't conflict with each other.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 2: Research Topic Domain with Context7

Query Context7 to gather comprehensive documentation and best practices for the topic.

```xml
<use_mcp_tool>
<server_name>github.com/upstash/context7-mcp</server_name>
<tool_name>resolve-library-id</tool_name>
<arguments>
{
  "libraryName": "[topic name or related library]"
}
</arguments>
</use_mcp_tool>
```

### Step 3: Retrieve Detailed Documentation

Once library ID is resolved, fetch detailed documentation and best practices.

```xml
<use_mcp_tool>
<server_name>github.com/upstash/context7-mcp</server_name>
<tool_name>get-library-docs</tool_name>
<arguments>
{
  "context7CompatibleLibraryID": "[resolved library ID from previous step]",
  "topic": "[topic] best practices, patterns, and conventions",
  "tokens": 5000
}
</arguments>
</use_mcp_tool>
```

### Step 4: Analyze and Categorize Findings

Use sequential thinking to analyze the gathered information and organize into rule categories.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Analyzing Context7 documentation for [topic]. Identifying key patterns: 1) Core principles and requirements, 2) Common anti-patterns to avoid, 3) Performance considerations, 4) Security best practices, 5) Testing requirements, 6) Integration patterns. Organizing these into logical rule categories that are actionable and specific.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 2,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 5: Generate YAML Frontmatter

Create proper YAML frontmatter with appropriate metadata for the rules file.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Generating YAML frontmatter: 1) Description - clear explanation of rules purpose, 2) applies_to - file glob patterns for when rules apply (e.g., **/*.tsx, **/*.graphql), 3) priority - high/critical/medium based on impact, 4) tags - relevant categorization tags. Ensuring metadata accurately reflects the scope and importance of these rules.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 3,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 6: Write Rule Sections by Category

Organize rules into logical sections with clear headings and concise rule statements.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Writing rule sections organized by category. Each rule must: 1) Start with 'Rule:' prefix, 2) Be concise and actionable (1-2 sentences max), 3) Focus on what/why not how, 4) Not include code examples or implementations, 5) Not conflict with other rules. Organizing sections logically: Core Principles, Implementation Requirements, Security, Testing, Anti-Patterns, etc.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 4,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 7: Validate Rules for Conflicts and Clarity

Review all rules to ensure they are non-conflicting, concise, and properly formatted.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating rules: 1) Checking each rule starts with 'Rule:' prefix, 2) Ensuring no conflicting directives exist, 3) Confirming rules are concise without verbose explanations, 4) Verifying no code examples included (only interface definitions if needed), 5) Ensuring all rules are actionable and specific, 6) Checking globs match intended file patterns.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 5,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 8: Final Review and Refinement

Perform final review of the complete rules file before saving.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Final review of rules file: 1) YAML frontmatter is properly formatted with valid description, applies_to, priority, 2) All sections have clear headings, 3) Rules are organized logically by category, 4) Each rule follows 'Rule:' prefix convention, 5) No code examples present, 6) Rules are comprehensive but concise, 7) File structure matches Cline rules conventions.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 6,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

### Step 9: Generate and Save Rules File

Create the complete rules file in the appropriate .oxenated.d/rules.d/ subdirectory based on category:

**Directory Selection by Category Prefix:**
- **.oxenated.d/rules.d/always-rules/** - For rules that should ALWAYS be active for every task:
  - `cline_`, `core_`, `bdd-test_`, `business-logic_`, `e2e-test_`, `integration_`, `type_`, `unit-test_`
- **.oxenated.d/rules.d/opt-rules/** - For optional rules activated only when needed:
  - `api_`, `bash_`, `database_`, `doc_`, `infra_`, `page_`, `security_`, `ui_`

Note: .clinerules/ is for ephemeral storage only - rules are activated there by the rules.sh script.

```xml
<write_to_file>
<path>.oxenated.d/rules.d/{always-rules|opt-rules}/{category}_{topic-name-kebab-case}.md</path>
<content>---
description: [Clear description of what these rules cover]
applies_to: ["**/*.{relevant-extensions}", "specific-patterns/**/*"]
priority: high
---

# [Topic] Best Practices

## Core Principles

**Rule:** [First core principle rule]
**Rule:** [Second core principle rule]
**Rule:** [Additional core rules...]

## Implementation Requirements

**Rule:** [Implementation requirement 1]
**Rule:** [Implementation requirement 2]
**Rule:** [Additional implementation rules...]

## Security Considerations

**Rule:** [Security rule 1]
**Rule:** [Security rule 2]
**Rule:** [Additional security rules...]

## Performance Optimization

**Rule:** [Performance rule 1]
**Rule:** [Performance rule 2]
**Rule:** [Additional performance rules...]

## Testing Requirements

**Rule:** [Testing rule 1]
**Rule:** [Testing rule 2]
**Rule:** [Additional testing rules...]

## Anti-Patterns to Avoid

**Rule:** [Anti-pattern 1]
**Rule:** [Anti-pattern 2]
**Rule:** [Additional anti-patterns...]

## [Additional relevant sections as needed]

**Rule:** [Additional rules organized by logical categories]
</content>
</write_to_file>
```

### Step 10: Confirm Completion

Use sequential thinking to confirm the rules file is complete and properly formatted.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Rules file generation complete. Verifying: 1) File saved to .oxenated.d/rules.d/opt-rules/ directory (permanent rules database), 2) YAML frontmatter present and valid, 3) All rules follow 'Rule:' prefix convention, 4) Rules organized into logical sections, 5) No code examples included, 6) Rules are concise and actionable, 7) No conflicting rules exist. Ready to present completion summary.",
  "nextThoughtNeeded": false,
  "thoughtNumber": 8,
  "totalThoughts": 8
}
</arguments>
</use_mcp_tool>
```

## Important Notes

**Do NOT:**
- Include actual code implementations or working code examples in rules
- Create rules that conflict with each other (e.g., "always use X" and "never use X")
- Be verbose - rules should be concise and actionable (1-2 sentences maximum)
- Forget the YAML frontmatter at the beginning of the file
- Use vague language like "consider" or "maybe" - be directive and specific
- Include detailed algorithm implementations - only high-level descriptions
- Create rules without researching the topic thoroughly first

**DO:**
- Use the 'Rule:' prefix for every rule statement
- Research the topic comprehensively using Context7
- Organize rules into logical categories with clear section headings
- Include proper glob patterns in applies_to for file matching
- Use descriptive tags for categorization
- Keep rules actionable and specific to the domain
- Ensure all rules are enforceable and verifiable
- Allow interface definitions and method signatures when needed for clarity
- Validate that no rules contradict each other
- Use consistent priority levels (critical, high, medium, low)

## Example Invocations

### Example 1: GraphQL API Development
```
Create a Cline rules file for GraphQL API development best practices
```

### Example 2: React Testing with Specific Focus
```
Create a Cline rules file for React Testing Library with specific focus on accessibility testing and user-centric test patterns
```

### Example 3: Database Optimization
```
Create a Cline rules file for PostgreSQL query optimization and performance tuning
```

### Example 4: Custom File Path
```
Create a Cline rules file for Kubernetes deployment patterns and save to .oxenated.d/rules.d/opt-rules/infra_k8s-deployment-practices.md
```

## Success Criteria

The workflow is complete when:
- [ ] Rules file created in appropriate `.oxenated.d/rules.d/` subdirectory (`always-rules/` or `opt-rules/`) based on category prefix
- [ ] Valid YAML frontmatter present with description, applies_to, and priority fields
- [ ] All rules properly prefixed with `**Rule:**` 
- [ ] Rules organized into logical sections with clear headings
- [ ] No code examples or implementations included (only interface definitions if needed)
- [ ] Rules are concise (1-2 sentences each) and actionable
- [ ] No conflicting rules exist within the file
- [ ] Proper glob patterns defined for file matching in applies_to
- [ ] Descriptive tags included in frontmatter for categorization
- [ ] File structure follows Cline rules conventions

For End Users: You can now easily generate standardized, well-organized rules files for any technology or domain by simply providing a topic name. The workflow automatically researches best practices, organizes them into clear categories, and ensures all rules follow proper formatting conventions without requiring manual YAML configuration or rule structuring.