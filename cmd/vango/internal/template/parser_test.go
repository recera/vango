package template

import (
	"strings"
	"testing"
)

func TestTemplateParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple template",
			source: `//vango:template
package routes

<div>Hello World</div>`,
			wantErr: false,
		},
		{
			name: "template with props",
			source: `//vango:template
package routes

//vango:props { Name string; Age int }

<div>Hello {{.Name}}</div>`,
			wantErr: false,
		},
		{
			name: "template with conditionals",
			source: `//vango:template
package routes

//vango:props { ShowGreeting bool }

{{#if .ShowGreeting}}
	<h1>Welcome!</h1>
{{/if}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewTemplateParser("test.vex.go", tt.source)
			err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateParser_GenerateCode(t *testing.T) {
	source := `//vango:template
package routes

//vango:props { Title string }

<h1>{{.Title}}</h1>`

	parser := NewTemplateParser("test.vex.go", source)
	if err := parser.Parse(); err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	code, err := parser.GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode() failed: %v", err)
	}

	// Check that generated code contains expected elements
	if !strings.Contains(code, "package routes") {
		t.Error("Generated code missing package declaration")
	}
	if !strings.Contains(code, "type PageProps struct") {
		t.Error("Generated code missing PageProps struct")
	}
	if !strings.Contains(code, "Title string") {
		t.Error("Generated code missing Title field")
	}
	if !strings.Contains(code, "func Page(") {
		t.Error("Generated code missing Page function")
	}
}