---
description: Workflow to extract and define features from MVP epics based on PRD requirements
applies_to: [".oxenated/docs/planning/**/*.md", "docs/_product_management_work_folder/**/*.md"]
priority: high
---

# PRD Feature Extraction Workflow

## Usage

This workflow extracts detailed features from MVP epics, creating individual feature.md files with complete context for AI implementation.

**When to use this workflow:**
- After epics have been created in Step 1 (epic extraction)
- When breaking down epics into implementable features
- When creating feature documentation for AI-driven development
- When mapping PRD requirements to individual feature specifications

**Prerequisites:**
- Completed epic extraction (prd-epic-extract.md workflow)
- Access to the MVP PRD document
- Epic directories exist in `.oxenated/docs/planning/[persona]/epics/`

**Example Usage:**
```
/prd-feature-extract.md
Extract features for: EPIC-OpsManager-SCHED-01
PRD Source: docs/_product_management_work_folder/mvp-prd.md
```

## Overview

This workflow systematically extracts features from PRD documentation, ensuring each feature:
- Has complete IAOOI framework (Inputs, Activities, Outputs, Outcomes, Impacts)
- Includes BDD scenarios where provided in PRD
- Is sized appropriately (3-7 days AI implementation)
- Has clear dependencies and integration points documented

The workflow uses:
- **Sequential Thinking MCP** - For structured feature analysis
- **File Operations** - Reading PRD and creating feature files
- **Validation** - Ensuring completeness against PRD requirements

## Parameters

When invoking this workflow, provide:

- **epic_id** (required): The epic identifier to extract features from (e.g., EPIC-ADMIN-SCHED-01)
- **prd_source** (optional): Path to PRD document. Default: `docs/_product_management_work_folder/mvp-prd.md`
- **persona** (optional): Target persona if known. Default: extracted from epic_id

**Format for invocation:**
```
/prd-feature-extract.md
Extract features for: [EPIC-ID]
PRD Source: [path/to/prd.md]
```

## Detailed Sequence of Steps

### Step 1: Initialize Feature Extraction Analysis

Begin systematic analysis of the epic and its feature requirements.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Starting feature extraction for specified epic. Plan: 1) Read epic documentation to understand scope, 2) Locate epic section in PRD, 3) Identify all features listed under this epic, 4) Extract complete feature details including IAOOI and BDD scenarios, 5) Create feature directories and files, 6) Validate completeness against PRD.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 2: Read Epic Documentation

Read the parent epic to understand scope and context.

```xml
<read_file>
<path>.oxenated/docs/planning/{persona}/epics/{epic-id}/epic.md</path>
</read_file>
```

### Step 3: Read PRD Source Document

Access the PRD to extract feature details.

```xml
<read_file>
<path>docs/_product_management_work_folder/mvp-prd.md</path>
</read_file>
```

### Step 4: Analyze PRD for Feature Definitions

Use sequential thinking to identify all features for this epic.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Analyzing PRD for features under this epic. Extracting: 1) Feature IDs matching pattern FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN], 2) Complete feature descriptions, 3) IAOOI framework components (Inputs, Activities, Outputs, Outcomes, Impacts), 4) BDD/Gherkin scenarios if provided, 5) Requirements mapping (R*.* references), 6) Integration points with other features/personas.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 2,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 5: User Confirmation - Feature List

Confirm the identified features before creating files.

```xml
<ask_followup_question>
<question>I've identified the following features from the PRD for this epic. Please confirm the list is complete:

[List features found with IDs and names]

Should I proceed with creating feature documentation for all of these?</question>
<options>["Proceed with all features", "Add missing features first", "Remove some features", "Review PRD section again"]</options>
</ask_followup_question>
```

### Step 6: Determine Feature Sizing

Validate each feature meets sizing guidelines (3-7 days implementation).

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating feature sizing. Checking: 1) Each feature represents 3-7 days of AI implementation work, 2) Features have clear business value for target persona, 3) Features are independently testable, 4) Features align with system boundaries, 5) If too large, identifying how to split while maintaining coherence.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 3,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 7: Create Feature Directory Structure

