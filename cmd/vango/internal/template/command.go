package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// ProcessTemplateFile processes a .vex file and generates Go code
func ProcessTemplateFile(filename string) error {
	// Read the source file
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create parser
	parser := NewTemplateParser(filename, string(source))
	
	// Parse the template
	if err := parser.Parse(); err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Generate Go code
	code, err := parser.GenerateCode()
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}
	
	// Write output file
	// Change .vex to .vex.go
	outputFile := strings.TrimSuffix(filename, ".vex") + ".vex.go"
	if err := ioutil.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	
	fmt.Printf("✅ Generated %s from %s\n", outputFile, filename)
	return nil
}

// ProcessDirectory processes all .vex files in a directory
func ProcessDirectory(dir string) error {
	// Use filepath.Walk to find .vex files recursively
	var templateFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".vex") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find template files: %w", err)
	}
	
	for _, file := range templateFiles {
		if err := ProcessTemplateFile(file); err != nil {
			return fmt.Errorf("failed to process %s: %w", file, err)
		}
	}
	
	return nil
}