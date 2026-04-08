// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"fmt"
	"regexp"
	"strings"
)

// Normalizer applies transformations to normalize output for comparison
type Normalizer struct {
	Name        string
	Description string
	Apply       func(string) string
}

// Comparator compares CLI outputs with normalization
type Comparator struct {
	normalizers []Normalizer
}

// NewComparator creates a new comparator with default normalizers
func NewComparator() *Comparator {
	return &Comparator{
		normalizers: DefaultNormalizers(),
	}
}

// NewComparatorWith creates a new comparator with custom normalizers
func NewComparatorWith(normalizers []Normalizer) *Comparator {
	return &Comparator{
		normalizers: normalizers,
	}
}

// AddNormalizer adds a normalizer to the comparator
func (c *Comparator) AddNormalizer(n Normalizer) {
	c.normalizers = append(c.normalizers, n)
}

// RemoveNormalizer removes a normalizer by name
func (c *Comparator) RemoveNormalizer(name string) {
	var filtered []Normalizer
	for _, n := range c.normalizers {
		if n.Name != name {
			filtered = append(filtered, n)
		}
	}
	c.normalizers = filtered
}

// ComparisonResult holds the result of a comparison
type ComparisonResult struct {
	Match       bool              // Whether outputs match after normalization
	Diff        string            // Human-readable diff
	Expected    string            // Normalized expected output
	Actual      string            // Normalized actual output
	Differences []DiffDetail      // Detailed differences
	Skipped     []string          // Normalizers that were skipped
}

// DiffDetail represents a single difference
type DiffDetail struct {
	Type        string // "line", "char", "regex", etc.
	Line        int    // Line number (1-indexed)
	Expected    string
	Actual      string
	Description string
}

// Compare compares two outputs and returns differences
func (c *Comparator) Compare(expected, actual string) *ComparisonResult {
	// Apply normalizers
	normalizedExpected := c.normalize(expected, nil)
	normalizedActual := c.normalize(actual, nil)

	result := &ComparisonResult{
		Expected: normalizedExpected,
		Actual:   normalizedActual,
		Match:    normalizedExpected == normalizedActual,
	}

	if !result.Match {
		result.Diff = c.generateDiff(normalizedExpected, normalizedActual)
		result.Differences = c.findDifferences(normalizedExpected, normalizedActual)
	}

	return result
}

// CompareWithSkip compares outputs while skipping specified normalizers
func (c *Comparator) CompareWithSkip(expected, actual string, skipNames []string) *ComparisonResult {
	// Apply normalizers, skipping specified ones
	normalizedExpected := c.normalize(expected, skipNames)
	normalizedActual := c.normalize(actual, skipNames)

	result := &ComparisonResult{
		Expected: normalizedExpected,
		Actual:   normalizedActual,
		Match:    normalizedExpected == normalizedActual,
		Skipped:  skipNames,
	}

	if !result.Match {
		result.Diff = c.generateDiff(normalizedExpected, normalizedActual)
		result.Differences = c.findDifferences(normalizedExpected, normalizedActual)
	}

	return result
}

// normalize applies all normalizers except those in skip list
func (c *Comparator) normalize(input string, skip []string) string {
	result := input
	skipMap := make(map[string]bool)
	for _, s := range skip {
		skipMap[s] = true
	}

	for _, n := range c.normalizers {
		if !skipMap[n.Name] {
			result = n.Apply(result)
		}
	}
	return result
}

// generateDiff creates a unified diff-like output
func (c *Comparator) generateDiff(expected, actual string) string {
	var sb strings.Builder
	sb.WriteString("--- Expected\n")
	sb.WriteString("+++ Actual\n")
	sb.WriteString("@@ -1,1 +1,1 @@\n")

	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		exp := ""
		act := ""
		if i < len(expectedLines) {
			exp = expectedLines[i]
		}
		if i < len(actualLines) {
			act = actualLines[i]
		}

		if exp != act {
			if exp != "" {
				sb.WriteString(fmt.Sprintf("-%s\n", exp))
			}
			if act != "" {
				sb.WriteString(fmt.Sprintf("+%s\n", act))
			}
		} else {
			sb.WriteString(fmt.Sprintf(" %s\n", exp))
		}
	}

	return sb.String()
}

// findDifferences finds detailed differences between outputs
func (c *Comparator) findDifferences(expected, actual string) []DiffDetail {
	var differences []DiffDetail
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		exp := ""
		act := ""
		if i < len(expectedLines) {
			exp = expectedLines[i]
		}
		if i < len(actualLines) {
			act = actualLines[i]
		}

		if exp != act {
			differences = append(differences, DiffDetail{
				Type:        "line",
				Line:        i + 1,
				Expected:    exp,
				Actual:      act,
				Description: fmt.Sprintf("Line %d differs", i+1),
			})
		}
	}

	return differences
}

