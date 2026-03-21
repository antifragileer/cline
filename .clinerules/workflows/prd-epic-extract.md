---
description: Extract ALL epics from the Product Requirements Document (PRD) using subagents, creating comprehensive documentation structure for MVP development
applies_to: [".oxenated/docs/planning/**/*.md", "docs/_product_management_work_folder/**/*.md"]
priority: high
---

# PRD Epic Extraction Workflow

Use subagents to extract ALL epics from the PRD in parallel, creating complete documentation structures with all necessary context for future development. By default, extracts all epics found in the PRD. Can optionally filter to specific epics.

## Usage

### When to Use This Workflow

Use this workflow when:
- You need to extract all epics from the Product Requirements Document
- You are preparing epic documentation for the development planning phase
- You want to create self-contained epic files that AI can use for feature extraction
- The PRD has been finalized and epics are ready for breakdown
- You want to extract multiple or all epics in parallel for efficiency

### Prerequisites

- The PRD file exists at `.oxenated/docs/planning/product_requirements.md`
- The PRD contains epics with documented features and IAOOI framework

### Expected Outcomes

- All epic directories created in the correct persona locations
- Comprehensive `epic.md` files containing all PRD context for each epic
- Empty `features/` subdirectories ready for Step 2 workflow
- Summary report of all extracted epics
- User confirmation of successful extraction

### Example Usage

```
Extract all epics from the PRD
```

```
Extract epics from the PRD for the Admin persona only
```

```
Extract only EPIC-ADMIN-SCHED-01 and EPIC-Guard-TA-01 from the PRD
```

## Parameters

- **filter** (optional): Filter criteria for which epics to extract. If not provided, extracts ALL epics.
  - **epic_ids** (optional): Array of specific epic IDs to extract (e.g., `["EPIC-ADMIN-SCHED-01", "EPIC-Guard-TA-01"]`)
  - **persona** (optional): Filter by persona - `ADMIN`, `OpsManager`, `Guard`
  - **domain** (optional): Filter by domain - `SCHED`, `TA`, `OPS`

- **epic_id** (legacy, optional): Single epic ID for backward compatibility (e.g., `EPIC-ADMIN-SCHED-01`)

## Overview

This workflow extracts epics from the PRD and creates their documentation structures. By default, it extracts ALL epics found in the PRD using parallel subagents for efficiency. Each subagent handles one epic extraction, following the detailed template to create a self-contained epic.md file.

The workflow:
1. Reads the PRD and identifies all epics matching the filter criteria
2. Launches subagents in parallel (one per epic)
3. Each subagent extracts one epic with complete IAOOI framework, features, BDD scenarios, and requirements mapping
4. Validates all extractions completed successfully
5. Reports summary to user

## Detailed Sequence of Steps

### Step 1: Read PRD and Identify All Epics

Read the Product Requirements Document to access all epic content:

```xml
<read_file>
<path>.oxenated/docs/planning/product_requirements.md</path>
</read_file>
```

Parse the PRD to identify all epics:
- Search for all epic IDs matching pattern `EPIC-[PERSONA]-[DOMAIN]-[NUMBER]`
- Extract epic names/titles
- Determine target persona for each epic
- Apply filters if specified (by epic_id, persona, or domain)

**Validation:** Confirm at least one epic is found. If filtering, confirm matching epics exist.

**Expected outcome:** List of all epics to extract with their IDs, names, and target personas.

### Step 2: Prepare Epic Extraction Tasks

For each identified epic, prepare the extraction context:

| Epic ID | Persona | Target Directory |
|---------|---------|------------------|
| `EPIC-ADMIN-*` | Admin | `.oxenated/docs/planning/admin/epics/[epic-id]/` |
| `EPIC-OpsManager-*` | Operations Manager | `.oxenated/docs/planning/operations/epics/[epic-id]/` |
| `EPIC-Guard-*` | Guard | `.oxenated/docs/planning/guard/epics/[epic-id]/` |

Create the base directory structure for each persona if it doesn't exist.

### Step 3: Launch Subagents for Parallel Extraction

Launch one subagent per epic to extract in parallel. Each subagent receives:

- The epic ID to extract
- The target persona
- The output directory path
- The PRD file path for citation references

**Subagent Instructions (provided to each subagent):**

