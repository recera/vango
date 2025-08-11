package tailwind

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config holds Tailwind configuration
type Config struct {
	ConfigPath   string // Path to tailwind.config.js
	InputPath    string // Path to input CSS file
	OutputPath   string // Path to output CSS file
	Watch        bool   // Whether to run in watch mode
	Strategy     string // auto|npm|standalone|vendor (vendor reserved)
	Version      string // version for standalone
	AutoDownload bool   // download standalone if missing
}

// Runner manages the Tailwind CSS process
type Runner struct {
	config    Config
	cmd       *exec.Cmd
	mu        sync.Mutex
	running   bool
	stopChan  chan struct{}
	doneChan  chan struct{}
	lastBuild time.Time
}

// NewRunner creates a new Tailwind runner
func NewRunner(config Config) *Runner {
	// Set defaults if not provided
	if config.ConfigPath == "" {
		config.ConfigPath = "tailwind.config.js"
	}
	if config.InputPath == "" {
		// Try common locations
		if _, err := os.Stat("app/styles/input.css"); err == nil {
			config.InputPath = "app/styles/input.css"
		} else if _, err := os.Stat("styles/input.css"); err == nil {
			config.InputPath = "styles/input.css"
		} else {
			config.InputPath = "./styles/input.css"
		}
	}
	if config.OutputPath == "" {
		config.OutputPath = "public/styles.css"
	}

	return &Runner{
		config:   config,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// BuildOnce performs a single, blocking Tailwind CSS build using the configured
// input/output/config paths. It does not start the watcher.
func (r *Runner) BuildOnce() error {
	// Ensure output directory exists
	outputDir := filepath.Dir(r.config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Resolve executable
	tailwindCmd, err := r.findTailwindExecutable()
	if err != nil {
		return fmt.Errorf("failed to find Tailwind CSS: %w", err)
	}
	// Build args without --watch
	args := []string{"-i", r.config.InputPath, "-o", r.config.OutputPath}
	if r.config.ConfigPath != "" {
		args = append(args, "-c", r.config.ConfigPath)
	}
	if os.Getenv("VANGO_ENV") == "production" {
		args = append(args, "--minify")
	}
	cmd := exec.Command(tailwindCmd[0], append(tailwindCmd[1:], args...)...)
	// Stream output minimally
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tailwind build failed: %w\n%s", err, string(out))
	}
	log.Println("🎨 Tailwind initial build complete")
	return nil
}

// Start starts the Tailwind process
func (r *Runner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("tailwind runner is already running")
	}

	// Check if Tailwind config exists
	if _, err := os.Stat(r.config.ConfigPath); os.IsNotExist(err) {
		log.Printf("⚠️  Tailwind config not found at %s, skipping Tailwind compilation", r.config.ConfigPath)
		return nil
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(r.config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Try to find Tailwind executable
	tailwindCmd, err := r.findTailwindExecutable()
	if err != nil {
		return fmt.Errorf("failed to find Tailwind CSS: %w", err)
	}

	// Build command arguments
	args := r.buildCommandArgs()

	// Create the command
	r.cmd = exec.Command(tailwindCmd[0], append(tailwindCmd[1:], args...)...)

	// Set up pipes for output
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Tailwind CSS: %w", err)
	}

	r.running = true
	r.lastBuild = time.Now()

	// Start output readers
	go r.readOutput(stdout, "🎨 Tailwind")
	go r.readOutput(stderr, "⚠️  Tailwind")

	// Start monitoring goroutine if in watch mode
	if r.config.Watch {
		go r.monitor()
		log.Printf("🎨 Tailwind CSS started in watch mode (config: %s)", r.config.ConfigPath)
	} else {
		go func() {
			r.cmd.Wait()
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
			close(r.doneChan)
		}()
		log.Printf("🎨 Building CSS with Tailwind (config: %s)", r.config.ConfigPath)
	}

	return nil
}

// Stop stops the Tailwind process
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	close(r.stopChan)

	if r.cmd != nil && r.cmd.Process != nil {
		// Try graceful shutdown first
		if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
			// Force kill if interrupt fails
			if err := r.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill Tailwind process: %w", err)
			}
		}

		// Wait for process to exit
		select {
		case <-r.doneChan:
		case <-time.After(5 * time.Second):
			// Force kill after timeout
			r.cmd.Process.Kill()
		}
	}

	r.running = false
	log.Println("🎨 Tailwind CSS stopped")
	return nil
}

