package styling

import (
	"strings"
	"sync"
)

// StyleRegistry collects all component styles for injection
type StyleRegistry struct {
	mu     sync.RWMutex
	styles map[string]*ComponentStyle
}

var (
	globalRegistry = &StyleRegistry{
		styles: make(map[string]*ComponentStyle),
	}
)

// Register adds a component style to the global registry
func Register(style *ComponentStyle) {
	if style == nil || style.CSS == "" {
		return
	}
	
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	
	// Use CSS content as key to avoid duplicates
	key := style.CSS
	if style.Hash != "" {
		key = style.Hash
	}
	
	globalRegistry.styles[key] = style
}

// GetAllCSS returns all registered CSS as a single string
func GetAllCSS() string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	var cssBuilder strings.Builder
	for _, style := range globalRegistry.styles {
		cssBuilder.WriteString(style.CSS)
		cssBuilder.WriteString("\n")
	}
	
	return cssBuilder.String()
}

// Reset clears all registered styles (useful for testing)
func Reset() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.styles = make(map[string]*ComponentStyle)
}

// StyleWithRegistry creates a new ComponentStyle and registers it
func StyleWithRegistry(css string) *ComponentStyle {
	style := Style(css)
	Register(style)
	return style
}