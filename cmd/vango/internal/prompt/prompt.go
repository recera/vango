package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompter handles interactive prompts
type Prompter struct {
	reader *bufio.Reader
}

// New creates a new prompter
func New() *Prompter {
	return &Prompter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Text prompts for text input
func (p *Prompter) Text(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, _ := p.reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" && defaultValue != "" {
		return defaultValue
	}
	
	return input
}

// Confirm prompts for yes/no confirmation
func (p *Prompter) Confirm(prompt string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}
	
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	
	input, _ := p.reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "" {
		return defaultYes
	}
	
	return input == "y" || input == "yes"
}

// Select prompts for selection from options
func (p *Prompter) Select(prompt string, options []string, defaultIndex int) int {
	fmt.Println(prompt)
	for i, option := range options {
		if i == defaultIndex {
			fmt.Printf("  > %d) %s (default)\n", i+1, option)
		} else {
			fmt.Printf("    %d) %s\n", i+1, option)
		}
	}
	
	fmt.Printf("Enter choice [%d]: ", defaultIndex+1)
	
	input, _ := p.reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return defaultIndex
	}
	
	// Try to parse as number
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
		if choice >= 1 && choice <= len(options) {
			return choice - 1
		}
	}
	
	// Try to match by name
	inputLower := strings.ToLower(input)
	for i, option := range options {
		if strings.ToLower(option) == inputLower {
			return i
		}
	}
	
	// Default to defaultIndex if invalid input
	fmt.Println("Invalid choice, using default.")
	return defaultIndex
}

// MultiSelect prompts for multiple selections
func (p *Prompter) MultiSelect(prompt string, options []string, defaults []bool) []bool {
	if len(defaults) != len(options) {
		defaults = make([]bool, len(options))
	}
	
	fmt.Println(prompt)
	fmt.Println("(Use space-separated numbers to select multiple, or 'all'/'none')")
	
	for i, option := range options {
		if defaults[i] {
			fmt.Printf("  [x] %d) %s\n", i+1, option)
		} else {
			fmt.Printf("  [ ] %d) %s\n", i+1, option)
		}
	}
	
	fmt.Print("Enter choices: ")
	
	input, _ := p.reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "" {
		return defaults
	}
	
	selected := make([]bool, len(options))
	
	if input == "all" {
		for i := range selected {
			selected[i] = true
		}
		return selected
	}
	
	if input == "none" {
		return selected
	}
	
	// Parse space-separated numbers
	parts := strings.Fields(input)
	for _, part := range parts {
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			if num >= 1 && num <= len(options) {
				selected[num-1] = true
			}
		}
	}
	
	return selected
}