```
You are an epic extraction specialist. Your task is to extract ONE specific epic from the PRD and create comprehensive documentation.

EPIC TO EXTRACT: [EPIC-ID]
TARGET PERSONA: [Persona Name]
OUTPUT DIRECTORY: [path]
PRD LOCATION: .oxenated/docs/planning/product_requirements.md

Follow these steps:

1. Read the PRD file at .oxenated/docs/planning/product_requirements.md

2. Locate the specific epic [EPIC-ID] and extract ALL of the following:
   - Epic description and overview
   - Complete IAOOI framework (Inputs → Activities → Outputs → Outcomes → Impacts)
   - All features listed under this epic with their exact IDs
   - Feature descriptions and specifications
   - BDD scenarios for the target persona (Gherkin format)
   - Requirements mapping (which R*.* requirements this addresses)
   - User journeys relevant to the persona
   - Technical considerations
   - Success metrics
   - Dependencies on other epics
   - Integration points with other personas

3. Create the epic directory if it doesn't exist:
   mkdir -p [OUTPUT DIRECTORY]/features

4. Create the epic.md file using this EXACT template. DO NOT deviate from this structure:

---

# [Epic Name - extract from PRD]

## Epic ID
[EPIC-ID]

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
[List of features from PRD with their exact IDs - preserve PRD format exactly]
- FEAT-[PERSONA]-[DOMAIN]-[NN]-[CODE]-[NN]: [Feature Name]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## Business Value & Requirements
[Direct copy from PRD showing which original requirements (R*.*) this addresses]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## User Journeys & Scenarios
[Relevant user journey from PRD for target persona]
[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]

## BDD Scenarios
[Include Gherkin scenarios from PRD if provided - exact copy]
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

---

IMPORTANT CITATION RULES:
- Content extracted from PRD: Use [Source: .oxenated/docs/planning/product_requirements.md:L##-L##] to cite the specific PRD sections
- References to existing code/APIs: Use [Source: path/to/file.ext:L##] format - verify existence first with read_file
- References to existing database models: Use [Source: prisma/schema.prisma:L## - ModelName]
- New functionality (to be created): Mark as "Proposed:" or "To be implemented" - no citation needed
- If any reference cannot be verified, mark as [UNVERIFIED - requires confirmation]

5. Report back with:
   - Confirmation of successful extraction
   - Path to created epic.md file
   - Count of features extracted
   - Any issues or missing content noted

DO NOT:
- Do NOT invent features not in the PRD
- Do NOT modify feature IDs or naming conventions
- Do NOT skip IAOOI components even if they seem obvious
- Do NOT omit BDD scenarios if they exist in PRD
- Do NOT create epic documentation for features outside MVP scope
- Do NOT include Client Persona epics (excluded from MVP)
```

Launch subagents using the use_subagents tool:

```xml
<use_subagents>
<prompt_1>[Subagent instructions for EPIC-ADMIN-SCHED-01]</prompt_1>
<prompt_2>[Subagent instructions for EPIC-ADMIN-TA-01]</prompt_2>
<prompt_3>[Subagent instructions for next epic...]</prompt_3>
...
</use_subagents>
```

**Note:** Launch up to 5 subagents at a time. If more than 5 epics, process in batches.

### Step 4: Validate Extractions

After all subagents complete:

1. Verify each epic.md file was created successfully
2. Check that all features/ subdirectories exist
3. Validate citation format consistency
4. Confirm no epics were missed

List all created epic directories to verify:

```xml
<list_files>
<path>.oxenated/docs/planning</path>
<recursive>true</recursive>
</list_files>
```

### Step 5: Report Results and Confirm with User

Present a summary of all extracted epics:

```xml
<ask_followup_question>
<question>I've extracted [N] epics from the PRD using parallel subagents.

**Extraction Summary:**

| Epic ID | Persona | Features | Location |
|---------|---------|----------|----------|
| [EPIC-ID-1] | [Persona] | [Count] | `.oxenated/docs/planning/[persona]/epics/[epic-id]/` |
| [EPIC-ID-2] | [Persona] | [Count] | `.oxenated/docs/planning/[persona]/epics/[epic-id]/` |
| ... | ... | ... | ... |

**By Persona:**
- **Admin:** [N] epics, [N] total features
- **Operations Manager:** [N] epics, [N] total features  
- **Guard:** [N] epics, [N] total features

**All epic directories include:**
- Complete IAOOI framework documentation
- All features with exact PRD IDs
- BDD scenarios (where provided)
- Requirements mapping (R*.*)
- Integration points with other personas
- Empty `features/` subdirectory for next workflow step

Would you like me to:
1. Proceed to extract features for all epics using /prd-feature-extract.md
2. Extract features for a specific epic only
3. Review a specific epic documentation first</question>
<options>["Extract features for all epics", "Extract features for specific epic", "Let me review first"]</options>
</ask_followup_question>
```

## Important Notes

### DO

