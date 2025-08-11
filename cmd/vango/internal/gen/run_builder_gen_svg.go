// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/recera/vango/cmd/vango/internal/gen"
)

func main() {
	// Get the project root
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Find the spec file
	specPath := filepath.Join(wd, "internal", "spec", "html.yml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		// Try from different locations
		specPath = filepath.Join(wd, "..", "..", "..", "internal", "spec", "html.yml")
		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			specPath = filepath.Join(wd, "..", "..", "internal", "spec", "html.yml")
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				log.Fatalf("Could not find html.yml spec file. Tried multiple paths from %s", wd)
			}
		}
	}

	fmt.Printf("Using spec file: %s\n", specPath)

	// Run the new builder generator with SVG support
	if err := gen.RunNew(specPath); err != nil {
		log.Fatalf("Failed to generate builders: %v", err)
	}

	fmt.Println("✅ Successfully generated builders with SVG support!")
}