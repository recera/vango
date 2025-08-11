package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Style definitions
var (
	// Colors
	primaryColor   = lipgloss.Color("#3b82f6") // Vango blue
	secondaryColor = lipgloss.Color("#64748b") // Gray
	successColor   = lipgloss.Color("#10b981") // Green
	warningColor   = lipgloss.Color("#f59e0b") // Yellow
	errorColor     = lipgloss.Color("#ef4444") // Red
	mutedColor     = lipgloss.Color("#94a3b8") // Muted gray

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)
)

// renderWelcome renders a simple, robust welcome screen with centered art
// and a single "Press Enter to Continue" prompt overlaid. The art has a
// fixed size, so we manually center the whole block and overlay text using
// absolute positioning relative to that block to avoid drift.
func (m Model) renderWelcome() string {
	const artWidth = 69
	const artHeight = 19

	// Build the art box with border
	boxWidth := artWidth + 4   // +4 for borders and padding
	boxHeight := artHeight + 2 // +2 for borders

	// Calculate centered position once
	boxStartX := (m.width - boxWidth) / 2
	if boxStartX < 0 {
		boxStartX = 0
	}
	boxStartY := (m.height - boxHeight) / 2
	if boxStartY < 0 {
		boxStartY = 0
	}

	// Compose the box content lines
	var lines []string
	// Top border
	lines = append(lines, "╭"+strings.Repeat("─", boxWidth-2)+"╮")
	// Art content
	for i := 0; i < artHeight; i++ {
		// Left border and space padding inside
		lineStr := "│ "
		if i < len(m.art) {
			// Use the art line as-is (it includes ANSI color codes and is
			// already designed to render to the correct visible width)
			lineStr += m.art[i]
		}
		// Right border
		lineStr += " │"
		lines = append(lines, lineStr)
	}
	// Bottom border
	lines = append(lines, "╰"+strings.Repeat("─", boxWidth-2)+"╯")

	// Render with manual centering (no lipgloss.Place) so our absolute
	// overlay coordinates stay in sync with the box position.
	var output strings.Builder
	// Top padding
	if boxStartY > 0 {
		output.WriteString(strings.Repeat("\n", boxStartY))
	}
	// Left padding per line + content
	leftPad := strings.Repeat(" ", boxStartX)
	for _, line := range lines {
		output.WriteString(leftPad)
		output.WriteString(line)
		output.WriteString("\n")
	}

	// Overlay: simple prompt centered within the art area
	prompt := "Press Enter to Continue"
	// Position within the inner area
	innerRow := artHeight - 5 // a bit above the bottom of the art
	promptCol := (artWidth - len(prompt)) / 2
	if promptCol < 0 {
		promptCol = 0
	}

	// Convert to absolute terminal coordinates (1-based)
	absRow := boxStartY + 1 + 1 + innerRow  // +1 for top border, +1 to move into content
	absCol := boxStartX + 1 + 1 + promptCol // left corner + border + space inside

	output.WriteString("\033[s")
	output.WriteString(fmt.Sprintf("\033[%d;%dH", absRow, absCol))
	output.WriteString("\033[1;33m") // bright yellow
	output.WriteString(prompt)
	output.WriteString("\033[0m")
	output.WriteString("\033[u")

	return output.String()
}

