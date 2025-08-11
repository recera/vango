// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/recera/vango/cmd/vango/internal/gen"
)

func main() {
	specPath := "internal/spec/html.yml"
	
	// Check if spec file exists
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		log.Fatal("HTML spec file not found at:", specPath)
	}
	
	fmt.Println("Generating builder API from spec:", specPath)
	
	if err := gen.Run(specPath); err != nil {
		log.Fatal("Failed to generate builder API:", err)
	}
	
	fmt.Println("✅ Builder API generated successfully!")
}