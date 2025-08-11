package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/recera/vango/cmd/vango/internal/gen"
	"github.com/recera/vango/cmd/vango/internal/router"
	"github.com/recera/vango/cmd/vango/internal/template"
	"github.com/spf13/cobra"
)

func newGenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate code for various Vango features",
		Long:  `Generate code for routing, components, templates, and other Vango features.`,
	}

	cmd.AddCommand(newGenRouterCommand())
	cmd.AddCommand(newGenTemplateCommand())
	cmd.AddCommand(newGenBuilderCommand())
	
	return cmd
}

func newGenRouterCommand() *cobra.Command {
	var (
		watch      bool
		jsonOutput bool
		printStats bool
		routesDir  string
		outputDir  string
	)

	cmd := &cobra.Command{
		Use:   "router",
		Short: "Generate router code from file-based routes",
		Long: `Scans the app/routes directory and generates:
  - Radix tree matcher (pkg/internal/router/tree_gen.go)
  - Typed parameter structs (router/params.go)
  - Path helper functions (router/paths.go)
  - Route table JSON (router/table.json)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			// Create scanner
			scanner := router.NewScanner(routesDir)
			
			// Scan routes
			if printStats {
				fmt.Println("🔍 Scanning routes directory...")
			}
			
			routes, err := scanner.Scan()
			if err != nil {
				return fmt.Errorf("failed to scan routes: %w", err)
			}

			// Report any non-fatal errors
			for _, err := range scanner.GetErrors() {
				log.Printf("⚠️  Warning: %v", err)
			}

			if printStats {
				fmt.Printf("📊 Found %d routes in %v\n", len(routes), time.Since(startTime))
			}

			// Output JSON if requested
			if jsonOutput {
				data, err := json.MarshalIndent(routes, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal routes: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			// Generate code
			generator := router.NewCodeGenerator(routesDir, outputDir)
			
			if printStats {
				fmt.Println("🔨 Generating router code...")
			}

			if err := generator.Generate(); err != nil {
				return fmt.Errorf("failed to generate code: %w", err)
			}

			// Calculate statistics
			if printStats {
				duration := time.Since(startTime)
				paramCount := 0
				for _, route := range routes {
					paramCount += len(route.Params)
				}

				fmt.Printf(`
✅ Router generation complete!

  Scanned %d pages    %v
  Generated %d param structs, %d matcher nodes
  
  Files generated:
    - pkg/internal/router/tree_gen.go
    - router/params.go  
    - router/paths.go
    - router/table.json
`,
					len(routes),
					duration,
					paramCount,
					len(routes),
				)
			}

			// Watch mode
			if watch {
				fmt.Println("\n👀 Watching for changes... (Press Ctrl+C to stop)")
				// TODO: Implement file watching
				select {} // Block forever for now
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch for file changes and regenerate")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON summary to stdout")
	cmd.Flags().BoolVar(&printStats, "print-stats", true, "Print generation statistics")
	cmd.Flags().StringVar(&routesDir, "routes-dir", "app/routes", "Directory containing route files")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Output directory for generated files")

	return cmd
}

func newGenTemplateCommand() *cobra.Command {
	var (
		watch     bool
		directory string
		verbose   bool
	)

	cmd := &cobra.Command{
		Use:   "template [files...]",
		Short: "Generate Go code from VEX template files",
		Long: `Compile VEX template files (.vex) into Go code.

If no files are specified, searches for .vex files in the current directory
and common locations (app/, app/routes/, app/components/).

Examples:
  vango gen template                     # Compile all .vex files
  vango gen template app/routes/home.vex # Compile specific file
  vango gen template --dir ./templates   # Compile all in directory
  vango gen template --watch             # Watch and recompile on changes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			if verbose {
				fmt.Println("🎨 Starting VEX template compilation...")
			}

			// If specific files provided, compile them
			if len(args) > 0 {
				for _, file := range args {
					if verbose {
						fmt.Printf("📝 Compiling %s...\n", file)
					}
					
					if err := template.ProcessTemplateFile(file); err != nil {
						return fmt.Errorf("failed to compile %s: %w", file, err)
					}
					
					if verbose {
						outputFile := filepath.Base(file)
						outputFile = outputFile[:len(outputFile)-4] + ".vex.go"
						fmt.Printf("✅ Generated %s\n", outputFile)
					}
				}
			} else {
				// Compile all templates in directory
				searchDirs := []string{}
				
				if directory != "" {
					searchDirs = []string{directory}
				} else {
					// Default search locations
					searchDirs = []string{".", "app", "app/routes", "app/components"}
				}
				
				totalCompiled := 0
				for _, dir := range searchDirs {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						continue
					}
					
					if verbose {
						fmt.Printf("🔍 Searching %s for .vex files...\n", dir)
					}
					
					count := 0
					err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						
						if !info.IsDir() && filepath.Ext(path) == ".vex" {
							if verbose {
								fmt.Printf("📝 Compiling %s...\n", path)
							}
							
							if err := template.ProcessTemplateFile(path); err != nil {
								log.Printf("⚠️  Failed to compile %s: %v\n", path, err)
								// Continue with other files
							} else {
								count++
								totalCompiled++
							}
						}
						
						return nil
					})
					
					if err != nil {
						return fmt.Errorf("failed to walk directory %s: %w", dir, err)
					}
					
					if verbose && count > 0 {
						fmt.Printf("✅ Compiled %d templates in %s\n", count, dir)
					}
				}
				
				if totalCompiled == 0 {
					fmt.Println("ℹ️  No .vex files found")
				} else {
					duration := time.Since(startTime)
					fmt.Printf("\n✨ Successfully compiled %d templates in %v\n", totalCompiled, duration)
				}
			}

			// Watch mode
			if watch {
				fmt.Println("\n👀 Watching for changes... (Press Ctrl+C to stop)")
				// TODO: Implement file watching for templates
				select {} // Block forever for now
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for file changes and recompile")
	cmd.Flags().StringVarP(&directory, "dir", "d", "", "Directory to search for .vex files")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", true, "Verbose output")

	return cmd
}

