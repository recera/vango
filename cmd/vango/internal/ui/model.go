package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Step represents the current step in the creation flow
type Step int

const (
	StepWelcome Step = iota
	StepPreflight
	StepProjectBasics
	StepTemplate
	StepRouting
	StepStyling
	StepSummary
	StepExecuting
	StepComplete
)

// ProjectConfig holds all configuration for the new project
type ProjectConfig struct {
	Name             string
	Module           string
	Directory        string
	Template         string
	RoutingStrategy  string
	Features         []string
	UseTailwind      bool
	TailwindStrategy string
	DarkMode         bool
	GitInit          bool
	OpenBrowser      bool
	Port             int
	LocalReplace     bool // For framework development
}

// PreflightCheck represents a system requirement check
type PreflightCheck struct {
	Name    string
	Status  CheckStatus
	Message string
	Fix     string
}

type CheckStatus int

const (
	CheckPending CheckStatus = iota
	CheckRunning
	CheckPassed
	CheckWarning
	CheckFailed
)

// Model represents the TUI application state
type Model struct {
	// Window dimensions
	width  int
	height int

	// Current step
	step Step

	// Project configuration
	config ProjectConfig

	// Preflight checks
	preflightChecks []PreflightCheck
	preflightDone   bool

	// UI components
	textInputs   []textinput.Model
	currentInput int
	selectedItem int
	checkboxes   map[string]bool
	spinner      spinner.Model
	progress     progress.Model

	// Execution state
	executionSteps  []ExecutionStep
	currentExecStep int
	executionError  error

	// Art and styling
	art      []string
	showHelp bool
	quitting bool
	err      error

	// Messages
	statusMessage string
	errorMessage  string
}

// ExecutionStep represents a step in the project creation process
type ExecutionStep struct {
	Name     string
	Status   ExecStatus
	Message  string
	Progress float64
}

type ExecStatus int

const (
	ExecPending ExecStatus = iota
	ExecRunning
	ExecComplete
	ExecFailed
)

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Space key.Binding
	Back  key.Binding
	Quit  key.Binding
	Help  key.Binding
	Tab   key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
}

// Messages
type preflightCompleteMsg struct{}
type executionCompleteMsg struct{}
type executionErrorMsg struct{ err error }
type tickMsg time.Time

// autoAdvanceMsg is emitted after preflight passes to advance automatically
type autoAdvanceMsg struct{}

// LoadArt loads the ASCII art from the embedded file
func LoadArt() []string {
	// Use the background art for enhanced visual display
	return LoadBackgroundArt()
}

