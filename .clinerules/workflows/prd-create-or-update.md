---
description: Generates or updates comprehensive Product Requirements Documents (PRDs) using IAOOI framework, Gherkin BDD scenarios, and AI-first execution plans
applies_to: [".oxenated/docs/planning/**/*.md", "docs/product/**/*.md"]
priority: high
---

# Generate or Update Product Requirements Document (PRD)

## Usage

This workflow generates comprehensive Product Requirements Documents (PRDs) for fintech, SaaS, and ERP platforms following industry best practices, or updates existing PRDs with expanded scope. PRDs are structured by persona-driven personas using the IAOOI (Inputs-Activities-Outputs-Outcomes-Impacts) systems model and include detailed Gherkin BDD scenarios for AI-first development.

**When to use this workflow:**
- When creating a new PRD for a platform or feature set from scratch
- When updating an existing PRD to expand scope with additional features or personas
- When defining functionality that addresses specific business problems and personas
- When preparing requirements for AI agent development teams

**Prerequisites:**
- Clear understanding of the business problem or opportunity
- Identified target personas and their roles
- Core requirements (can be high-level or detailed)
- Optional: Positioning statement, persona documents, latent demand research

**Expected Outcomes:**
- Complete PRD document with system-level IAOOI framework
- Persona-driven persona structure with epics and features
- Gherkin BDD scenarios for all features
- Traceability matrix linking features to requirements
- AI execution plan for implementation

## Overview

This workflow systematically analyzes business requirements and generates or updates comprehensive PRDs structured for AI development teams. It applies product management best practices including:

- **IAOOI Framework**: Every system, epic, and feature documents Inputs, Activities, Outputs, Outcomes, and Impacts to ensure complete understanding
- **Persona-Driven Design**: Each persona serves specific personas with self-contained functionality
- **Gherkin BDD Scenarios**: All features include executable user journey specifications
- **AI-First Execution**: PRDs include detailed execution plans optimized for AI agent development
- **Traceability**: Complete mapping between epics/features and original requirements

The workflow uses:
- **Sequential Thinking MCP** - For thorough multi-step analysis of requirements and structure
- **Context7 MCP** - For accessing best practices in PRD creation (when applicable)
- **File Operations** - For reading existing PRDs and writing updated documents

## Parameters

When invoking this workflow, provide:

### Required Parameters

- **business_problem** (required): Clear description of what the platform should solve. Include specific pain points and goals.
- **target_personas** (required): Who will use the system and their roles. Format: persona name and primary responsibilities.
- **core_requirements** (required): Key functionality needed. Can be high-level bullet points or detailed specifications.

### Optional Parameters

- **existing_prd_path** (optional): File path if updating existing PRD. Default: none (creates new). Example: `.oxenated/docs/planning/product_requirements.md`
- **positioning_statement** (optional): How the platform is positioned in market. Default: none.
- **persona_documents** (optional): Detailed persona descriptions and goals. Default: none.
- **latent_demand_research** (optional): Customer research, interviews, pain points. Default: none.
- **competitive_landscape** (optional): Competitor analysis and differentiation. Default: none.

**Format for invocation:**
```
Use sequential thinking to [generate a PRD | update the PRD at <path>] for <platform description>.

Context: <business problem and requirements>

Optional Attachments:
- Positioning: <positioning statement>
- Personas: <persona descriptions>
- Latent Demand Research: <research findings>
- Competitive Landscape: <competitive analysis>
```

## Detailed Sequence of Steps

### Step 1: Initialize Sequential Thinking and Detect Mode

Begin analysis and determine whether creating new PRD or updating existing one. This step establishes the workflow context and approach.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Starting PRD workflow. First, detecting mode: if user provided existing PRD path, this is an UPDATE; otherwise NEW creation. For NEW: will gather business context, define personas, map requirements. For UPDATE: will read existing PRD, understand current scope, integrate new requirements seamlessly. Both modes require: IAOOI framework at all levels, proper ID formats (EPIC-[PERSONA]-[SCOPE]-[ID]), Gherkin BDD scenarios, traceability matrix, AI execution plan.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 2: Read Existing PRD (If Updating)

