package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/recera/vango/cmd/vango/internal/config"
	"github.com/recera/vango/cmd/vango/internal/pragma"
	"github.com/recera/vango/cmd/vango/internal/routes"
	"github.com/recera/vango/cmd/vango/internal/tailwind"
	"github.com/recera/vango/internal/assets"
	"github.com/spf13/cobra"
)

func newBuildCommand() *cobra.Command {
	var output string
	var optimize bool
	var sourcemap bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the application for production",
		Long:  `Creates an optimized production build of your Vango application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(output, optimize, sourcemap)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "dist", "Output directory")
	cmd.Flags().BoolVar(&optimize, "optimize", true, "Optimize the build")
	cmd.Flags().BoolVar(&sourcemap, "sourcemap", false, "Generate source maps")

	return cmd
}

func runBuild(output string, optimize bool, sourcemap bool) error {
	log.Println("🚀 Building Vango application for production...")

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		log.Printf("⚠️  Failed to load vango.json: %v (using defaults)", err)
		cfg = config.DefaultConfig()
	}

	// Clean output directory
	if err := os.RemoveAll(output); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Create output directories
	if err := os.MkdirAll(filepath.Join(output, "assets"), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run pragma scanner to detect and inject build tags
	log.Println("🔍 Scanning for build pragmas...")
	scanner, err := pragma.NewScanner(pragma.ScannerConfig{
		AutoInjectTags: true,
		Verbose:        false,
		CacheDir:       filepath.Join(os.Getenv("HOME"), ".cache", "vango", "pragma"),
	})
	if err != nil {
		return fmt.Errorf("failed to create pragma scanner: %w", err)
	}

	manifest, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("pragma scan failed: %w", err)
	}

	// Save manifest for reference
	manifestPath := filepath.Join(output, "vango.manifest.json")
	if err := manifest.Save(manifestPath); err != nil {
		log.Printf("Warning: failed to save manifest: %v", err)
	}

	log.Printf("  Found %d server files, %d client files, %d shared files",
		len(manifest.ServerFiles), len(manifest.ClientFiles), len(manifest.SharedFiles))

	// Build CSS with Tailwind if configured
	// First check vango.json configuration, then fall back to checking for tailwind.config.js
	shouldBuildTailwind := false
	var tailwindRunner *tailwind.Runner

	// Check if Tailwind is explicitly configured in vango.json
	if cfg.Styling != nil && cfg.Styling.Tailwind != nil && cfg.Styling.Tailwind.Enabled {
		log.Println("🎨 Building CSS with Tailwind (using vango.json config)...")

		tw := cfg.Styling.Tailwind
		tailwindRunner = tailwind.NewRunner(tailwind.Config{
			ConfigPath:   tw.ConfigPath,
			InputPath:    tw.InputPath,
			OutputPath:   filepath.Join(output, "assets/styles.css"),
			Watch:        false, // Never watch in build mode
			Strategy:     tw.Strategy,
			Version:      tw.Version,
			AutoDownload: tw.AutoDownload,
		})
		shouldBuildTailwind = true
	} else if _, err := os.Stat("tailwind.config.js"); err == nil {
		// Fallback: if tailwind.config.js exists but not configured in vango.json
		log.Println("🎨 Building CSS with Tailwind (found tailwind.config.js)...")

		// Use default paths that match common conventions
		inputPath := "./styles/input.css"
		if _, err := os.Stat("app/styles/input.css"); err == nil {
			inputPath = "app/styles/input.css"
		}

		tailwindRunner = tailwind.NewRunner(tailwind.Config{
			ConfigPath:   "tailwind.config.js",
			InputPath:    inputPath,
			OutputPath:   filepath.Join(output, "assets/styles.css"),
			Watch:        false,
			Strategy:     "auto", // Auto-detect best strategy
			AutoDownload: true,   // Allow auto-download for better DX
		})
		shouldBuildTailwind = true
	}

	// Build Tailwind CSS if needed
	if shouldBuildTailwind && tailwindRunner != nil {
		// Set environment to production for minification
		os.Setenv("VANGO_ENV", "production")

		// Use BuildOnce for a single blocking build (not Start which runs in background)
		if err := tailwindRunner.BuildOnce(); err != nil {
			// Log warning but don't fail the entire build
			log.Printf("⚠️  Tailwind CSS build failed: %v", err)
			log.Println("    Continuing build without Tailwind CSS...")
		} else {
			log.Println("✅ Tailwind CSS build complete")
		}
	}

	// Build WASM with TinyGo (client-side code)
	log.Println("🔨 Building WASM with TinyGo...")

	wasmPath := filepath.Join(output, "assets/app.wasm")

	args := []string{
		"build",
		"-o", wasmPath,
		"-target", "wasm",
		"-tags", "vango_client", // Build with client tag for WASM
	}

	if !sourcemap {
		args = append(args, "-no-debug")
	}

	if optimize {
		args = append(args, "-opt", "z")      // Optimize for size
		args = append(args, "-gc", "leaking") // Use simpler GC for smaller size
	} else {
		args = append(args, "-opt", "2")
	}

	// Prefer app/client/main.go when present, fallback to app/main.go
	wasmMainPath := "./app/client/main.go"
	if _, err := os.Stat(wasmMainPath); os.IsNotExist(err) {
		wasmMainPath = "./app/main.go"
	}
	args = append(args, wasmMainPath)

	cmd := exec.Command("tinygo", args...)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("TinyGo build failed: %w\nOutput: %s", err, cmdOutput)
	}

	// Copy wasm_exec.js
	log.Println("📄 Copying wasm_exec.js...")
	if err := copyWasmExec(output); err != nil {
		return fmt.Errorf("failed to copy wasm_exec.js: %w", err)
	}

	// Copy bootstrap.js
	log.Println("📄 Copying bootstrap.js...")
	bootstrapContent := assets.BootstrapJS
	if len(bootstrapContent) == 0 {
		return fmt.Errorf("failed to read bootstrap.js: %w", fmt.Errorf("embedded asset missing"))
	}

	// Replace development checks
	bootstrapStr := string(bootstrapContent)
	bootstrapStr = strings.ReplaceAll(bootstrapStr, "process.env.NODE_ENV", "'production'")

	if err := os.WriteFile(filepath.Join(output, "assets/bootstrap.js"), []byte(bootstrapStr), 0644); err != nil {
		return fmt.Errorf("failed to write bootstrap.js: %w", err)
	}

	// Copy static files
	if err := copyStaticFiles(output); err != nil {
		return fmt.Errorf("failed to copy static files: %w", err)
	}

	// Generate index.html if it doesn't exist
	if _, err := os.Stat(filepath.Join(output, "index.html")); os.IsNotExist(err) {
		if err := generateIndexHTML(output); err != nil {
			return fmt.Errorf("failed to generate index.html: %w", err)
		}
	}

	// Generate production routing code
	if pb, err := routes.NewProductionBuilder("app/routes"); err == nil {
		log.Println("🧭 Generating production routing code...")
		if err := pb.Build(); err != nil {
			log.Printf("Warning: production routing codegen failed: %v", err)
		} else {
			if err := pb.GenerateProductionServer(); err != nil {
				log.Printf("Warning: production server main generation failed: %v", err)
			}
		}
	} else {
		log.Printf("Warning: could not initialize production builder: %v", err)
	}

	// Build server binary if there are server files
	if len(manifest.ServerFiles) > 0 {
		log.Println("🖥️  Building server binary...")

		serverPath := filepath.Join(output, "server")
		serverArgs := []string{
			"build",
			"-o", serverPath,
			"-tags", "vango_server", // Build with server tag
		}

		if optimize {
			serverArgs = append(serverArgs, "-ldflags", "-s -w") // Strip debug info
		}

		// Build project including generated main_gen.go if present
		if _, err := os.Stat("main_gen.go"); err == nil {
			serverArgs = append(serverArgs, "main_gen.go")
		} else {
			serverArgs = append(serverArgs, ".")
		}

		serverCmd := exec.Command("go", serverArgs...)
		serverOutput, err := serverCmd.CombinedOutput()
		if err != nil {
			log.Printf("Warning: Server build failed: %v\nOutput: %s", err, serverOutput)
			// Don't fail the entire build if server build fails
		} else {
			log.Printf("  Server binary: %s", serverPath)
		}
	}

	// Report build sizes
	log.Println("\n📊 Build complete!")
	reportBuildSizes(output)

	return nil
}

func copyWasmExec(output string) error {
	// Prefer embedded wasm_exec.js if available
	if len(assets.WasmExecJS) > 0 {
		return os.WriteFile(filepath.Join(output, "assets/wasm_exec.js"), assets.WasmExecJS, 0644)
	}

	// Fallback to TinyGo's wasm_exec.js
	content, err := os.ReadFile("internal/assets/wasm_exec.js")
	if err != nil {
		tinygoRoot, err := exec.Command("tinygo", "env", "TINYGOROOT").Output()
		if err != nil {
			return fmt.Errorf("failed to get TinyGo root: %w", err)
		}

		wasmExecPath := filepath.Join(strings.TrimSpace(string(tinygoRoot)), "targets/wasm_exec.js")
		content, err = os.ReadFile(wasmExecPath)
		if err != nil {
			return fmt.Errorf("failed to read wasm_exec.js: %w", err)
		}
	}

	return os.WriteFile(filepath.Join(output, "assets/wasm_exec.js"), content, 0644)
}

func copyStaticFiles(output string) error {
	// Copy public directory if it exists
	if info, err := os.Stat("public"); err == nil && info.IsDir() {
		return filepath.Walk("public", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Skip WASM files (already built)
			if strings.HasSuffix(path, ".wasm") {
				return nil
			}

			relPath, err := filepath.Rel("public", path)
			if err != nil {
				return err
			}

			destPath := filepath.Join(output, relPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			return os.WriteFile(destPath, input, 0644)
		})
	}

	return nil
}

func generateIndexHTML(output string) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Vango App</title>
    <link rel="stylesheet" href="/assets/styles.css">
    <script src="/assets/wasm_exec.js"></script>
    <script src="/assets/bootstrap.js"></script>
</head>
<body>
    <div id="app"></div>
</body>
</html>`

	return os.WriteFile(filepath.Join(output, "index.html"), []byte(html), 0644)
}

func reportBuildSizes(output string) {
	// Get WASM size
	wasmPath := filepath.Join(output, "assets/app.wasm")
	if info, err := os.Stat(wasmPath); err == nil {
		size := info.Size()
		log.Printf("  WASM:        %s", formatSize(size))

		// Check gzipped size
		gzSize := getGzippedSize(wasmPath)
		log.Printf("  WASM (gzip): %s", formatSize(gzSize))
	}

	// Get CSS size
	cssPath := filepath.Join(output, "assets/styles.css")
	if info, err := os.Stat(cssPath); err == nil {
		size := info.Size()
		log.Printf("  CSS:         %s", formatSize(size))
	}

	// Total size
	var totalSize int64
	filepath.Walk(output, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	log.Printf("  Total:       %s", formatSize(totalSize))
	log.Printf("\n✨ Build output: %s", output)
}

func getGzippedSize(path string) int64 {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	gz.Write(content)
	gz.Close()

	return int64(buf.Len())
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