- Extract ALL epics by default unless filtered
- Use subagents to process epics in parallel for efficiency
- Ensure each subagent follows the exact epic.md template provided
- Extract ALL content from PRD for each epic - do not summarize or abbreviate
- Include complete IAOOI framework with all five components for each epic
- Preserve exact feature IDs from PRD (e.g., FEAT-ADMIN-SCHED-01-ANA-01)
- Include BDD scenarios exactly as written in PRD
- Document all integration points with other personas
- Create self-contained documentation that AI can use without reading full PRD
- Add `[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]` citations for all PRD-extracted content
- Verify any referenced existing code/APIs/models with read_file before citing
- Mark unverified references as `[UNVERIFIED - requires confirmation]`
- Process up to 5 epics in parallel per batch

### DO NOT

- Do NOT extract epics sequentially - always use subagents for parallel processing
- Do NOT invent features not in the PRD
- Do NOT modify feature IDs or naming conventions
- Do NOT skip IAOOI components even if they seem obvious
- Do NOT omit BDD scenarios if they exist in PRD
- Do NOT create epic documentation for features outside MVP scope
- Do NOT include Client Persona epics (excluded from MVP)
- Do NOT include speculative features not documented in MVP PRD

## Example Invocations

### Example 1: Extract All Epics (Default)

```
Extract all epics from the PRD
```

This extracts ALL epics found in the PRD and creates documentation for each in parallel using subagents.

### Example 2: Filter by Persona

```
Extract all Admin persona epics from the PRD
```

This extracts only epics with IDs matching `EPIC-ADMIN-*`.

### Example 3: Filter by Specific Epics

```
Extract only EPIC-ADMIN-SCHED-01 and EPIC-Guard-TA-01 from the PRD
```

This extracts only the two specified epics.

### Example 4: Extract All Operations Manager Epics

```
Extract all Operations Manager epics from the PRD
```

This extracts all `EPIC-OpsManager-*` epics in parallel.

## Success Criteria

- [ ] PRD successfully read and all epics identified
- [ ] Filters applied correctly (if specified)
- [ ] Subagents launched in parallel for each epic (up to 5 at a time)
- [ ] Each subagent created epic.md following the exact template structure
- [ ] All epic directories created in correct persona locations (admin/operations/guard)
- [ ] Complete IAOOI framework included for each epic (all 5 components)
- [ ] All features listed with exact PRD feature IDs
- [ ] BDD scenarios included where provided in PRD
- [ ] Requirements mapping (R*.*) documented for each epic
- [ ] Integration points with other personas identified
- [ ] Dependencies on other epics documented
- [ ] All PRD-extracted content has `[Source: .oxenated/docs/planning/product_requirements.md:L##-L##]` citations
- [ ] Any existing code references verified and cited with line numbers
- [ ] Empty `features/` directory created for each epic
- [ ] Summary report presented to user with clear next step options

## Error Handling

### No Epics Found in PRD

If no epics are found in the PRD:
1. Report that no epics matching the pattern were found
2. Show a sample of what epic IDs should look like
3. Ask user to verify the PRD contains properly formatted epics

### Filter Returns No Matches

If filters are specified but no epics match:
1. Report available epics in the PRD
2. Show which filters were applied
3. Ask user to adjust filters or extract all epics

### Subagent Failure

If a subagent fails to extract an epic:
1. Report which epic failed
2. Capture the error message
3. Retry the failed epic extraction with a new subagent
4. If repeated failures, report to user and suggest manual review

### Incomplete IAOOI Framework

If any IAOOI component is missing from PRD for a specific epic:
1. Subagent should document what was found
2. Mark missing components as "[Not documented in PRD]"
3. Notify user in the final summary of any incomplete extractions
4. Proceed with available content

## Related Workflows

- **Next Step:** `/prd-feature-extract.md` - Extract features from the epics
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
│       │   ├── epic.md
│       │   └── features/
│       ├── EPIC-ADMIN-TA-01/
│       │   ├── epic.md
│       │   └── features/
│       └── EPIC-ADMIN-OPS-01/
│           ├── epic.md
│           └── features/
├── operations/
│   └── epics/
│       ├── EPIC-OpsManager-SCHED-01/
│       │   ├── epic.md
│       │   └── features/
│       ├── EPIC-OpsManager-SCHED-02/
│       │   ├── epic.md
│       │   └── features/
│       ├── EPIC-OpsManager-TA-01/
│       │   ├── epic.md
│       │   └── features/
│       └── EPIC-OpsManager-OPS-01/
│           ├── epic.md
│           └── features/
└── guard/
    └── epics/
        ├── EPIC-Guard-TA-01/
        │   ├── epic.md
        │   └── features/
        ├── EPIC-Guard-SCHED-01/
        │   ├── epic.md
        │   └── features/
        └── EPIC-Guard-OPS-01/
            ├── epic.md
            └── features/