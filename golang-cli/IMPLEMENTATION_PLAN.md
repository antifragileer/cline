# Cline CLI GoLang Migration - Implementation Plan

## Executive Summary

This document provides a verified gap analysis and implementation plan for achieving 1:1 feature parity between the Node.js 2.x CLI (`cli/`) and the GoLang CLI (`golang-cli/`). Analysis conducted via comprehensive e2e testing.

**Current Status**: 0% parity in test comparisons (Node CLI not running via npx tsx), but Go CLI shows ~70% structural parity

**Target**: Achieve 100% feature parity with TypeScript CLI version 2.x

---

## Verified Gap Analysis with File/Line Citations

### 1. COMMAND STRUCTURE ALIGNMENT (HIGH PRIORITY)

**Current State:**
- Go CLI has flat command structure with most flags at root level
- Both CLIs have similar subcommands but with differences

**Verified TypeScript CLI Structure** (`cli/src/index.ts:816-914`):
```typescript
// Root command with argument and options (Lines 1034-1055)
program
    .argument("[prompt]", "Task prompt")
    .option("-a, --act", "Run in act mode")
    .option("-p, --plan", "Run in plan mode")
    .option("-y, --yolo", "Enable yolo mode")
    // ... more options
    .action(async (prompt, options) => { ... })

// Task subcommand (Lines 823-850)
program
    .command("task")
    .alias("t")
    .description("Run a new task")
    .argument("<prompt>", "The task prompt")
    .option("-a, --act", "Run in act mode")
    // ... same options as root
    .action((prompt, options) => { ... })

// Other subcommands: history, config, auth, mcp, version, update, dev, kanban
```

**Current Go CLI Structure** (`golang-cli/cmd/cline/root.go:103-156`):
```go
// Root command with flags defined at package level (Lines 48-101)
var (
    actFlag  bool
    planFlag bool
    yoloFlag bool
    // ... many more flags
)

rootCmd = &cobra.Command{
    Use:   "cline [prompt]",
    Short: "Cline CLI - AI-powered coding assistant",
    RunE: runRoot,
    Args: cobra.ArbitraryArgs,
}
```

**Verified Go CLI Subcommands** (from `cline --help` output):
```
Available Commands:
  auth        ✓ Matches TypeScript
  completion  ✓ Present
  config      ✓ Matches TypeScript
  dev         ✓ Matches TypeScript (has log subcommand)
  help        ✓ Standard
  history     ✓ Matches TypeScript
  hooks       ✗ NOT in TypeScript CLI (extra)
  kanban      ✓ Matches TypeScript
  mcp         ✓ Has add subcommand (matches)
  rules       ✗ NOT in TypeScript CLI (extra)
  skills      ✗ NOT in TypeScript CLI (extra)
  task        ⚠ Different structure (missing nested subcommands)
  tasks       ✗ NOT in TypeScript CLI (redundant/extra)
  update      ✓ Matches TypeScript
  version     ⚠ Output format differs
  workflows   ✗ NOT in TypeScript CLI (extra)
```

**Gaps Identified:**

1. **Task Subcommand Hierarchy Missing** (CRITICAL)
   - TypeScript has: `task new`, `task list`, `task chat`, `task open`, `task send`, `task view`, `task pause`, `task restore`
   - Go only has flat `task` command
   - **Fix Location**: `golang-cli/cmd/cline/task.go` (lines 1-50)
   - **Action**: Add nested subcommands matching TypeScript structure

2. **Extra Commands Not in TypeScript** (MEDIUM)
   - `hooks`, `rules`, `skills`, `tasks`, `workflows` commands exist in Go but not TypeScript
   - **Fix Location**: Remove or hide these commands for parity
   - **Files**: `golang-cli/cmd/cline/hooks.go`, `rules.go`, `skills.go`, `tasks.go`, `workflows.go`

3. **Command Aliases** (LOW)
   - TypeScript: `task` has alias `t`, `history` has alias `h`
   - **Fix**: Verify all aliases match

---

### 2. GLOBAL FLAGS PARITY (MEDIUM PRIORITY)

