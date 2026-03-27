// cline-test is a dual testing framework for comparing Go and TypeScript CLI behavior.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cline/cline/golang-cli/tests/dual"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define flags
	var (
		goBinary     = flag.String("go-binary", "", "Path to Go CLI binary (auto-detected if not specified)")
		tsBinary     = flag.String("ts-binary", "", "Path to TypeScript CLI binary (auto-detected if not specified)")
		outputFormat = flag.String("format", "text", "Output format: text, json, junit")
		outputPath   = flag.String("output", "", "Output file path (stdout if not specified)")
		parallel     = flag.Bool("parallel", true, "Run tests in parallel")
		verbose      = flag.Bool("verbose", false, "Enable verbose output")
		include      = flag.String("include", "", "Comma-separated list of test patterns to include")
		exclude      = flag.String("exclude", "", "Comma-separated list of test patterns to exclude")
		stopOnFail   = flag.Bool("stop-on-failure", false, "Stop on first failure")
		timeout      = flag.Duration("timeout", 30*time.Second, "Test timeout")
		compare      = flag.String("compare", "", "Compare specific command (format: 'args...')")
		baseline     = flag.String("baseline", "", "Load baseline report for regression detection")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Dual testing framework for Cline CLI parity testing.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Run all tests\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run specific tests\n")
		fmt.Fprintf(os.Stderr, "  %s -include='version*,config*'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Compare specific command\n")
		fmt.Fprintf(os.Stderr, "  %s -compare='version --json'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Detect regressions\n")
		fmt.Fprintf(os.Stderr, "  %s -baseline=baseline.json -format=json -output=report.json\n\n", os.Args[0])
	}

	flag.Parse()

	// Handle compare mode
	if *compare != "" {
		return runCompareMode(*goBinary, *tsBinary, *compare, *verbose)
	}

	// Parse include/exclude patterns
	var includePatterns, excludePatterns []string
	if *include != "" {
		includePatterns = strings.Split(*include, ",")
	}
	if *exclude != "" {
		excludePatterns = strings.Split(*exclude, ",")
	}

	// Create test runner config
	config := &dual.TestRunnerConfig{
		GoBinaryPath:  *goBinary,
		TSBinaryPath:  *tsBinary,
		OutputFormat:  *outputFormat,
		OutputPath:    *outputPath,
		Parallel:      *parallel,
		Verbose:       *verbose,
		IncludeTests:  includePatterns,
		ExcludeTests:  excludePatterns,
		StopOnFailure: *stopOnFail,
		Timeout:       *timeout,
	}

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Run tests
	runner := dual.NewTestRunner(config)
	report, err := runner.Run(ctx)
	if err != nil {
		return fmt.Errorf("test run failed: %w", err)
	}

	// Handle regression detection
	if *baseline != "" {
		return runRegressionDetection(ctx, report, *baseline, config)
	}

	// Exit with appropriate code
	if !report.Success {
		os.Exit(1)
	}

	return nil
}

func runCompareMode(goBinary, tsBinary, compareStr string, verbose bool) error {
	// Auto-detect binaries if not specified
	if goBinary == "" {
		goBinary = dual.FindGoBinary()
	}
	if tsBinary == "" {
		tsBinary = dual.FindTSBinary()
	}

	if goBinary == "" {
		return fmt.Errorf("Go CLI binary not found")
	}
	if tsBinary == "" {
		return fmt.Errorf("TypeScript CLI binary not found")
	}

	args := strings.Fields(compareStr)
	if len(args) == 0 {
		return fmt.Errorf("no command specified for comparison")
	}

	ctx := context.Background()
	tool := dual.NewComparisonTool(goBinary, tsBinary)
	tool.Verbose = verbose

	comparison, err := tool.CompareCommand(ctx, args)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	comparison.Print(os.Stdout, verbose)

	if !comparison.Match {
		os.Exit(1)
	}

	return nil
}

func runRegressionDetection(ctx context.Context, current *dual.DualTestReport, baselinePath string, config *dual.TestRunnerConfig) error {
	harness := dual.NewDualTestHarness(config.GoBinaryPath, config.TSBinaryPath)
	detector := dual.NewRegressionDetector(harness)

	if err := detector.LoadBaseline(baselinePath); err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}

	tests := dual.StandardDualTests()
	report, err := detector.DetectRegressions(ctx, tests)
	if err != nil {
		return fmt.Errorf("regression detection failed: %w", err)
	}

	report.Print(os.Stdout)

	os.Exit(report.ExitCode())
	return nil
}