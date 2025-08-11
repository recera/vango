package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recera/vango/cmd/vango/cli_templates"
	"github.com/recera/vango/cmd/vango/internal/ui"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var (
		// Top-level flags
		template      string
		interactive   bool
		noInteractive bool
		localDev      bool
		cwd           string

		// Base-only flags
		routing          string
		styling          string
		tailwindStrategy string
		darkMode         bool
		gitInit          bool
		port             int
		host             string
		openBrowser      bool
	)

	cmd := &cobra.Command{
		Use:   "create [project-name]",
		Short: "Create a new Vango project",
		Long:  `Creates a new Vango project with a configurable base template or showcase templates.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]

			// Change working directory if specified
			if cwd != "" {
				if err := os.Chdir(cwd); err != nil {
					return fmt.Errorf("failed to change directory: %w", err)
				}
			}

			// Determine if we should use interactive mode
			isTerminal := false
			if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
				isTerminal = true
			}

			useInteractive := !noInteractive && isTerminal && (interactive || template == "base")

			if useInteractive {
				// Use TUI for base template or if explicitly requested
				config, err := ui.RunCreateTUI(projectName)
				if err != nil {
					return fmt.Errorf("TUI error: %w", err)
				}

				fmt.Printf("\n✨ Project '%s' created successfully!\n", config.Name)
				fmt.Printf("\n📚 Next steps:\n")
				fmt.Printf("   cd %s\n", config.Directory)
				fmt.Printf("   vango dev\n")
				fmt.Printf("\n📖 Documentation: https://vango.dev/docs\n")

				if localDev {
					return addLocalReplace(projectName)
				}
				return nil
			}

			// Non-interactive mode - build config from flags
			config := &cli_templates.ProjectConfig{
				Name:         projectName,
				Module:       projectName,
				Directory:    projectName,
				Template:     template,
				LocalReplace: localDev,
			}

			// Apply base template configuration if template is "base"
			if template == "base" {
				config.RoutingStrategy = routing
				if styling == "tailwind" {
					config.UseTailwind = true
					config.TailwindStrategy = tailwindStrategy
				} else {
					config.UseTailwind = false
				}
				config.DarkMode = darkMode
				config.GitInit = gitInit
				config.Port = port
				config.OpenBrowser = openBrowser
			} else {
				// Showcase templates use their own defaults
				switch template {
				case "blog":
					config.UseTailwind = true
					config.TailwindStrategy = "npm"
					config.DarkMode = true
					config.RoutingStrategy = "file-based"
				case "counter":
					config.UseTailwind = true
					config.TailwindStrategy = "npm"
					config.RoutingStrategy = "minimal"
				case "graphviewer":
					config.UseTailwind = true
					// Prefer standalone so users don't need Node/npm installed
					config.TailwindStrategy = "standalone"
					config.RoutingStrategy = "minimal"
				default:
					// Unknown template, use base defaults
					config.RoutingStrategy = "file-based"
					config.UseTailwind = true
					config.TailwindStrategy = "auto"
					config.DarkMode = true
					config.GitInit = true
					config.Port = 5173
				}

				// Apply optional port/host for showcase templates
				if cmd.Flags().Changed("port") {
					config.Port = port
				} else {
					config.Port = 5173
				}
			}

			// Generate the project
			if err := cli_templates.Generate(config); err != nil {
				return fmt.Errorf("failed to generate project: %w", err)
			}

			// Git init if requested (base template) or for showcase templates that want it
			if config.GitInit {
				if err := initGitRepo(projectName); err != nil {
					fmt.Printf("⚠️  Failed to initialize git: %v\n", err)
				}
			}

			// Success message
			fmt.Printf("\n✨ Project '%s' created successfully!\n", projectName)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %s\n", projectName)
			fmt.Printf("  go mod tidy\n")
			fmt.Printf("  vango dev\n")

			if localDev {
				return addLocalReplace(projectName)
			}

			return nil
		},
	}

	// Top-level flags
	cmd.Flags().StringVarP(&template, "template", "t", "base", "Template to use (base, blog, counter, graphviewer)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Force interactive TUI")
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Force non-interactive mode")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Change working directory before generation")
	cmd.Flags().BoolVar(&localDev, "local-replace", false, "Add replace directive to use local Vango source (framework dev only)")

	// Base-only flags
	cmd.Flags().StringVar(&routing, "routing", "file-based", "Routing strategy: file-based, programmatic, minimal (base template only)")
	cmd.Flags().StringVar(&styling, "styling", "tailwind", "Styling method: tailwind, none (base template only)")
	cmd.Flags().StringVar(&tailwindStrategy, "tailwind-strategy", "auto", "Tailwind strategy: auto, npm, standalone (base template only)")
	cmd.Flags().BoolVar(&darkMode, "dark-mode", true, "Enable dark mode scaffolding (base template only)")
	cmd.Flags().BoolVar(&gitInit, "git-init", true, "Initialize git repository and initial commit (base template only)")
	cmd.Flags().IntVar(&port, "port", 5173, "Dev server port")
	cmd.Flags().StringVar(&host, "host", "localhost", "Dev server host")
	cmd.Flags().BoolVar(&openBrowser, "open-browser", false, "Open browser on vango dev start (base template only)")

	return cmd
}

// addLocalReplace appends a replace directive to the generated app's go.mod so it uses the
// framework source from the local filesystem. Intended for framework developers.
func addLocalReplace(projectName string) error {
	gomodPath := filepath.Join(projectName, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return err
	}
	content := string(data)
	if strings.Contains(content, "replace github.com/recera/vango") {
		return nil
	}
	// Detect repo root by walking up from current working directory until we find go.mod containing module github.com/recera/vango
	root := findFrameworkRoot()
	if root == "" {
		// Fallback to current working directory
		cwd, _ := os.Getwd()
		root = cwd
	}
	content += "\nreplace github.com/recera/vango => " + root + "\n"
	return os.WriteFile(gomodPath, []byte(content), 0644)
}

func findFrameworkRoot() string {
	dirs := []string{".", "..", "../..", "../../..", "../../../.."}
	for _, d := range dirs {
		p := filepath.Join(d, "go.mod")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if strings.Contains(string(b), "module github.com/recera/vango") {
			abs, _ := filepath.Abs(filepath.Dir(p))
			return abs
		}
	}
	return ""
}
