---
description: Bash scripting standards and security rules
applies_to: ["**/*.sh", "scripts/**/*", "deploy/**/*.sh"]
priority: high
---

# Bash Scripting Best Practices

## Script Structure

**Rule:** Always start scripts with `#!/bin/bash` as first line
**Rule:** Include header comment block with purpose, author, date
**Rule:** Document script dependencies and required environment variables at top
**Rule:** Specify minimum bash version requirements if using advanced features
**Rule:** Place function definitions before main script logic
**Rule:** Group related functions together
**Rule:** Use meaningful file names indicating script purpose

## Error Handling

**Rule:** Enable strict error handling in production scripts with set options
**Rule:** Always check exit status of critical commands
**Rule:** Implement error messages with context about what failed
**Rule:** Use exit codes consistently (0 success, non-zero failure)
**Rule:** Redirect errors to STDERR, not STDOUT
**Rule:** Implement trap handlers for cleanup on script exit
**Rule:** Use debugging flags during development
**Rule:** Test scripts with shellcheck before deployment

## Security

**Rule:** Always validate and sanitize user input
**Rule:** Never trust external input - treat as potentially malicious
**Rule:** Use quotes around variables to prevent word splitting and globbing
**Rule:** Validate file paths and check for directory traversal attempts
**Rule:** Never hardcode passwords, API keys, or sensitive data in scripts
**Rule:** Use environment variables or secure config files for credentials
**Rule:** Set appropriate file permissions on scripts with sensitive logic
**Rule:** Avoid logging sensitive information
**Rule:** Clear sensitive variables after use
**Rule:** Avoid using eval with user-supplied input
**Rule:** Be cautious with command substitution when handling external data
**Rule:** Use full paths for critical system commands in production
**Rule:** Validate and sanitize data used in command construction

## Variable and Parameter Handling

**Rule:** Use descriptive, meaningful variable names
**Rule:** Use UPPERCASE for environment variables and constants
**Rule:** Use lowercase for local variables
**Rule:** Use snake_case for multi-word variable names
**Rule:** Prefix global variables with namespace to avoid conflicts
**Rule:** Always use curly braces when referencing variables
**Rule:** Initialize variables before use
**Rule:** Use readonly for constants
**Rule:** Declare variables with appropriate scope (local for functions)
**Rule:** Use default values for optional parameters
**Rule:** Provide default values for potentially unset variables
**Rule:** Use proper quoting to handle spaces and special characters
**Rule:** Check if required variables are set and non-empty

## Functions

**Rule:** Create functions for any code used more than once
**Rule:** Keep functions focused on single task
**Rule:** Limit function length for readability
**Rule:** Use meaningful function names describing their action
**Rule:** Return meaningful exit codes from functions
**Rule:** Document each function with comment block including purpose, parameters, output, return values
**Rule:** Place documentation immediately before function definition
**Rule:** Validate function parameters at beginning
**Rule:** Use local variables for function parameters
**Rule:** Document expected parameter formats
**Rule:** Provide helpful error messages for invalid parameters

## File Operations

**Rule:** Check file existence and permissions before operations
**Rule:** Use proper error handling for file operations
**Rule:** Clean up temporary files in trap handlers
**Rule:** Use mktemp for creating temporary files securely
**Rule:** Validate file paths to prevent directory traversal

## Output and Logging

**Rule:** Understand difference between STDOUT and STDERR
**Rule:** Use appropriate redirection operators
**Rule:** Be explicit about redirection targets
**Rule:** Consider using tee for logging while displaying output
**Rule:** Provide clear prompts for user input
**Rule:** Validate user responses
**Rule:** Offer sensible defaults where appropriate
**Rule:** Implement confirmation prompts for destructive operations

## Portability

**Rule:** Avoid bash-specific features if portability required
**Rule:** Document any non-POSIX features used
**Rule:** Test scripts on target systems
**Rule:** Use feature detection rather than version detection
**Rule:** Check for required commands before using them
**Rule:** Provide helpful error messages for missing dependencies
**Rule:** Document all external command requirements
**Rule:** Use command -v to check for command availability

## Performance

**Rule:** Minimize subprocess creation in loops
**Rule:** Use built-in bash features when possible
**Rule:** Avoid unnecessary cat usage
**Rule:** Cache expensive operations results
**Rule:** Clean up resources (files, processes) on exit
**Rule:** Implement timeouts for long-running operations
**Rule:** Monitor and limit resource usage where appropriate

## Testing

**Rule:** Write test cases for critical functions
**Rule:** Test edge cases and error conditions
**Rule:** Validate script behavior with different inputs
**Rule:** Test scripts in safe environment before production
**Rule:** Use consistent indentation
**Rule:** Keep line length reasonable (under 80-100 characters)
**Rule:** Remove commented-out code before committing

## Documentation

**Rule:** Comment complex logic and non-obvious code
**Rule:** Explain why, not what (code shows what)
**Rule:** Keep comments up-to-date with code changes
**Rule:** Use TODO/FIXME markers for known issues
**Rule:** Document assumptions and limitations
**Rule:** Maintain README for script collections
**Rule:** Provide usage examples

## Environment and Configuration

**Rule:** Document all required environment variables
**Rule:** Provide sensible defaults where possible
**Rule:** Validate environment variables at startup
**Rule:** Use consistent naming convention
**Rule:** Separate configuration from code
**Rule:** Support multiple environments (dev, staging, prod)