// IsRunning returns whether the runner is currently running
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Rebuild triggers a manual rebuild (for non-watch mode)
func (r *Runner) Rebuild() error {
	if r.config.Watch {
		// In watch mode, Tailwind handles rebuilds automatically
		return nil
	}

	// Debounce rapid rebuilds
	if time.Since(r.lastBuild) < 100*time.Millisecond {
		return nil
	}

	log.Println("🎨 Rebuilding CSS with Tailwind...")
	return r.Start()
}

// findTailwindExecutable tries to find the Tailwind CSS executable
func (r *Runner) findTailwindExecutable() ([]string, error) {
	// Honor explicit strategy if provided
	switch strings.ToLower(r.config.Strategy) {
	case "npm":
		if r.commandExists("npx") {
			return []string{"npx", "tailwindcss"}, nil
		}
	case "standalone":
		if _, err := os.Stat(r.expandPath("~/.vango/tools/tailwindcss")); err == nil {
			return []string{r.expandPath("~/.vango/tools/tailwindcss")}, nil
		}
		if r.config.AutoDownload {
			if err := r.downloadStandaloneBinary(); err == nil {
				return []string{r.expandPath("~/.vango/tools/tailwindcss")}, nil
			}
		}
	}
	// 1) Project-local binary
	if _, err := os.Stat(r.expandPath("./node_modules/.bin/tailwindcss")); err == nil {
		return []string{r.expandPath("./node_modules/.bin/tailwindcss")}, nil
	}

	// 2) Cached standalone binary (~/.vango/tools/tailwindcss)
	standaloneLocations := []string{
		"./node_modules/.bin/tailwindcss",
		"~/.vango/tools/tailwindcss",
		"./bin/tailwindcss",
	}

	for _, location := range standaloneLocations {
		expanded := r.expandPath(location)
		if _, err := os.Stat(expanded); err == nil {
			return []string{expanded}, nil
		}
	}

	// 3) Download standalone binary automatically (preferred over npx when missing)
	if r.config.AutoDownload {
		if err := r.downloadStandaloneBinary(); err == nil {
			return []string{r.expandPath("~/.vango/tools/tailwindcss")}, nil
		}
	}

	// 4) As a last resort, try npm/npx/yarn
	if r.commandExists("npx") {
		return []string{"npx", "tailwindcss"}, nil
	}
	if r.commandExists("pnpm") {
		return []string{"pnpm", "exec", "tailwindcss"}, nil
	}
	if r.commandExists("yarn") {
		return []string{"yarn", "tailwindcss"}, nil
	}
	if r.commandExists("tailwindcss") {
		return []string{"tailwindcss"}, nil
	}

	return nil, fmt.Errorf("Tailwind CSS not available and auto-download failed")
}

