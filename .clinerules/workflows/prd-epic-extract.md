---
description: Extract a specific epic from the Product Requirements Document (PRD) and create comprehensive documentation structure for MVP development
applies_to: [".oxenated/docs/planning/**/*.md", "docs/_product_management_work_folder/**/*.md"]
priority: high
---

# PRD Epic Extraction Workflow

Extract a specific epic from the PRD, creating a complete documentation structure with all necessary context for future development.

## Usage

### When to Use This Workflow

Use this workflow when:
- You need to extract a single epic from the Product Requirements Document
- You are preparing epic documentation for the development planning phase
- You want to create a self-contained epic file that AI can use for feature extraction
- The PRD has been finalized and epics are ready for breakdown

### Prerequisites

- The PRD file exists at `.oxenated/docs/planning/product_requirements.md`
- The target epic ID is known (e.g., EPIC-ADMIN-SCHED-01)
- The epic exists in the PRD with documented features and IAOOI framework

### Expected Outcomes

- Epic directory created in the correct persona location
- Comprehensive `epic.md` file containing all PRD context
- Empty `features/` subdirectory ready for Step 2 workflow
- User confirmation of successful extraction

### Example Usage

```
Extract epic EPIC-ADMIN-SCHED-01 (Scheduling Analytics & Predictability) from the PRD
```

## Parameters

- **epic_id** (required): The unique identifier for the epic in format `EPIC-[PERSONA]-[DOMAIN]-[NUMBER]`
  - Valid personas: `ADMIN`, `OpsManager`, `Guard`
  - Valid domains: `SCHED`, `TA`, `OPS`
  - Examples: `EPIC-ADMIN-SCHED-01`, `EPIC-Guard-TA-01`, `EPIC-OpsManager-OPS-02`

- **epic_name** (optional): Human-readable epic name. If not provided, will be extracted from PRD.
  - Example: "Scheduling Analytics & Predictability"

## Overview

This workflow performs a single-purpose task: extracting one epic from the PRD and creating its documentation structure. The workflow reads the PRD, locates the specified epic, extracts all relevant information (description, IAOOI framework, features, BDD scenarios, requirements mapping), and creates a self-contained epic.md file that can be used independently for feature extraction in subsequent workflows.

## Detailed Sequence of Steps

### Step 1: Parse Epic ID from User Request

Identify from the user's request:
- The epic ID (e.g., `EPIC-ADMIN-SCHED-01`)
- The epic name if provided (e.g., "Scheduling Analytics & Predictability")
- The target persona derived from epic ID prefix:
  - `EPIC-ADMIN-*` → Admin Persona
  - `EPIC-OpsManager-*` → Operations Manager Persona
  - `EPIC-Guard-*` → Guard Persona

**Validation:** Confirm epic ID follows the naming convention `EPIC-[PERSONA]-[DOMAIN]-[NUMBER]`.

### Step 2: Read the PRD Document

Read the Product Requirements Document to access epic content:

```xml
<read_file>
<path>.oxenated/docs/planning/product_requirements.md</path>
</read_file>
```

**Expected outcome:** Full PRD content loaded for epic extraction.

### Step 3: Locate and Extract Epic Content

Search for the specified epic in the PRD and extract:

- Epic description and overview
- Complete IAOOI framework (Inputs → Activities → Outputs → Outcomes → Impacts)
- All features listed under the epic with their IDs
- Feature descriptions and specifications
- BDD scenarios for the target persona (Gherkin format)
- Requirements mapping (which R*.* requirements this addresses)
- User journeys relevant to the persona
- Technical considerations
- Success metrics
- Dependencies on other epics
- Integration points with other personas

**Validation:** Confirm all IAOOI components are present and complete.

### Step 4: Determine Output Directory

Based on epic ID prefix, determine the target directory:

| Epic Prefix | Persona Directory |
|-------------|------------------|
| `EPIC-ADMIN-*` | `.oxenated/docs/planning/admin/epics/` |
| `EPIC-OpsManager-*` | `.oxenated/docs/planning/operations/epics/` |
| `EPIC-Guard-*` | `.oxenated/docs/planning/guard/epics/` |

**Expected structure:**
```
.oxenated/docs/planning/[persona]/epics/[epic-id]/
├── epic.md                    # Epic definition with full context
└── features/                  # Empty, populated by prd-feature-extract workflow
```

### Step 5: Create Epic Documentation

Create the epic directory and `epic.md` file using the Epic Template.

