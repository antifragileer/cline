# Epic: Structured Output (JSON)

## Epic ID
EPIC-AUTO-OUT-007

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1243-L1262]

## Persona Context
**Target Persona:** DevOps/Automation User
**Persona Pain Points:**
- Need for reliable exit codes
- JSON output for parsing
- No interactive prompts breaking scripts
- Timeout controls for long-running tasks

## Epic Overview
**Objective:** Implement structured JSON output formatting for the GoLang CLI to enable machine-readable output for integration with other tools, CI/CD pipelines, and automation workflows.
**Business Value:** Enables programmatic access, tool ecosystem integration, and automation workflows that parse CLI output programmatically.
**PRD References:** REQ-007

## Epic IAOOI Framework
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1243-L1262]

**Inputs:**
- Message objects (type, text, ts, reasoning, say, ask, partial, images, files)
- Output format flag (`--json`)
- JSON schema requirements

**Activities:**
- Serialize messages to JSON
- Stream JSON lines for real-time output
- Handle errors in JSON format
- Marshal message objects with proper field inclusion

**Outputs:**
- JSON-formatted messages
- Structured error responses
- Streaming JSON lines for long-running tasks

**Outcomes:**
- Machine-readable output for integration with other tools
- Parseable output for scripts and automation tools
- Real-time JSON output for long-running task monitoring

**Impacts:**
- Tool ecosystem integration
- Programmatic access to Cline CLI
- Automation workflow enablement
- CI/CD pipeline integration

## Features in This Epic
1. **FEAT-AUTO-OUT-007-JSON-001**: JSON Output Formatting
2. **FEAT-AUTO-OUT-007-STREAM-002**: JSON Streaming for Long Tasks

## Dependencies
- Epic 6 (Automated Execution & Yolo Mode) - for non-interactive mode foundation
- Epic 11 (Core Extension Integration) - for message streaming from core

## Success Criteria
- [ ] JSON output format matches existing TypeScript CLI exactly
- [ ] All message types serialize correctly to JSON
- [ ] Optional fields included only when present
- [ ] Error output follows JSON format
- [ ] Streaming JSON works for long-running tasks
- [ ] Partial message streaming supported
- [ ] Dual testing shows byte-for-byte parity with existing CLI

## Implementation Notes
- Message struct must match existing TypeScript message format exactly
- JSON marshaling must preserve field names and types
- Streaming must output JSON lines immediately as messages arrive
- Partial flag must be included for streaming chunks