// NewModel creates a new TUI model
func NewModel(projectName string) Model {
	// Initialize text inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "my-vango-app"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 40
	if projectName != "" {
		nameInput.SetValue(projectName)
	}

	moduleInput := textinput.New()
	moduleInput.Placeholder = "github.com/username/my-vango-app"
	moduleInput.CharLimit = 100
	moduleInput.Width = 40

	portInput := textinput.New()
	portInput.Placeholder = "5173"
	portInput.CharLimit = 5
	portInput.Width = 10
	portInput.SetValue("5173")

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize progress bar
	p := progress.New(progress.WithDefaultGradient())

	return Model{
		// Start at preflight as per new workflow
		step:       StepPreflight,
		art:        LoadArt(),
		textInputs: []textinput.Model{nameInput, moduleInput, portInput},
		spinner:    s,
		progress:   p,
		checkboxes: make(map[string]bool),
		config: ProjectConfig{
			Name:             projectName,
			Port:             5173,
			RoutingStrategy:  "file-based",
			Template:         "base",
			UseTailwind:      true,
			TailwindStrategy: "auto",
			DarkMode:         true,
			GitInit:          true,
			OpenBrowser:      true,
		},
		preflightChecks: []PreflightCheck{
			{Name: "Go Installation", Status: CheckPending},
			{Name: "TinyGo Installation", Status: CheckPending},
			{Name: "Tailwind Strategy", Status: CheckPending},
			{Name: "Directory Writable", Status: CheckPending},
			{Name: "Port Available", Status: CheckPending},
		},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Kick off preflight automatically in the new flow
	cmds := []tea.Cmd{m.spinner.Tick, textinput.Blink}
	if m.step == StepPreflight {
		cmds = append(cmds, m.runPreflight())
	}
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 40
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil

	case tea.KeyMsg:
		// Global key handling
		if key.Matches(msg, DefaultKeyMap.Quit) && m.step != StepExecuting {
			m.quitting = true
			return m, tea.Quit
		}

		if key.Matches(msg, DefaultKeyMap.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Step-specific key handling
		switch m.step {
		case StepWelcome:
			if key.Matches(msg, DefaultKeyMap.Enter) {
				m.step = StepPreflight
				return m, m.runPreflight()
			}

		case StepPreflight:
			if m.preflightDone && key.Matches(msg, DefaultKeyMap.Enter) {
				m.step = StepTemplate
				return m, nil
			}

		case StepProjectBasics:
			cmd := m.handleProjectBasicsKeys(msg)
			if cmd != nil {
				return m, cmd
			}

		case StepTemplate:
			cmd := m.handleTemplateKeys(msg)
			if cmd != nil {
				return m, cmd
			}

		case StepRouting:
			cmd := m.handleRoutingKeys(msg)
			if cmd != nil {
				return m, cmd
			}

		case StepStyling:
			cmd := m.handleStylingKeys(msg)
			if cmd != nil {
				return m, cmd
			}

		case StepSummary:
			if key.Matches(msg, DefaultKeyMap.Enter) {
				m.step = StepExecuting
				return m, m.executeCreation()
			}
			if key.Matches(msg, DefaultKeyMap.Back) {
				m.step = StepStyling
				return m, nil
			}

		case StepComplete:
			if key.Matches(msg, DefaultKeyMap.Enter) {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case preflightCompleteMsg:
		m.preflightDone = true
		// If all checks passed, auto-advance after 1.5s to template selector
		allPassed := true
		for _, c := range m.preflightChecks {
			if c.Status != CheckPassed {
				allPassed = false
				break
			}
		}
		if allPassed {
			return m, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return autoAdvanceMsg{} })
		}
		return m, nil

	case executionCompleteMsg:
		m.step = StepComplete
		return m, nil

	case executionErrorMsg:
		m.executionError = msg.err
		return m, nil

	case tickMsg:
		if m.step == StepExecuting {
			return m, m.tickExecution()
		}
	case autoAdvanceMsg:
		// Move from preflight to the art-based template selector
		if m.step == StepPreflight {
			m.step = StepTemplate
		}
		return m, nil
	}

	// Update text inputs
	if m.step == StepProjectBasics && m.currentInput < len(m.textInputs) {
		var cmd tea.Cmd
		m.textInputs[m.currentInput], cmd = m.textInputs[m.currentInput].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// Main content based on current step
	var content string
	switch m.step {
	case StepWelcome:
		content = m.renderWelcome()
	case StepPreflight:
		content = m.renderPreflight()
	case StepProjectBasics:
		content = m.renderProjectBasics()
	case StepTemplate:
		content = m.renderTemplateOverlay()
	case StepRouting:
		content = m.renderRouting()
	case StepStyling:
		content = m.renderStyling()
	case StepSummary:
		content = m.renderSummary()
	case StepExecuting:
		content = m.renderExecution()
	case StepComplete:
		content = m.renderComplete()
	}

	// Add footer with keybindings
	footer := m.renderFooter()

	// Combine content and footer
	fullHeight := strings.Count(content, "\n") + strings.Count(footer, "\n") + 2
	if fullHeight < m.height {
		padding := m.height - fullHeight - 1
		content += strings.Repeat("\n", padding)
	}

	return content + "\n" + footer
}

// Utility function to validate project name
func isValidProjectName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}
	// Check for valid characters (letters, numbers, hyphens, underscores)
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

// GetConfig returns the final project configuration
func (m Model) GetConfig() ProjectConfig {
	// Update config from text inputs
	if len(m.textInputs) > 0 {
		m.config.Name = m.textInputs[0].Value()
		if m.textInputs[1].Value() != "" {
			m.config.Module = m.textInputs[1].Value()
		} else {
			// Default module path
			m.config.Module = fmt.Sprintf("github.com/example/%s", m.config.Name)
		}
		if len(m.textInputs) > 2 {
			portStr := m.textInputs[2].Value()
			if portStr != "" {
				fmt.Sscanf(portStr, "%d", &m.config.Port)
			}
		}
	}

	// Set directory to project name if not specified
	if m.config.Directory == "" {
		m.config.Directory = m.config.Name
	}

	return m.config
}
