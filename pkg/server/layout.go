package server

import (
	"github.com/recera/vango/pkg/vango/vdom"
)

// Layout is the interface that all layout components must implement
type Layout interface {
	// Wrap wraps the given child component with the layout
	Wrap(child *vdom.VNode) *vdom.VNode
}

// LayoutFunc is a function type that implements the Layout interface
type LayoutFunc func(child *vdom.VNode) *vdom.VNode

// Wrap implements the Layout interface for LayoutFunc
func (f LayoutFunc) Wrap(child *vdom.VNode) *vdom.VNode {
	return f(child)
}

// LayoutRegistry manages layouts for different routes
type LayoutRegistry struct {
	layouts map[string]Layout
}

// NewLayoutRegistry creates a new layout registry
func NewLayoutRegistry() *LayoutRegistry {
	return &LayoutRegistry{
		layouts: make(map[string]Layout),
	}
}

// Register registers a layout for a specific path pattern
func (r *LayoutRegistry) Register(pattern string, layout Layout) {
	r.layouts[pattern] = layout
}

// RegisterFunc registers a layout function for a specific path pattern
func (r *LayoutRegistry) RegisterFunc(pattern string, layoutFunc func(child *vdom.VNode) *vdom.VNode) {
	r.Register(pattern, LayoutFunc(layoutFunc))
}

// GetLayout returns the layout for a given path
func (r *LayoutRegistry) GetLayout(path string) Layout {
	// First try exact match
	if layout, ok := r.layouts[path]; ok {
		return layout
	}
	
	// Then try directory-level layouts
	// For example, /blog/post would match /blog/* layout
	for pattern, layout := range r.layouts {
		if matchesPattern(path, pattern) {
			return layout
		}
	}
	
	// Finally, check for root layout
	if layout, ok := r.layouts["/"]; ok {
		return layout
	}
	
	return nil
}

// ApplyLayout applies the appropriate layout to a VNode
func (r *LayoutRegistry) ApplyLayout(path string, content *vdom.VNode) *vdom.VNode {
	layout := r.GetLayout(path)
	if layout != nil {
		return layout.Wrap(content)
	}
	return content
}

// matchesPattern checks if a path matches a pattern
// Patterns can include:
// - Exact matches: "/about"
// - Directory matches: "/blog/*"
// - Root layout: "/"
func matchesPattern(path, pattern string) bool {
	// Handle wildcard patterns
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	
	// Handle directory patterns (e.g., /blog matches /blog/*)
	if len(pattern) > 1 && pattern[len(pattern)-1] == '/' {
		return len(path) >= len(pattern) && path[:len(pattern)] == pattern
	}
	
	return path == pattern
}