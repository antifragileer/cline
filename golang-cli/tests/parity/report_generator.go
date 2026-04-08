// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Reporter generates test reports
type Reporter struct {
	outputDir string
}

// NewReporter creates a new reporter
func NewReporter(outputDir string) *Reporter {
	return &Reporter{
		outputDir: outputDir,
	}
}

// Generate creates test reports in multiple formats
func (r *Reporter) Generate(result *SuiteResult) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate Markdown report
	if err := r.generateMarkdown(result); err != nil {
		return fmt.Errorf("failed to generate Markdown report: %w", err)
	}

	// Generate JSON report
	if err := r.generateJSON(result); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	// Generate HTML report
	if err := r.generateHTML(result); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	// Generate JUnit XML report
	if err := r.generateJUnit(result); err != nil {
		return fmt.Errorf("failed to generate JUnit report: %w", err)
	}

	return nil
}

// GenerateMarkdown generates a Markdown report
func (r *Reporter) generateMarkdown(result *SuiteResult) error {
	filename := filepath.Join(r.outputDir, "parity_report.md")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# Cline CLI Parity Test Report\n\n")
	fmt.Fprintf(file, "**Generated:** %s\n\n", result.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(file, "**Duration:** %s\n\n", result.Duration.Round(time.Millisecond))

	// Summary
	fmt.Fprintf(file, "## Summary\n\n")
	fmt.Fprintf(file, "| Metric | Value |\n")
	fmt.Fprintf(file, "|--------|-------|\n")
	fmt.Fprintf(file, "| Total Tests | %d |\n", result.Total)
	fmt.Fprintf(file, "| Passed | %d ✅ |\n", result.Passed)
	fmt.Fprintf(file, "| Failed | %d ❌ |\n", result.Failed)
	fmt.Fprintf(file, "| Skipped | %d ⏭️ |\n", result.Skipped)
	fmt.Fprintf(file, "| Success Rate | %.1f%% |\n\n", result.SuccessRate*100)

	// Results by category
	fmt.Fprintf(file, "## Results by Category\n\n")
	categories := make(map[Category][]TestResult)
	for _, tr := range result.Results {
		categories[tr.Scenario.Category] = append(categories[tr.Scenario.Category], tr)
	}

	// Sort categories for consistent output
	var catKeys []Category
	for cat := range categories {
		catKeys = append(catKeys, cat)
	}
	sort.Slice(catKeys, func(i, j int) bool { return string(catKeys[i]) < string(catKeys[j]) })

	for _, cat := range catKeys {
		results := categories[cat]
		passCount := 0
		for _, r := range results {
			if r.Passed {
				passCount++
			}
		}
		fmt.Fprintf(file, "### %s\n\n", cat)
		fmt.Fprintf(file, "**Status:** %d/%d passed\n\n", passCount, len(results))
		fmt.Fprintf(file, "| Test | Status | Duration | Exit Code |\n")
		fmt.Fprintf(file, "|------|--------|----------|-----------|\n")
		for _, r := range results {
			status := "❌ FAIL"
			if r.Passed {
				status = "✅ PASS"
			} else if r.Skipped {
				status = "⏭️ SKIP"
			}
			fmt.Fprintf(file, "| %s | %s | %s | %d/%d |\n",
				r.Scenario.Name,
				status,
				r.Duration.Round(time.Millisecond),
				r.Actual.ExitCode,
				r.Scenario.ExpectExitCode,
			)
		}
		fmt.Fprintln(file)
	}

	// Failed tests details
	fmt.Fprintf(file, "## Failed Tests\n\n")
	hasFailures := false
	for _, tr := range result.Results {
		if !tr.Passed && !tr.Skipped {
			hasFailures = true
			fmt.Fprintf(file, "### %s (%s)\n\n", tr.Scenario.Name, tr.Scenario.Category)
			fmt.Fprintf(file, "**Description:** %s\n\n", tr.Scenario.Description)
			fmt.Fprintf(file, "**Command:** `cline %s`\n\n", strings.Join(tr.Scenario.Args, " "))
			fmt.Fprintf(file, "**Exit Codes:** Expected %d, Got %d\n\n",
				tr.Scenario.ExpectExitCode, tr.Actual.ExitCode)

			if tr.Comparison != nil && !tr.Comparison.Match {
				fmt.Fprintf(file, "**Output Diff:**\n\n```diff\n%s\n```\n\n", tr.Comparison.Diff)
			}

			if tr.Error != "" {
				fmt.Fprintf(file, "**Error:** %s\n\n", tr.Error)
			}

		fmt.Fprintln(file, "---")
		fmt.Fprintln(file)
	}
	}

	if !hasFailures {
		fmt.Fprintf(file, "All tests passed! 🎉\n\n")
	}

	// Environment info
	fmt.Fprintf(file, "## Environment\n\n")
	if result.NodeCLIInfo != nil {
		fmt.Fprintf(file, "### Node.js CLI\n")
		for k, v := range result.NodeCLIInfo {
			fmt.Fprintf(file, "- **%s:** %s\n", k, v)
		}
		fmt.Fprintln(file)
	}
	if result.GoLangCLIInfo != nil {
		fmt.Fprintf(file, "### GoLang CLI\n")
		for k, v := range result.GoLangCLIInfo {
			fmt.Fprintf(file, "- **%s:** %s\n", k, v)
		}
	}

	return nil
}

// generateJSON generates a JSON report
func (r *Reporter) generateJSON(result *SuiteResult) error {
	filename := filepath.Join(r.outputDir, "parity_report.json")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// generateHTML generates an HTML dashboard
func (r *Reporter) generateHTML(result *SuiteResult) error {
	filename := filepath.Join(r.outputDir, "parity_report.html")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Cline CLI Parity Test Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin: 20px 0; }
        .stat-box { background: #f8f9fa; padding: 15px; border-radius: 6px; text-align: center; }
        .stat-box .number { font-size: 32px; font-weight: bold; color: #333; }
        .stat-box .label { font-size: 14px; color: #666; margin-top: 5px; }
        .stat-box.pass .number { color: #28a745; }
        .stat-box.fail .number { color: #dc3545; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #f8f9fa; font-weight: 600; color: #555; }
        tr:hover { background: #f8f9fa; }
        .status-pass { color: #28a745; font-weight: 600; }
        .status-fail { color: #dc3545; font-weight: 600; }
        .status-skip { color: #6c757d; font-weight: 600; }
        .diff { background: #f8f9fa; padding: 15px; border-radius: 4px; font-family: monospace; white-space: pre-wrap; overflow-x: auto; }
        .diff .minus { color: #dc3545; }
        .diff .plus { color: #28a745; }
        .progress-bar { width: 100%; height: 30px; background: #e9ecef; border-radius: 4px; overflow: hidden; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #28a745 0%, #28a745 {{.SuccessRate}}%, #dc3545 {{.SuccessRate}}%, #dc3545 100%); }
        .timestamp { color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🧪 Cline CLI Parity Test Report</h1>
        <p class="timestamp">Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}} | Duration: {{.Duration}}</p>
        
        <div class="progress-bar">
            <div class="progress-fill"></div>
        </div>
        <p style="text-align: center; margin-top: 10px;">{{printf "%.1f" .SuccessRate}}% Success Rate</p>

        <div class="summary">
            <div class="stat-box">
                <div class="number">{{.Total}}</div>
                <div class="label">Total Tests</div>
            </div>
            <div class="stat-box pass">
                <div class="number">{{.Passed}}</div>
                <div class="label">Passed</div>
            </div>
            <div class="stat-box fail">
                <div class="number">{{.Failed}}</div>
                <div class="label">Failed</div>
            </div>
            <div class="stat-box">
                <div class="number">{{.Skipped}}</div>
                <div class="label">Skipped</div>
            </div>
        </div>

        <h2>Test Results</h2>
        <table>
            <thead>
                <tr>
                    <th>Category</th>
                    <th>Test Name</th>
                    <th>Status</th>
                    <th>Duration</th>
                    <th>Exit Code</th>
                </tr>
            </thead>
            <tbody>
                {{range .Results}}
                <tr>
                    <td>{{.Scenario.Category}}</td>
                    <td>{{.Scenario.Name}}</td>
                    <td class="{{if .Passed}}status-pass{{else if .Skipped}}status-skip{{else}}status-fail{{end}}">
                        {{if .Passed}}✅ PASS{{else if .Skipped}}⏭️ SKIP{{else}}❌ FAIL{{end}}
                    </td>
                    <td>{{.Duration}}</td>
                    <td>{{.Actual.ExitCode}} / {{.Scenario.ExpectExitCode}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <h2>Failed Tests</h2>
        {{$hasFailures := false}}
        {{range .Results}}{{if and (not .Passed) (not .Skipped)}}{{$hasFailures = true}}{{end}}{{end}}
        
        {{if not $hasFailures}}
        <p style="color: #28a745; font-weight: 600;">🎉 All tests passed!</p>
        {{else}}
        {{range .Results}}
        {{if and (not .Passed) (not .Skipped)}}
        <div style="margin: 20px 0; padding: 20px; background: #fff5f5; border-radius: 6px; border-left: 4px solid #dc3545;">
            <h3>{{.Scenario.Name}} ({{.Scenario.Category}})</h3>
            <p><strong>Description:</strong> {{.Scenario.Description}}</p>
            <p><strong>Command:</strong> <code>cline {{.Scenario.Args}}</code></p>
            <p><strong>Exit Codes:</strong> Expected {{.Scenario.ExpectExitCode}}, Got {{.Actual.ExitCode}}</p>
            {{if .Comparison}}
            <div class="diff">{{.Comparison.Diff}}</div>
            {{end}}
            {{if .Error}}
            <p style="color: #dc3545;"><strong>Error:</strong> {{.Error}}</p>
            {{end}}
        </div>
        {{end}}
        {{end}}
        {{end}}
    </div>
</body>
</html>
`

	type templateData struct {
		*SuiteResult
		SuccessRate float64
	}

	data := templateData{
		SuiteResult: result,
		SuccessRate: result.SuccessRate * 100,
	}

	t := template.Must(template.New("report").Parse(tmpl))
	return t.Execute(file, data)
}

// JUnitXMLReport represents a JUnit XML report structure
type JUnitXMLReport struct {
	XMLName   xml.Name         `xml:"testsuites"`
	Name      string           `xml:"name,attr"`
	Tests     int              `xml:"tests,attr"`
	Failures  int              `xml:"failures,attr"`
	Errors    int              `xml:"errors,attr"`
	Time      float64          `xml:"time,attr"`
	TestSuite []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite in JUnit format
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a test case in JUnit format
type JUnitTestCase struct {
	Name      string           `xml:"name,attr"`
	ClassName string           `xml:"classname,attr"`
	Time      float64          `xml:"time,attr"`
	Failure   *JUnitFailure    `xml:"failure,omitempty"`
	Skipped   *JUnitSkipped    `xml:"skipped,omitempty"`
	SystemOut string           `xml:"system-out,omitempty"`
}

// JUnitFailure represents a test failure in JUnit format
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test in JUnit format
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// generateJUnit generates a JUnit XML report
func (r *Reporter) generateJUnit(result *SuiteResult) error {
	filename := filepath.Join(r.outputDir, "parity_report.xml")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Group results by category
	categories := make(map[Category][]TestResult)
	for _, tr := range result.Results {
		categories[tr.Scenario.Category] = append(categories[tr.Scenario.Category], tr)
	}

	var suites []JUnitTestSuite
	for cat, results := range categories {
		suite := JUnitTestSuite{
			Name:  string(cat),
			Tests: len(results),
			Time:  result.Duration.Seconds(),
		}

		for _, tr := range results {
			tc := JUnitTestCase{
				Name:      tr.Scenario.Name,
				ClassName: fmt.Sprintf("parity.%s", cat),
				Time:      tr.Duration.Seconds(),
			}

			if tr.Skipped {
				suite.Tests--
				tc.Skipped = &JUnitSkipped{Message: "Test skipped"}
			} else if !tr.Passed {
				suite.Failures++
				tc.Failure = &JUnitFailure{
					Message: fmt.Sprintf("Exit code mismatch: expected %d, got %d",
						tr.Scenario.ExpectExitCode, tr.Actual.ExitCode),
					Type:    "AssertionError",
					Content: tr.Comparison.Diff,
				}
			}

			suite.TestCases = append(suite.TestCases, tc)
		}

		suites = append(suites, suite)
	}

	report := JUnitXMLReport{
		Name:      "Cline CLI Parity Tests",
		Tests:     result.Total,
		Failures:  result.Failed,
		Errors:    0,
		Time:      result.Duration.Seconds(),
		TestSuite: suites,
	}

	fmt.Fprintf(file, xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	return encoder.Encode(report)
}

// ConsoleReporter prints results to console
type ConsoleReporter struct{}

// NewConsoleReporter creates a new console reporter
func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{}
}

// PrintSummary prints a summary to stdout
func (c *ConsoleReporter) PrintSummary(result *SuiteResult) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          Cline CLI Parity Test Results                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Statistics
	fmt.Printf("  Total Tests:  %d\n", result.Total)
	fmt.Printf("  Passed:       %d ✅\n", result.Passed)
	fmt.Printf("  Failed:       %d ❌\n", result.Failed)
	fmt.Printf("  Skipped:      %d ⏭️\n", result.Skipped)
	fmt.Printf("  Success Rate: %.1f%%\n", result.SuccessRate*100)
	fmt.Printf("  Duration:     %s\n", result.Duration.Round(time.Millisecond))
	fmt.Println()

	// Failed tests
	if result.Failed > 0 {
		fmt.Println("Failed Tests:")
		for _, tr := range result.Results {
			if !tr.Passed && !tr.Skipped {
				fmt.Printf("  ❌ %s/%s\n", tr.Scenario.Category, tr.Scenario.Name)
				if tr.Error != "" {
					fmt.Printf("     Error: %s\n", tr.Error)
				}
			}
		}
		fmt.Println()
	}

	// Overall status
	if result.Failed == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Printf("❌ %d test(s) failed\n", result.Failed)
	}
	fmt.Println()
}