**Conditional Step:** Only execute if updating an existing PRD. Skip to Step 3 if creating new PRD.

Read the current PRD to understand existing structure, personas, epics, and scope before integrating new requirements.

```xml
<read_file>
<path>{existing_prd_path provided by user}</path>
</read_file>
```

### Step 3: Analyze Business Context and Personas

Use sequential thinking to understand the business problem, target personas, and requirements comprehensively.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Analyzing business context: 1) What problem does this platform solve? 2) Who are the target personas and what are their pain points? 3) What are the core requirements and desired outcomes? 4) What is the competitive landscape and positioning? 5) What latent demand exists based on research? Building persona-to-persona mapping: each primary persona gets a dedicated persona with self-contained functionality.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 2,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 4: User Confirmation - Context Understanding

Confirm the analysis findings with the user before proceeding with structure design.

```xml
<ask_followup_question>
<question>Based on the business context analysis, I identified [summary of personas, problems, and requirements]. Is this understanding correct, or would you like to adjust the scope before I proceed with the PRD structure?</question>
<options>["Yes, proceed with PRD structure", "Adjust scope - let me clarify", "Add more context first", "Start over with different focus"]</options>
</ask_followup_question>
```

### Step 5: Define System-Level IAOOI

Establish the complete system IAOOI that will guide all epic and feature design.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Defining system-level IAOOI framework: System Inputs (what data/triggers enter the platform), System Activities (core processing and workflows), System Outputs (what the system produces), System Outcomes (immediate business improvements), System Impacts (long-term strategic value). This high-level IAOOI will cascade down to each epic and feature, ensuring complete value chain visibility.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 3,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 6: Identify Personas and Primary Personas

Map personas to personas, ensuring each persona serves specific user types with self-contained functionality.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Identifying personas: Each primary persona gets a dedicated persona. Examples: Admin Persona (owner/executive), Operations Manager Persona (dispatcher/supervisor), Guard Persona (field worker), Client Persona (customer). Each persona must be self-contained - users shouldn't need to switch personas unless they serve multiple roles. Determining which personas are in MVP scope vs future phases.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 4,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 7: Break Down Requirements into Epics

Identify major epics for each persona, ensuring each epic addresses significant business value.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Breaking requirements into epics by persona. Each epic represents a major functional area delivering cohesive value. Format: EPIC-[PERSONA]-[SCOPE]-[ID] where PERSONA = ADMIN/OpsManager/Guard/Client, SCOPE = functional area like SCHED/TA/OPS, ID = sequential number. Each epic must: 1) Serve the persona's persona, 2) Have clear business value, 3) Include complete IAOOI, 4) Reference original requirements, 5) Be sized appropriately (multiple features within).",
  "nextThoughtNeeded": true,
  "thoughtNumber": 5,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 8: Define Epic-Level IAOOI for Each Epic

For each identified epic, establish detailed IAOOI showing value creation.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "For each epic, defining comprehensive IAOOI: Inputs (what data/triggers this epic needs), Activities (what processing/workflows happen), Outputs (what this epic produces - UI, reports, integrations), Outcomes (immediate business improvements - efficiency, accuracy, satisfaction), Impacts (long-term strategic value - cost savings, retention, growth). Each epic IAOOI must align with and support the system-level IAOOI.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 6,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 9: User Confirmation - Epic Structure

Confirm the epic structure before decomposing into features.

```xml
<ask_followup_question>
<question>I've identified [N] epics across [N] personas: [list epic names by persona]. Does this structure align with your expectations, or would you like to adjust the epic organization?</question>
<options>["Proceed with feature decomposition", "Add more epics", "Combine some epics", "Reorganize persona structure", "Review epic details first"]</options>
</ask_followup_question>
```

### Step 10: Decompose Epics into Features

Break each epic into 1-4 specific features with clear implementation scope.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Decomposing epics into features. Format: [EPICID]-[FEATURE_SCOPE]-[ID] where FEATURE_SCOPE describes the feature type (UI, API, RULE, ANA, etc.) and ID is sequential. Each feature must: 1) Be implementable within 3-7 days by AI, 2) Have independent business value, 3) Include complete IAOOI showing where it fits in epic context, 4) Include Gherkin BDD scenarios, 5) Be testable independently. Features should be concrete and actionable.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 7,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 11: Define Feature-Level IAOOI

