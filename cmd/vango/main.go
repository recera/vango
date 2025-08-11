package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0-preview"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "vango",
		Short: "Vango - The Go Frontend Framework",
		Long: `Vango is a Go-native, hybrid-rendered UI framework designed to provide
a first-class developer experience, exceptional performance, and a robust,
type-safe environment for building modern web applications.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add commands
	rootCmd.AddCommand(newDevCommand())
	rootCmd.AddCommand(newBuildCommand())
	rootCmd.AddCommand(newCreateCommand())
	rootCmd.AddCommand(newGenCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}