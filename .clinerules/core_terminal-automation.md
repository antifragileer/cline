---
description: Terminal automation best practices for safe and reliable script execution including heredoc avoidance and text buffer management
applies_to: ["**/*.sh", "**/*.bash", "**/*.zsh", "**/cli/**/*.ts", "**/terminal/**/*.ts"]
priority: high
---

# Terminal Automation Best Practices

## Heredoc Anti-Patterns

**Rule:** NEVER send heredocs to automated terminal sessions; heredocs cause delimiter collision, buffer synchronization failures, and cannot be reliably terminated programmatically.

**Rule:** NEVER assume heredoc delimiters are unique; automated content may contain the delimiter string causing premature termination or infinite hangs.

**Rule:** NEVER use heredocs when exit code detection is required; heredoc syntax errors and failures cannot be distinguished from successful execution in piped terminal sessions.

**Rule:** NEVER nest heredocs in automated terminal commands; nested delimiters create ambiguous parsing states that are impossible to resolve without human intervention.

## Text Size Limits

**Rule:** NEVER write more than 4KB of text in a single terminal write operation; terminal buffers truncate large writes causing silent data loss.

**Rule:** NEVER rely on implicit terminal buffering for content delivery; implicit buffering causes race conditions between write and read operations.

**Rule:** ALWAYS verify terminal buffer capacity before large output operations; buffer sizes vary across terminal emulators and can be as small as 1KB.

**Rule:** NEVER stream unbounded output directly to terminal without pagination or truncation; unbounded output causes memory exhaustion and UI freezing.

## Alternative Approaches

**Rule:** ALWAYS use temporary files for multi-line input instead of heredocs; files provide explicit boundaries, error detection, and work reliably in all terminal contexts.

**Rule:** ALWAYS use `printf '%s\n'` for precise character-by-character terminal input; printf provides explicit formatting control and predictable behavior.

**Rule:** ALWAYS use proper IPC mechanisms when available instead of terminal automation; sockets, pipes, and APIs are more reliable than terminal simulation.

**Rule:** ALWAYS use `expect` or `script` utilities for interactive terminal sessions; these tools provide proper synchronization primitives that raw terminal writes lack.

## Buffer Management

**Rule:** ALWAYS explicitly flush terminal buffers after write operations; unflushed buffers cause timing-dependent race conditions.

**Rule:** ALWAYS implement chunked writing with explicit synchronization for content exceeding 1KB; chunking prevents buffer overflow and enables progress tracking.

**Rule:** NEVER assume terminal state between write operations; terminal state changes asynchronously due to user interaction, window resizing, or signal handling.

**Rule:** ALWAYS wait for terminal ready state before subsequent writes; writes issued during terminal processing are dropped or corrupted.

## Error Handling

**Rule:** ALWAYS set operation timeouts for terminal write operations; unbounded waits cause indefinite hangs when terminals become unresponsive.

**Rule:** ALWAYS detect and handle partial writes; terminal buffers may accept only a portion of data requiring retry logic.

**Rule:** ALWAYS implement cleanup handlers for terminal session failures; abandoned terminal states leak resources and block subsequent operations.

**Rule:** NEVER retry failed terminal operations without state verification; retrying into an unknown terminal state compounds errors.

## Security Considerations

**Rule:** NEVER send untrusted content directly to terminal; terminal escape sequences in content enable command injection and terminal manipulation attacks.

**Rule:** ALWAYS sanitize content for terminal control characters; control characters in output cause unintended terminal behavior and potential data exfiltration.

**Rule:** NEVER expose terminal session state to untrusted processes; terminal state includes sensitive information like environment variables and command history.