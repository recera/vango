package tailwind

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	tailwindVersion = "v3.4.0"
	cacheDir        = ".vango/tools"
)

// Manager handles Tailwind CSS binary management
type Manager struct {
	homeDir    string
	binaryPath string
}

// NewManager creates a new Tailwind manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Manager{
		homeDir: home,
	}, nil
}

// EnsureTailwind ensures Tailwind CSS is available
func (m *Manager) EnsureTailwind() (string, error) {
	// First, try to use npm/npx if available
	if path, err := m.tryNPX(); err == nil {
		fmt.Println("✅ Using Tailwind CSS via npx")
		return path, nil
	}

	// Check if standalone binary is already downloaded
	if path, err := m.getStandalonePath(); err == nil {
		if _, err := os.Stat(path); err == nil {
			fmt.Println("✅ Using cached Tailwind CSS standalone binary")
			m.binaryPath = path
			return path, nil
		}
	}

	// Download standalone binary
	fmt.Println("📥 Downloading Tailwind CSS standalone binary...")
	if err := m.downloadStandalone(); err != nil {
		return "", fmt.Errorf("failed to download Tailwind: %w", err)
	}

	return m.binaryPath, nil
}

// tryNPX attempts to use Tailwind via npx
func (m *Manager) tryNPX() (string, error) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return "", fmt.Errorf("npx not found")
	}

	// Test if tailwindcss is available via npx
	cmd := exec.Command("npx", "--no-install", "tailwindcss", "--help")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tailwindcss not available via npx")
	}

	return "npx", nil
}

// getStandalonePath returns the path where the standalone binary should be
func (m *Manager) getStandalonePath() (string, error) {
	platform := getPlatform()
	if platform == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	filename := fmt.Sprintf("tailwindcss-%s", platform)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	path := filepath.Join(m.homeDir, cacheDir, filename)
	return path, nil
}

// downloadStandalone downloads the Tailwind standalone binary
func (m *Manager) downloadStandalone() error {
	platform := getPlatform()
	if platform == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Construct download URL
	url := fmt.Sprintf(
		"https://github.com/tailwindlabs/tailwindcss/releases/download/%s/tailwindcss-%s",
		tailwindVersion,
		platform,
	)

	// Create cache directory
	cacheFullPath := filepath.Join(m.homeDir, cacheDir)
	if err := os.MkdirAll(cacheFullPath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Determine output path
	outputPath, err := m.getStandalonePath()
	if err != nil {
		return err
	}

	// Download the file
	fmt.Printf("Downloading from %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Make executable
	if err := os.Chmod(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	fmt.Printf("✅ Downloaded Tailwind CSS standalone to %s\n", outputPath)
	m.binaryPath = outputPath
	return nil
}

// Run runs Tailwind CSS with the given arguments
func (m *Manager) Run(args ...string) error {
	if m.binaryPath == "" {
		path, err := m.EnsureTailwind()
		if err != nil {
			return err
		}
		m.binaryPath = path
	}

	var cmd *exec.Cmd
	if m.binaryPath == "npx" {
		// Use npx
		cmdArgs := append([]string{"tailwindcss"}, args...)
		cmd = exec.Command("npx", cmdArgs...)
	} else {
		// Use standalone binary
		cmd = exec.Command(m.binaryPath, args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Watch runs Tailwind in watch mode
func (m *Manager) Watch(input, output string, config string) error {
	args := []string{
		"-i", input,
		"-o", output,
		"--watch",
	}

	if config != "" {
		args = append(args, "-c", config)
	}

	return m.Run(args...)
}

// Build runs Tailwind in build mode
func (m *Manager) Build(input, output string, config string, minify bool) error {
	args := []string{
		"-i", input,
		"-o", output,
	}

	if config != "" {
		args = append(args, "-c", config)
	}

	if minify {
		args = append(args, "--minify")
	}

	return m.Run(args...)
}

// getPlatform returns the platform string for the current OS/arch
func getPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "macos-x64"
		case "arm64":
			return "macos-arm64"
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "linux-x64"
		case "arm64":
			return "linux-arm64"
		case "arm":
			if strings.Contains(runtime.GOARCH, "v7") {
				return "linux-armv7"
			}
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "windows-x64.exe"
		case "arm64":
			return "windows-arm64.exe"
		}
	}

	return ""
}