## Brief overview
Guidelines for setting up and optimizing Go projects to ensure efficient builds, tests, and development workflows. These rules apply when working with Go codebases to prevent common performance pitfalls.

## Build optimization
- Always use `-s -w` ldflags to strip debug info from production binaries
- Use `-trimpath` to remove file system paths from binaries
- Prefer `make build` over raw `go build` to ensure consistent flags
- Use `make dev` for debug builds during development (faster compilation)
- Set up `air` for live reload during active development

## Test execution
- Never run `go test ./...` without timeouts or parallelism limits
- Use the test runner script (`scripts/test.sh`) for proper timeout handling
- Limit parallel tests to 4 by default (`-parallel=4`) to prevent resource exhaustion
- Set appropriate timeouts: 2m for unit tests, 5m for integration, 10m for e2e
- Separate test types: unit, integration, e2e should be runnable independently

## Cache management
- Monitor build cache size regularly with `make cache-info`
- Clean caches when they exceed 5GB or cause performance issues
- Use `go clean -cache` for build cache, `go clean -modcache` for module cache
- Document cache locations for troubleshooting

## Makefile structure
- Organize targets into logical sections (build, test, quality, release)
- Provide `help` target documenting all available commands
- Use consistent variable naming (GOCMD, GOBUILD, GOTEST, etc.)
- Include cache management targets in all Go projects

## Development workflow
- Use watch mode (`make watch`) for rapid iteration
- Test individual packages during development, not entire codebase
- Profile slow tests to identify bottlenecks
- Set resource limits to prevent system sluggishness

## Project setup checklist
When setting up a new Go project:
1. Create optimized Makefile with build/test targets
2. Add `.air.toml` for live reload
3. Create test runner script with timeouts
4. Document cache management commands
5. Set up test configuration file
6. Add test documentation (TESTING.md)