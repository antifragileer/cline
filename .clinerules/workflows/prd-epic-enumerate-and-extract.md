---
description: Enumerate all epics in a PRD file and spawn dependency-ordered subagents to extract each epic using the prd-epic-extract workflow
applies_to: [".oxenated/docs/planning/**/*.md", "docs/_product_management_work_folder/**/*.md", "**/*prd*.md"]
priority: high
---

# PRD Epic Enumeration and Extraction Workflow

Enumerate all epics from a Product Requirements Document (PRD), analyze their dependencies, and spawn subagents in topological dependency order to extract each epic using the `/prd-epic-extract.md` workflow.

## Usage

### When to Use This Workflow

Use this workflow when:
- You need to extract ALL epics from a PRD file in one operation
- You want to parallelize epic extraction while respecting dependency constraints
- Epics have explicit dependencies that must be extracted in order
- You're bootstrapping a new project and need to extract the complete epic hierarchy
- You want to ensure dependent epics are created before their dependents

### Prerequisites

- The PRD file exists and contains epic definitions with dependency information
- Epics follow the standard naming convention: `EPIC-[PERSONA]-[DOMAIN]-[NUMBER]`
- Dependencies are documented within each epic section
- The `/prd-epic-extract.md` workflow is available in `.clinerules/workflows/`

### Expected Outcomes

- All epics extracted from the PRD in dependency order
- Parallel execution where dependencies allow
- Complete epic documentation structure created for each epic
- Summary report of all extracted epics and their status
- Failed extractions reported for retry or manual intervention

### Example Usage

```
Enumerate and extract all epics from .oxenated/docs/planning/cline-cli-golang-migration-prd.md
```

## Parameters

- **prd_path** (required): Absolute or relative path to the PRD markdown file
  - Example: `.oxenated/docs/planning/product_requirements.md`
  - Example: `docs/_product_management_work_folder/migration-prd.md`

- **max_parallel** (optional): Maximum number of subagents to spawn in parallel per dependency batch
  - Default: 5 (also the maximum allowed)
  - Minimum: 1
  - Note: Cline supports up to 5 parallel subagents via `use_subagents`

## Overview

This workflow automates the bulk extraction of epics from a PRD by:

1. **Parsing**: Reading the PRD and identifying all epic definitions
2. **Dependency Analysis**: Extracting dependency relationships between epics
3. **Topological Sorting**: Ordering epics so dependencies are processed before dependents
4. **Batching**: Grouping epics into parallelizable batches based on dependency levels
5. **Subagent Spawning**: Using `use_subagents` to execute epic extractions in parallel within each batch
6. **Progress Tracking**: Monitoring completion status and handling failures
7. **Reporting**: Providing a comprehensive summary of extraction results

The workflow respects the constraint that an epic must be fully extracted before any epic that depends on it can be processed, while maximizing parallelism where dependencies allow.

## Detailed Sequence of Steps

### Step 1: Parse PRD Path and Validate Input

Extract the PRD file path from the user's request and validate it exists.

```xml
<read_file>
<path>{prd_path}</path>
</read_file>
```

**Validation:**
- Confirm the file exists and is readable
- Verify it contains epic definitions (search for `EPIC-` patterns)
- Confirm at least one epic is present

**Error Handling:**
If the PRD file doesn't exist, report the error and suggest checking the path.

### Step 2: Enumerate All Epics in the PRD

Parse the PRD content to identify all epic definitions and extract their metadata.

```xml
<search_files>
<path>.</path>
<regex>^#{1,3}\s+.*EPIC-[A-Z]+-[A-Z]+-\d+|^#{1,3}\s+EPIC-[A-Z]+-[A-Z]+-\d+</regex>
<file_pattern>*</file_pattern>
</search_files>
```

**Extract for each epic:**
- Epic ID (e.g., `EPIC-ADMIN-SCHED-01`)
- Epic name/title
- Persona (from ID: ADMIN, OpsManager, Guard)
- Domain (from ID: SCHED, TA, OPS)
- Dependencies listed in the epic section
- Priority or sequence hints if present

**Output:** List of all epics with their dependency information.

### Step 3: Build Dependency Graph