For each feature, establish detailed IAOOI within epic context.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "For each feature, defining detailed IAOOI showing how it contributes to epic goals: Inputs (specific data this feature consumes), Activities (precise operations performed), Outputs (UI components, API responses, events produced), Outcomes (specific improvements - reduced time, eliminated errors, enabled capability), Impacts (contribution to epic and system impacts). Feature IAOOI must clearly show value chain from inputs through to strategic impact.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 8,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 12: Generate Gherkin BDD Scenarios

Create detailed user journey scenarios in Gherkin syntax for each feature.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Generating Gherkin BDD scenarios for each feature. Format: Scenario: [descriptive name] with Given/When/Then steps. Each scenario must: 1) Show complete user journey from persona perspective, 2) Use concrete examples with realistic data, 3) Cover happy path and key edge cases, 4) Be executable as acceptance tests, 5) Use ubiquitous language from the domain. Including comments in parentheses for alternative flows where helpful. These become the acceptance criteria for AI development.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 9,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 13: Create Traceability Matrix

Build complete mapping from epics/features back to original requirements.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Creating traceability matrix linking every epic and feature to original requirement IDs. Format: table with Epic/Feature ID in first column, Original Requirement IDs in second column. This ensures: 1) Complete requirements coverage, 2) No orphaned requirements, 3) Clear justification for each feature, 4) Easy impact analysis for requirement changes. If requirements weren't formally numbered, create implicit mapping to business needs stated in context.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 10,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 14: Design AI Execution Plan

Create detailed implementation strategy for AI development agents.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Designing AI execution plan: 1) Foundational Setup (architecture, data model, tech stack), 2) Persona Development Streams (parallel development with coordination points), 3) Integration & Testing (end-to-end scenarios), 4) Security & Compliance Review, 5) User Training & Feedback Loops. Each stream identifies: which AI agents work on which epics, dependencies between epics, testing strategies, coordination points. Plan optimizes for parallel development while managing dependencies.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 11,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 15: Validate PRD Completeness

Verify all required sections are present and properly structured.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating PRD completeness checklist: 1) System Overview with complete IAOOI, 2) Each persona has title and description, 3) All epics follow ID format and include IAOOI, 4) All features follow ID format and include IAOOI, 5) All features have Gherkin BDD scenarios, 6) Traceability matrix is complete, 7) AI execution plan is detailed and actionable, 8) No timelines or roadmap dates (AI-focused), 9) Language is clear and technical where needed, 10) Consistent terminology throughout.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 12,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

### Step 16: Write PRD Document

Generate the complete PRD following the established structure. Use the PRD template format from `.oxenated/docs/planning/` directory conventions.

```xml
<write_to_file>
<path>.oxenated/docs/planning/product_requirements.md</path>
<content>{Generated PRD content following IAOOI structure with all sections}</content>
</write_to_file>
```

**Note:** The PRD content should follow the template structure including:
- H1 title with platform name
- Scope description
- System Overview with IAOOI framework
- Persona sections (one per persona)
- Epics with IAOOI and ID format
- Features with IAOOI, Gherkin scenarios, and ID format
- Traceability Matrix table
- AI Execution Plan with numbered phases
- End User summary paragraph

### Step 17: Final Validation and Completion

