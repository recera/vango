package tailwind

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Watcher manages Tailwind CSS watching during development
type Watcher struct {
	manager    *Manager
	configPath string
	inputPath  string
	outputPath string
	cancel     context.CancelFunc
}

// NewWatcher creates a new Tailwind watcher
func NewWatcher() (*Watcher, error) {
	manager, err := NewManager()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		manager: manager,
	}, nil
}

// Start starts the Tailwind watcher if a config file exists
func (w *Watcher) Start(projectDir string) error {
	// Look for Tailwind config file
	configPaths := []string{
		filepath.Join(projectDir, "tailwind.config.js"),
		filepath.Join(projectDir, "tailwind.config.ts"),
		filepath.Join(projectDir, "tailwind.config.mjs"),
		filepath.Join(projectDir, "tailwind.config.cjs"),
	}

	var configFound string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFound = path
			break
		}
	}

	if configFound == "" {
		// No Tailwind config found, skip
		return nil
	}

	fmt.Printf("🎨 Found Tailwind config at %s\n", configFound)
	w.configPath = configFound

	// Determine input and output paths
	w.inputPath = filepath.Join(projectDir, "styles", "input.css")
	w.outputPath = filepath.Join(projectDir, "public", "styles.css")

	// Check if custom input file exists
	customInputs := []string{
		filepath.Join(projectDir, "styles", "tailwind.css"),
		filepath.Join(projectDir, "styles", "main.css"),
		filepath.Join(projectDir, "app", "styles.css"),
	}

	for _, path := range customInputs {
		if _, err := os.Stat(path); err == nil {
			w.inputPath = path
			break
		}
	}

	// Create default input file if it doesn't exist
	if _, err := os.Stat(w.inputPath); os.IsNotExist(err) {
		if err := w.createDefaultInput(); err != nil {
			return fmt.Errorf("failed to create default input file: %w", err)
		}
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(w.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Start watching
	fmt.Printf("⚡ Starting Tailwind CSS in watch mode...\n")
	fmt.Printf("   Input:  %s\n", w.inputPath)
	fmt.Printf("   Output: %s\n", w.outputPath)

	// Run in a goroutine so it doesn't block
	go func() {
		if err := w.manager.Watch(w.inputPath, w.outputPath, w.configPath); err != nil {
			fmt.Printf("❌ Tailwind watcher error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the Tailwind watcher
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// createDefaultInput creates a default Tailwind input file
func (w *Watcher) createDefaultInput() error {
	dir := filepath.Dir(w.inputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := `@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom styles can be added here */
`

	return os.WriteFile(w.inputPath, []byte(content), 0644)
}

// Build runs Tailwind in production build mode
func (w *Watcher) Build(projectDir string, minify bool) error {
	// Look for config just like in Start
	configPaths := []string{
		filepath.Join(projectDir, "tailwind.config.js"),
		filepath.Join(projectDir, "tailwind.config.ts"),
		filepath.Join(projectDir, "tailwind.config.mjs"),
		filepath.Join(projectDir, "tailwind.config.cjs"),
	}

	var configFound string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFound = path
			break
		}
	}

	if configFound == "" {
		// No Tailwind config, nothing to build
		return nil
	}

	// Set paths
	w.configPath = configFound
	w.inputPath = filepath.Join(projectDir, "styles", "input.css")
	w.outputPath = filepath.Join(projectDir, "dist", "styles.css")

	// Look for custom input
	customInputs := []string{
		filepath.Join(projectDir, "styles", "tailwind.css"),
		filepath.Join(projectDir, "styles", "main.css"),
		filepath.Join(projectDir, "app", "styles.css"),
	}

	for _, path := range customInputs {
		if _, err := os.Stat(path); err == nil {
			w.inputPath = path
			break
		}
	}

	if _, err := os.Stat(w.inputPath); os.IsNotExist(err) {
		// No input file, skip
		return nil
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(w.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("🎨 Building Tailwind CSS...\n")
	fmt.Printf("   Input:  %s\n", w.inputPath)
	fmt.Printf("   Output: %s\n", w.outputPath)
	
	if minify {
		fmt.Println("   Minification: enabled")
	}

	return w.manager.Build(w.inputPath, w.outputPath, w.configPath, minify)
}