Analyze the extracted epic data to build a dependency graph.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Building dependency graph from extracted epics. For each epic, identify its dependencies by parsing the Dependencies section. Create a directed graph where edges point from dependency to dependent (A -> B means B depends on A). Track epics with no dependencies as roots. Detect any cycles in the graph. Prepare for topological sorting to determine execution order.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 1,
  "totalThoughts": 5
}
</arguments>
</use_mcp_tool>
```

**Dependency Parsing:**
- Look for "Dependencies" sections in each epic
- Extract referenced epic IDs (e.g., "Depends on: EPIC-ADMIN-SCHED-01")
- Handle multiple dependencies per epic
- Identify epics with no dependencies (root nodes)

**Cycle Detection:**
If a circular dependency is detected, report it and request manual resolution before proceeding.

### Step 4: Perform Topological Sort

Order epics so all dependencies are processed before their dependents.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Performing topological sort on the dependency graph. Using Kahn's algorithm: start with all root nodes (no dependencies), process them in parallel batches. After each batch completes, remove those nodes from the graph and identify new nodes that now have all dependencies satisfied. Continue until all nodes processed. If nodes remain with unmet dependencies, there's a cycle or missing dependency. Track the batch number for each epic to maximize parallelism.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 2,
  "totalThoughts": 5
}
</arguments>
</use_mcp_tool>
```

**Algorithm:**
1. Initialize batch 0 with all epics having no dependencies
2. For each batch, spawn subagents to extract those epics
3. After batch completes, identify epics whose dependencies are all satisfied
4. Add those epics to the next batch
5. Repeat until all epics are assigned to a batch

**Output:** List of batches, where each batch contains epics that can be processed in parallel.

### Step 5: Present Execution Plan to User

Before spawning subagents, present the analysis and execution plan for confirmation.

```xml
<ask_followup_question>
<question>I've analyzed the PRD and prepared an execution plan for epic extraction.

**PRD File:** {prd_path}

**Total Epics Found:** {count}

**Dependency Analysis:**
- Root epics (no dependencies): {list}
- Epics with dependencies: {count}
- Execution batches: {number_of_batches}

**Execution Order (by batch):**
{batch_1_list}
{batch_2_list}
...

**Estimated Parallelism:**
- Batch 1: {count} epics in parallel
- Batch 2: {count} epics in parallel
...

Would you like me to proceed with extraction, or would you prefer to:</question>
<options>["Proceed with full extraction", "Review dependency details first", "Extract specific batch only", "Cancel and adjust PRD dependencies"]</options>
</ask_followup_question>
```

### Step 6: Execute Batch 1 - Root Epics (No Dependencies)

Spawn subagents to extract all epics with no dependencies in parallel.

```xml
<use_subagents>
<prompt_1>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID-1}</prompt_1>
<prompt_2>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID-2}</prompt_2>
<prompt_3>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID-3}</prompt_3>
<prompt_4>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID-4}</prompt_4>
<prompt_5>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID-5}</prompt_5>
</use_subagents>
```

**Note:** Include up to 5 prompts (the maximum). If more than 5 root epics exist, execute multiple `use_subagents` calls sequentially, or wait for completion and then spawn the next set.

**Progress Tracking:**
- Record successful extractions
- Note any failures with error details
- Track which epics are complete before proceeding to next batch

### Step 7: Validate Batch 1 Completion and Identify Next Batch

Confirm Batch 1 epics are extracted and identify epics ready for Batch 2.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Validating Batch 1 completion. Checking which epics were successfully extracted. For each epic in Batch 1, verify the epic directory and epic.md file were created. If any failed, note them for retry. Now identifying Batch 2 epics - those whose dependencies are all in Batch 1 and completed successfully. If a Batch 2 epic depends on a failed Batch 1 epic, flag it as blocked.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 3,
  "totalThoughts": 5
}
</arguments>
</use_subagents>
```

**Validation:**
- Check epic directories were created: `.oxenated/docs/planning/[persona]/epics/[epic-id]/`
- Verify `epic.md` files exist in each directory
- Identify any failures for retry or reporting

**Blocked Epic Handling:**
If an epic's dependency failed extraction, mark it as blocked and report to user.

### Step 8: Execute Subsequent Batches

Repeat the subagent spawning process for each subsequent batch.

```xml
<use_subagents>
<prompt_1>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {BATCH_N_EPIC_1}</prompt_1>
<prompt_2>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {BATCH_N_EPIC_2}</prompt_2>
<prompt_3>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {BATCH_N_EPIC_3}</prompt_3>
<prompt_4>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {BATCH_N_EPIC_4}</prompt_4>
<prompt_5>/prd-epic-extract.md PRD: `{prd_path}` EPIC: {BATCH_N_EPIC_5}</prompt_5>
</use_subagents>
```

**Continue for each batch:**
- Batch 2: Epics depending only on Batch 1
- Batch 3: Epics depending on Batch 1 and/or 2
- And so on...

**Between batches:**
- Validate previous batch completion
- Report any failures
- Identify blocked epics (those with failed dependencies)

### Step 9: Handle Failures and Retries

If any epic extraction fails, decide whether to retry or report.

```xml
<ask_followup_question>
<question>Some epic extractions encountered issues:

**Failed Epics:**
{list of failed epics with error details}

**Blocked Epics (depend on failed epics):**
{list of blocked epics}

