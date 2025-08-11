package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/recera/vango/cmd/vango/cli_templates"
	"github.com/recera/vango/cmd/vango/internal/prompt"
)

// runInteractiveCreate runs the interactive project creation wizard
func runInteractiveCreate(projectName string) error {
	p := prompt.New()

	fmt.Println("\n🚀 Welcome to Vango Project Creator!")
	fmt.Println("====================================")
	fmt.Printf("Creating project: %s\n\n", projectName)

	// Template selection
	templates := []string{
		"Basic - Minimal starter template",
		"Counter - Interactive counter example",
		"Todo - Todo list application",
		"Blog - Blog with routing",
		"Full Stack - Complete application with all features",
	}

	templateIdx := p.Select("Select a project template:", templates, 0)
	templateNames := []string{"basic", "counter", "todo", "blog", "fullstack"}
	selectedTemplate := templateNames[templateIdx]

	// Feature selection
	fmt.Println("\n📦 Select features to include:")
	features := []string{
		"Tailwind CSS",
		"Dark Mode",
		"Routing",
		"State Management",
		"Server-Side Rendering",
		"WebSocket Live Updates",
		"Authentication",
		"Database Integration",
	}

	// Default selections based on template
	defaults := make([]bool, len(features))
	switch selectedTemplate {
	case "fullstack":
		// Select all features for full stack
		for i := range defaults {
			defaults[i] = true
		}
	case "blog":
		defaults[0] = true // Tailwind
		defaults[1] = true // Dark Mode
		defaults[2] = true // Routing
		defaults[3] = true // State Management
		defaults[4] = true // SSR
	case "counter", "todo":
		defaults[0] = true // Tailwind
		defaults[3] = true // State Management
	}

	selectedFeatures := p.MultiSelect("", features, defaults)
	var selectedFeatureNames []string
	useTailwind := false
	darkMode := false

	for i, selected := range selectedFeatures {
		if selected {
			selectedFeatureNames = append(selectedFeatureNames, features[i])
			if strings.Contains(features[i], "Tailwind") {
				useTailwind = true
			}
			if strings.Contains(features[i], "Dark Mode") {
				darkMode = true
			}
		}
	}

	// Development configuration
	fmt.Println("\n⚙️  Development Configuration:")
	portStr := p.Text("Development server port", "5173")
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	if port == 0 {
		port = 5173
	}

	// Routing strategy
	routingStrategy := "file-based"
	for _, feature := range selectedFeatureNames {
		if strings.Contains(feature, "Routing") {
			strategies := []string{
				"File-based (recommended) - Routes based on file structure",
				"Programmatic - Define routes in code",
			}
			strategyIdx := p.Select("Select routing strategy:", strategies, 0)
			if strategyIdx == 0 {
				routingStrategy = "file-based"
			} else {
				routingStrategy = "programmatic"
			}
			break
		}
	}

	// Tailwind strategy
	tailwindStrategy := "npm"
	if useTailwind {
		tailStrategies := []string{
			"npm - Use npm to run Tailwind (requires Node.js)",
			"standalone - Download standalone binary (no Node.js required)",
			"auto - Automatically choose best available option",
		}
		tailIdx := p.Select("Select Tailwind strategy:", tailStrategies, 2)
		if tailIdx == 0 {
			tailwindStrategy = "npm"
		} else if tailIdx == 1 {
			tailwindStrategy = "standalone"
		} else {
			tailwindStrategy = "auto"
		}
	}

	// Git initialization
	gitInit := p.Confirm("\n📁 Initialize git repository?", true)

	// Dependencies installation
	installDeps := p.Confirm("📦 Install dependencies now?", true)

	// Open browser
	openBrowser := p.Confirm("🌐 Open browser when dev server starts?", false)

	// Editor opening
	openEditor := p.Confirm("💻 Open in VS Code when done?", false)

	// Confirm configuration
	fmt.Println("\n📋 Project Configuration:")
	fmt.Println("========================")
	fmt.Printf("  Name:       %s\n", projectName)
	fmt.Printf("  Template:   %s\n", selectedTemplate)
	fmt.Printf("  Port:       %d\n", port)
	fmt.Printf("  Features:   %s\n", strings.Join(selectedFeatureNames, ", "))
	if gitInit {
		fmt.Println("  Git:        ✓ Will initialize")
	}
	if installDeps {
		fmt.Println("  Deps:       ✓ Will install")
	}

	if !p.Confirm("\nProceed with creation?", true) {
		fmt.Println("❌ Project creation cancelled.")
		return nil
	}

	// Create the project using shared templates
	fmt.Println("\n🔨 Creating project structure...")

	// Create cli_templates.ProjectConfig
	config := &cli_templates.ProjectConfig{
		Name:             projectName,
		Module:           projectName,
		Directory:        projectName,
		Template:         selectedTemplate,
		RoutingStrategy:  routingStrategy,
		Features:         selectedFeatureNames,
		UseTailwind:      useTailwind,
		TailwindStrategy: tailwindStrategy,
		DarkMode:         darkMode,
		GitInit:          false, // We'll handle git separately
		OpenBrowser:      openBrowser,
		Port:             port,
		LocalReplace:     false,
	}

	// Generate the project
	if err := cli_templates.Generate(config); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Initialize git if requested
	if gitInit {
		fmt.Println("📁 Initializing git repository...")
		if err := initGitRepo(projectName); err != nil {
			fmt.Printf("⚠️  Failed to initialize git: %v\n", err)
		}
	}

	// Install dependencies if requested
	if installDeps {
		fmt.Println("📦 Installing dependencies...")
		if err := installDependencies(projectName, useTailwind); err != nil {
			fmt.Printf("⚠️  Failed to install dependencies: %v\n", err)
		}
	}

	// Success message
	fmt.Println("\n✨ Project created successfully!")
	fmt.Println("\n📚 Next steps:")
	fmt.Printf("   cd %s\n", projectName)
	if !installDeps {
		fmt.Println("   go mod tidy")
		if useTailwind && tailwindStrategy == "npm" {
			fmt.Println("   npm install")
		}
	}
	fmt.Println("   vango dev")

	fmt.Println("\n📖 Documentation: https://vango.dev/docs")
	fmt.Println("💬 Discord: https://discord.gg/vango")

	// Open in editor if requested
	if openEditor {
		fmt.Println("\n💻 Opening in VS Code...")
		exec.Command("code", projectName).Start()
	}

	return nil
}

// initGitRepo initializes a git repository
func initGitRepo(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit from Vango")
	cmd.Dir = projectPath
	return cmd.Run()
}

// installDependencies installs project dependencies
func installDependencies(projectPath string, useTailwind bool) error {
	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	// Install npm dependencies if using Tailwind
	if useTailwind {
		// Check if npm is available
		if _, err := exec.LookPath("npm"); err != nil {
			fmt.Println("⚠️  npm not found. Please install Node.js and run 'npm install' manually.")
			return nil
		}

		cmd = exec.Command("npm", "install")
		cmd.Dir = projectPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run npm install: %w", err)
		}
	}

	return nil
}