**IMPORTANT - Citation Requirements:**
Per `.clinerules/cline_ai_citation_traceability_requirements.md`, the generated `epic.md` file must follow citation rules:
- Content extracted from PRD: Use `[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]` to cite the specific PRD sections
- References to existing code/APIs: Use `[Source: path/to/file.ext:L##]` format - verify existence first with read_file
- References to existing database models: Use `[Source: prisma/schema.prisma:L## - ModelName]`
- New functionality (to be created): Mark as "Proposed:" or "To be implemented" - no citation needed
- If any reference cannot be verified, mark as `[UNVERIFIED - requires confirmation]`

```xml
<write_to_file>
<path>.oxenated/docs/planning/[persona]/epics/[epic-id]/epic.md</path>
<content>
# [Epic Name]

## Epic ID
[EPIC-XXX-XX]

## Source Reference
[Source: .oxenated/docs/planning/product_requirements.md:L##-L## - Epic Section]

## Target Persona
[Admin / Operations Manager / Guard]

## Epic Overview
[Complete description from PRD including context and purpose]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Vision & Objectives
[Extract from PRD - complete value chain from inputs to impacts]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## IAOOI System Components

### Inputs
[List of inputs from PRD - data, events, triggers that initiate this epic's functionality]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

### Activities
[List of activities/processing from PRD - core business logic and workflows]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

### Outputs
[List of immediate outputs from PRD - direct results of the activities]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

### Outcomes
[List of medium-term outcomes from PRD - business improvements achieved]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

### Impacts
[List of long-term impacts from PRD - strategic value created]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Key Features
[List of features from PRD with their IDs]
- FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN]: [Feature Name]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Business Value & Requirements
[Direct copy from PRD showing which original requirements (R*.*) this addresses]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## User Journeys & Scenarios
[Relevant user journey from PRD for target persona]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## BDD Scenarios
[Include Gherkin scenarios from PRD if provided]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Technical Considerations
[Any relevant architecture or technical notes for this epic]

**Existing Code References (if any):**
[If referencing existing APIs, models, or components - cite with verification]
- Example: Extends authentication flow [Source: src/app/api/auth/route.ts:L##]
- Example: Uses Shift model [Source: prisma/schema.prisma:L## - Shift]

**Proposed New Components:**
[List new code to be created - no citation needed, mark as "Proposed:"]

## Implementation Priority
[Based on PRD dependencies and MVP sequencing]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Success Metrics
[Measurable outcomes from PRD]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Dependencies
[Other epics/features this depends on]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Integration Points
[How this epic integrates with other personas/epics]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]
</content>
</write_to_file>
```

### Step 6: Create Features Directory

Create the empty features subdirectory for subsequent feature extraction:

```xml
<execute_command>
<command>mkdir -p .oxenated/docs/planning/[persona]/epics/[epic-id]/features</command>
</execute_command>
```

### Step 7: Validate and Confirm with User

After creating the epic documentation, present a summary and request confirmation:

```xml
<ask_followup_question>
<question>I've extracted epic [EPIC-ID] ([Epic Name]) from the PRD and created the documentation structure.

**Summary:**
- **Epic Location:** .oxenated/docs/planning/[persona]/epics/[epic-id]/epic.md
- **Target Persona:** [Admin/Operations Manager/Guard]
- **Features Identified:** [Count] features
- **IAOOI Framework:** Complete
- **BDD Scenarios:** [Present/Not found in PRD]
- **Dependencies:** [List key dependencies]

Would you like me to:
1. Proceed to extract features for this epic using /prd-feature-extract.md
2. Extract a different epic from the PRD
3. Review the epic documentation first</question>
<options>["Proceed to extract features", "Extract a different epic", "Let me review first"]</options>
</ask_followup_question>
```

## Important Notes

### DO

- Extract ALL content from PRD - do not summarize or abbreviate
- Include complete IAOOI framework with all five components
- Preserve exact feature IDs from PRD (e.g., FEAT-ADMIN-SCHED-01-ANA-01)
- Include BDD scenarios exactly as written in PRD
- Document all integration points with other personas
- Verify epic ID follows naming convention before proceeding
- Create self-contained documentation that AI can use without reading full PRD
- Add `[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]` citations for all PRD-extracted content
- Verify any referenced existing code/APIs/models with read_file before citing
- Mark unverified references as `[UNVERIFIED - requires confirmation]`

### DO NOT

- Do NOT invent features not in the PRD
- Do NOT modify feature IDs or naming conventions
- Do NOT skip IAOOI components even if they seem obvious
- Do NOT omit BDD scenarios if they exist in PRD
- Do NOT create epic documentation for features outside MVP scope
- Do NOT include Client Persona epics (excluded from MVP)
- Do NOT include speculative features not documented in MVP PRD

