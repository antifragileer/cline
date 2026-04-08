// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Execution represents the result of executing a CLI command
type Execution struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// TestResult represents the result of a single parity test
type TestResult struct {
	Scenario   Scenario
	Expected   *Execution
	Actual     *Execution
	Comparison *ComparisonResult
	Passed     bool
	Skipped    bool
	Error      string
	Duration   time.Duration
}

// SuiteResult represents the result of running all tests in a suite
type SuiteResult struct {
	Results      []TestResult
	Total        int
	Passed       int
	Failed       int
	Skipped      int
	SuccessRate  float64
	Duration     time.Duration
	Timestamp    time.Time
	NodeCLIInfo  map[string]string
	GoLangCLIInfo map[string]string
}

// Suite orchestrates parity tests between Node.js and GoLang CLIs
type Suite struct {
	nodeCli       *NodeCLIAdapter
	golangCli     *GoLangCLIAdapter
	comparator    *Comparator
	reporter      *Reporter
	registry      *Registry
	outputDir     string
	baselineDir   string
}

// SuiteOptions contains options for creating a new suite
type SuiteOptions struct {
	NodeCliPath    string
	GoLangCliPath  string
	OutputDir      string
	BaselineDir    string
	CustomNormalizers []Normalizer
}

// NewSuite creates a new parity test suite
func NewSuite(opts SuiteOptions) (*Suite, error) {
	// Initialize Node.js CLI adapter
	var nodeCli *NodeCLIAdapter
	var err error
	if opts.NodeCliPath != "" {
		nodeCli, err = NewNodeCLIAdapterWithPath(opts.NodeCliPath)
	} else {
		nodeCli, err = NewNodeCLIAdapter()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Node.js CLI adapter: %w", err)
	}

	// Initialize GoLang CLI adapter
	var golangCli *GoLangCLIAdapter
	if opts.GoLangCliPath != "" {
		golangCli, err = NewGoLangCLIAdapterWithPath(opts.GoLangCliPath)
	} else {
		golangCli, err = NewGoLangCLIAdapter()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GoLang CLI adapter: %w", err)
	}

	// Build GoLang CLI if needed
	if !golangCli.IsAvailable() {
		if err := golangCli.Build(); err != nil {
			return nil, fmt.Errorf("failed to build GoLang CLI: %w", err)
		}
	}

	// Initialize comparator
	var comparator *Comparator
	if len(opts.CustomNormalizers) > 0 {
		comparator = NewComparatorWith(opts.CustomNormalizers)
	} else {
		comparator = NewComparator()
	}

	// Set default directories
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "parity-reports"
	}
	
	baselineDir := opts.BaselineDir
	if baselineDir == "" {
		baselineDir = "baselines"
	}

	return &Suite{
		nodeCli:     nodeCli,
		golangCli:   golangCli,
		comparator:  comparator,
		reporter:    NewReporter(outputDir),
		registry:    NewRegistry(),
		outputDir:   outputDir,
		baselineDir: baselineDir,
	}, nil
}

// RunScenario executes a single test scenario against both CLIs
func (s *Suite) RunScenario(scenario Scenario) (*TestResult, error) {
	result := &TestResult{
		Scenario: scenario,
		Duration: 0,
	}

	// Record start time
	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	// Run Node.js CLI
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
	expected, err := s.nodeCli.Execute(ctx, scenario.Args, scenario.Stdin)
	cancel()

	if err != nil {
		result.Error = fmt.Sprintf("Node.js CLI execution failed: %v", err)
		return result, nil
	}
	result.Expected = expected

	// Run GoLang CLI
	ctx, cancel = context.WithTimeout(context.Background(), scenario.Timeout)
	actual, err := s.golangCli.Execute(ctx, scenario.Args, scenario.Stdin)
	cancel()

	if err != nil {
		result.Error = fmt.Sprintf("GoLang CLI execution failed: %v", err)
		return result, nil
	}
	result.Actual = actual

	// Compare exit codes
	if actual.ExitCode != scenario.ExpectExitCode {
		result.Error = fmt.Sprintf("exit code mismatch: expected %d, got %d", 
			scenario.ExpectExitCode, actual.ExitCode)
		return result, nil
	}

	// Compare outputs (if we have a baseline comparison)
	if scenario.ExpectBaseline {
		comparison := s.comparator.CompareWithSkip(expected.Stdout, actual.Stdout, scenario.SkipNormalization)
		result.Comparison = comparison
		result.Passed = comparison.Match && actual.ExitCode == scenario.ExpectExitCode
	} else {
		// For non-baseline tests, just check exit code
		result.Passed = actual.ExitCode == scenario.ExpectExitCode
	}

	return result, nil
}

