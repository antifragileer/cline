---
description: Bash console UI scripting standards
applies_to: ["**/*.sh", "scripts/**/*.sh", "deploy/**/*.sh"]
priority: medium
---

# Bash Console UI Scripting Best Practices

## Terminal Control

**Rule:** Detect terminal capabilities before using advanced features with tput
**Rule:** Check for color support, cursor control, terminal dimensions at initialization
**Rule:** Implement fallback mechanisms for limited terminals
**Rule:** Use tput commands instead of hardcoded ANSI sequences for portability
**Rule:** Always reset terminal attributes at exit using trap handlers
**Rule:** Test escape sequences across multiple terminal emulators
**Rule:** Handle terminal resizing with trap on SIGWINCH signal

## Screen Management

**Rule:** Save cursor position before drawing UI elements, restore appropriately
**Rule:** Clear screen sections efficiently, avoid full screen clears
**Rule:** Use alternate screen buffer for full-screen apps
**Rule:** Implement double-buffering for flicker-free updates
**Rule:** Document screen coordinate systems and boundaries

## Layout and Design

**Rule:** Implement responsive layouts adapting to terminal dimensions
**Rule:** Use consistent spacing and alignment
**Rule:** Define clear boundaries using box-drawing characters
**Rule:** Handle content overflow properly
**Rule:** Test layouts at various terminal sizes (80x24, 120x40, etc.)

## Color and Styling

**Rule:** Check for color support before applying colors
**Rule:** Provide monochrome fallbacks for limited terminals
**Rule:** Use semantic colors (red=error, green=success)
**Rule:** Implement configurable color themes
**Rule:** Don't rely solely on color to convey information
**Rule:** Test on both dark and light backgrounds
**Rule:** Ensure sufficient contrast ratios

## Input Handling

**Rule:** Implement non-blocking input reading for responsive interfaces
**Rule:** Handle special keys consistently across terminals
**Rule:** Provide keyboard shortcuts for common operations
**Rule:** Implement input buffering and debouncing
**Rule:** Support both vi-style and arrow key navigation
**Rule:** Document all keyboard shortcuts
**Rule:** Test across different terminal emulators and SSH

## Navigation

**Rule:** Implement intuitive navigation consistent with common CLI tools
**Rule:** Provide visual feedback for current selection/focus
**Rule:** Support keyboard and mouse navigation where appropriate
**Rule:** Handle menu overflow with scrolling/pagination
**Rule:** Test navigation with keyboard-only interaction

## Progress Indicators

**Rule:** Use appropriate progress indicators for operation types
**Rule:** Implement smooth updates without flickering
**Rule:** Provide time estimates when possible (ETA, elapsed)
**Rule:** Include textual progress alongside visual indicators
**Rule:** Handle terminal width constraints

## Error Handling

**Rule:** Always restore terminal state on exit using trap handlers
**Rule:** Implement cleanup for all terminal modifications
**Rule:** Handle interruption signals gracefully
**Rule:** Reset cursor visibility and position on exit
**Rule:** Provide clear, actionable error messages
**Rule:** Implement error logging separate from UI display

## Performance

**Rule:** Minimize terminal I/O operations
**Rule:** Batch terminal commands to reduce system calls
**Rule:** Implement dirty region tracking for selective redraws
**Rule:** Avoid unnecessary full screen redraws
**Rule:** Profile and optimize critical rendering paths

## Cross-Platform

**Rule:** Test across major terminal emulators (xterm, iTerm2, Windows Terminal)
**Rule:** Handle OS-specific terminal behaviors
**Rule:** Support UTF-8 as default character encoding
**Rule:** Provide ASCII fallback for limited environments
**Rule:** Test over SSH and terminal multiplexers

## Accessibility

**Rule:** Structure output in logical, sequential manner
**Rule:** Provide text alternatives for visual elements
**Rule:** Ensure keyboard accessibility for all functions
**Rule:** Avoid keyboard traps
**Rule:** Provide high contrast color schemes
**Rule:** Allow color customization
**Rule:** Test with screen readers when possible

## Testing

**Rule:** Implement unit tests for UI component logic
**Rule:** Create integration tests for interaction flows
**Rule:** Test edge cases and boundary conditions
**Rule:** Test on physical terminals and consoles
**Rule:** Verify behavior over slow network connections
**Rule:** Validate international character support

## Documentation

**Rule:** Document all keyboard shortcuts and commands
**Rule:** Include installation and setup instructions
**Rule:** Provide troubleshooting guides
**Rule:** Document configuration options
**Rule:** Maintain up-to-date help screens

## Security

**Rule:** Sanitize all user input before processing
**Rule:** Prevent terminal escape sequence injection
**Rule:** Validate input lengths
**Rule:** Mask password and sensitive input
**Rule:** Clear sensitive information from screen buffers
**Rule:** Test with malicious input patterns

## Configuration

**Rule:** Provide configuration files for customization
**Rule:** Implement sensible defaults
**Rule:** Support environment variables for configuration
**Rule:** Validate configuration before applying
**Rule:** Support multiple themes (light/dark modes)
