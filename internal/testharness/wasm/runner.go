// Package wasm provides a test runner for executing WASM tests.
package wasm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner executes WASM tests using the test harness
type Runner struct {
	harness *Harness
	config  RunnerConfig
}

// RunnerConfig holds runner configuration
type RunnerConfig struct {
	TinyGoPath   string        // Path to TinyGo compiler
	BuildTags    []string      // Build tags to pass to TinyGo
	BuildFlags   []string      // Additional build flags
	TestFlags    []string      // Flags to pass to test binary
	Verbose      bool          // Enable verbose output
	Timeout      time.Duration // Test timeout
	WorkDir      string        // Working directory for builds
	CacheEnabled bool          // Enable build caching
}

// DefaultRunnerConfig returns default runner configuration
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		TinyGoPath:   "tinygo",
		BuildTags:    []string{"wasm"},
		BuildFlags:   []string{"-opt", "2", "-no-debug"},
		TestFlags:    []string{},
		Verbose:      false,
		Timeout:      5 * time.Minute,
		WorkDir:      os.TempDir(),
		CacheEnabled: true,
	}
}

// NewRunner creates a new test runner
func NewRunner(config RunnerConfig) (*Runner, error) {
	if config.TinyGoPath == "" {
		config.TinyGoPath = "tinygo"
	}

	// Verify TinyGo is available
	if _, err := exec.LookPath(config.TinyGoPath); err != nil {
		return nil, fmt.Errorf("TinyGo not found: %w", err)
	}

	// Create harness
	harness, err := New(Config{
		Verbose: config.Verbose,
		Timeout: config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create harness: %w", err)
	}

	return &Runner{
		harness: harness,
		config:  config,
	}, nil
}

// RunPackage runs tests for a Go package
func (r *Runner) RunPackage(packagePath string) (*TestReport, error) {
	// Build test WASM
	wasmPath, err := r.buildTestWASM(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to build test WASM: %w", err)
	}
	defer os.Remove(wasmPath) // Clean up

	// Run tests in harness
	return r.harness.Run(wasmPath)
}

// RunFile runs tests for a specific test file
func (r *Runner) RunFile(testFile string) (*TestReport, error) {
	// Build test WASM for specific file
	wasmPath, err := r.buildTestWASMForFile(testFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build test WASM: %w", err)
	}
	defer os.Remove(wasmPath)

	// Run tests in harness
	return r.harness.Run(wasmPath)
}

// buildTestWASM compiles test code to WASM
func (r *Runner) buildTestWASM(packagePath string) (string, error) {
	// Generate output path
	wasmPath := filepath.Join(r.config.WorkDir, fmt.Sprintf("test_%d.wasm", time.Now().UnixNano()))

	// Build TinyGo command
	args := []string{"test", "-c"} // -c compiles but doesn't run
	args = append(args, "-o", wasmPath)
	args = append(args, "-target", "wasm")

	// Add build tags
	if len(r.config.BuildTags) > 0 {
		args = append(args, "-tags", strings.Join(r.config.BuildTags, " "))
	}

	// Add build flags
	args = append(args, r.config.BuildFlags...)

	// Add package path
	args = append(args, packagePath)

	// Add test flags
	if len(r.config.TestFlags) > 0 {
		args = append(args, r.config.TestFlags...)
	}

	cmd := exec.Command(r.config.TinyGoPath, args...)
	cmd.Dir = packagePath

	if r.config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("Building WASM: %s %s\n", r.config.TinyGoPath, strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("TinyGo build failed: %w", err)
	}

	return wasmPath, nil
}

// buildTestWASMForFile compiles a specific test file to WASM
func (r *Runner) buildTestWASMForFile(testFile string) (string, error) {
	// Generate output path
	wasmPath := filepath.Join(r.config.WorkDir, fmt.Sprintf("test_%d.wasm", time.Now().UnixNano()))

	// Build TinyGo command
	args := []string{"test", "-c"}
	args = append(args, "-o", wasmPath)
	args = append(args, "-target", "wasm")

	// Add build tags
	if len(r.config.BuildTags) > 0 {
		args = append(args, "-tags", strings.Join(r.config.BuildTags, " "))
	}

	// Add build flags
	args = append(args, r.config.BuildFlags...)

	// Add specific test file
	args = append(args, testFile)

	cmd := exec.Command(r.config.TinyGoPath, args...)
	cmd.Dir = filepath.Dir(testFile)

	if r.config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("TinyGo build failed: %w", err)
	}

	return wasmPath, nil
}

// StreamOutput streams test output to a writer
func (r *Runner) StreamOutput(w io.Writer) error {
	return r.harness.StreamOutput(w)
}

// Close cleans up resources
func (r *Runner) Close() error {
	// Clean up any temporary files
	if r.config.WorkDir != "" && r.config.WorkDir != os.TempDir() {
		os.RemoveAll(r.config.WorkDir)
	}
	return nil
}