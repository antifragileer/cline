## Brief overview
Guidelines for preventing process leaks and managing rate limits when developing scripts and tests that invoke live AI connections through the cline CLI. The key principle: tests using real AI connections must run serially, while other tests can run in parallel.

## AI Connection Test Detection
- Tests that invoke `cline task`, `cline ask`, or any command that makes live API calls are "AI connection tests"
- Tests that only check local state (config, version, help) are "local tests" and can run in parallel
- When in doubt, assume a test uses AI connections and run it serially

## Serial Execution for AI Tests
- AI connection tests must run serially (one at a time) to prevent rate limiting and process accumulation
- Use file locks or semaphores to ensure only one AI test runs at a time
- Never run AI connection tests in parallel with each other
- Local tests (non-AI) can still run in parallel with each other

## Process Spawning Controls for AI Tests
- Implement rate limiting for AI API calls (e.g., max 50 calls per hour with automatic throttling)
- Limit concurrent AI process execution to 1 (strictly serial)
- Use `timeout` command for all AI process invocations (e.g., `timeout 300 cline task -y "prompt"`)
- Implement global runtime limits (e.g., 1 hour max total execution time)
- Track and limit retry attempts for AI calls (max 2 retries with increasing delays)

## Concurrency Management
- Use a single global semaphore for all AI connection tests
- Allow unlimited parallelism for local/non-AI tests
- Queue AI tests behind each other, never concurrently
- Use backoff strategies when rate limits are approached

## Process Cleanup Requirements
- Track all spawned PIDs in an array for cleanup
- Implement `trap` handlers for EXIT, INT, TERM, and HUP signals
- Send SIGTERM first, wait 1-2 seconds, then send SIGKILL if needed
- Clean up zombie processes by terminating their parent processes
- Use `t.Cleanup()` in Go tests to ensure subprocess termination

## Monitoring and Detection
- Use `scripts/cline-process-monitor.sh` to check process counts before/after test runs
- Run process monitor in CI/CD pipelines: `./scripts/cline-process-monitor.sh check || exit 1`
- Set thresholds: Warning at >5 AI processes, Critical at >10 total processes
- Monitor for processes exceeding 30 minutes runtime
- Use dry-run mode first to validate scripts without executing
- Track which tests are AI vs non-AI in test manifests

## Rate Limiting Best Practices
- Track total API calls with a counter variable
- Implement sliding window or fixed window rate limiting
- Wait for quota reset when limits are exceeded
- Reduce parallelism and retry counts when approaching limits
- Log rate limit status for debugging

## Emergency Procedures
- Only use emergency kills for stuck/zombie processes, not running tests
- Kill long-running processes: `./scripts/cline-process-monitor.sh kill-long-running`
- Clean up zombies: `./scripts/cline-process-monitor.sh kill-zombies`
- Emergency kill all: `./scripts/cline-process-monitor.sh kill-all` (requires confirmation)
- Stop all scripts immediately: `pkill -f "cline-feature-evaluator"`

## Test Classification
- Tag tests in manifest files as `"type": "ai"` or `"type": "local"`
- AI tests: Involve `cline task`, `cline ask`, or API calls
- Local tests: Config, version, help, internal logic only
- Executor should respect tags and serialize AI tests only

## Script Development Checklist
Before committing scripts that spawn cline processes:
- [ ] Tests classified as AI or Local in manifest
- [ ] AI tests use global semaphore to ensure serial execution
- [ ] Rate limiting implemented for AI connections
- [ ] Timeout set for all AI process calls
- [ ] Max 1 concurrent AI process (serial execution)
- [ ] Proper cleanup handlers registered with trap
- [ ] Process monitor check passes: `./scripts/cline-process-monitor.sh check`
- [ ] Dry-run mode tested before actual execution
- [ ] Parallel local tests verified to not use AI
