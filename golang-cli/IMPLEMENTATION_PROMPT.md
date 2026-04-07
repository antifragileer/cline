# Cline CLI GoLang Migration - Implementation Prompt

## Context
You are implementing the GoLang rewrite of the Cline CLI to achieve 1:1 feature parity with the Node.js 2.x CLI (`cli/`). The implementation must be a pure Go solution with NO dependencies on the existing TypeScript CLI code.

## Source of Truth: TypeScript CLI v2.x

All behavior must match exactly with these verified source locations:

**Critical Source Files:**
- `cli/src/index.ts:816-1154` - Main entry with all commands and flags
- `cli/src/index.ts:1034-1055` - Root command with all global flags
- `cli/src/index.ts:823-850` - Task command with nested subcommands
- `cli/src/index.ts:904-913` - Dev command with log subcommand
- `cli/src/utils/plain-text-task.ts:45-187` - Output format definitions

## Implementation Requirements

### 1. Task Subcommand Hierarchy (CRITICAL - Week 1 Priority)

**TypeScript Structure** (`cli/src/index.ts:823-850`):
```typescript
program
    .command("task")
    .alias("t")
    .description("Run a new task")
    .argument("<prompt>", "The task prompt")
    // ... all root options
    .action((prompt, options) => { ... })

// Nested subcommands from TypeScript help:
// task new [prompt]     - Create a new task
// task list             - List recent task history  
// task chat             - Chat with the current task
// task open <id>        - Open a task by ID
// task send [message]   - Send a message to the current task
// task view <id>        - View a task conversation
// task pause            - Pause the current task
// task restore <id>     - Restore a task to a checkpoint
```

**Implementation Target**: `golang-cli/cmd/cline/task.go`

Add these subcommands to the existing task command:
```go
var (
    taskNewCmd = &cobra.Command{
        Use:   "new [prompt]",
        Short: "Create a new task",
        RunE:  runTaskNew,
    }
    
    taskListCmd = &cobra.Command{
        Use:   "list",
        Short: "List recent task history",
        RunE:  runTaskList,
    }
    
    taskChatCmd = &cobra.Command{
        Use:   "chat",
        Short: "Chat with the current task",
        RunE:  runTaskChat,
    }
    
    taskOpenCmd = &cobra.Command{
        Use:   "open <id>",
        Short: "Open a task by ID",
        Args:  cobra.ExactArgs(1),
        RunE:  runTaskOpen,
    }
    
    taskSendCmd = &cobra.Command{
        Use:   "send [message]",
        Short: "Send a message to the current task",
        RunE:  runTaskSend,
    }
    
    taskViewCmd = &cobra.Command{
        Use:   "view <id>",
        Short: "View a task conversation",
        Args:  cobra.ExactArgs(1),
        RunE:  runTaskView,
    }
    
    taskPauseCmd = &cobra.Command{
        Use:   "pause",
        Short: "Pause the current task",
        RunE:  runTaskPause,
    }
    
    taskRestoreCmd = &cobra.Command{
        Use:   "restore <id>",
        Short: "Restore a task to a checkpoint",
        Args:  cobra.ExactArgs(1),
        RunE:  runTaskRestore,
    }
)

func init() {
    taskCmd.AddCommand(taskNewCmd)
    taskCmd.AddCommand(taskListCmd)
    taskCmd.AddCommand(taskChatCmd)
    taskCmd.AddCommand(taskOpenCmd)
    taskCmd.AddCommand(taskSendCmd)
    taskCmd.AddCommand(taskViewCmd)
    taskCmd.AddCommand(taskPauseCmd)
    taskCmd.AddCommand(taskRestoreCmd)
}
```

### 2. Remove Extra Commands (MEDIUM Priority)

These commands exist in Go CLI but NOT in TypeScript CLI:
- `hooks` (`hooks.go`)
- `rules` (`rules.go`)  
- `skills` (`skills.go`)
- `tasks` (`tasks.go` - redundant with `task`)
- `workflows` (`workflows.go`)