Create directories for each identified feature.

```xml
<execute_command>
<command>mkdir -p .oxenated/docs/planning/{persona}/epics/{epic-id}/features/{feat-id-01} .oxenated/docs/planning/{persona}/epics/{epic-id}/features/{feat-id-01}/tasks</command>
<requires_approval>false</requires_approval>
</execute_command>
```

### Step 8: Extract Feature Content from PRD

For each feature, extract complete content following the template.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Extracting complete feature content: 1) Feature Header - ID and name exactly as in PRD, 2) Feature Description - full paragraph from PRD, 3) IAOOI Framework - all five components verbatim, 4) Gherkin Scenarios - BDD scenarios if provided, 5) Requirements Mapping - which R*.* items addressed, 6) Integration Notes - connections to other features/personas. Ensuring nothing from PRD is missed.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 4,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 9: Create Feature Documentation with Citations

Write the feature.md file with complete content and proper citations.

**CRITICAL - Citation Requirements per `.clinerules/cline_ai_citation_traceability_requirements.md`:**
- All content extracted from PRD MUST cite the PRD source with line numbers
- All references to existing code, APIs, or database schemas MUST cite their source files
- All references to existing components MUST cite the component file and line
- New functionality to be created MUST be marked as "Proposed:" or "To be created:"
- Integration points with existing systems MUST cite the existing interface definitions

```xml
<write_to_file>
<path>.oxenated/docs/planning/{persona}/epics/{epic-id}/features/{feat-id}/feature.md</path>
<content># [Feature Name]

## Feature ID
[FEAT-XXX-XXX-XX]

## Source
**PRD Source:** [Source: docs/_product_management_work_folder/mvp-prd.md:L##-L##]

## Epic Context
**Parent Epic:** [EPIC-XXX-XX] - [Epic Name] [Source: .oxenated/docs/planning/{persona}/epics/{epic-id}/epic.md:L##]
**Target Persona:** [Admin / Operations Manager / Guard]
**Epic Objective:** [Brief summary of parent epic goal] [Source: mvp-prd.md:L##]
**Business Impact:** [How this feature contributes to operational goals] [Source: mvp-prd.md:L##]

## Feature Overview
**Purpose:** [What this feature accomplishes for the persona] [Source: mvp-prd.md:L##]
**Scope:** [What is included and excluded] [Source: mvp-prd.md:L##]
**PRD References:** [List of R*.* items this implements] [Source: mvp-prd.md:L##]
**PRD Feature ID:** [The exact feature ID from the MVP PRD] [Source: mvp-prd.md:L##]
**Dependencies:** [Required features/components - cite existing if applicable]

## IAOOI Components
[Source: mvp-prd.md:L##-L## - IAOOI section for this feature]
**Inputs:** [Data/events this feature receives]
**Activities:** [Core processing/business logic]
**Outputs:** [Data/events this feature produces]
**Outcomes:** [Business improvements from this feature]
**Impacts:** [Long-term value created]

## Technical Requirements
**Architecture Layer:** [UI/Application/Domain/Infrastructure]
**Integration Points:** 
- [Existing API: cite source] [Source: src/app/api/path/route.ts:L##] OR
- [Proposed: New API endpoint to be created]
**Data Requirements:** 
- [Existing model: cite schema] [Source: prisma/schema.prisma:L## - ModelName] OR
- [Proposed: New entities to be created]
**Performance Requirements:** [Response time, throughput, scalability]
**Security Requirements:** [Authentication, authorization, data protection]

## User Experience
**User Personas:** [Which persona(s) interact with this feature]
**User Actions:** [Key workflows and interactions]
**UI Components:** 
- [Existing component: cite source] [Source: src/components/path/Component.tsx:L##] OR
- [Proposed: New components to be created]
**Mobile Considerations:** [For Guard persona features - offline, geolocation, etc.]

## BDD Scenarios
[Source: mvp-prd.md:L##-L## - BDD scenarios for this feature]
[Include Gherkin scenarios from PRD if provided for this feature]

## Success Criteria
[Source: mvp-prd.md:L##-L## - Success criteria from PRD]
**Functional:** [Feature works as specified in PRD]
**Performance:** [Meets timing/resource requirements]
**Quality:** [Reliability, accuracy, usability measures]
**Integration:** [Works seamlessly with other features]
**Business Value:** [Measurable impact on operations]

## Testing Strategy
**Unit Testing:** [Component-level test requirements]
**Integration Testing:** [Cross-feature test scenarios]
**User Acceptance:** [Business validation criteria]
**Performance Testing:** [Load and stress test requirements if applicable]

## Tasks Overview
[Brief list of 3+ tasks that will implement this feature]

## Implementation Notes
[Any special considerations, architectural decisions, or constraints]

## Citation Verification
- [ ] All PRD content includes source line numbers
- [ ] Existing code references cite actual file paths and lines
- [ ] New functionality clearly marked as "Proposed:"
- [ ] Integration points cite existing interfaces or mark as new
</content>
</write_to_file>
```