// commandExists checks if a command exists in PATH
func (r *Runner) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// expandPath expands ~ in paths
func (r *Runner) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// downloadStandaloneBinary downloads the Tailwind standalone binary
func (r *Runner) downloadStandaloneBinary() error {
	version := os.Getenv("VANGO_TAILWIND_VERSION")
	if version == "" {
		version = "3.4.0"
	}
	url, binName, err := tailwindDownloadURL(version)
	if err != nil {
		return err
	}

	// Prepare destination
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home dir: %w", err)
	}
	toolsDir := filepath.Join(home, ".vango", "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir tools: %w", err)
	}
	dst := filepath.Join(toolsDir, "tailwindcss")
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}

	// If already exists, nothing to do
	if _, statErr := os.Stat(dst); statErr == nil {
		return nil
	}

	log.Printf("⬇️  Downloading Tailwind CSS v%s (%s) ...", version, binName)
	// Download
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)
	if _, err := io.Copy(mw, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if expected := expectedChecksum(version); expected != "" && !strings.EqualFold(sum, expected) {
		return fmt.Errorf("checksum mismatch: got %s", sum)
	}
	if err := f.Chmod(0o755); err != nil {
		_ = os.Chmod(dst, 0o755)
	}
	log.Printf("✅ Tailwind downloaded to %s", dst)
	return nil
}

func tailwindDownloadURL(version string) (url string, bin string, err error) {
	var osPart, archPart string
	switch runtime.GOOS {
	case "darwin":
		osPart = "macos"
	case "linux":
		osPart = "linux"
	case "windows":
		osPart = "windows"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64":
		archPart = "x64"
	case "arm64":
		archPart = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
	}
	bin = fmt.Sprintf("tailwindcss-%s-%s", osPart, archPart)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	url = fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/v%s/%s", version, bin)
	return url, bin, nil
}

// expectedChecksum returns optional checksum for the selected platform/version.
// TODO: populate with published checksums for selected releases.
func expectedChecksum(version string) string {
	// Known checksums (partial). Keys are version+platform.
	// Example populated for local darwin-arm64 testing.
	table := map[string]string{
		"3.4.0-darwin-arm64": "9b7e5a771851484155e812299fcf5529541192523ec562b8b0d3b8bb6728ac28",
	}
	var osPart, archPart string
	switch runtime.GOOS {
	case "darwin":
		osPart = "darwin"
	case "linux":
		osPart = "linux"
	case "windows":
		osPart = "windows"
	default:
		return ""
	}
	switch runtime.GOARCH {
	case "arm64":
		archPart = "arm64"
	case "amd64":
		archPart = "x64"
	default:
		return ""
	}
	return table[version+"-"+osPart+"-"+archPart]
}

// buildCommandArgs builds the command arguments for Tailwind
func (r *Runner) buildCommandArgs() []string {
	args := []string{
		"-i", r.config.InputPath,
		"-o", r.config.OutputPath,
	}

	if r.config.ConfigPath != "" {
		args = append(args, "-c", r.config.ConfigPath)
	}

	if r.config.Watch {
		args = append(args, "--watch")
	}

	// Add minification in production mode
	if os.Getenv("VANGO_ENV") == "production" {
		args = append(args, "--minify")
	}

	return args
}

// readOutput reads and logs output from the Tailwind process
func (r *Runner) readOutput(pipe io.ReadCloser, prefix string) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		// Filter out noise
		if !strings.Contains(line, "Done in") && !strings.Contains(line, "Watching for changes") {
			if strings.Contains(line, "error") || strings.Contains(line, "Error") {
				log.Printf("%s Error: %s", prefix, line)
			} else if strings.Contains(line, "warn") || strings.Contains(line, "Warning") {
				log.Printf("%s Warning: %s", prefix, line)
			} else if strings.Contains(line, "Rebuilding") || strings.Contains(line, "Building") {
				log.Printf("%s: %s", prefix, line)
			}
		}
	}
}

// monitor monitors the Tailwind process in watch mode
func (r *Runner) monitor() {
	defer close(r.doneChan)

	// Wait for process to exit or stop signal
	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case err := <-done:
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		if err != nil {
			log.Printf("⚠️  Tailwind CSS exited with error: %v", err)
		}
	case <-r.stopChan:
		// Stop requested
		return
	}
}

// GetOutputPath returns the output CSS file path
func (r *Runner) GetOutputPath() string {
	return r.config.OutputPath
}

// GetLastBuildTime returns the time of the last build
func (r *Runner) GetLastBuildTime() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastBuild
}