**Successful Extractions:**
{count} of {total} epics completed

How would you like to proceed?</question>
<options>["Retry failed epics", "Skip failed epics and continue with unblocked", "Review failed epic details first", "Cancel remaining extractions", "Mark failed epics as manual extraction needed"]</options>
</ask_followup_question>
```

**Retry Logic:**
- For transient failures (file system, permissions), retry immediately
- For PRD parsing errors, report to user for PRD fixes
- Allow up to 2 retry attempts per epic

### Step 10: Final Validation and Summary Report

After all batches complete, generate a comprehensive summary.

```xml
<list_files>
<path>.oxenated/docs/planning</path>
<recursive>true</recursive>
</list_files>
```

**Summary Report Includes:**
- Total epics found in PRD: {count}
- Successfully extracted: {count}
- Failed extractions: {count} (with details)
- Blocked epics (unextracted due to failed dependencies): {count}
- Execution time and parallelism statistics
- Directory structure created
- Next steps recommendation

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Generating final summary. Compiling statistics: total epics, success rate, failures by batch, blocked epics. Verifying directory structure matches expected output. Checking that all epic.md files were created with proper citations. Preparing recommendations for next steps - whether to proceed with feature extraction, retry failures, or investigate issues.",
  "nextThoughtNeeded": true,
  "thoughtNumber": 4,
  "totalThoughts": 5
}
</arguments>
</use_mcp_tool>
```

### Step 11: Present Final Results to User

Display the comprehensive extraction report.

```xml
<ask_followup_question>
<question>## PRD Epic Extraction Complete

**PRD Source:** {prd_path}

### Extraction Summary
| Metric | Count |
|--------|-------|
| Total Epics Found | {total} |
| Successfully Extracted | {success} |
| Failed | {failed} |
| Blocked (dependency failures) | {blocked} |

### Batch Execution Details
- **Batch 1 (Root Epics):** {batch1_success}/{batch1_total} completed
- **Batch 2:** {batch2_success}/{batch2_total} completed
- **...**

### Created Directory Structure
```
.oxenated/docs/planning/
├── admin/epics/
│   ├── EPIC-ADMIN-XXX-XX/...
├── operations/epics/
│   ├── EPIC-OpsManager-XXX-XX/...
└── guard/epics/
    ├── EPIC-Guard-XXX-XX/...
```

### Failed Extractions (if any)
{details or "None - all epics extracted successfully"}

### Next Steps
All epics are now ready for feature extraction using `/prd-feature-extract.md`.

What would you like to do next?</question>
<options>["Proceed to feature extraction for all epics", "Review specific epic documentation", "Retry any failed extractions", "Export extraction report", "Complete workflow"]</options>
</ask_followup_question>
```

### Step 12: Workflow Completion

Mark workflow as complete after user confirmation.

```xml
<use_mcp_tool>
<server_name>github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking</server_name>
<tool_name>sequentialthinking</tool_name>
<arguments>
{
  "thought": "Workflow complete. All epics have been enumerated from the PRD, dependencies analyzed, and subagents spawned in topological order to extract each epic. Successfully extracted {success} of {total} epics. Directory structure created and ready for feature extraction workflow. User has been presented with summary and next step options.",
  "nextThoughtNeeded": false,
  "thoughtNumber": 5,
  "totalThoughts": 5
}
</arguments>
</use_mcp_tool>
```

## Important Notes

### DO

- Always validate the PRD file exists before attempting extraction
- Parse dependencies carefully to build accurate dependency graph
- Use topological sorting to ensure correct execution order
- Maximize parallelism within each dependency batch
- Verify each epic extraction succeeded before marking dependencies as satisfied
- Report blocked epics clearly when dependencies fail
- Allow user to review execution plan before spawning subagents
- Track and report detailed statistics for transparency
- Handle up to 5 parallel subagents per `use_subagents` call
- Retry transient failures before reporting permanent errors
- Verify created directory structure matches expected output

### DO NOT

- Do NOT spawn subagents without first analyzing dependencies
- Do NOT attempt to extract epics with unsatisfied dependencies
- Do NOT exceed 5 parallel subagents in a single `use_subagents` call
- Do NOT skip validation between batches - always confirm previous batch completion
- Do NOT ignore failed extractions - always report and handle them
- Do NOT assume dependency information is well-formed - validate epic IDs
- Do NOT proceed with blocked epics without user confirmation
- Do NOT modify the PRD file during extraction
- Do NOT create epics out of dependency order
- Do NOT hide failures or errors from the user

## Example Invocations

### Example 1: Extract All Epics from a Migration PRD

```
Enumerate and extract all epics from .oxenated/docs/planning/cline-cli-golang-migration-prd.md
```