### Step 10: Validate Feature Completeness

Verify all required content was extracted from PRD.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating feature completeness: 1) Feature uses exact PRD feature ID, 2) Complete IAOOI framework included, 3) BDD scenarios present if in PRD, 4) Operational impact documented, 5) Integration points identified, 6) Feature is independently testable, 7) Sufficient context for AI implementation without reading full PRD.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 5,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 11: Document Cross-Persona Integration

For features spanning multiple personas, ensure integration points are clear.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Documenting cross-persona integration for features like: Shift Marketplace (Ops Manager creates, Guard claims), Incident Management (Guard submits, Ops Manager reviews), Welfare Checks (Guard responds, Ops Manager monitors), Communications (Ops Manager broadcasts, Guard receives), Time & Attendance (Guard clocks, Ops Manager approves, Admin exports). Ensuring both sides of integration documented.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 6,
  "totalThoughts": 12
}
</arguments>
</use_mcp_tool>
```

### Step 12: User Confirmation - Final Validation

Confirm all features have been extracted correctly.

```xml
<ask_followup_question>
<question>Feature extraction complete for this epic. Created the following feature files:

[List all feature.md files created with paths]

Please verify:
- All features from PRD are included
- IAOOI frameworks are complete
- BDD scenarios are present where applicable

