package styling

import (
	"strings"
	"testing"
)

func TestStyle(t *testing.T) {
	css := `
		.card {
			background: white;
			padding: 1rem;
		}
		.card.active {
			background: blue;
		}
	`
	
	style := Style(css)
	
	// Check that hash was generated
	if style.Hash == "" {
		t.Error("Expected hash to be generated")
	}
	
	// Check that CSS is stored
	if style.CSS != css {
		t.Errorf("Expected CSS to be stored, got %s", style.CSS)
	}
	
	// Check that class names were extracted
	if len(style.names) != 2 {
		t.Errorf("Expected 2 class names, got %d", len(style.names))
	}
	
	// Check specific class names
	cardClass := style.Class("card")
	if !strings.HasPrefix(cardClass, "_") {
		t.Errorf("Expected hashed class name to start with _, got %s", cardClass)
	}
	
	activeClass := style.Class("card.active")
	if !strings.HasPrefix(activeClass, "_") {
		t.Errorf("Expected hashed class name to start with _, got %s", activeClass)
	}
}

func TestStyleRegistry(t *testing.T) {
	// Clear registry first
	Reset()
	
	// Create and register styles
	style1 := StyleWithRegistry(`.test1 { color: red; }`)
	_ = StyleWithRegistry(`.test2 { color: blue; }`) // style2 registered but not directly used
	
	// Check that styles were registered
	allCSS := GetAllCSS()
	
	if !strings.Contains(allCSS, "color: red") {
		t.Error("Expected style1 CSS in registry")
	}
	
	if !strings.Contains(allCSS, "color: blue") {
		t.Error("Expected style2 CSS in registry")
	}
	
	// Verify no duplicates
	style3 := StyleWithRegistry(`.test1 { color: red; }`)
	if style3.Hash != style1.Hash {
		t.Error("Expected same hash for identical CSS")
	}
	
	// We can't directly access globalRegistry.styles as it's private
	// Instead, let's verify uniqueness by checking that adding the same
	// style doesn't change the output
	allCSSBefore := GetAllCSS()
	_ = StyleWithRegistry(`.test1 { color: red; }`) // Add duplicate
	allCSSAfter := GetAllCSS()
	if allCSSBefore != allCSSAfter {
		t.Error("Registry should not contain duplicate styles")
	}
}

func TestComponentStyle_Classes(t *testing.T) {
	style := Style(`
		.btn { padding: 1rem; }
		.primary { background: blue; }
		.secondary { background: gray; }
	`)
	
	// Test single class
	btnClass := style.Class("btn")
	if btnClass == "" {
		t.Error("Expected class name for 'btn'")
	}
	
	// Test multiple classes
	combined := style.Classes("btn", "primary")
	if !strings.Contains(combined, style.Class("btn")) {
		t.Error("Expected 'btn' class in combined classes")
	}
	if !strings.Contains(combined, style.Class("primary")) {
		t.Error("Expected 'primary' class in combined classes")
	}
	
	// Classes should be space-separated
	parts := strings.Fields(combined)
	if len(parts) != 2 {
		t.Errorf("Expected 2 classes, got %d", len(parts))
	}
}

func TestExtractClassNames(t *testing.T) {
	tests := []struct {
		css      string
		expected []string
	}{
		{
			css:      `.card { color: red; }`,
			expected: []string{"card"},
		},
		{
			css:      `.btn-primary { background: blue; }`,
			expected: []string{"btn-primary"},
		},
		{
			css: `.card { } .card.active { } .card:hover { }`,
			expected: []string{"card", "card.active", "card:hover"},
		},
		{
			css: `
				.container { }
				.container .item { }
				@media (min-width: 768px) {
					.container { }
				}
			`,
			expected: []string{"container", "item"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.css, func(t *testing.T) {
			names := extractClassNames(tt.css)
			
			for _, expected := range tt.expected {
				found := false
				for _, name := range names {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected class name %s not found", expected)
				}
			}
		})
	}
}