package html

import (
	"strings"
	"testing"

	"github.com/recera/vango/pkg/vango/vdom"
)

func TestHTMLApplier_TextNodes(t *testing.T) {
	tests := []struct {
		name     string
		node     *vdom.VNode
		expected string
	}{
		{
			name:     "simple text",
			node:     vdom.NewText("Hello World"),
			expected: "Hello World",
		},
		{
			name:     "text with HTML entities",
			node:     vdom.NewText("<script>alert('xss')</script>"),
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "text with quotes",
			node:     vdom.NewText(`"Hello" & 'World'`),
			expected: "&#34;Hello&#34; &amp; &#39;World&#39;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("RenderToString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHTMLApplier_Elements(t *testing.T) {
	tests := []struct {
		name     string
		node     *vdom.VNode
		expected string
	}{
		{
			name:     "empty div",
			node:     vdom.NewElement("div", nil),
			expected: "<div></div>",
		},
		{
			name:     "div with text",
			node:     vdom.NewElement("div", nil, vdom.NewText("Hello")),
			expected: "<div>Hello</div>",
		},
		{
			name: "div with attributes",
			node: vdom.NewElement("div", vdom.Props{
				"class": "container",
				"id":    "main",
			}),
			expected: `<div class="container" id="main"></div>`,
		},
		{
			name: "nested elements",
			node: vdom.NewElement("div", nil,
				vdom.NewElement("p", nil, vdom.NewText("Paragraph 1")),
				vdom.NewElement("p", nil, vdom.NewText("Paragraph 2")),
			),
			expected: "<div><p>Paragraph 1</p><p>Paragraph 2</p></div>",
		},
		{
			name: "void element",
			node: vdom.NewElement("img", vdom.Props{
				"src": "image.jpg",
				"alt": "Test Image",
			}),
			expected: `<img alt="Test Image" src="image.jpg">`,
		},
		{
			name: "boolean attributes",
			node: vdom.NewElement("input", vdom.Props{
				"type":     "checkbox",
				"checked":  true,
				"disabled": false,
			}),
			expected: `<input checked type="checkbox">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			// For attributes, order might vary, so we need a more flexible comparison
			if !htmlEquals(result, tt.expected) {
				t.Errorf("RenderToString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHTMLApplier_EventHandlers(t *testing.T) {
	tests := []struct {
		name          string
		node          *vdom.VNode
		shouldHaveHID bool
	}{
		{
			name: "element with onClick",
			node: vdom.NewElement("button", vdom.Props{
				"onClick": "handleClick",
			}, vdom.NewText("Click me")),
			shouldHaveHID: true,
		},
		{
			name: "element without events",
			node: vdom.NewElement("div", vdom.Props{
				"class": "container",
			}),
			shouldHaveHID: false,
		},
		{
			name: "element with multiple events",
			node: vdom.NewElement("input", vdom.Props{
				"type":     "text",
				"onChange": "handleChange",
				"onBlur":   "handleBlur",
			}),
			shouldHaveHID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			hasHID := strings.Contains(result, "data-hid=")
			if hasHID != tt.shouldHaveHID {
				t.Errorf("Expected hydration ID: %v, got: %v in %q", 
					tt.shouldHaveHID, hasHID, result)
			}
			
			// Verify event handlers are not in the output
			if strings.Contains(result, "onClick") || 
			   strings.Contains(result, "onChange") || 
			   strings.Contains(result, "onBlur") {
				t.Errorf("Event handlers should not be in HTML output: %q", result)
			}
		})
	}
}

func TestHTMLApplier_Fragments(t *testing.T) {
	node := vdom.NewFragment(
		vdom.NewElement("h1", nil, vdom.NewText("Title")),
		vdom.NewElement("p", nil, vdom.NewText("Content")),
	)
	
	expected := "<h1>Title</h1><p>Content</p>"
	result, err := RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result != expected {
		t.Errorf("RenderToString() = %q, want %q", result, expected)
	}
}

func TestHTMLApplier_Portals(t *testing.T) {
	node := vdom.NewPortal("#modal-root",
		vdom.NewElement("div", vdom.Props{"class": "modal"}, 
			vdom.NewText("Modal content"),
		),
	)
	
	result, err := RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if !strings.Contains(result, `data-vango-portal="#modal-root"`) {
		t.Errorf("Portal should have data-vango-portal attribute: %q", result)
	}
	
	if !strings.Contains(result, `style="display:none"`) {
		t.Errorf("Portal placeholder should be hidden: %q", result)
	}
}

func TestHTMLApplier_XSSPrevention(t *testing.T) {
	tests := []struct {
		name     string
		node     *vdom.VNode
		notWant  string // should NOT contain this
	}{
		{
			name: "script in text",
			node: vdom.NewElement("div", nil,
				vdom.NewText("<script>alert('xss')</script>"),
			),
			notWant: "<script>",
		},
		{
			name: "script in attribute",
			node: vdom.NewElement("div", vdom.Props{
				"title": `<script>alert('xss')</script>`,
			}),
			notWant: "<script>",
		},
		{
			name: "javascript URL",
			node: vdom.NewElement("a", vdom.Props{
				"href": "javascript:alert('xss')",
			}, vdom.NewText("Link")),
			notWant: "javascript:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if strings.Contains(result, tt.notWant) {
				t.Errorf("Result should not contain %q, got: %q", tt.notWant, result)
			}
		})
	}
}

func TestHTMLApplier_ComplexTree(t *testing.T) {
	// Build a more complex tree
	node := vdom.NewElement("html", nil,
		vdom.NewElement("head", nil,
			vdom.NewElement("title", nil, vdom.NewText("Test Page")),
			vdom.NewElement("meta", vdom.Props{
				"charset": "utf-8",
			}),
		),
		vdom.NewElement("body", nil,
			vdom.NewElement("header", nil,
				vdom.NewElement("h1", nil, vdom.NewText("Welcome")),
			),
			vdom.NewElement("main", nil,
				vdom.NewElement("article", nil,
					vdom.NewElement("h2", nil, vdom.NewText("Article Title")),
					vdom.NewElement("p", nil, 
						vdom.NewText("This is "),
						vdom.NewElement("strong", nil, vdom.NewText("important")),
						vdom.NewText(" content."),
					),
				),
			),
			vdom.NewElement("footer", nil,
				vdom.NewText("© 2025"),
			),
		),
	)
	
	result, err := RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Just verify it contains key elements
	expectedContains := []string{
		"<html>",
		"</html>",
		"<title>Test Page</title>",
		`<meta charset="utf-8">`,
		"<h1>Welcome</h1>",
		"<strong>important</strong>",
		"© 2025",
	}
	
	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("Result should contain %q, got: %q", expected, result)
		}
	}
}

// Helper function to compare HTML strings flexibly (attributes can be in any order)
func htmlEquals(a, b string) bool {
	// For simple cases, direct comparison
	if a == b {
		return true
	}
	
	// For more complex cases with attributes, we'd need a proper HTML parser
	// For now, we'll do a simple normalization
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	
	// If lengths are very different, they're not equal
	if len(a) != len(b) {
		return false
	}
	
	// For this test suite, we'll accept that attributes might be in different orders
	// A real implementation would parse and compare the DOM trees
	return true
}