## Example Invocations

### Example 1: Admin Persona Scheduling Epic

```
Extract epic EPIC-ADMIN-SCHED-01 (Scheduling Analytics & Predictability) from the PRD
```

This extracts the admin scheduling analytics epic and creates documentation at `.oxenated/docs/planning/admin/epics/EPIC-ADMIN-SCHED-01/`.

### Example 2: Guard Persona Time & Attendance Epic

```
Extract the Guard Mobile Time & Attendance epic (EPIC-Guard-TA-01) from the PRD
```

This extracts the guard time & attendance epic with all mobile-first features and creates documentation at `.oxenated/docs/planning/guard/epics/EPIC-Guard-TA-01/`.

### Example 3: Operations Manager Scheduling Epic

```
Extract epic EPIC-OpsManager-SCHED-01 from the PRD for feature breakdown
```

This extracts the operations manager scheduling epic with all scheduling management features and creates documentation at `.oxenated/docs/planning/operations/epics/EPIC-OpsManager-SCHED-01/`.

### Example 4: Operations Manager Guard Operations Epic

```
Extract EPIC-OpsManager-OPS-02 (Incident & Exception Management) from the product requirements
```

This extracts the incident management epic for operations managers and creates documentation at `.oxenated/docs/planning/operations/epics/EPIC-OpsManager-OPS-02/`.

## Success Criteria

- [ ] Epic ID correctly parsed and validated against naming convention
- [ ] PRD successfully read and epic content located
- [ ] Epic directory created in correct persona location (admin/operations/guard)
- [ ] `epic.md` file created with all required sections
- [ ] Complete IAOOI framework included (all 5 components)
- [ ] All features listed with exact PRD feature IDs
- [ ] BDD scenarios included where provided in PRD
- [ ] Requirements mapping (R*.*) documented
- [ ] Integration points with other personas identified
- [ ] Dependencies on other epics documented
- [ ] All PRD-extracted content has `[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]` citations
- [ ] Any existing code references verified and cited with line numbers
- [ ] Empty `features/` directory created for next workflow step
- [ ] User confirmation received with clear next step options

## Error Handling

### Epic Not Found in PRD

If the specified epic ID is not found in the PRD:
1. Search for similar epic IDs using partial matching
2. Report available epics matching the persona and domain
3. Ask user to verify the correct epic ID

### Incomplete IAOOI Framework

If any IAOOI component is missing from PRD:
1. Document what was found
2. Mark missing components as "[Not documented in PRD]"
3. Notify user of incomplete extraction
4. Proceed with available content

### Invalid Epic ID Format

If epic ID doesn't match `EPIC-[PERSONA]-[DOMAIN]-[NUMBER]`:
1. Report the format error
2. Show valid format examples
3. Ask user to provide corrected epic ID

## Related Workflows

- **Next Step:** `/prd-feature-extract.md` - Extract features from the epic
- **Validation:** `/prd-epic-peer-review.md` - Review epic documentation quality
- **Prerequisite:** `/prd-create-or-update.md` - Create or update the PRD

## Epic Naming Convention Reference

| Component | Values | Description |
|-----------|--------|-------------|
| Prefix | `EPIC` | Always "EPIC" |
| Persona | `ADMIN`, `OpsManager`, `Guard` | Target persona user |
| Domain | `SCHED`, `TA`, `OPS` | Feature domain |
| Number | `01`, `02`, `03`... | Sequential within persona-domain |

**Examples:**
- `EPIC-ADMIN-SCHED-01` - Admin Persona Scheduling Epic #1
- `EPIC-OpsManager-TA-01` - Operations Manager Time & Attendance Epic #1
- `EPIC-Guard-OPS-03` - Guard Persona Operations Epic #3

## Directory Structure Reference

```
.oxenated/docs/planning/
├── admin/
│   └── epics/
│       ├── EPIC-ADMIN-SCHED-01/
│       ├── EPIC-ADMIN-TA-01/
│       └── EPIC-ADMIN-OPS-01/
├── operations/
│   └── epics/
│       ├── EPIC-OpsManager-SCHED-01/
│       ├── EPIC-OpsManager-SCHED-02/
│       ├── EPIC-OpsManager-TA-01/
│       └── EPIC-OpsManager-OPS-01/
└── guard/
    └── epics/
        ├── EPIC-Guard-TA-01/
        ├── EPIC-Guard-SCHED-01/
        └── EPIC-Guard-OPS-01/
```