How would you like to proceed?</question>
<options>["Mark extraction complete", "Review specific feature", "Add missing content", "Proceed to next epic"]</options>
</ask_followup_question>
```

## Feature Categories Reference

**Admin Persona Features:**
- Analytics & Dashboards
- Configuration & Rules Management
- Reporting & Exports
- Oversight & Monitoring Tools
- Approval & Review Interfaces

**Operations Manager Persona Features:**
- Interactive UI Components (Calendars, Boards)
- Workflow Management & Automation
- Approval & Review Systems
- Real-time Monitoring & Alerts
- Communication & Broadcasting Tools
- AI-Assisted Optimization
- Exception Handling & Corrections

**Guard Persona Features (Mobile-First):**
- Mobile Time Clock Components
- Schedule Viewing & Management
- Field Operations Tools (Incident, Patrol)
- Safety & Communication Features
- Offline-Capable Functionality
- Geolocation & Verification
- Media Capture & Upload

## Feature ID Naming Convention

Format: `FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN]`

Examples:
- `FEAT-ADMIN-SCHED-01-ANA-01`: Admin Scheduling Analytics feature
- `FEAT-OpsManager-SCHED-01-UI-01`: Ops Manager Scheduling UI feature
- `FEAT-Guard-TA-01-LOC-01`: Guard Time & Attendance Location feature

## Important Notes

**Do NOT:**
- Create features not explicitly listed in MVP PRD
- Include features from full OS platform not in MVP scope
- Include Client Persona features (excluded from MVP)
- Include features beyond Time & Attendance, Scheduling, and Guard Operations
- Skip IAOOI framework extraction - it's critical for context
- Omit BDD scenarios when they exist in PRD
- Create overly large features (>7 days implementation)
- Mix feature content from different epics
- Assume feature details - extract exactly from PRD
- Create feature.md files without PRD source citations (CRITICAL)
- Reference existing code without verifying and citing file paths
- Describe existing APIs or schemas without citing prisma/schema.prisma or route.ts files
- Invent function names, types, or file paths that may not exist

**DO:**
- Use exact PRD feature IDs (e.g., FEAT-OpsManager-SCHED-01-UI-01)
- Extract complete IAOOI frameworks verbatim from PRD
- Include all BDD/Gherkin scenarios exactly as written
- Document cross-persona integration points clearly
- Ensure each feature has sufficient context for standalone implementation
- Map features to specific R*.* requirements from PRD
- Size features appropriately (3-7 days AI implementation)
- Validate completeness against PRD source
- Create tasks/ subdirectory for each feature
- **CITE all PRD content with line numbers** [Source: mvp-prd.md:L##]
- **CITE existing code references** [Source: src/path/file.ts:L##]
- **CITE database schemas** [Source: prisma/schema.prisma:L## - ModelName]
- **Mark new functionality as "Proposed:"** when it doesn't exist yet
- **Verify existing code exists** before citing - use read_file or search_files
- Follow `.clinerules/cline_ai_citation_traceability_requirements.md` rules

## Example Invocations

### Example 1: Admin Persona Scheduling Epic
```
/prd-feature-extract.md
Extract features for: EPIC-ADMIN-SCHED-01
PRD Source: docs/_product_management_work_folder/mvp-prd.md
```

### Example 2: Operations Manager Scheduling Epic
```
/prd-feature-extract.md
Extract features for: EPIC-OpsManager-SCHED-01
```
Expected features:
- FEAT-OpsManager-SCHED-01-UI-01: Drag-and-Drop Calendar Interface
- FEAT-OpsManager-SCHED-01-RULE-02: Integrated Qualification & Compliance Checks
- FEAT-OpsManager-SCHED-01-OT-03: Overtime Visibility & Filtering

### Example 3: Guard Persona Time & Attendance Epic
```
/prd-feature-extract.md
Extract features for: EPIC-Guard-TA-01
```
Expected features:
- FEAT-Guard-TA-01-LOC-01: Geofenced Clock-In/Out with GPS Verification
- FEAT-Guard-TA-01-IVR-02: Phone/IVR Time Clock as Fallback
- FEAT-Guard-TA-01-OFF-03: Offline Clock-In/Out Support

## Success Criteria

The workflow is complete when:
- [ ] All features listed in PRD for the epic have been extracted
- [ ] Each feature uses exact PRD feature ID
- [ ] Each feature includes complete IAOOI framework from PRD
- [ ] BDD scenarios are included where provided in PRD
- [ ] Features are appropriately sized (3-7 day AI implementation)
- [ ] Operational impact is clear for each feature
- [ ] Feature dependencies are identified
- [ ] All features are independently testable
- [ ] Cross-persona integration points are documented
- [ ] Feature directories created with tasks/ subdirectory
- [ ] No PRD content is missing from feature documentation
- [ ] **CITATION: All PRD content includes [Source: mvp-prd.md:L##] citations**
- [ ] **CITATION: Existing code references verified and cited with file:line format**
- [ ] **CITATION: New functionality marked as "Proposed:" or "To be created:"**
- [ ] **CITATION: Integration points cite existing interfaces or marked as new**
- [ ] **CITATION: Citation Verification checklist completed in each feature.md**

For End Users: This workflow helps product managers and developers systematically break down high-level epic requirements into detailed, implementable features. Each feature file contains everything an AI needs to understand the requirement, including the complete IAOOI framework, BDD scenarios, and integration context, without needing to reference the full PRD document. All content is traceable back to the original PRD through explicit citations, ensuring accuracy and preventing AI hallucination.