func newGenBuilderCommand() *cobra.Command {
	var (
		specPath string
		verbose  bool
	)

	cmd := &cobra.Command{
		Use:   "builder",
		Short: "Generate fluent builder API from HTML specification",
		Long: `Generate the fluent builder API (Layer 1) and functional API (Layer 0)
from the HTML element specification.

This command reads the HTML specification from internal/spec/html.yml
and generates:
  - Functional API (Layer 0) - pkg/vex/functional/elements.go
  - Builder API (Layer 1) - pkg/vex/builder/elements.go
  - Props helpers - pkg/vex/functional/props.go

Example:
  vango gen builder                    # Use default spec path
  vango gen builder --spec custom.yml  # Use custom spec file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			if verbose {
				fmt.Println("🔨 Starting builder API generation...")
				fmt.Printf("📖 Reading spec from: %s\n", specPath)
			}

			// Run the builder generator
			if err := gen.Run(specPath); err != nil {
				return fmt.Errorf("builder generation failed: %w", err)
			}

			duration := time.Since(startTime)
			
			if verbose {
				fmt.Printf(`
✅ Builder API generated successfully!

  Generated files:
    - pkg/vex/functional/elements.go
    - pkg/vex/builder/elements.go  
    - pkg/vex/functional/props.go
    
  Duration: %v
`, duration)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&specPath, "spec", "internal/spec/html.yml", "Path to HTML specification YAML file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", true, "Verbose output")

	return cmd
}

// Helper function to check if running in a Vango project
func isVangoProject() bool {
	// Check for vango.json or go.mod with vango dependency
	if _, err := os.Stat("vango.json"); err == nil {
		return true
	}
	
	if _, err := os.Stat("app/routes"); err == nil {
		return true
	}
	
	return false
}