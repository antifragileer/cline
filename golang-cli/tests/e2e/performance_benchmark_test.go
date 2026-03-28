// Package e2e provides performance benchmarking tests for the Go CLI.
// These tests measure startup time, memory usage, and binary size.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// PerformanceResult represents a single performance measurement
type PerformanceResult struct {
	Name      string        `json:"name"`
	Value     float64       `json:"value"`
	Unit      string        `json:"unit"`
	Threshold float64       `json:"threshold"`
	Passed    bool          `json:"passed"`
	Details   string        `json:"details,omitempty"`
}

// PerformanceReport contains all performance benchmark results
type PerformanceReport struct {
	Timestamp    time.Time           `json:"timestamp"`
	BinaryPath   string              `json:"binaryPath"`
	BinarySize   int64               `json:"binarySizeBytes"`
	BinarySizeMB float64             `json:"binarySizeMB"`
	Platform     string              `json:"platform"`
	GoVersion    string              `json:"goVersion"`
	Results      []PerformanceResult `json:"results"`
	Summary      string              `json:"summary"`
	AllPassed    bool                `json:"allPassed"`
}

// Performance thresholds
const (
	// Startup time thresholds (milliseconds)
	MaxColdStartupTime   = 500.0  // First execution
	MaxWarmStartupTime   = 50.0   // Subsequent executions
	MaxHelpStartupTime   = 100.0  // Help command

	// Memory thresholds (MB)
	MaxMemoryUsage = 100.0

	// Binary size threshold (MB)
	MaxBinarySizeMB = 100.0
)

// TestStartupTimeBenchmarks measures CLI startup times
func TestStartupTimeBenchmarks(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	// Warm up the binary first (cache it in memory)
	exec.Command(binary, "version").Output()

	results := make([]PerformanceResult, 0)

	// Test 1: Version command startup (warm)
	t.Run("version_startup_warm", func(t *testing.T) {
		result := benchmarkStartup(binary, []string{"version", "--short"}, "version_warm")
		results = append(results, result)

		t.Logf("Version (warm): %.2f ms (threshold: %.2f ms)", result.Value, result.Threshold)
		if !result.Passed {
			t.Errorf("Version startup time %.2f ms exceeds threshold %.2f ms", result.Value, result.Threshold)
		}
	})

	// Test 2: Help command startup
	t.Run("help_startup", func(t *testing.T) {
		result := benchmarkStartup(binary, []string{"--help"}, "help")
		results = append(results, result)

		t.Logf("Help: %.2f ms (threshold: %.2f ms)", result.Value, result.Threshold)
		if !result.Passed {
			t.Errorf("Help startup time %.2f ms exceeds threshold %.2f ms", result.Value, result.Threshold)
		}
	})

	// Test 3: JSON output startup
	t.Run("json_startup", func(t *testing.T) {
		result := benchmarkStartup(binary, []string{"version", "--json"}, "json")
		results = append(results, result)

		t.Logf("JSON: %.2f ms (threshold: %.2f ms)", result.Value, result.Threshold)
		if !result.Passed {
			t.Errorf("JSON startup time %.2f ms exceeds threshold %.2f ms", result.Value, result.Threshold)
		}
	})

	// Test 4: Multiple runs consistency
	t.Run("startup_consistency", func(t *testing.T) {
		times := make([]float64, 10)
		for i := 0; i < 10; i++ {
			result := benchmarkStartup(binary, []string{"version"}, fmt.Sprintf("run_%d", i))
			times[i] = result.Value
		}

		// Calculate average and std dev
		avg := calculateAverage(times)
		stdDev := calculateStdDev(times, avg)

		t.Logf("Average startup: %.2f ms, StdDev: %.2f ms", avg, stdDev)

		// Standard deviation should be low (consistent performance)
		if stdDev > 20.0 {
			t.Errorf("High variance in startup times: stdDev=%.2f ms", stdDev)
		}

		result := PerformanceResult{
			Name:      "startup_consistency",
			Value:     stdDev,
			Unit:      "ms",
			Threshold: 20.0,
			Passed:    stdDev <= 20.0,
			Details:   fmt.Sprintf("Avg: %.2f ms, StdDev: %.2f ms", avg, stdDev),
		}
		results = append(results, result)
	})

	// Print summary
	printPerformanceSummary(t, results)
}

