package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectConfig holds the configuration for a new project
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

// TemplateGenerator is the interface for all template generators
type TemplateGenerator interface {
	Generate(config *ProjectConfig) error
	Name() string
	Description() string
}

// Registry holds all available templates
var Registry = make(map[string]TemplateGenerator)

// Register adds a template to the registry
func Register(name string, generator TemplateGenerator) {
	Registry[name] = generator
}

// Generate creates a project using the specified template
func Generate(config *ProjectConfig) error {
	generator, exists := Registry[config.Template]
	if !exists {
		return fmt.Errorf("unknown template: %s", config.Template)
	}

	// Ensure directory is set
	if config.Directory == "" {
		config.Directory = config.Name
	}

	// Ensure module is set
	if config.Module == "" {
		config.Module = config.Name
	}

	// Create base directory structure
	if err := createBaseStructure(config); err != nil {
		return err
	}

	// Generate template-specific files
	if err := generator.Generate(config); err != nil {
		return err
	}

	// Create common files
	if err := createCommonFiles(config); err != nil {
		return err
	}

	// Create Tailwind config if enabled
	if config.UseTailwind {
		if err := createTailwindConfig(config); err != nil {
			return err
		}
	}

	return nil
}

// createBaseStructure creates the base directory structure for all templates
func createBaseStructure(config *ProjectConfig) error {
	dirs := []string{
		"app",
		"public",
		"internal/assets",
		"styles", // Always create styles directory for CSS files
	}

	// Add routing-specific directories
	switch config.RoutingStrategy {
	case "file-based":
		dirs = append(dirs,
			"app/routes",
			"app/components",
			"app/layouts",
			"app/styles",
		)
	case "programmatic":
		dirs = append(dirs,
			"app/handlers",
			"app/components",
			"server",
		)
	case "minimal":
		// Just the basics
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(config.Directory, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// WriteFile is a helper to write content to a file
func WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// GetTemplateNames returns a list of available template names
func GetTemplateNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// GetTemplateDescriptions returns template names with descriptions
func GetTemplateDescriptions() []string {
	descriptions := make([]string, 0, len(Registry))
	for name, gen := range Registry {
		descriptions = append(descriptions, fmt.Sprintf("%s - %s", name, gen.Description()))
	}
	return descriptions
}