**Action**: Either:
1. Remove these files entirely, OR
2. Hide them with `Hidden: true` in cobra command definitions

Example for hiding:
```go
var hooksCmd = &cobra.Command{
    Use:    "hooks",
    Short:  "Manage Cline hooks",
    Hidden: true,  // Hide from help
    RunE:   runHooks,
}
```

### 3. Dev Log Subcommand (MEDIUM Priority)

**TypeScript Implementation** (`cli/src/index.ts:904-913`):
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

**Implementation Target**: `golang-cli/cmd/cline/dev.go`

Add:
```go
var devLogCmd = &cobra.Command{
    Use:   "log",
    Short: "Open the log file",
    RunE:  runDevLog,
}

func runDevLog(cmd *cobra.Command, args []string) error {
    logFile := filepath.Join(os.Getenv("HOME"), ".cline", "logs", "cline.log")
    // Open file with default application
    return openFile(logFile)
}

func init() {
    devCmd.AddCommand(devLogCmd)
}
```

### 4. Version Output Format (MEDIUM Priority)

**TypeScript Output Format**:
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

**Current Go Output**:
```
Cline CLI
Cline CLI Version:  0.1.0
Cline Core Version: unknown
Commit:             unknown
Built:  ...
```

**Implementation Target**: `golang-cli/cmd/cline/version.go`

Required changes:
1. Add build-time variables at top of file:
```go
var (
    Version      = "0.1.0"
    BuildDate    = "unknown"
    GitCommit    = "unknown"
    GitBranch    = "unknown"
    BuildHost    = "unknown"
)
```

2. Update runVersion to fetch CoreVersion from gRPC:
```go
func runVersion(cmd *cobra.Command, args []string) error {
    // Try to get core version via gRPC
    coreVersion := "unknown"
    client, err := host.NewClient("localhost:50052")
    if err == nil {
        defer client.Stop()
        // Call version endpoint if available
        // coreVersion = client.GetVersion()
    }
    
    // Print formatted output matching TypeScript
    fmt.Fprintln(cmd.OutOrStdout(), "Cline CLI")
    fmt.Fprintf(cmd.OutOrStdout(), "Cline CLI Version:  %s\n", Version)
    fmt.Fprintf(cmd.OutOrStdout(), "Cline Core Version: %s\n", coreVersion)
    fmt.Fprintf(cmd.OutOrStdout(), "Commit:             %s\n", GitCommit)
    fmt.Fprintf(cmd.OutOrStdout(), "Built:              %s\n", BuildDate)
    fmt.Fprintf(cmd.OutOrStdout(), "Built by:           %s\n", BuildHost)
    fmt.Fprintf(cmd.OutOrStdout(), "Go version:         %s\n", runtime.Version())
    fmt.Fprintf(cmd.OutOrStdout(), "OS/Arch:            %s/%s\n", runtime.GOOS, runtime.GOARCH)
    
    return nil
}
```

3. Update Makefile to pass ldflags:
```makefile
VERSION := 0.1.0
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS := -ldflags "\
    -X main.Version=$(VERSION) \
    -X main.BuildDate=$(BUILD_DATE) \
    -X main.GitCommit=$(GIT_COMMIT) \
    -X main.GitBranch=$(GIT_BRANCH) \
"

build:
    go build $(LDFLAGS) -o bin/cline ./cmd/cline
```

### 5. Flag Verification Checklist

Verify all TypeScript flags exist with identical behavior:

