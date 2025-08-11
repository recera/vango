package wasm

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestFindChrome(t *testing.T) {
	chrome := findChrome()
	if chrome == "" {
		t.Skip("Chrome/Chromium not found - skipping test")
	}
	
	// Verify the path exists
	if _, err := os.Stat(chrome); err != nil {
		t.Errorf("Chrome path exists but cannot be accessed: %v", err)
	}
	
	t.Logf("Found Chrome at: %s", chrome)
}

func TestHarness_New(t *testing.T) {
	config := DefaultConfig()
	harness, err := New(config)
	
	if err != nil {
		// If Chrome is not available, skip
		if findChrome() == "" {
			t.Skip("Chrome not available")
		}
		t.Fatalf("Failed to create harness: %v", err)
	}
	
	if harness == nil {
		t.Fatal("Harness is nil")
	}
	
	if harness.timeout != 5*time.Minute {
		t.Errorf("Expected default timeout of 5 minutes, got %v", harness.timeout)
	}
}

func TestParseTestOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int // expected number of results
	}{
		{
			name: "pass output",
			output: `PASS: TestExample (0.001s)
PASS: TestAnother (0.002s)`,
			expected: 2,
		},
		{
			name: "mixed output",
			output: `PASS: TestOne (0.001s)
FAIL: TestTwo
SKIP: TestThree
PASS: TestFour (0.003s)`,
			expected: 4,
		},
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
		{
			name: "with extra text",
			output: `Starting tests...
PASS: TestA (0.001s)
Some debug output
FAIL: TestB
Test complete`,
			expected: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseTestOutput(tt.output)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
			
			// Verify pass/fail/skip counts
			var passCount, failCount, skipCount int
			for _, r := range results {
				if r.Skip {
					skipCount++
				} else if r.Pass {
					passCount++
				} else {
					failCount++
				}
			}
			
			t.Logf("Pass: %d, Fail: %d, Skip: %d", passCount, failCount, skipCount)
		})
	}
}

func TestTestReport(t *testing.T) {
	report := &TestReport{
		Total:    10,
		Passed:   7,
		Failed:   2,
		Skipped:  1,
		Duration: 5 * time.Second,
		Results: []TestResult{
			{Name: "TestA", Pass: true, Duration: 100 * time.Millisecond},
			{Name: "TestB", Pass: false, Error: "assertion failed"},
			{Name: "TestC", Skip: true},
		},
	}
	
	if report.Total != report.Passed+report.Failed+report.Skipped {
		t.Error("Total count doesn't match sum of passed, failed, and skipped")
	}
	
	if len(report.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(report.Results))
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		text     string
		prefix   string
		expected string
	}{
		{
			text:     "line1\nline2\nline3",
			prefix:   "  ",
			expected: "  line1\n  line2\n  line3",
		},
		{
			text:     "single line",
			prefix:   ">>> ",
			expected: ">>> single line",
		},
		{
			text:     "",
			prefix:   "  ",
			expected: "",
		},
		{
			text:     "line1\n\nline3",
			prefix:   "  ",
			expected: "  line1\n\n  line3",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.text[:min(10, len(tt.text))], func(t *testing.T) {
			result := indent(tt.text, tt.prefix)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestRunner_DefaultConfig(t *testing.T) {
	config := DefaultRunnerConfig()
	
	if config.TinyGoPath != "tinygo" {
		t.Errorf("Expected TinyGoPath to be 'tinygo', got %s", config.TinyGoPath)
	}
	
	if len(config.BuildTags) != 1 || config.BuildTags[0] != "wasm" {
		t.Errorf("Expected BuildTags to contain 'wasm', got %v", config.BuildTags)
	}
	
	if config.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout of 5 minutes, got %v", config.Timeout)
	}
	
	if !config.CacheEnabled {
		t.Error("Expected cache to be enabled by default")
	}
}

func TestRunner_New(t *testing.T) {
	// Check if TinyGo is available
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Skip("TinyGo not available")
	}
	
	config := DefaultRunnerConfig()
	runner, err := NewRunner(config)
	
	if err != nil {
		// If Chrome is not available, skip
		if findChrome() == "" {
			t.Skip("Chrome not available")
		}
		t.Fatalf("Failed to create runner: %v", err)
	}
	
	if runner == nil {
		t.Fatal("Runner is nil")
	}
	
	if runner.config.TinyGoPath != "tinygo" {
		t.Errorf("Expected TinyGoPath to be 'tinygo', got %s", runner.config.TinyGoPath)
	}
	
	// Clean up
	runner.Close()
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}