// RunAll executes all registered scenarios
func (s *Suite) RunAll() (*SuiteResult, error) {
	return s.RunScenarios(s.registry.GetAll())
}

// RunScenarios executes a specific set of scenarios
func (s *Suite) RunScenarios(scenarios []Scenario) (*SuiteResult, error) {
	result := &SuiteResult{
		Timestamp: time.Now(),
		Results:   make([]TestResult, 0, len(scenarios)),
	}

	// Get CLI info for reporting
	var err error
	result.NodeCLIInfo, err = GetNodeCLIInfo()
	if err != nil {
		result.NodeCLIInfo = map[string]string{"error": err.Error()}
	}

	result.GoLangCLIInfo, err = GetGoLangCLIInfo()
	if err != nil {
		result.GoLangCLIInfo = map[string]string{"error": err.Error()}
	}

	// Record start time
	start := time.Now()

	// Execute all scenarios
	for _, scenario := range scenarios {
		testResult, err := s.RunScenario(scenario)
		if err != nil {
			testResult.Error = err.Error()
		}
		result.Results = append(result.Results, *testResult)
	}

	result.Duration = time.Since(start)

	// Calculate statistics
	for _, tr := range result.Results {
		result.Total++
		if tr.Passed {
			result.Passed++
		} else if tr.Skipped {
			result.Skipped++
		} else {
			result.Failed++
		}
	}

	if result.Total > 0 {
		result.SuccessRate = float64(result.Passed) / float64(result.Total-result.Skipped)
	}

	return result, nil
}

// RunCategory executes all scenarios in a category
func (s *Suite) RunCategory(cat Category) (*SuiteResult, error) {
	scenarios := s.registry.GetByCategory(cat)
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios found for category: %s", cat)
	}
	return s.RunScenarios(scenarios)
}

// RunByName executes a specific scenario by name
func (s *Suite) RunByName(name string) (*TestResult, error) {
	scenario, found := s.registry.GetByName(name)
	if !found {
		return nil, fmt.Errorf("scenario not found: %s", name)
	}
	return s.RunScenario(scenario)
}