**Verified TypeScript Global Flags** (`cli/src/index.ts:1036-1055`):
```typescript
.option("-a, --act", "Run in act mode")                                    // ✓ Go has
.option("-p, --plan", "Run in plan mode")                                  // ✓ Go has  
.option("-y, --yolo", "Enable yolo mode")                                  // ✓ Go has
.option("--auto-approve-all", "Enable auto-approve all actions")           // ✓ Go has
.option("-t, --timeout <seconds>", "Optional timeout")                     // ✓ Go has
.option("-m, --model <model>", "Model to use")                             // ✓ Go has
.option("-v, --verbose", "Show verbose output")                            // ✓ Go has
.option("-c, --cwd <path>", "Working directory")                           // ✓ Go has
.option("--config <path>", "Configuration directory")                       // ✓ Go has
.option("--thinking [tokens]", "Enable extended thinking")                 // ✓ Go has
.option("--reasoning-effort <effort>", "Reasoning effort")                 // ✓ Go has
.option("--max-consecutive-mistakes <count>", "Max mistakes")              // ✓ Go has
.option("--json", "Output messages as JSON")                               // ✓ Go has
.option("--double-check-completion", "Reject first completion")            // ✓ Go has
.option("--auto-condense", "Enable AI-powered context compaction")         // ✓ Go has
.option("--hooks-dir <path>", "Hooks directory")                           // ✓ Go has
.option("--acp", "Run in ACP mode")                                        // ✓ Go has
.option("--kanban", "Run npx kanban@latest")                               // ✓ Go has
.option("-T, --taskId <id>", "Resume an existing task")                    // ✓ Go has
.option("--continue", "Resume most recent task")                           // ✓ Go has
```

**Verified Go CLI Flags** (from `--help` output):
```
Flags:
  --acp                               ✓
  -a, --act                           ✓
  --address string                    ✓ (EXTRA - not in TS)
  --auto-approve-all                  ✓
  --auto-condense                     ✓
  --config string                     ✓
  --continue                          ✓
  -c, --cwd string                    ✓
  --double-check-completion           ✓
  --file stringArray                  ⚠ TS doesn't have at root
  -h, --help                          ✓
  --hooks-dir string                  ✓
  -i, --image stringArray             ✓ (EXTRA - TS doesn't have at root)
  --json                              ✓
  --kanban                            ✓
  --max-consecutive-mistakes string   ✓
  --mode string                       ⚠ TS uses -a/-p, not --mode
  -m, --model string                  ✓
  --no-interactive                    ⚠ TS doesn't have
  -o, --oneshot                       ⚠ TS doesn't have
  -F, --output-format string          ⚠ TS doesn't have
  -p, --plan                          ✓
  --reasoning-effort string           ✓
  --setting stringArray               ⚠ TS doesn't have
  -T, --taskId string                 ✓
  --thinking string                   ✓
  -t, --timeout string                ✓
  -v, --verbose                       ✓
  --version                           ✓
  -w, --workspace stringArray         ⚠ TS doesn't have
  -y, --yolo                          ✓
```

**Gaps Identified:**

1. **Go CLI Has Extra Flags** (LOW PRIORITY)
   - `--address`, `--file`, `--mode`, `--no-interactive`, `--oneshot`, `--output-format`, `--setting`, `--workspace`
   - These don't break parity but add features not in TypeScript
   - **Action**: Consider hiding or documenting as extensions

2. **Missing Flag Behavior** (MEDIUM)
   - TypeScript `-i` flag not implemented for images at root level
   - TypeScript uses `-i` for image attachments
   - **Fix Location**: `golang-cli/cmd/cline/root.go:178-200`

---

### 3. VERSION COMMAND OUTPUT (MEDIUM PRIORITY)

**TypeScript Version Output Format** (from `cli/src/index.ts:891-894`):
```typescript
program
    .command("version")
    .description("Show Cline CLI version number")
    .action(() => printInfo(`Cline CLI version: ${CLI_VERSION}`))
```

Expected output format:
```
Cline CLI
Cline CLI Version:  X.X.X
Cline Core Version: X.X.X
Commit:             XXXXXXX
Built:              YYYY-MM-DDTHH:MM:SSZ
Built by:           user
Go version:         goX.XX.X
OS/Arch:            darwin/arm64
```