This reads the migration PRD, finds all epics (e.g., EPIC-MIGRATION-SETUP-01, EPIC-MIGRATION-CORE-01), analyzes their dependencies, and spawns subagents to extract each one in dependency order.

### Example 2: Extract Epics with Limited Parallelism

```
Enumerate and extract all epics from docs/planning/product_requirements.md with max_parallel: 3
```

Same as Example 1 but limits parallel subagents to 3 per batch instead of the default 5.

### Example 3: Large PRD with Many Epics

```
Extract all epics from .oxenated/docs/planning/enterprise-features-prd.md
```

For a PRD with 15+ epics, the workflow will:
1. Identify all 15+ epics and their dependencies
2. Group them into 4-5 batches based on dependencies
3. Spawn subagents in parallel within each batch
4. Process batches sequentially respecting dependencies
5. Report final status for all 15+ epics

### Example 4: PRD with Complex Dependencies

```
Enumerate and extract epics from docs/planning/multi-persona-prd.md
```

For a PRD with complex cross-persona dependencies (e.g., Admin epics depending on OpsManager epics), the workflow ensures proper ordering so shared foundation epics are extracted before dependent persona-specific epics.

## Success Criteria

The workflow is complete when:
- [ ] PRD file exists and is readable
- [ ] All epics enumerated from PRD with IDs and dependencies identified
- [ ] Dependency graph built without circular dependencies
- [ ] Topological sort produces valid execution order
- [ ] All epics assigned to appropriate dependency batches
- [ ] User confirmed execution plan or chose alternative
- [ ] Subagents spawned for all batches in dependency order
- [ ] Each subagent receives exact prompt: `/prd-epic-extract.md PRD: \`[path]\` EPIC: [ID]`
- [ ] No more than 5 subagents spawned per `use_subagents` call
- [ ] Validation performed between batches confirming previous completions
- [ ] Failed extractions identified and reported
- [ ] Blocked epics (due to failed dependencies) identified and reported
- [ ] Final summary report generated with statistics
- [ ] User presented with results and next step options
- [ ] Directory structure created at `.oxenated/docs/planning/[persona]/epics/` for all extracted epics

## Error Handling

### Circular Dependencies Detected

If the dependency graph contains cycles:
1. Report the cycle: "Circular dependency detected: EPIC-A -> EPIC-B -> EPIC-A"
2. Do not proceed with extraction
3. Ask user to resolve the cycle in the PRD before retrying

### PRD Parse Errors

If the PRD cannot be parsed or contains no epics:
1. Report parse error details
2. Suggest checking PRD format against expected epic structure
3. Offer to retry or cancel

### Subagent Failures

If a subagent fails to extract an epic:
1. Capture error details from subagent output
2. Do not mark dependency as satisfied
3. Block dependent epics from processing
4. Offer retry or skip options to user

### File System Errors

If directory creation or file writing fails:
1. Check permissions and disk space
2. Retry once after brief delay
3. Report persistent errors to user

## Related Workflows

- **Individual Epic Extraction:** `/prd-epic-extract.md` - Used by subagents spawned by this workflow
- **Feature Extraction:** `/prd-feature-extract.md` - Next step after epic extraction
- **PRD Creation:** `/prd-create-or-update.md` - Create or modify the source PRD
- **Epic Peer Review:** `/prd-epic-peer-review.md` - Review extracted epic quality

## Technical Details

### Dependency Parsing

Dependencies are identified by searching for patterns like:
- "Depends on: EPIC-XXX-XX"
- "Dependencies: EPIC-XXX-XX, EPIC-YYY-YY"
- "Requires: EPIC-XXX-XX"

### Batch Size Limitations

The `use_subagents` tool supports up to 5 parallel prompts. For batches larger than 5 epics:
- Option A: Execute multiple `use_subagents` calls sequentially for the same batch
- Option B: Wait for each batch of 5 to complete before spawning the next
- Default: Option A for maximum parallelism

### Prompt Format

Each subagent receives the exact prompt format:
```
/prd-epic-extract.md PRD: `{prd_path}` EPIC: {EPIC-ID}
```

The backticks around the PRD path ensure proper handling of paths containing spaces or special characters.

## Performance Considerations

- **Parallelism**: Maximum throughput achieved with high fan-out in early batches (root epics)
- **Latency**: Dependency chains create minimum execution time floor (sum of batch processing times)
- **Resource Usage**: Each subagent consumes resources; limit parallelism if resource-constrained
- **Retry Overhead**: Failed extractions add latency; validate PRD quality before execution

For End Users: This workflow automates the tedious process of manually extracting each epic from a PRD. Instead of running `/prd-epic-extract.md` repeatedly for each epic, this workflow analyzes dependencies, parallelizes execution, and ensures proper ordering - saving time and reducing errors when bootstrapping a new project from a PRD.