// DefaultNormalizers returns the standard set of normalizers
func DefaultNormalizers() []Normalizer {
	return []Normalizer{
		// Timestamp normalization
		{
			Name:        "timestamp",
			Description: "Normalizes various timestamp formats",
			Apply: func(s string) string {
				// ISO 8601: 2024-01-15T10:30:00Z
				iso8601 := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?`)
				s = iso8601.ReplaceAllString(s, "[TIMESTAMP]")
				
				// Date: 2024-01-15
				date := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
				s = date.ReplaceAllString(s, "[DATE]")
				
				// Time: 10:30:00 or 10:30:00.123
				time := regexp.MustCompile(`\d{2}:\d{2}:\d{2}(?:\.\d+)?`)
				s = time.ReplaceAllString(s, "[TIME]")
				
				return s
			},
		},
		// UUID normalization
		{
			Name:        "uuid",
			Description: "Normalizes UUIDs",
			Apply: func(s string) string {
				uuid := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
				return uuid.ReplaceAllString(s, "[UUID]")
			},
		},
		// Task ID normalization (short UUIDs or task IDs)
		{
			Name:        "task_id",
			Description: "Normalizes task IDs",
			Apply: func(s string) string {
				// Task IDs are often alphanumeric strings
				taskID := regexp.MustCompile(`task[_-]?[a-zA-Z0-9]{8,}`)
				return taskID.ReplaceAllString(s, "[TASK_ID]")
			},
		},
		// Duration normalization
		{
			Name:        "duration",
			Description: "Normalizes duration strings",
			Apply: func(s string) string {
				// Patterns like "5.2s", "1m30s", "2h15m"
				duration := regexp.MustCompile(`\d+\.?\d*\s*(ns|us|µs|ms|s|m|h)`)
				return duration.ReplaceAllString(s, "[DURATION]")
			},
		},
		// Path normalization
		{
			Name:        "path",
			Description: "Normalizes absolute paths",
			Apply: func(s string) string {
				// Home directory
				home := regexp.MustCompile(`/Users/[^/]+|/home/[^/]+|C:\\\\Users\\\\[^\\\\]+`)
				s = home.ReplaceAllString(s, "[HOME]")
				
				// Absolute paths
				absPath := regexp.MustCompile(`(?:/[^/\s]+)+|(?:[A-Za-z]:\\\\[^\\\\\s]+)`)
				s = absPath.ReplaceAllString(s, "[PATH]")
				
				return s
			},
		},
		// Version number normalization (for version strings that may differ)
		{
			Name:        "version",
			Description: "Normalizes version numbers",
			Apply: func(s string) string {
				// Semantic version: 1.2.3, 1.2.3-beta.1, etc.
				semver := regexp.MustCompile(`\d+\.\d+\.\d+(?:[-+.]?[a-zA-Z0-9.-]+)?`)
				return semver.ReplaceAllString(s, "[VERSION]")
			},
		},
		// ANSI escape code stripping
		{
			Name:        "ansi",
			Description: "Removes ANSI escape codes",
			Apply: func(s string) string {
				// ANSI escape sequences
				ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
				return ansi.ReplaceAllString(s, "")
			},
		},
		// Whitespace normalization
		{
			Name:        "whitespace",
			Description: "Normalizes whitespace",
			Apply: func(s string) string {
				// Trim trailing whitespace from each line
				lines := strings.Split(s, "\n")
				for i, line := range lines {
					lines[i] = strings.TrimRight(line, " \t\r")
				}
				// Remove extra blank lines at end
				for len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				return strings.Join(lines, "\n")
			},
		},
		// Memory address normalization (for pointer outputs)
		{
			Name:        "memory_address",
			Description: "Normalizes memory addresses",
			Apply: func(s string) string {
				// Go pointer format: 0xc0000b4000
				ptr := regexp.MustCompile(`0x[0-9a-fA-F]+`)
				return ptr.ReplaceAllString(s, "[ADDR]")
			},
		},
		// Process ID normalization
		{
			Name:        "pid",
			Description: "Normalizes process IDs",
			Apply: func(s string) string {
				pid := regexp.MustCompile(`\bpid[=:]\s*\d+\b|\bpid\s+\d+\b`)
				return pid.ReplaceAllString(s, "[PID]")
			},
		},
		// Port number normalization
		{
			Name:        "port",
			Description: "Normalizes port numbers",
			Apply: func(s string) string {
				// Port patterns: :8080, port 8080, etc.
				port := regexp.MustCompile(`\bport[=:]?\s*\d{2,5}\b|:\d{2,5}\b`)
				return port.ReplaceAllString(s, "[PORT]")
			},
		},
	}
}

// JSONComparator compares JSON outputs
type JSONComparator struct {
	*Comparator
}

// NewJSONComparator creates a comparator for JSON outputs
func NewJSONComparator() *JSONComparator {
	return &JSONComparator{
		Comparator: NewComparator(),
	}
}

// CompareJSON compares two JSON strings
func (jc *JSONComparator) CompareJSON(expected, actual string) *ComparisonResult {
	// First normalize whitespace in JSON
	normalizeJSON := func(s string) string {
		// Remove extra whitespace while preserving structure
		s = strings.TrimSpace(s)
		// This is a simple normalization - for more complex JSON comparison,
		// consider using a proper JSON diff library
		return s
	}

	normalizedExpected := normalizeJSON(expected)
	normalizedActual := normalizeJSON(actual)

	return jc.Comparator.Compare(normalizedExpected, normalizedActual)
}