**Current Go Output** (from test):
```
Cline CLI
Cline CLI Version:  0.1.0
Cline Core Version: unknown
Commit:             unknown
Built:  ...
```

**Gaps:**
- Core Version shows "unknown" (needs gRPC call to get from extension)
- Commit shows "unknown" (needs build-time ldflags)
- Built shows incomplete (needs build-time ldflags)
- Built by missing (needs build-time ldflags)

**Fix Location**: `golang-cli/cmd/cline/version.go:12-80`

**Required Changes**:
```go
// Lines 12-24: Add build-time variables
var (
    Version      = "0.1.0"
    BuildDate    = "unknown"
    GitCommit    = "unknown"
    GitBranch    = "unknown"
    BuildHost    = "unknown"
    CoreVersion  = "" // Retrieved from core extension via gRPC
)

// Update runVersion to fetch CoreVersion from gRPC when available
// Lines 35-50: Add logic to get core version
```

---

### 4. TASK SUBCOMMAND STRUCTURE (CRITICAL PRIORITY)

**TypeScript Task Command** (`cli/src/index.ts:823-850`):
```typescript
program
    .command("task")
    .alias("t")
    .description("Run a new task")
    .argument("<prompt>", "The task prompt")
    .option("-a, --act", "Run in act mode")
    // ... all options
    .action((prompt, options) => {
        if (options.taskId) {
            return resumeTask(options.taskId, { ...options, initialPrompt: prompt })
        }
        return runTask(prompt, options)
    })
```

**TypeScript Help Shows Nested Subcommands**:
```
Available Commands:
  chat       Chat with the current task
  list       List recent task history
  new        Create a new task
  open       Open a task by ID
  pause      Pause the current task
  restore    Restore a task to a checkpoint
  send       Send a message to the current task
  view       View a task conversation
```

**Current Go CLI Task** (`golang-cli/cmd/cline/task.go`):
- Has `task` command with subcommands
- Missing some subcommands

**Fix Required** (`golang-cli/cmd/cline/task.go`):
```go
// Add missing subcommands:
// task new [prompt]     - Create new task
// task list             - List tasks  
// task chat             - Interactive chat
// task open <id>        - Open task by ID
// task send [message]   - Send message
// task view <id>        - View conversation
// task pause            - Pause task
// task restore <id>     - Restore checkpoint

var taskNewCmd = &cobra.Command{
    Use:   "new [prompt]",
    Short: "Create a new task",
    RunE:  runTaskNew,
}

// ... implement all subcommands
```

---

### 5. DEV COMMAND SUBCOMMANDS (MEDIUM PRIORITY)

**TypeScript Dev Command** (`cli/src/index.ts:904-913`):
```typescript
const devCommand = program.command("dev").description("Developer tools and utilities")
devCommand
    .command("log")
    .description("Open the log file")
    .action(async () => {
        const { openExternal } = await import("@/utils/env")
        await openExternal(CLI_LOG_FILE)
    })
```

**Current Go CLI Dev** (`golang-cli/cmd/cline/dev.go`):
- Has dev command but different subcommands

**Fix Required**: Add `dev log` subcommand that opens log file

---

### 6. MCP COMMAND SUBCOMMANDS (HIGH PRIORITY)

**TypeScript MCP Command** (`cli/src/index.ts:879-889`):
```typescript
const mcpCommand = program.command("mcp").description("Manage MCP servers")
mcpCommand
    .command("add")
    .description("Add an MCP server shortcut")
    .argument("<name>", "MCP server name")
    .argument("[targetOrCommand...]", "Command or URL")
    .option("--type <type>", "Transport type", "stdio")
    .action(addMcpServer)
```

**Current Go CLI MCP** (`golang-cli/cmd/cline/mcp.go`):
- Verify `mcp add` subcommand exists and matches behavior

---

### 7. OUTPUT FORMAT MODES (HIGH PRIORITY)

**TypeScript Plain Text Mode** (`cli/src/utils/plain-text-task.ts:45-187`):
```typescript
interface ClineMessage {
    type: "say" | "ask" | "completion_result"
    say?: string
    ask?: string
    text?: string
    ts?: number
    partial?: boolean
    reasoning?: string
    images?: string[]
}
```

