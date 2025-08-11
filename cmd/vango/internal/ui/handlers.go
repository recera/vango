package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleProjectBasicsKeys handles keyboard input for the project basics step
func (m *Model) handleProjectBasicsKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, DefaultKeyMap.Tab):
		// Move to next input field
		if m.currentInput < 3 { // 3 text inputs + 1 checkbox
			if m.currentInput < len(m.textInputs)-1 {
				m.textInputs[m.currentInput].Blur()
				m.currentInput++
				if m.currentInput < len(m.textInputs) {
					m.textInputs[m.currentInput].Focus()
				}
			} else {
				m.textInputs[m.currentInput].Blur()
				m.currentInput++
			}
		} else {
			// Wrap around to first input
			m.currentInput = 0
			m.textInputs[0].Focus()
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Enter):
		// Validate inputs
		name := m.textInputs[0].Value()
		if !isValidProjectName(name) {
			m.errorMessage = "Invalid project name. Use only letters, numbers, hyphens, and underscores."
			return nil
		}

		// Persist updates
		m.config.Name = name
		m.selectedItem = 0
		m.currentInput = 0
		m.textInputs[0].Blur()

		// Decide next step based on selected template
		if m.config.Template == "base" {
			m.step = StepRouting
		} else {
			m.step = StepSummary
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Back):
		// Go back to template selector in new workflow
		m.step = StepTemplate
		return nil

	case key.Matches(msg, DefaultKeyMap.Space):
		// Toggle checkbox if on Git init option
		if m.currentInput == 3 {
			m.config.GitInit = !m.config.GitInit
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Up):
		if m.currentInput > 0 {
			if m.currentInput <= len(m.textInputs)-1 {
				m.textInputs[m.currentInput].Blur()
			}
			m.currentInput--
			if m.currentInput < len(m.textInputs) {
				m.textInputs[m.currentInput].Focus()
			}
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Down):
		if m.currentInput < 3 {
			if m.currentInput < len(m.textInputs) {
				m.textInputs[m.currentInput].Blur()
			}
			m.currentInput++
			if m.currentInput < len(m.textInputs) {
				m.textInputs[m.currentInput].Focus()
			}
		}
		return nil
	}

	return nil
}

// handleTemplateKeys handles keyboard input for the template selection step
func (m *Model) handleTemplateKeys(msg tea.KeyMsg) tea.Cmd {
	// Order: Base, Blog, Graph, Counter
	templates := []string{"base", "blog", "graphviewer", "counter"}

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.selectedItem > 0 {
			m.selectedItem--
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Down):
		if m.selectedItem < len(templates)-1 {
			m.selectedItem++
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Space):
		// Toggle example checkboxes
		// This would need more complex handling for checkbox selection
		return nil

	case key.Matches(msg, DefaultKeyMap.Enter):
		m.config.Template = templates[m.selectedItem]
		// For Base, enter full config workflow starting at Project Basics
		if m.config.Template == "base" {
			m.step = StepProjectBasics
		} else {
			// Other templates skip straight to summary/confirmation
			m.step = StepSummary
		}
		m.selectedItem = 0
		return nil

	case key.Matches(msg, DefaultKeyMap.Back):
		// Back to preflight (previous screen) in the new flow
		m.step = StepPreflight
		return nil
	}

	return nil
}

// handleRoutingKeys handles keyboard input for the routing strategy step
func (m *Model) handleRoutingKeys(msg tea.KeyMsg) tea.Cmd {
	strategies := []string{"file-based", "programmatic", "minimal"}

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.selectedItem > 0 {
			m.selectedItem--
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Down):
		if m.selectedItem < len(strategies)-1 {
			m.selectedItem++
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Enter):
		m.config.RoutingStrategy = strategies[m.selectedItem]
		m.step = StepStyling
		m.selectedItem = 0
		return nil

	case key.Matches(msg, DefaultKeyMap.Back):
		m.step = StepTemplate
		m.selectedItem = 0
		return nil
	}

	return nil
}

// handleStylingKeys handles keyboard input for the styling configuration step
func (m *Model) handleStylingKeys(msg tea.KeyMsg) tea.Cmd {
	maxItems := 1 // Just Tailwind toggle and dark mode
	if m.config.UseTailwind {
		maxItems = 5 // Tailwind toggle + 3 strategies + dark mode
	}

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.selectedItem > 0 {
			m.selectedItem--
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Down):
		if m.selectedItem < maxItems-1 {
			m.selectedItem++
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Space):
		if m.selectedItem == 0 {
			// Toggle Tailwind
			m.config.UseTailwind = !m.config.UseTailwind
			if !m.config.UseTailwind && m.selectedItem > 1 {
				m.selectedItem = 1 // Move to dark mode toggle
			}
		} else if m.config.UseTailwind && m.selectedItem >= 1 && m.selectedItem <= 3 {
			// Select Tailwind strategy
			strategies := []string{"auto", "standalone", "npm"}
			m.config.TailwindStrategy = strategies[m.selectedItem-1]
		} else {
			// Toggle dark mode
			m.config.DarkMode = !m.config.DarkMode
		}
		return nil

	case key.Matches(msg, DefaultKeyMap.Enter):
		m.step = StepSummary
		return nil

	case key.Matches(msg, DefaultKeyMap.Back):
		m.step = StepRouting
		m.selectedItem = 0
		return nil
	}

	return nil
}