// renderTemplateOverlay shows the art background with an overlaid
// template selector (Base, Blog, Graph, Counter). The title includes
// the project name if available.
func (m Model) renderTemplateOverlay() string {
	const artWidth = 69
	const artHeight = 19

	// Build the art box with border
	boxWidth := artWidth + 4
	boxHeight := artHeight + 2

	// Centered position
	boxStartX := (m.width - boxWidth) / 2
	if boxStartX < 0 {
		boxStartX = 0
	}
	boxStartY := (m.height - boxHeight) / 2
	if boxStartY < 0 {
		boxStartY = 0
	}

	// Compose the box content
	var lines []string
	lines = append(lines, "╭"+strings.Repeat("─", boxWidth-2)+"╮")
	for i := 0; i < artHeight; i++ {
		lineStr := "│ "
		if i < len(m.art) {
			lineStr += m.art[i]
		}
		lineStr += " │"
		lines = append(lines, lineStr)
	}
	lines = append(lines, "╰"+strings.Repeat("─", boxWidth-2)+"╯")

	// Manual centering render
	var output strings.Builder
	if boxStartY > 0 {
		output.WriteString(strings.Repeat("\n", boxStartY))
	}
	leftPad := strings.Repeat(" ", boxStartX)
	for _, line := range lines {
		output.WriteString(leftPad)
		output.WriteString(line)
		output.WriteString("\n")
	}

	// Overlay selector
	projectName := m.config.Name
	if projectName == "" {
		projectName = "Your App"
	}
	title := fmt.Sprintf("Create %s", projectName)
	options := []string{"Base", "Blog", "Graph", "Counter"}

	// Position block roughly centered within art
	innerTop := 6
	absRow := boxStartY + 1 + 1 + innerTop
	absCol := boxStartX + 1 + 1 + (artWidth-40)/2

	output.WriteString("\033[s")
	// Title
	output.WriteString(fmt.Sprintf("\033[%d;%dH", absRow, absCol))
	output.WriteString("\033[1;36m")
	output.WriteString(title)
	output.WriteString("\033[0m")

	// Subtitle
	output.WriteString(fmt.Sprintf("\033[%d;%dH", absRow+2, absCol))
	output.WriteString("\033[1;37mChoose a template:\033[0m")

	// Options list
	for i, opt := range options {
		output.WriteString(fmt.Sprintf("\033[%d;%dH", absRow+4+i, absCol))
		if i == m.selectedItem {
			output.WriteString("\033[1;33m▶ ")
			output.WriteString(opt)
			output.WriteString("\033[0m")
		} else {
			output.WriteString("  ")
			output.WriteString(opt)
		}
	}

	// Help
	output.WriteString(fmt.Sprintf("\033[%d;%dH\033[2;37m↑/↓ Select • Enter Continue • q Quit\033[0m", absRow+10, absCol))
	output.WriteString("\033[u")

	return output.String()
}