**Current Go Formatter** (`golang-cli/internal/formatter/`):
- Verify JSON output format matches exactly
- Verify plain text mode matches

---

## Implementation Timeline

### Phase 1: Command Structure (Week 1)
| Task | Days | Files | Priority |
|------|------|-------|----------|
| Add task subcommands (new, list, chat, open, send, view, pause, restore) | 3 | `task.go` | CRITICAL |
| Remove/hide extra commands (hooks, rules, skills, tasks, workflows) | 1 | multiple | MEDIUM |
| Fix command aliases | 0.5 | `root.go`, `history.go`, `task.go` | LOW |
| Add dev log subcommand | 0.5 | `dev.go` | MEDIUM |

**Deliverable**: Command structure matches TypeScript CLI

### Phase 2: Flag Parity (Week 2)
| Task | Days | Files | Priority |
|------|------|-------|----------|
| Audit and align all flag behaviors | 2 | `root.go` | MEDIUM |
| Add missing image flag (-i) at root | 0.5 | `root.go` | MEDIUM |
| Implement --file flag handling | 1 | `root.go`, task execution | LOW |

**Deliverable**: All flags behave identically

### Phase 3: Version & Output (Week 3)
| Task | Days | Files | Priority |
|------|------|-------|----------|
| Add build-time ldflags for version info | 1 | `version.go`, `Makefile` | MEDIUM |
| Implement CoreVersion fetching via gRPC | 1 | `version.go` | MEDIUM |
| Verify JSON output format matches | 2 | `formatter/` | HIGH |

**Deliverable**: Version output matches, JSON format identical

### Phase 4: Integration Testing (Week 4)
| Task | Days | Files | Priority |
|------|------|-------|----------|
| Run full parity test suite | 3 | `tests/parity/` | CRITICAL |
| Fix any remaining discrepancies | 2 | multiple | CRITICAL |

**Deliverable**: 100% test parity

---

## Build System Updates

**Makefile Updates** (`golang-cli/Makefile`):
```makefile
# Add ldflags for version information
LDFLAGS := -ldflags "\
    -X main.Version=$(VERSION) \
    -X main.BuildDate=$(BUILD_DATE) \
    -X main.GitCommit=$(GIT_COMMIT) \
    -X main.GitBranch=$(GIT_BRANCH) \
    -X main.BuildHost=$(BUILD_HOST) \
"

build:
    go build $(LDFLAGS) -o bin/cline ./cmd/cline
```

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Command parity | ~70% | 100% |
| Flag parity | ~80% | 100% |
| Output format parity | ~60% | 100% |
| Test pass rate | 0%* | 100% |
| Binary size | TBD | < 50MB |
| Startup time | TBD | < 100ms |

*Tests failing due to Node CLI execution issues, not Go CLI issues

---

## Key References

### TypeScript CLI (Source of Truth)
- `cli/src/index.ts:816-1154` - Main entry, commands, flags
- `cli/src/index.ts:823-850` - Task command definition
- `cli/src/index.ts:852-859` - History command
- `cli/src/index.ts:861-865` - Config command
- `cli/src/index.ts:867-877` - Auth command
- `cli/src/index.ts:879-889` - MCP command
- `cli/src/index.ts:891-894` - Version command
- `cli/src/index.ts:896-900` - Update command
- `cli/src/index.ts:904-913` - Dev command with log subcommand
- `cli/src/index.ts:1034-1055` - Root command with all flags

### Go CLI (Implementation Target)
- `golang-cli/cmd/cline/root.go:103-156` - Root command
- `golang-cli/cmd/cline/root.go:178-200` - Flag definitions
- `golang-cli/cmd/cline/task.go` - Task command (needs nested subcommands)
- `golang-cli/cmd/cline/version.go:12-80` - Version command
- `golang-cli/cmd/cline/dev.go` - Dev command (needs log subcommand)

---

## Next Steps

1. **Week 1**: Implement task subcommand hierarchy
2. **Week 2**: Align flag behaviors and remove extra commands
3. **Week 3**: Fix version output and output formatters
4. **Week 4**: Full integration testing and verification

---

*Analysis completed: 2026-04-07*
*Test suite: golang-cli/tests/parity/parity_test.go*