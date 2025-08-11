package ui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recera/vango/cmd/vango/cli_templates"
)

// RunCreateTUI starts the interactive TUI for project creation
func RunCreateTUI(projectName string) (ProjectConfig, error) {
	// Check if we're in a TTY
	if !isatty() {
		return ProjectConfig{}, fmt.Errorf("not running in a terminal, use --no-interactive flag")
	}

	// Create the model
	model := NewModel(projectName)

	// Run the TUI
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("TUI error: %w", err)
	}

	// Get the final configuration
	m := finalModel.(Model)
	if m.quitting && m.step != StepComplete {
		return ProjectConfig{}, fmt.Errorf("project creation cancelled")
	}

	config := m.GetConfig()

	// Ensure directory is set
	if config.Directory == "" {
		config.Directory = config.Name
	}

	// Actually create the project
	if err := CreateProject(config); err != nil {
		return config, fmt.Errorf("failed to create project: %w", err)
	}

	return config, nil
}

// CreateProject creates the actual project files based on configuration
func CreateProject(config ProjectConfig) error {
	// Convert UI ProjectConfig to cli_templates.ProjectConfig
	templateConfig := &cli_templates.ProjectConfig{
		Name:             config.Name,
		Module:           config.Module,
		Directory:        config.Directory,
		Template:         config.Template,
		RoutingStrategy:  config.RoutingStrategy,
		Features:         config.Features,
		UseTailwind:      config.UseTailwind,
		TailwindStrategy: config.TailwindStrategy,
		DarkMode:         config.DarkMode,
		GitInit:          config.GitInit,
		OpenBrowser:      config.OpenBrowser,
		Port:             config.Port,
		LocalReplace:     config.LocalReplace,
	}

	// Use the shared template generator
	if err := cli_templates.Generate(templateConfig); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Initialize git if requested
	if config.GitInit {
		if err := initGitRepo(config.Directory); err != nil {
			// Don't fail the whole process for git init failure
			fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
		}
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = config.Directory
	if err := cmd.Run(); err != nil {
		// Don't fail for go mod tidy
		fmt.Printf("Warning: Failed to run go mod tidy: %v\n", err)
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

// isatty checks if we're running in a terminal
func isatty() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
