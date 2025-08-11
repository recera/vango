package html

import (
	"fmt"
	"html"
	"io"
	"strings"
	"sync"

	"github.com/recera/vango/pkg/vango/vdom"
)

// voidElements are HTML elements that cannot have children
var voidElements = map[string]bool{
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"link":   true,
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,
}

// booleanAttributes are HTML attributes that are boolean flags
var booleanAttributes = map[string]bool{
	"checked":   true,
	"disabled":  true,
	"readonly":  true,
	"required":  true,
	"selected":  true,
	"defer":     true,
	"async":     true,
	"multiple":  true,
	"autofocus": true,
}

// HTMLApplier renders VNodes to HTML
type HTMLApplier struct {
	w              io.Writer
	hydrationIDGen *HydrationIDGenerator
	err            error
}

// HydrationIDGenerator generates unique IDs for hydration
type HydrationIDGenerator struct {
	mu      sync.Mutex
	counter uint32
}

// NewHydrationIDGenerator creates a new hydration ID generator
func NewHydrationIDGenerator() *HydrationIDGenerator {
	return &HydrationIDGenerator{counter: 1}
}

// Next returns the next hydration ID
func (g *HydrationIDGenerator) Next() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.counter
	g.counter++
	return fmt.Sprintf("h%d", id)
}

// NewHTMLApplier creates a new HTML applier
func NewHTMLApplier(w io.Writer) *HTMLApplier {
	return &HTMLApplier{
		w:              w,
		hydrationIDGen: NewHydrationIDGenerator(),
	}
}

// Apply renders a VNode tree to HTML
func (a *HTMLApplier) Apply(prev, next *vdom.VNode) error {
	if prev != nil {
		return fmt.Errorf("htmlApplier does not support incremental updates")
	}

	if next == nil {
		return nil
	}

	a.renderNode(next)
	return a.err
}

// write helper that tracks errors
func (a *HTMLApplier) write(s string) {
	if a.err != nil {
		return
	}
	_, a.err = io.WriteString(a.w, s)
}

// renderNode renders a single VNode
func (a *HTMLApplier) renderNode(node *vdom.VNode) {
	if node == nil || a.err != nil {
		return
	}

	switch node.Kind {
	case vdom.KindText:
		// HTML escape text content to prevent XSS
		a.write(html.EscapeString(node.Text))

	case vdom.KindElement:
		a.renderElement(node)

	case vdom.KindFragment:
		// Fragments just render their children
		for i := range node.Kids {
			a.renderNode(&node.Kids[i])
		}

	case vdom.KindPortal:
		// Portals need special handling - for SSR we render a placeholder
		a.write(fmt.Sprintf(`<div data-vango-portal="%s" style="display:none"></div>`,
			html.EscapeString(node.PortalTarget)))
	}
}

// renderElement renders an element node
func (a *HTMLApplier) renderElement(node *vdom.VNode) {
	// Start tag
	a.write("<")
	a.write(node.Tag)

	// Check if this node needs a hydration ID
	needsHydrationID := false
	if node.Props != nil {
		for key := range node.Props {
			if len(key) > 2 && key[0] == 'o' && key[1] == 'n' {
				needsHydrationID = true
				break
			}
		}
	}

	// Add hydration ID if needed
	var hydrationID string
	if needsHydrationID {
		hydrationID = a.hydrationIDGen.Next()
		a.write(fmt.Sprintf(` data-hid="%s"`, hydrationID))
	}

	// Render attributes
	if node.Props != nil {
		for key, value := range node.Props {
			// Skip event handlers and special props
			if key == "key" || key == "ref" || (len(key) > 2 && key[0] == 'o' && key[1] == 'n') {
				continue
			}

			// Handle boolean attributes
			if booleanAttributes[key] {
				if v, ok := value.(bool); ok && v {
					a.write(" ")
					a.write(key)
				}
				continue
			}

			// Regular attributes
			valueStr := fmt.Sprintf("%v", value)

			// Security: prevent javascript: URLs in href/src attributes
			if (key == "href" || key == "src") && strings.HasPrefix(strings.ToLower(valueStr), "javascript:") {
				valueStr = "#"
			}

			a.write(" ")
			a.write(key)
			a.write(`="`)
			a.write(html.EscapeString(valueStr))
			a.write(`"`)
		}
	}

	// Close opening tag
	a.write(">")

	// Void elements don't have closing tags or children
	if voidElements[node.Tag] {
		return
	}

	// Render children with special handling for script/style tags
	// Script and style tags should not have their content escaped
	isRawTextElement := node.Tag == "script" || node.Tag == "style"
	for i := range node.Kids {
		if isRawTextElement {
			a.renderRawNode(&node.Kids[i])
		} else {
			a.renderNode(&node.Kids[i])
		}
	}

	// Closing tag
	a.write("</")
	a.write(node.Tag)
	a.write(">")
}

// renderRawNode renders a node without HTML escaping (for script/style content)
func (a *HTMLApplier) renderRawNode(node *vdom.VNode) {
	if node == nil || a.err != nil {
		return
	}

	switch node.Kind {
	case vdom.KindText:
		// Don't escape text inside script/style tags
		a.write(node.Text)

	case vdom.KindElement:
		// This shouldn't happen inside script/style but handle anyway
		a.renderElement(node)

	case vdom.KindFragment:
		// Fragments just render their children
		for i := range node.Kids {
			a.renderRawNode(&node.Kids[i])
		}
	}
}

// RenderToString is a convenience function to render a VNode to a string
func RenderToString(node *vdom.VNode) (string, error) {
	var buf strings.Builder
	applier := NewHTMLApplier(&buf)
	err := applier.Apply(nil, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