// renderPreflight renders the preflight checks screen
func (m Model) renderPreflight() string {
	title := titleStyle.Render("🔍 Preflight Checks")
	subtitle := subtitleStyle.Render("Verifying your development environment...")

	var checks []string
	for _, check := range m.preflightChecks {
		var icon, status string
		switch check.Status {
		case CheckPending:
			icon = "⏳"
			status = mutedStyle.Render("Pending")
		case CheckRunning:
			icon = m.spinner.View()
			status = "Checking..."
		case CheckPassed:
			icon = "✅"
			status = successStyle.Render("Passed")
		case CheckWarning:
			icon = "⚠️"
			status = warningStyle.Render("Warning")
		case CheckFailed:
			icon = "❌"
			status = errorStyle.Render("Failed")
		}

		line := fmt.Sprintf("%s  %-25s %s", icon, check.Name, status)
		if check.Message != "" {
			line += "\n    " + mutedStyle.Render(check.Message)
		}
		if check.Fix != "" && check.Status == CheckFailed {
			line += "\n    " + warningStyle.Render("Fix: "+check.Fix)
		}
		checks = append(checks, line)
	}

	checksBox := boxStyle.Render(strings.Join(checks, "\n"))

	var footer string
	if m.preflightDone {
		allPassed := true
		for _, check := range m.preflightChecks {
			if check.Status == CheckFailed {
				allPassed = false
				break
			}
		}

		if allPassed {
			footer = successStyle.Render("\n✨ All checks passed! Press Enter to continue.")
		} else {
			footer = warningStyle.Render("\n⚠️  Some checks failed. You may encounter issues. Press Enter to continue anyway.")
		}
	} else {
		footer = mutedStyle.Render("\nRunning checks...")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		checksBox,
		footer,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderProjectBasics renders the project configuration screen
func (m Model) renderProjectBasics() string {
	title := titleStyle.Render("📁 Project Configuration")
	subtitle := subtitleStyle.Render("Let's set up your project basics")

	var fields []string

	// Project name
	nameLabel := "Project Name:"
	if m.currentInput == 0 {
		nameLabel = selectedStyle.Render("▶ " + nameLabel)
	} else {
		nameLabel = normalStyle.Render("  " + nameLabel)
	}
	fields = append(fields, fmt.Sprintf("%s\n  %s", nameLabel, m.textInputs[0].View()))

	// Module path
	moduleLabel := "Module Path:"
	if m.currentInput == 1 {
		moduleLabel = selectedStyle.Render("▶ " + moduleLabel)
	} else {
		moduleLabel = normalStyle.Render("  " + moduleLabel)
	}
	fields = append(fields, fmt.Sprintf("%s\n  %s", moduleLabel, m.textInputs[1].View()))

	// Port
	portLabel := "Dev Server Port:"
	if m.currentInput == 2 {
		portLabel = selectedStyle.Render("▶ " + portLabel)
	} else {
		portLabel = normalStyle.Render("  " + portLabel)
	}
	fields = append(fields, fmt.Sprintf("%s\n  %s", portLabel, m.textInputs[2].View()))

	// Git init checkbox
	gitLabel := "Initialize Git repository"
	gitCheck := "☐"
	if m.config.GitInit {
		gitCheck = "☑"
	}
	if m.currentInput == 3 {
		gitLabel = selectedStyle.Render(fmt.Sprintf("▶ %s %s", gitCheck, gitLabel))
	} else {
		gitLabel = normalStyle.Render(fmt.Sprintf("  %s %s", gitCheck, gitLabel))
	}
	fields = append(fields, gitLabel)

	fieldsBox := boxStyle.Render(strings.Join(fields, "\n\n"))

	help := helpStyle.Render("\nTab: Next field • Space: Toggle checkbox • Enter: Continue • Esc: Back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		fieldsBox,
		help,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderTemplate renders the template selection screen
func (m Model) renderTemplate() string {
	title := titleStyle.Render("🎨 Template Selection")
	subtitle := subtitleStyle.Render("Choose a starting template for your project")

	templates := []struct {
		name        string
		description string
		features    []string
	}{
		{
			name:        "base",
			description: "Configurable base template with customizable routing and styling",
			features:    []string{"Flexible routing", "Optional Tailwind", "Dark mode support", "Git integration"},
		},
		{
			name:        "blog",
			description: "Blog with routing and content management",
			features:    []string{"Dynamic routing", "Markdown support", "Dark mode", "Tailwind CSS"},
		},
		{
			name:        "counter",
			description: "Interactive counter demonstrating state management",
			features:    []string{"Reactive signals", "Client interactivity", "State persistence"},
		},
		{
			name:        "graphviewer",
			description: "Interactive knowledge graph visualization with multiple graph examples",
			features:    []string{"Canvas rendering", "Physics simulation", "Dark mode", "Multiple datasets"},
		},
	}

	var items []string
	for i, tmpl := range templates {
		var item string
		if i == m.selectedItem {
			item = selectedStyle.Render(fmt.Sprintf("▶ %s", strings.ToUpper(tmpl.name)))
			item += "\n  " + normalStyle.Render(tmpl.description)
			item += "\n  " + mutedStyle.Render("Features: "+strings.Join(tmpl.features, ", "))

			// Update config
			m.config.Template = tmpl.name
		} else {
			item = normalStyle.Render(fmt.Sprintf("  %s", strings.ToUpper(tmpl.name)))
			item += "\n  " + mutedStyle.Render(tmpl.description)
		}
		items = append(items, item)
	}

	templatesBox := boxStyle.Render(strings.Join(items, "\n\n"))

	// Feature checkboxes
	featuresTitle := normalStyle.Render("\n📦 Additional Examples:")
	features := []string{
		"Include SSR example route",
		"Include client-only example",
		"Include server-driven (live) example",
	}

	var featureItems []string
	for i, feature := range features {
		check := "☐"
		key := fmt.Sprintf("example_%d", i)
		if m.checkboxes[key] {
			check = "☑"
		}
		item := normalStyle.Render(fmt.Sprintf("  %s %s", check, feature))
		featureItems = append(featureItems, item)
	}

	featuresBox := strings.Join(featureItems, "\n")

	help := helpStyle.Render("\n↑/↓: Select template • Space: Toggle examples • Enter: Continue • Esc: Back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		templatesBox,
		featuresTitle,
		featuresBox,
		help,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderRouting renders the routing strategy selection
func (m Model) renderRouting() string {
	title := titleStyle.Render("🛣️  Routing Strategy")
	subtitle := subtitleStyle.Render("How would you like to handle routing?")

	strategies := []struct {
		name        string
		description string
		benefits    []string
	}{
		{
			name:        "file-based",
			description: "Zero config, automatic route discovery (Recommended)",
			benefits: []string{
				"Routes defined by file structure",
				"Automatic code generation",
				"Hot reload support",
				"Type-safe params",
			},
		},
		{
			name:        "programmatic",
			description: "Full control, manual configuration",
			benefits: []string{
				"Explicit route registration",
				"Custom middleware chains",
				"Dynamic routing logic",
				"Integration flexibility",
			},
		},
		{
			name:        "minimal",
			description: "Just the basics, configure everything yourself",
			benefits: []string{
				"Complete freedom",
				"No opinions imposed",
				"Good for experiments",
				"Smallest footprint",
			},
		},
	}

	var items []string
	for i, strategy := range strategies {
		var item string
		name := strings.ToUpper(strings.ReplaceAll(strategy.name, "-", " "))
		if i == m.selectedItem {
			item = selectedStyle.Render(fmt.Sprintf("▶ %s", name))
			item += "\n  " + normalStyle.Render(strategy.description)
			item += "\n  " + mutedStyle.Render("✓ "+strings.Join(strategy.benefits, "\n  ✓ "))

			// Update config
			m.config.RoutingStrategy = strategy.name
		} else {
			item = normalStyle.Render(fmt.Sprintf("  %s", name))
			item += "\n  " + mutedStyle.Render(strategy.description)
		}
		items = append(items, item)
	}

	strategiesBox := boxStyle.Render(strings.Join(items, "\n\n"))

	// Note about template compatibility
	note := warningStyle.Render("\nℹ️  Note: Blog template works best with file-based routing")

	help := helpStyle.Render("\n↑/↓: Select strategy • Enter: Continue • Esc: Back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		strategiesBox,
		note,
		help,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderStyling renders the styling configuration screen
func (m Model) renderStyling() string {
	title := titleStyle.Render("🎨 Styling Configuration")
	subtitle := subtitleStyle.Render("Configure how you want to style your application")

	var fields []string

	// Tailwind toggle
	tailwindCheck := "☐"
	if m.config.UseTailwind {
		tailwindCheck = "☑"
	}
	tailwindLabel := fmt.Sprintf("%s Enable Tailwind CSS", tailwindCheck)
	if m.selectedItem == 0 {
		tailwindLabel = selectedStyle.Render("▶ " + tailwindLabel)
	} else {
		tailwindLabel = normalStyle.Render("  " + tailwindLabel)
	}
	fields = append(fields, tailwindLabel)

	// Tailwind strategy (only if enabled)
	if m.config.UseTailwind {
		fields = append(fields, normalStyle.Render("\n  Tailwind Strategy:"))

		strategies := []struct {
			value       string
			label       string
			description string
		}{
			{"auto", "Auto", "Automatically detect best option"},
			{"standalone", "Standalone", "Download Tailwind binary (no Node.js required)"},
			{"npm", "NPM", "Use npm/yarn/pnpm (requires Node.js)"},
		}

		for i, strategy := range strategies {
			radio := "○"
			if m.config.TailwindStrategy == strategy.value {
				radio = "●"
			}
			label := fmt.Sprintf("    %s %s - %s", radio, strategy.label, mutedStyle.Render(strategy.description))
			if m.selectedItem == i+1 {
				label = selectedStyle.Render("▶ " + label[2:])
			} else {
				label = normalStyle.Render(label)
			}
			fields = append(fields, label)
		}
	}

	// Dark mode toggle
	darkCheck := "☐"
	if m.config.DarkMode {
		darkCheck = "☑"
	}
	darkLabel := fmt.Sprintf("%s Enable Dark/Light mode toggle", darkCheck)
	idx := 4
	if !m.config.UseTailwind {
		idx = 1
	}
	if m.selectedItem == idx {
		darkLabel = selectedStyle.Render("▶ " + darkLabel)
	} else {
		darkLabel = normalStyle.Render("  " + darkLabel)
	}
	fields = append(fields, "\n"+darkLabel)

	fieldsBox := boxStyle.Render(strings.Join(fields, "\n"))

	// CSS features info
	info := mutedStyle.Render(`
Additional CSS features available:
• Scoped styles with vango.Style()
• CSS-in-Go with automatic extraction
• Global stylesheets support
• Hot reload for all CSS changes`)

	help := helpStyle.Render("\n↑/↓: Navigate • Space: Toggle • Enter: Continue • Esc: Back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		fieldsBox,
		info,
		help,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderSummary renders the configuration summary
func (m Model) renderSummary() string {
	title := titleStyle.Render("📋 Configuration Summary")
	subtitle := subtitleStyle.Render("Review your project configuration")

	config := m.GetConfig()

	summary := []string{
		fmt.Sprintf("Project Name:     %s", selectedStyle.Render(config.Name)),
		fmt.Sprintf("Module Path:      %s", normalStyle.Render(config.Module)),
		fmt.Sprintf("Template:         %s", normalStyle.Render(strings.ToUpper(config.Template))),
		fmt.Sprintf("Routing:          %s", normalStyle.Render(config.RoutingStrategy)),
		fmt.Sprintf("Dev Port:         %s", normalStyle.Render(fmt.Sprintf("%d", config.Port))),
	}

	if config.UseTailwind {
		summary = append(summary, fmt.Sprintf("Tailwind CSS:     %s (%s)",
			successStyle.Render("Enabled"),
			mutedStyle.Render(config.TailwindStrategy)))
	}

	if config.DarkMode {
		summary = append(summary, fmt.Sprintf("Dark Mode:        %s", successStyle.Render("Enabled")))
	}

	if config.GitInit {
		summary = append(summary, fmt.Sprintf("Git Repository:   %s", successStyle.Render("Will initialize")))
	}

	summaryBox := boxStyle.Render(strings.Join(summary, "\n"))

	// Project structure preview
	structureTitle := normalStyle.Render("\n📁 Project Structure:")
	structure := m.getProjectStructure(config)
	structureBox := mutedStyle.Render(structure)

	confirm := selectedStyle.Render("\n✨ Press Enter to create your project")
	back := helpStyle.Render("Press Esc to go back and modify")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		summaryBox,
		structureTitle,
		structureBox,
		confirm,
		back,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderExecution renders the project creation progress
func (m Model) renderExecution() string {
	title := titleStyle.Render("🚀 Creating Your Project")
	subtitle := subtitleStyle.Render("Setting up " + m.config.Name + "...")

	var steps []string
	for _, step := range m.executionSteps {
		var icon, status string
		switch step.Status {
		case ExecPending:
			icon = "⏳"
			status = mutedStyle.Render(step.Name)
		case ExecRunning:
			icon = m.spinner.View()
			status = normalStyle.Render(step.Name)
		case ExecComplete:
			icon = "✅"
			status = successStyle.Render(step.Name)
		case ExecFailed:
			icon = "❌"
			status = errorStyle.Render(step.Name)
		}

		line := fmt.Sprintf("%s  %s", icon, status)
		if step.Message != "" {
			line += "\n    " + mutedStyle.Render(step.Message)
		}
		steps = append(steps, line)
	}

	stepsBox := boxStyle.Render(strings.Join(steps, "\n"))

	// Progress bar
	var progressBar string
	if m.currentExecStep > 0 && len(m.executionSteps) > 0 {
		progress := float64(m.currentExecStep) / float64(len(m.executionSteps))
		progressBar = "\n" + m.progress.ViewAs(progress)
	}

	// Error message if any
	var errorMsg string
	if m.executionError != nil {
		errorMsg = "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.executionError))
		errorMsg += "\n" + helpStyle.Render("Press Ctrl+C to exit")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		stepsBox,
		progressBar,
		errorMsg,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderComplete renders the completion screen
func (m Model) renderComplete() string {
	title := titleStyle.Render("✨ Project Created Successfully!")

	config := m.GetConfig()

	success := successStyle.Render(fmt.Sprintf(`
Your Vango project "%s" has been created!
`, config.Name))

	nextSteps := normalStyle.Render(`
📚 Next Steps:
`) + mutedStyle.Render(fmt.Sprintf(`
   cd %s
   vango dev
`, config.Name))

	features := normalStyle.Render(`

🎯 What's included:`)

	included := []string{
		"Complete project structure",
		"Development server configuration",
		"Hot module reloading",
	}

	if config.UseTailwind {
		included = append(included, "Tailwind CSS setup")
	}
	if config.DarkMode {
		included = append(included, "Dark/light mode support")
	}
	if config.GitInit {
		included = append(included, "Git repository initialized")
	}

	for _, item := range included {
		features += "\n   " + successStyle.Render("✓") + " " + mutedStyle.Render(item)
	}

	resources := normalStyle.Render(`

📖 Resources:`) + mutedStyle.Render(`
   Documentation: https://vango.dev/docs
   Examples:      https://github.com/recera/vango/examples
   Discord:       https://discord.gg/vango`)

	footer := selectedStyle.Render("\n\nPress Enter to exit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		success,
		nextSteps,
		features,
		resources,
		footer,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderHelp renders the help screen
func (m Model) renderHelp() string {
	title := titleStyle.Render("⌨️  Keyboard Shortcuts")

	shortcuts := [][]string{
		{"↑/↓, j/k", "Navigate up/down"},
		{"←/→, h/l", "Navigate left/right"},
		{"Tab", "Next field"},
		{"Space", "Toggle checkbox/selection"},
		{"Enter", "Confirm/Continue"},
		{"Esc", "Go back"},
		{"?", "Toggle this help"},
		{"q, Ctrl+C", "Quit"},
	}

	var helpText []string
	for _, shortcut := range shortcuts {
		key := selectedStyle.Render(fmt.Sprintf("%-12s", shortcut[0]))
		desc := normalStyle.Render(shortcut[1])
		helpText = append(helpText, fmt.Sprintf("%s  %s", key, desc))
	}

	helpBox := boxStyle.Render(strings.Join(helpText, "\n"))

	footer := mutedStyle.Render("\nPress ? to close help")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		helpBox,
		footer,
	)

	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderFooter renders the footer with context-sensitive keybindings
func (m Model) renderFooter() string {
	var keys []string

	switch m.step {
	case StepWelcome:
		keys = []string{"Enter: Begin", "q: Quit"}
	case StepPreflight:
		if m.preflightDone {
			keys = []string{"Enter: Continue", "q: Quit"}
		} else {
			keys = []string{"Checking...", "q: Quit"}
		}
	case StepProjectBasics:
		keys = []string{"Tab: Next", "Enter: Continue", "Esc: Back", "q: Quit"}
	case StepTemplate, StepRouting:
		keys = []string{"↑/↓: Select", "Space: Toggle", "Enter: Continue", "Esc: Back"}
	case StepStyling:
		keys = []string{"↑/↓: Navigate", "Space: Toggle", "Enter: Continue", "Esc: Back"}
	case StepSummary:
		keys = []string{"Enter: Create", "Esc: Back", "q: Quit"}
	case StepExecuting:
		keys = []string{"Creating project...", "Ctrl+C: Cancel"}
	case StepComplete:
		keys = []string{"Enter: Exit"}
	default:
		keys = []string{"?: Help", "q: Quit"}
	}

	// Always add help unless in execution
	if m.step != StepExecuting && m.step != StepComplete {
		keys = append(keys, "?: Help")
	}

	return footerStyle.Render(strings.Join(keys, " • "))
}

// getProjectStructure returns a string representation of the project structure
func (m Model) getProjectStructure(config ProjectConfig) string {
	structure := config.Name + "/\n"

	if config.RoutingStrategy == "file-based" {
		structure += `├── app/
│   ├── routes/
│   │   ├── index.go
│   │   ├── _layout.go
│   │   └── _404.go
│   └── main.go
├── public/
│   └── index.html`
	} else if config.RoutingStrategy == "programmatic" {
		structure += `├── app/
│   ├── handlers/
│   │   └── home.go
│   └── main.go
├── server/
│   └── routes.go
├── public/`
	} else {
		structure += `├── main.go
├── public/`
	}

	structure += `
├── vango.json
├── go.mod`

	if config.UseTailwind {
		structure += `
├── tailwind.config.js
├── package.json`
	}

	if config.GitInit {
		structure += `
├── .gitignore
└── README.md`
	} else {
		structure += `
└── README.md`
	}

	return structure
}