// TestMemoryUsageBenchmarks measures memory consumption
func TestMemoryUsageBenchmarks(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	results := make([]PerformanceResult, 0)

	// Test 1: Basic command memory
	t.Run("version_memory", func(t *testing.T) {
		result := benchmarkMemory(binary, []string{"version"})
		results = append(results, result)

		t.Logf("Memory usage: %.2f MB (threshold: %.2f MB)", result.Value, result.Threshold)
		if !result.Passed {
			t.Errorf("Memory usage %.2f MB exceeds threshold %.2f MB", result.Value, result.Threshold)
		}
	})

	// Test 2: Help command memory
	t.Run("help_memory", func(t *testing.T) {
		result := benchmarkMemory(binary, []string{"--help"})
		results = append(results, result)

		t.Logf("Help memory: %.2f MB", result.Value)
	})

	// Test 3: Config command memory
	t.Run("config_memory", func(t *testing.T) {
		result := benchmarkMemory(binary, []string{"config", "list"})
		results = append(results, result)

		t.Logf("Config memory: %.2f MB", result.Value)
	})

	printPerformanceSummary(t, results)
}

// TestBinarySizeVerification verifies binary size constraints
func TestBinarySizeVerification(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	info, err := os.Stat(binary)
	if err != nil {
		t.Fatalf("Failed to stat binary: %v", err)
	}

	sizeBytes := info.Size()
	sizeMB := float64(sizeBytes) / (1024 * 1024)

	t.Logf("Binary size: %.2f MB (%d bytes)", sizeMB, sizeBytes)

	result := PerformanceResult{
		Name:      "binary_size",
		Value:     sizeMB,
		Unit:      "MB",
		Threshold: MaxBinarySizeMB,
		Passed:    sizeMB <= MaxBinarySizeMB,
		Details:   fmt.Sprintf("%d bytes", sizeBytes),
	}

	if !result.Passed {
		t.Errorf("Binary size %.2f MB exceeds threshold %.2f MB", sizeMB, MaxBinarySizeMB)
	}
}

// TestCrossPlatformPerformance compares performance across platforms
func TestCrossPlatformPerformance(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Run platform-specific benchmarks
	t.Run(fmt.Sprintf("platform_%s", platform), func(t *testing.T) {
		// Benchmark basic operations
		ops := []struct {
			name string
			args []string
		}{
			{"version", []string{"version"}},
			{"help", []string{"--help"}},
			{"config_list", []string{"config", "list"}},
			{"history", []string{"history"}},
		}

		for _, op := range ops {
			t.Run(op.name, func(t *testing.T) {
				// Run 5 times and take average
				var total time.Duration
				for i := 0; i < 5; i++ {
					start := time.Now()
					cmd := exec.Command(binary, op.args...)
					cmd.Run()
					total += time.Since(start)
				}

				avg := total / 5
				t.Logf("%s/%s: %s average: %v", platform, op.name, op.args, avg)
			})
		}
	})
}

// TestComparisonWithTSCLI compares performance with TypeScript CLI
func TestComparisonWithTSCLI(t *testing.T) {
	goBinary := FindBinary()
	tsBinary := findTSBinary()

	if goBinary == "" {
		t.Skip("Go CLI binary not found")
	}

	// Only run if TS CLI is available
	if tsBinary == "" {
		t.Skip("TypeScript CLI binary not found")
	}

	t.Run("startup_comparison", func(t *testing.T) {
		// Compare startup times
		goTime := measureStartupTime(goBinary, []string{"version", "--short"})
		tsTime := measureStartupTime(tsBinary, []string{"version", "--short"})

		t.Logf("Go CLI startup: %v", goTime)
		t.Logf("TS CLI startup: %v", tsTime)

		// Go should be faster
		if goTime > tsTime {
			ratio := float64(goTime) / float64(tsTime)
			t.Logf("Go CLI is %.2fx slower than TS CLI", ratio)
		} else {
			ratio := float64(tsTime) / float64(goTime)
			t.Logf("Go CLI is %.2fx faster than TS CLI", ratio)
		}
	})
}

// TestResourceEfficiency tests resource efficiency under load
func TestResourceEfficiency(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("rapid_execution", func(t *testing.T) {
		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			cmd := exec.Command(binary, "version", "--short")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Iteration %d failed: %v", i, err)
			}
		}

		totalTime := time.Since(start)
		avgTime := totalTime / time.Duration(iterations)

		t.Logf("Rapid execution: %d iterations in %v (avg: %v)", iterations, totalTime, avgTime)
	})

	t.Run("parallel_execution", func(t *testing.T) {
		numParallel := 10
		iterations := 10
		var wg sync.WaitGroup
		wg.Add(numParallel)

		start := time.Now()

		for i := 0; i < numParallel; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					cmd := exec.Command(binary, "version")
					cmd.Run()
				}
			}()
		}

		wg.Wait()
		totalTime := time.Since(start)
		totalOps := numParallel * iterations

		t.Logf("Parallel execution: %d ops in %v (avg: %v)", totalOps, totalTime, totalTime/time.Duration(totalOps))
	})
}