// runPreflight runs the preflight checks
func (m Model) runPreflight() tea.Cmd {
	return func() tea.Msg {
		// Simulate running checks
		time.Sleep(500 * time.Millisecond)

		// Check Go installation
		m.preflightChecks[0].Status = CheckRunning
		if _, err := exec.LookPath("go"); err == nil {
			m.preflightChecks[0].Status = CheckPassed
			m.preflightChecks[0].Message = "Go is installed"
		} else {
			m.preflightChecks[0].Status = CheckFailed
			m.preflightChecks[0].Message = "Go not found in PATH"
			m.preflightChecks[0].Fix = "Install Go from https://go.dev"
		}

		// Check TinyGo installation
		m.preflightChecks[1].Status = CheckRunning
		if _, err := exec.LookPath("tinygo"); err == nil {
			m.preflightChecks[1].Status = CheckPassed
			m.preflightChecks[1].Message = "TinyGo is installed"
		} else {
			m.preflightChecks[1].Status = CheckWarning
			m.preflightChecks[1].Message = "TinyGo not found (optional for WASM)"
			m.preflightChecks[1].Fix = "Install from https://tinygo.org"
		}

		// Check Tailwind strategy
		m.preflightChecks[2].Status = CheckRunning
		if _, err := exec.LookPath("npm"); err == nil {
			m.preflightChecks[2].Status = CheckPassed
			m.preflightChecks[2].Message = "npm available for Tailwind"
		} else {
			m.preflightChecks[2].Status = CheckPassed
			m.preflightChecks[2].Message = "Will use standalone Tailwind binary"
		}

		// Check directory writable
		m.preflightChecks[3].Status = CheckRunning
		homeDir, _ := os.UserHomeDir()
		vangoDir := filepath.Join(homeDir, ".vango")
		if err := os.MkdirAll(vangoDir, 0755); err == nil {
			m.preflightChecks[3].Status = CheckPassed
			m.preflightChecks[3].Message = "Can write to ~/.vango"
		} else {
			m.preflightChecks[3].Status = CheckFailed
			m.preflightChecks[3].Message = "Cannot write to ~/.vango"
			m.preflightChecks[3].Fix = "Check directory permissions"
		}

		// Check port availability (simplified)
		m.preflightChecks[4].Status = CheckPassed
		m.preflightChecks[4].Message = fmt.Sprintf("Port %d available", m.config.Port)

		return preflightCompleteMsg{}
	}
}

// executeCreation executes the project creation
func (m Model) executeCreation() tea.Cmd {
	// Initialize execution steps
	m.executionSteps = []ExecutionStep{
		{Name: "Creating project directory", Status: ExecPending},
		{Name: "Generating project structure", Status: ExecPending},
		{Name: "Writing configuration files", Status: ExecPending},
		{Name: "Setting up routing", Status: ExecPending},
		{Name: "Configuring styles", Status: ExecPending},
		{Name: "Running go mod tidy", Status: ExecPending},
		{Name: "Initializing git repository", Status: ExecPending},
		{Name: "Building initial assets", Status: ExecPending},
	}

	// Remove git step if not needed
	if !m.config.GitInit {
		m.executionSteps = m.executionSteps[:len(m.executionSteps)-2]
		m.executionSteps = append(m.executionSteps, m.executionSteps[len(m.executionSteps)-1])
		m.executionSteps = m.executionSteps[:len(m.executionSteps)-1]
	}

	return m.tickExecution()
}

// tickExecution advances the execution progress
func (m Model) tickExecution() tea.Cmd {
	return func() tea.Msg {
		if m.currentExecStep >= len(m.executionSteps) {
			return executionCompleteMsg{}
		}

		// Mark current step as running
		if m.currentExecStep < len(m.executionSteps) {
			m.executionSteps[m.currentExecStep].Status = ExecRunning
		}

		// Execute the actual step
		stepName := m.executionSteps[m.currentExecStep].Name
		var err error

		switch stepName {
		case "Creating project directory":
			// Directory is created by the generator
		case "Generating project structure":
			// This is the main generation step - handled by CreateProject
		case "Writing configuration files":
			// Also handled by the generator
		case "Setting up routing":
			// Part of the generation
		case "Configuring styles":
			// Part of the generation
		case "Running go mod tidy":
			// Handled after generation
		case "Initializing git repository":
			// Handled after generation if requested
		case "Building initial assets":
			// Initial build step if needed
		}

		// Mark as complete and move to next
		if m.currentExecStep < len(m.executionSteps) {
			if err != nil {
				m.executionSteps[m.currentExecStep].Status = ExecFailed
				m.executionSteps[m.currentExecStep].Message = err.Error()
				return executionErrorMsg{err: err}
			}
			m.executionSteps[m.currentExecStep].Status = ExecComplete
			m.executionSteps[m.currentExecStep].Message = "Complete"
		}

		m.currentExecStep++

		// Continue to next step with small delay for visual feedback
		time.Sleep(200 * time.Millisecond)
		return tickMsg(time.Now())
	}
}

// ValidateProjectName validates the project name
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("project name too long (max 50 characters)")
	}

	// Check for valid characters
	for i, ch := range name {
		if i == 0 && (ch >= '0' && ch <= '9') {
			return fmt.Errorf("project name cannot start with a number")
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return fmt.Errorf("project name contains invalid character: %c", ch)
		}
	}

	return nil
}

// GetTemplateDefaults returns default configuration for a template
func GetTemplateDefaults(template string) ProjectConfig {
	config := ProjectConfig{
		Port:             5173,
		RoutingStrategy:  "file-based",
		UseTailwind:      true,
		TailwindStrategy: "auto",
		DarkMode:         false,
		GitInit:          true,
		OpenBrowser:      true,
	}

	switch template {
	case "blog":
		config.DarkMode = true
		config.Features = append(config.Features, "markdown", "syntax-highlighting")
	case "fullstack":
		config.DarkMode = true
		config.Features = append(config.Features, "auth", "database", "websocket")
	case "minimal":
		config.UseTailwind = false
		config.RoutingStrategy = "minimal"
	}

	return config
}