Confirm the PRD meets all requirements and provide summary to user.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "PRD workflow complete. Final validation: 1) Document written to correct path, 2) All sections present and properly formatted, 3) IAOOI applied at system, epic, and feature levels, 4) All features have Gherkin scenarios, 5) Traceability matrix complete, 6) AI execution plan actionable, 7) Consistent terminology throughout. Ready to present completion summary to user.",
  "nextThoughtNeeded": false,
  "thoughtNumber": 13,
  "totalThoughts": 13
}
</arguments>
</use_mcp_tool>
```

## Important Notes

**DO:**
- Always start with Sequential Thinking to structure analysis
- Define system-level IAOOI before drilling into personas and epics
- Map each persona to exactly one primary persona
- Use consistent ID formats: EPIC-[PERSONA]-[SCOPE]-[ID] and [EPICID]-[FEATURE_SCOPE]-[ID]
- Include complete IAOOI at every level (system, epic, feature)
- Write Gherkin scenarios using concrete, realistic data
- Create traceability matrix linking every feature to requirements
- Confirm understanding with user at key checkpoints
- Keep language clear, technical where needed, jargon-free where possible
- Focus on outcomes and value, not implementation details

**DO NOT:**
- Skip the persona-to-persona mapping step
- Create features without Gherkin BDD scenarios
- Write vague IAOOI sections without specific details
- Include timeline dates or roadmap schedules (AI-focused, not time-boxed)
- Create orphaned requirements not linked in traceability matrix
- Mix multiple personas in a single persona (unless explicitly justified)
- Write implementation code in the PRD (outcomes and requirements only)
- Skip user confirmation checkpoints
- Create epics that span multiple personas
- Use inconsistent terminology across the document

## Example Invocations

### Example 1: New Security Guard Management Platform PRD
```
Use sequential thinking to generate a PRD for a security guard management platform.

Context: We need a platform for scheduling security guards, tracking time & attendance, and managing field operations. Target personas are Admin (company owner), Operations Manager (dispatcher), and Guard (field worker).

Optional Attachments:
- Positioning: Cloud-based security operations platform
- Personas: See attached persona documents
- Latent Demand Research: Customer interviews show need for mobile time clock and shift marketplace
```

### Example 2: Update Existing PRD with Payroll Integration
```
Use sequential thinking to update the existing PRD at .oxenated/docs/planning/product_requirements.md with expanded scope.

New Requirements: Add payroll integration features including automated payroll export, overtime classification rules, and wage advance calculation. This expands the Admin persona and adds financial services capabilities.

Context: Customers are requesting tighter integration with payroll providers and early wage access for guards.
```

### Example 3: New Client Persona PRD
```
Use sequential thinking to generate a PRD for a client-facing persona.

Context: Security company clients need to view guard schedules, request coverage changes, review incident reports, and approve invoices. The persona should integrate with the existing Admin and Operations personas.

Target Personas: Client Manager (primary contact), Client Executive (approval authority)

Requirements: Schedule visibility, coverage requests, incident reports, invoice approval, communication with operations
```

### Example 4: Minimal MVP PRD for Startup
```
Use sequential thinking to generate an MVP PRD for a field service scheduling platform.

Context: Early-stage startup needs core scheduling functionality only. Single Admin persona managing technicians. Must be implementable in 2 weeks by AI agents.

Target Personas: Admin (owner/operator)
Core Requirements: Technician profiles, job scheduling, calendar view, basic reporting
```

## Success Criteria

The workflow is complete when:
- [ ] PRD document is written to `.oxenated/docs/planning/` directory
- [ ] System Overview section includes complete IAOOI framework
- [ ] Each identified persona has a dedicated persona section
- [ ] All epics follow ID format: EPIC-[PERSONA]-[SCOPE]-[ID]
- [ ] All epics include complete IAOOI (Inputs, Activities, Outputs, Outcomes, Impacts)
- [ ] All features follow ID format: [EPICID]-[FEATURE_SCOPE]-[ID]
- [ ] All features include complete IAOOI
- [ ] All features have at least one Gherkin BDD scenario
- [ ] Traceability matrix links all epics/features to original requirements
- [ ] AI Execution Plan includes numbered phases with clear action items
- [ ] No timeline dates or roadmap schedules included
- [ ] Terminology is consistent throughout the document
- [ ] User confirmed understanding at checkpoints

For End Users: This workflow helps you create comprehensive, AI-ready Product Requirements Documents that clearly define what needs to be built, for whom, and why. The resulting PRD uses the IAOOI framework to ensure every feature is traceable from business need to strategic impact, includes Gherkin scenarios that serve as executable acceptance criteria, and provides an AI execution plan that enables development teams to work in parallel efficiently.