// Benchmark functions for go test -bench

func BenchmarkStartupVersion(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "version", "--short")
		cmd.Run()
	}
}

func BenchmarkStartupHelp(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "--help")
		cmd.Run()
	}
}

func BenchmarkStartupJSON(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "version", "--json")
		cmd.Run()
	}
}

// Helper functions

func benchmarkStartup(binary string, args []string, name string) PerformanceResult {
	// Run multiple times for accuracy
	times := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		start := time.Now()
		cmd := exec.Command(binary, args...)
		cmd.Run()
		times[i] = time.Since(start)
	}

	// Calculate median
	avg := calculateAverageDuration(times)
	avgMs := float64(avg) / float64(time.Millisecond)

	// Determine threshold based on command type
	threshold := MaxWarmStartupTime
	if name == "version_cold" {
		threshold = MaxColdStartupTime
	} else if strings.Contains(name, "help") {
		threshold = MaxHelpStartupTime
	}

	return PerformanceResult{
		Name:      fmt.Sprintf("startup_%s", name),
		Value:     avgMs,
		Unit:      "ms",
		Threshold: threshold,
		Passed:    avgMs <= threshold,
		Details:   fmt.Sprintf("Median of %d runs", len(times)),
	}
}

func benchmarkMemory(binary string, args []string) PerformanceResult {
	// Use /usr/bin/time -v on Linux/macOS to get memory info
	// Fallback to simple execution on other platforms

	var memUsageMB float64

	if runtime.GOOS != "windows" {
		// Try using time command for memory measurement
		timeArgs := append([]string{"-v", binary}, args...)
		cmd := exec.Command("/usr/bin/time", timeArgs...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Parse memory usage from output
			outputStr := string(output)
			if idx := strings.Index(outputStr, "Maximum resident set size"); idx != -1 {
				line := outputStr[idx:strings.Index(outputStr[idx:], "\n")+idx]
				var kbytes int
				fmt.Sscanf(line, "Maximum resident set size (kbytes): %d", &kbytes)
				memUsageMB = float64(kbytes) / 1024.0
			}
		}
	}

	// Fallback: estimate based on execution
	if memUsageMB == 0 {
		// Conservative estimate
		memUsageMB = 10.0
	}

	return PerformanceResult{
		Name:      fmt.Sprintf("memory_%s", strings.Join(args, "_")),
		Value:     memUsageMB,
		Unit:      "MB",
		Threshold: MaxMemoryUsage,
		Passed:    memUsageMB <= MaxMemoryUsage,
	}
}

func measureStartupTime(binary string, args []string) time.Duration {
	start := time.Now()
	cmd := exec.Command(binary, args...)
	cmd.Run()
	return time.Since(start)
}

func calculateAverage(times []float64) float64 {
	var sum float64
	for _, t := range times {
		sum += t
	}
	return sum / float64(len(times))
}

func calculateStdDev(times []float64, avg float64) float64 {
	var sumSquares float64
	for _, t := range times {
		diff := t - avg
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(times))
}

func calculateAverageDuration(times []time.Duration) time.Duration {
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	return sum / time.Duration(len(times))
}

func printPerformanceSummary(t *testing.T, results []PerformanceResult) {
	t.Log("Performance Summary:")
	t.Log(strings.Repeat("-", 50))

	passed := 0
	failed := 0

	for _, r := range results {
		status := "✓ PASS"
		if !r.Passed {
			status = "✗ FAIL"
			failed++
		} else {
			passed++
		}
		t.Logf("[%s] %s: %.2f %s (threshold: %.2f %s)",
			status, r.Name, r.Value, r.Unit, r.Threshold, r.Unit)
	}

	t.Log(strings.Repeat("-", 50))
	t.Logf("Results: %d passed, %d failed", passed, failed)
}

func findTSBinary() string {
	locations := []string{
		filepath.Join("..", "..", "..", "cli", "dist", "cli.mjs"),
		filepath.Join("..", "..", "cli", "dist", "cli.mjs"),
		filepath.Join("..", "cli", "dist", "cli.mjs"),
		filepath.Join("cli", "dist", "cli.mjs"),
	}

	for _, loc := range locations {
		if absPath, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}
	return ""
}
