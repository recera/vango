// Package wasm implements a test harness for running WASM tests in headless Chrome.
// It uses the Chrome DevTools Protocol to execute TinyGo-compiled test binaries
// and stream results back to the test runner.
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Harness runs WASM tests in a headless Chrome browser
type Harness struct {
	mu           sync.Mutex
	chromePath   string
	chromeProc   *exec.Cmd
	debugURL     string
	wsConn       net.Conn
	httpServer   *http.Server
	serverPort   int
	results      chan TestResult
	outputWriter io.Writer
	verbose      bool
	timeout      time.Duration
}

// TestResult represents the outcome of a single test
type TestResult struct {
	Name     string        `json:"name"`
	Package  string        `json:"package"`
	Pass     bool          `json:"pass"`
	Skip     bool          `json:"skip"`
	Output   string        `json:"output"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// TestReport contains all test results
type TestReport struct {
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
	Results  []TestResult  `json:"results"`
}

// Config holds harness configuration
type Config struct {
	ChromePath   string        // Path to Chrome executable
	Verbose      bool          // Enable verbose output
	Timeout      time.Duration // Test timeout (default: 5 minutes)
	OutputWriter io.Writer     // Where to write test output (default: os.Stdout)
	Port         int           // HTTP server port (0 for random)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		ChromePath:   findChrome(),
		Verbose:      false,
		Timeout:      5 * time.Minute,
		OutputWriter: os.Stdout,
		Port:         0,
	}
}

// New creates a new test harness
func New(config Config) (*Harness, error) {
	if config.ChromePath == "" {
		config.ChromePath = findChrome()
		if config.ChromePath == "" {
			return nil, fmt.Errorf("Chrome/Chromium not found. Please install Chrome or set CHROME_PATH")
		}
	}

	if config.OutputWriter == nil {
		config.OutputWriter = os.Stdout
	}

	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	return &Harness{
		chromePath:   config.ChromePath,
		results:      make(chan TestResult, 100),
		outputWriter: config.OutputWriter,
		verbose:      config.Verbose,
		timeout:      config.Timeout,
		serverPort:   config.Port,
	}, nil
}

// Run executes WASM tests and returns a report
func (h *Harness) Run(wasmPath string) (*TestReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Start HTTP server to serve WASM and test runner
	if err := h.startHTTPServer(ctx, wasmPath); err != nil {
		return nil, fmt.Errorf("failed to start HTTP server: %w", err)
	}
	defer h.stopHTTPServer()

	// Start headless Chrome
	if err := h.startChrome(ctx); err != nil {
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}
	defer h.stopChrome()

	// Connect to Chrome DevTools
	if err := h.connectDevTools(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to DevTools: %w", err)
	}
	defer h.disconnect()

	// Navigate to test page
	testURL := fmt.Sprintf("http://localhost:%d/test.html", h.serverPort)
	if err := h.navigateToPage(ctx, testURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to test page: %w", err)
	}

	// Collect test results
	report := h.collectResults(ctx)
	return report, nil
}

// StreamOutput writes test output to the configured writer in real-time
func (h *Harness) StreamOutput(w io.Writer) error {
	if w == nil {
		w = h.outputWriter
	}

	for result := range h.results {
		if result.Skip {
			fmt.Fprintf(w, "SKIP: %s\n", result.Name)
		} else if result.Pass {
			fmt.Fprintf(w, "PASS: %s (%.3fs)\n", result.Name, result.Duration.Seconds())
		} else {
			fmt.Fprintf(w, "FAIL: %s\n", result.Name)
			if result.Error != "" {
				fmt.Fprintf(w, "  Error: %s\n", result.Error)
			}
		}

		if h.verbose && result.Output != "" {
			fmt.Fprintf(w, "  Output:\n%s\n", indent(result.Output, "    "))
		}
	}

	return nil
}

// Private methods

func (h *Harness) startHTTPServer(ctx context.Context, wasmPath string) error {
	// Find available port if not specified
	if h.serverPort == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return err
		}
		h.serverPort = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	mux := http.NewServeMux()

	// Serve WASM file
	mux.HandleFunc("/test.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(w, r, wasmPath)
	})

	// Serve wasm_exec.js
	mux.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(wasmExecJS))
	})

	// Serve test runner HTML
	mux.HandleFunc("/test.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testRunnerHTML))
	})

	// Test results endpoint
	mux.HandleFunc("/results", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var result TestResult
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		select {
		case h.results <- result:
			w.WriteHeader(http.StatusOK)
		case <-ctx.Done():
			http.Error(w, "Context cancelled", http.StatusServiceUnavailable)
		}
	})

	h.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.serverPort),
		Handler: mux,
	}

	go func() {
		if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (h *Harness) stopHTTPServer() {
	if h.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.httpServer.Shutdown(ctx)
	}
}

func (h *Harness) startChrome(ctx context.Context) error {
	// Chrome flags for headless mode
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--remote-debugging-port=9222",
		"--disable-web-security", // Allow WASM loading
		"--allow-file-access-from-files",
		"about:blank",
	}

	h.chromeProc = exec.CommandContext(ctx, h.chromePath, args...)
	
	if h.verbose {
		h.chromeProc.Stdout = os.Stdout
		h.chromeProc.Stderr = os.Stderr
	}

	if err := h.chromeProc.Start(); err != nil {
		return fmt.Errorf("failed to start Chrome: %w", err)
	}

	// Wait for Chrome to start
	time.Sleep(2 * time.Second)

	// Get debugging URL
	resp, err := http.Get("http://localhost:9222/json/version")
	if err != nil {
		return fmt.Errorf("failed to get Chrome debug URL: %w", err)
	}
	defer resp.Body.Close()

	var info struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to decode Chrome debug info: %w", err)
	}

	h.debugURL = info.WebSocketDebuggerURL
	return nil
}

func (h *Harness) stopChrome() {
	if h.chromeProc != nil && h.chromeProc.Process != nil {
		h.chromeProc.Process.Kill()
		h.chromeProc.Wait()
	}
}

func (h *Harness) connectDevTools(ctx context.Context) error {
	// Simple WebSocket connection (would use a proper CDP library in production)
	// For now, we'll use HTTP polling instead of WebSocket for simplicity
	return nil
}

func (h *Harness) disconnect() {
	if h.wsConn != nil {
		h.wsConn.Close()
	}
}

func (h *Harness) navigateToPage(ctx context.Context, url string) error {
	// Use Chrome DevTools HTTP API to navigate
	endpoint := "http://localhost:9222/json/new?" + url
	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to navigate to page: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (h *Harness) collectResults(ctx context.Context) *TestReport {
	report := &TestReport{
		Results: []TestResult{},
	}

	startTime := time.Now()
	timeout := time.After(h.timeout)

	for {
		select {
		case result := <-h.results:
			report.Results = append(report.Results, result)
			report.Total++
			
			if result.Skip {
				report.Skipped++
			} else if result.Pass {
				report.Passed++
			} else {
				report.Failed++
			}

			// Stream output if configured
			if h.outputWriter != nil {
				h.StreamOutput(h.outputWriter)
			}

		case <-timeout:
			// Timeout reached
			report.Duration = time.Since(startTime)
			return report

		case <-ctx.Done():
			// Context cancelled
			report.Duration = time.Since(startTime)
			return report
		}
	}
}

// Helper functions

func findChrome() string {
	// Check environment variable first
	if path := os.Getenv("CHROME_PATH"); path != "" {
		return path
	}

	// Common Chrome/Chromium paths
	paths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", // macOS
		"/Applications/Chromium.app/Contents/MacOS/Chromium",           // macOS
		"/usr/bin/google-chrome",                                       // Linux
		"/usr/bin/chromium-browser",                                    // Linux
		"/usr/bin/chromium",                                           // Linux
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",   // Windows
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", // Windows
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try to find using 'which' command
	if path, err := exec.LookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium-browser"); err == nil {
		return path
	}

	return ""
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// parseTestOutput parses Go test output format
func parseTestOutput(output string) []TestResult {
	var results []TestResult
	
	// Regular expressions for Go test output
	passRegex := regexp.MustCompile(`^(PASS|pass): (\S+) \(([0-9.]+)s\)`)
	failRegex := regexp.MustCompile(`^(FAIL|fail): (\S+)`)
	skipRegex := regexp.MustCompile(`^(SKIP|skip): (\S+)`)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if matches := passRegex.FindStringSubmatch(line); len(matches) > 0 {
			duration, _ := time.ParseDuration(matches[3] + "s")
			results = append(results, TestResult{
				Name:     matches[2],
				Pass:     true,
				Duration: duration,
			})
		} else if matches := failRegex.FindStringSubmatch(line); len(matches) > 0 {
			results = append(results, TestResult{
				Name: matches[2],
				Pass: false,
			})
		} else if matches := skipRegex.FindStringSubmatch(line); len(matches) > 0 {
			results = append(results, TestResult{
				Name: matches[2],
				Skip: true,
			})
		}
	}
	
	return results
}

// Embedded assets

const testRunnerHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>WASM Test Runner</title>
    <script src="/wasm_exec.js"></script>
</head>
<body>
    <h1>WASM Test Runner</h1>
    <div id="output"></div>
    <script>
        const go = new Go();
        
        // Capture console output
        const originalLog = console.log;
        const originalError = console.error;
        let testOutput = '';
        
        console.log = function(...args) {
            const msg = args.join(' ');
            testOutput += msg + '\n';
            originalLog.apply(console, args);
            
            // Parse and send test results
            if (msg.includes('PASS:') || msg.includes('FAIL:') || msg.includes('SKIP:')) {
                sendResult(msg);
            }
        };
        
        console.error = function(...args) {
            const msg = args.join(' ');
            testOutput += 'ERROR: ' + msg + '\n';
            originalError.apply(console, args);
        };
        
        function sendResult(output) {
            const result = parseTestLine(output);
            if (result) {
                fetch('/results', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(result)
                });
            }
        }
        
        function parseTestLine(line) {
            if (line.includes('PASS:')) {
                const match = line.match(/PASS:\s+(\S+)\s+\(([0-9.]+)s\)/);
                if (match) {
                    return {
                        name: match[1],
                        pass: true,
                        duration: parseFloat(match[2]) * 1000000000 // Convert to nanoseconds
                    };
                }
            } else if (line.includes('FAIL:')) {
                const match = line.match(/FAIL:\s+(\S+)/);
                if (match) {
                    return {
                        name: match[1],
                        pass: false
                    };
                }
            } else if (line.includes('SKIP:')) {
                const match = line.match(/SKIP:\s+(\S+)/);
                if (match) {
                    return {
                        name: match[1],
                        skip: true
                    };
                }
            }
            return null;
        }
        
        // Load and run WASM
        WebAssembly.instantiateStreaming(fetch("/test.wasm"), go.importObject).then((result) => {
            go.run(result.instance);
        }).catch((err) => {
            console.error('Failed to load WASM:', err);
            sendResult('FAIL: WASM_LOAD - ' + err.message);
        });
    </script>
</body>
</html>`

// Minimal wasm_exec.js for TinyGo (would be loaded from TinyGo in production)
const wasmExecJS = `/* wasm_exec.js - TinyGo runtime support */
// This would be the actual wasm_exec.js from TinyGo
// Placeholder for now - in production, serve the real file
`