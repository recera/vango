package ui

import (
	_ "embed"
	"strings"
)

//go:embed vango_art.txt
var vangoArtData string

// VangoArt contains the ASCII art for the Vango logo
var VangoArt = []string{
	`                                                                     `,
	`                                                                     `,
	`    ██╗   ██╗ █████╗ ███╗   ██╗ ██████╗  ██████╗                   `,
	`    ██║   ██║██╔══██╗████╗  ██║██╔════╝ ██╔═══██╗                  `,
	`    ██║   ██║███████║██╔██╗ ██║██║  ███╗██║   ██║                  `,
	`    ╚██╗ ██╔╝██╔══██║██║╚██╗██║██║   ██║██║   ██║                  `,
	`     ╚████╔╝ ██║  ██║██║ ╚████║╚██████╔╝╚██████╔╝                  `,
	`      ╚═══╝  ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝  ╚═════╝                   `,
	`                                                                     `,
	`         ╔═══════════════════════════════════════════╗              `,
	`         ║  The Go-Native UI Framework for the Web   ║              `,
	`         ╚═══════════════════════════════════════════╝              `,
	`                                                                     `,
	`              Build • Ship • Scale with Confidence                  `,
	`                                                                     `,
}

// VangoArtCompact is a more compact version for smaller terminals
var VangoArtCompact = []string{
	`╦  ╦┌─┐┌┐┌┌─┐┌─┐`,
	`╚╗╔╝├─┤││││ ┬│ │`,
	` ╚╝ ┴ ┴┘└┘└─┘└─┘`,
	`Go-Native UI Framework`,
}

// GetArt returns the appropriate ASCII art based on terminal size
func GetArt(width, height int) []string {
	if width < 70 || height < 20 {
		return VangoArtCompact
	}
	return VangoArt
}

// LoadBackgroundArt loads the background art from embedded file
func LoadBackgroundArt() []string {
	lines := strings.Split(vangoArtData, "\n")
	// Ensure we have exactly 19 lines
	if len(lines) > 19 {
		lines = lines[:19]
	}
	return lines
}