| Flag | TypeScript Location | Go Status | Action |
|------|---------------------|-----------|--------|
| `-a, --act` | `index.ts:1036` | ✓ Exists | Verify behavior |
| `-p, --plan` | `index.ts:1037` | ✓ Exists | Verify behavior |
| `-y, --yolo` | `index.ts:1038` | ✓ Exists | Verify behavior |
| `--auto-approve-all` | `index.ts:1039` | ✓ Exists | Verify behavior |
| `-t, --timeout` | `index.ts:1040` | ✓ Exists | Verify behavior |
| `-m, --model` | `index.ts:1041` | ✓ Exists | Verify behavior |
| `-v, --verbose` | `index.ts:1042` | ✓ Exists | Verify behavior |
| `-c, --cwd` | `index.ts:1043` | ✓ Exists | Verify behavior |
| `--config` | `index.ts:1044` | ✓ Exists | Verify behavior |
| `--thinking` | `index.ts:1045` | ✓ Exists | Verify behavior |
| `--reasoning-effort` | `index.ts:1046` | ✓ Exists | Verify behavior |
| `--max-consecutive-mistakes` | `index.ts:1047` | ✓ Exists | Verify behavior |
| `--json` | `index.ts:1048` | ✓ Exists | Verify behavior |
| `--double-check-completion` | `index.ts:1049` | ✓ Exists | Verify behavior |
| `--auto-condense` | `index.ts:1050` | ✓ Exists | Verify behavior |
| `--hooks-dir` | `index.ts:1051` | ✓ Exists | Verify behavior |
| `--acp` | `index.ts:1052` | ✓ Exists | Verify behavior |
| `--kanban` | `index.ts:1053` | ✓ Exists | Verify behavior |
| `-T, --taskId` | `index.ts:1054` | ✓ Exists | Verify behavior |
| `--continue` | `index.ts:1055` | ✓ Exists | Verify behavior |

### 6. Command Aliases

Add missing aliases:
```go
// In task.go init()
taskCmd.Aliases = []string{"t"}

// In history.go init()
historyCmd.Aliases = []string{"h"}
```

## Testing Requirements

After implementing changes, run the parity test suite:

```bash
cd golang-cli/tests/parity
go test -v -run TestParitySuite
```

The tests will generate:
- `parity_execution_report.json` - JSON results
- `parity_gap_report.md` - Markdown report

## File/Line Reference Summary

**TypeScript CLI (Source)**:
- Main commands: `cli/src/index.ts:816-914`
- Root flags: `cli/src/index.ts:1036-1055`
- Task command: `cli/src/index.ts:823-850`
- Dev command: `cli/src/index.ts:904-913`
- MCP command: `cli/src/index.ts:879-889`

**Go CLI (Target)**:
- Root command: `golang-cli/cmd/cline/root.go:103-156`
- Task command: `golang-cli/cmd/cline/task.go`
- Dev command: `golang-cli/cmd/cline/dev.go`
- Version command: `golang-cli/cmd/cline/version.go:12-80`
- Flag definitions: `golang-cli/cmd/cline/root.go:178-200`

## Success Criteria

- [ ] All task subcommands (new, list, chat, open, send, view, pause, restore) implemented
- [ ] Extra commands (hooks, rules, skills, tasks, workflows) removed or hidden
- [ ] Dev log subcommand added
- [ ] Version output format matches TypeScript exactly
- [ ] Command aliases match (task->t, history->h)
- [ ] All flags behave identically to TypeScript
- [ ] Parity test pass rate > 90%

## Important Constraints

1. **NO TypeScript dependencies** - This must be a pure Go implementation
2. **File/line citations required** - Every requirement must cite the TypeScript source
3. **No new external dependencies** - Use existing Go modules
4. **Maintain gRPC compatibility** - Ensure all gRPC calls still work
5. **State compatibility** - Must work with existing `~/.cline/data/` directories

## Deliverables

1. Updated Go source files with required changes
2. All tests passing
3. Updated documentation if needed
4. Build verification with `go build ./cmd/cline`

---

*Generated from: golang-cli/IMPLEMENTATION_PLAN.md*
*TypeScript Source: cli/src/index.ts*