// CaptureBaselines captures expected outputs from Node.js CLI
func (s *Suite) CaptureBaselines(scenarios []Scenario) error {
	if err := os.MkdirAll(s.baselineDir, 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	for _, scenario := range scenarios {
		if !scenario.ExpectBaseline {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
		result, err := s.nodeCli.Execute(ctx, scenario.Args, scenario.Stdin)
		cancel()

		if err != nil {
			fmt.Printf("Warning: failed to capture baseline for %s: %v\n", scenario.Name, err)
			continue
		}

		// Normalize output before saving (apply all normalizers except skipped ones)
		comparison := s.comparator.CompareWithSkip(result.Stdout, result.Stdout, scenario.SkipNormalization)
		normalized := comparison.Expected

		filename := GetScenarioFilename(scenario, "txt")
		filepath := filepath.Join(s.baselineDir, filename)

		if err := os.WriteFile(filepath, []byte(normalized), 0644); err != nil {
			fmt.Printf("Warning: failed to write baseline for %s: %v\n", scenario.Name, err)
			continue
		}

		fmt.Printf("Captured baseline: %s\n", filename)
	}

	return nil
}

// CompareWithBaseline compares GoLang CLI output with saved baseline
func (s *Suite) CompareWithBaseline(scenario Scenario) (*TestResult, error) {
	result := &TestResult{
		Scenario: scenario,
	}

	// Read baseline
	filename := GetScenarioFilename(scenario, "txt")
	baselinePath := filepath.Join(s.baselineDir, filename)

	baseline, err := os.ReadFile(baselinePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read baseline: %v", err)
		return result, nil
	}

	// Run GoLang CLI
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
	actual, err := s.golangCli.Execute(ctx, scenario.Args, scenario.Stdin)
	cancel()

	if err != nil {
		result.Error = fmt.Sprintf("GoLang CLI execution failed: %v", err)
		return result, nil
	}
	result.Actual = actual

	// Compare
	comparison := s.comparator.CompareWithSkip(string(baseline), actual.Stdout, scenario.SkipNormalization)
	result.Comparison = comparison
	result.Passed = comparison.Match && actual.ExitCode == scenario.ExpectExitCode

	return result, nil
}

// GetRegistry returns the scenario registry
func (s *Suite) GetRegistry() *Registry {
	return s.registry
}

// GetReporter returns the reporter
func (s *Suite) GetReporter() *Reporter {
	return s.reporter
}

// GetNodeCLI returns the Node.js CLI adapter
func (s *Suite) GetNodeCLI() *NodeCLIAdapter {
	return s.nodeCli
}

// GetGoLangCLI returns the GoLang CLI adapter
func (s *Suite) GetGoLangCLI() *GoLangCLIAdapter {
	return s.golangCli
}

// SuitePool manages concurrent suite execution with rate limiting
type SuitePool struct {
	suites     []*Suite
	semaphore  chan struct{}
	rateLimiter *time.Ticker
	mu         sync.Mutex
	results    []TestResult
}

// NewSuitePool creates a new suite pool for parallel execution
func NewSuitePool(suites []*Suite, maxConcurrent int) *SuitePool {
	return &SuitePool{
		suites:     suites,
		semaphore:  make(chan struct{}, maxConcurrent),
		rateLimiter: time.NewTicker(time.Second / 10), // 10 requests per second max
		results:    make([]TestResult, 0),
	}
}

// Run executes all suites in the pool with concurrency control
func (p *SuitePool) Run(scenarios []Scenario) (*SuiteResult, error) {
	var wg sync.WaitGroup
	resultChan := make(chan TestResult, len(scenarios)*len(p.suites))

	for _, suite := range p.suites {
		for _, scenario := range scenarios {
			wg.Add(1)
			go func(s *Suite, sc Scenario) {
				defer wg.Done()
				
				// Rate limiting
				<-p.rateLimiter.C
				
				// Concurrency control
				p.semaphore <- struct{}{}
				defer func() { <-p.semaphore }()

				testResult, err := s.RunScenario(sc)
				if err != nil {
					testResult.Error = err.Error()
				}
				resultChan <- *testResult
			}(suite, scenario)
		}
	}

	// Close channel when all done
	go func() {
		wg.Wait()
		close(resultChan)
		p.rateLimiter.Stop()
	}()

	// Collect results
	suiteResult := &SuiteResult{
		Timestamp: time.Now(),
	}

	for tr := range resultChan {
		suiteResult.Results = append(suiteResult.Results, tr)
		suiteResult.Total++
		if tr.Passed {
			suiteResult.Passed++
		} else if tr.Skipped {
			suiteResult.Skipped++
		} else {
			suiteResult.Failed++
		}
	}

	if suiteResult.Total > 0 {
		suiteResult.SuccessRate = float64(suiteResult.Passed) / float64(suiteResult.Total-suiteResult.Skipped)
	}

	return suiteResult, nil
}

// ExecutionConfig provides configuration for test execution
type ExecutionConfig struct {
	Parallelism    int           // Number of concurrent tests (1 for AI tests)
	Timeout        time.Duration // Default timeout
	RateLimit      int           // Max requests per second
	StopOnFailure  bool          // Stop on first failure
	CaptureOutput  bool          // Capture and compare output
	SkipCategories []Category    // Categories to skip
}

// DefaultExecutionConfig returns default execution configuration
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		Parallelism:   1, // Serial by default for AI tests
		Timeout:       60 * time.Second,
		RateLimit:     10,
		StopOnFailure: false,
		CaptureOutput: true,
	}
}

// ExecuteWithConfig runs tests with specific configuration
func (s *Suite) ExecuteWithConfig(config ExecutionConfig) (*SuiteResult, error) {
	scenarios := s.registry.GetAll()
	
	// Filter out skipped categories
	var filtered []Scenario
	for _, sc := range scenarios {
		skip := false
		for _, cat := range config.SkipCategories {
			if sc.Category == cat {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, sc)
		}
	}

	if config.Parallelism <= 1 {
		return s.RunScenarios(filtered)
	}

	// Parallel execution
	pool := NewSuitePool([]*Suite{s}, config.Parallelism)
	return pool.Run(filtered)
}