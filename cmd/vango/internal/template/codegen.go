package template

import (
	"fmt"
	"strings"
)

// Generate generates Go code for the entire template AST
func (ast *TemplateAST) Generate() string {
	var code strings.Builder
	
	// Generate children array
	code.WriteString("\tvar children []vdom.VNode\n")
	
	// Generate code for all nodes
	for _, node := range ast.Nodes {
		nodeCode := node.Generate()
		if nodeCode != "" {
			code.WriteString(nodeCode)
		}
	}
	
	// Return the root element or fragment
	if len(ast.Nodes) == 1 {
		// Single root element
		return ast.Nodes[0].Generate()
	} else {
		// Multiple elements - wrap in fragment
		code.WriteString("\treturn vdom.Fragment(children...)\n")
		return code.String()
	}
}

// Generate for TextNode
func (n *TextNode) Generate() string {
	// Skip empty text nodes
	trimmed := strings.TrimSpace(n.Content)
	if trimmed == "" {
		return ""
	}
	
	// Escape the content
	escaped := strings.ReplaceAll(n.Content, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\t", `\t`)
	
	return fmt.Sprintf("functional.Text(%q)", escaped)
}

// Generate for ElementNode
func (n *ElementNode) Generate() string {
	var code strings.Builder
	
	// Use builder pattern for elements
	tagMethod := capitalizeFirst(n.Tag)
	code.WriteString(fmt.Sprintf("builder.%s()", tagMethod))
	
	// Add attributes
	if n.Attributes != nil {
		for name, value := range n.Attributes {
			switch name {
			case "class":
				code.WriteString(fmt.Sprintf(".Class(%q)", value))
			case "id":
				code.WriteString(fmt.Sprintf(".ID(%q)", value))
			case "href":
				code.WriteString(fmt.Sprintf(".Href(%q)", value))
			case "src":
				code.WriteString(fmt.Sprintf(".Src(%q)", value))
			case "alt":
				code.WriteString(fmt.Sprintf(".Alt(%q)", value))
			case "type":
				code.WriteString(fmt.Sprintf(".Type(%q)", value))
			case "value":
				code.WriteString(fmt.Sprintf(".Value(%q)", value))
			case "placeholder":
				code.WriteString(fmt.Sprintf(".Placeholder(%q)", value))
			default:
				// Generic attribute
				code.WriteString(fmt.Sprintf(".Attr(%q, %q)", name, value))
			}
		}
	}
	
	// Add event handlers
	if n.Events != nil {
		for event, handler := range n.Events {
			switch event {
			case "click":
				code.WriteString(fmt.Sprintf(".OnClick(func() { %s })", handler))
			case "input":
				code.WriteString(fmt.Sprintf(".OnInput(func(e vdom.Event) { %s })", handler))
			case "submit":
				code.WriteString(fmt.Sprintf(".OnSubmit(func(e vdom.Event) { %s })", handler))
			case "change":
				code.WriteString(fmt.Sprintf(".OnChange(func(e vdom.Event) { %s })", handler))
			default:
				code.WriteString(fmt.Sprintf(".On(%q, func(e vdom.Event) { %s })", event, handler))
			}
		}
	}
	
	// Add children
	if len(n.Children) > 0 {
		code.WriteString(".Children(\n")
		for i, child := range n.Children {
			childCode := child.Generate()
			if childCode != "" {
				// Remove leading tabs from child code if present
				childCode = strings.TrimPrefix(childCode, "\t")
				code.WriteString("\t\t" + childCode)
				
				// Add comma between children
				if i < len(n.Children)-1 {
					code.WriteString(",")
				}
				code.WriteString("\n")
			}
		}
		code.WriteString("\t)")
	}
	
	code.WriteString(".Build()")
	
	return code.String()
}

// Generate for ComponentNode
func (n *ComponentNode) Generate() string {
	var code strings.Builder
	
	// Component invocation
	code.WriteString(fmt.Sprintf("\t%s(", n.Name))
	
	// Add props
	if len(n.Props) > 0 {
		code.WriteString(fmt.Sprintf("%sProps{\n", n.Name))
		for name, value := range n.Props {
			propName := capitalizeFirst(name)
			// Check if value is an expression or literal
			if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
				// Expression - remove braces
				expr := value[1 : len(value)-1]
				code.WriteString(fmt.Sprintf("\t\t%s: %s,\n", propName, expr))
			} else {
				// Literal string
				code.WriteString(fmt.Sprintf("\t\t%s: %q,\n", propName, value))
			}
		}
		code.WriteString("\t}")
	}
	
	// Add children if any
	if len(n.Children) > 0 {
		if len(n.Props) > 0 {
			code.WriteString(", ")
		}
		code.WriteString("[]vdom.VNode{\n")
		for _, child := range n.Children {
			childCode := child.Generate()
			if childCode != "" {
				code.WriteString("\t\t" + childCode + ",\n")
			}
		}
		code.WriteString("\t}")
	}
	
	code.WriteString(")")
	
	return code.String()
}

// Generate for ExpressionNode
func (n *ExpressionNode) Generate() string {
	content := strings.TrimSpace(n.Content)
	
	// Handle prop references
	if n.IsProp && strings.HasPrefix(content, ".") {
		// Remove the dot and reference props
		propName := content[1:]
		return fmt.Sprintf("functional.Text(fmt.Sprint(props.%s))", propName)
	}
	
	// Handle regular expressions
	return fmt.Sprintf("functional.Text(fmt.Sprint(%s))", content)
}

// Generate for IfNode
func (n *IfNode) Generate() string {
	var code strings.Builder
	
	// Generate if statement
	code.WriteString(fmt.Sprintf("\tif %s {\n", n.Condition))
	
	// Generate then branch
	for _, node := range n.Then {
		nodeCode := node.Generate()
		if nodeCode != "" {
			// Indent the code
			lines := strings.Split(nodeCode, "\n")
			for _, line := range lines {
				if line != "" {
					code.WriteString("\t" + line + "\n")
				}
			}
		}
	}
	
	// Generate else-if branches
	for _, elseIf := range n.ElseIf {
		code.WriteString(fmt.Sprintf("\t} else if %s {\n", elseIf.Condition))
		for _, node := range elseIf.Then {
			nodeCode := node.Generate()
			if nodeCode != "" {
				lines := strings.Split(nodeCode, "\n")
				for _, line := range lines {
					if line != "" {
						code.WriteString("\t" + line + "\n")
					}
				}
			}
		}
	}
	
	// Generate else branch
	if len(n.Else) > 0 {
		code.WriteString("\t} else {\n")
		for _, node := range n.Else {
			nodeCode := node.Generate()
			if nodeCode != "" {
				lines := strings.Split(nodeCode, "\n")
				for _, line := range lines {
					if line != "" {
						code.WriteString("\t" + line + "\n")
					}
				}
			}
		}
	}
	
	code.WriteString("\t}\n")
	
	return code.String()
}

// Generate for ElseIfNode (not used directly, handled in IfNode)
func (n *ElseIfNode) Generate() string {
	return ""
}

// Generate for ForNode
func (n *ForNode) Generate() string {
	var code strings.Builder
	
	// Generate for loop
	code.WriteString(fmt.Sprintf("\tfor _, %s := range %s {\n", n.Variable, n.Iterator))
	
	// Generate body
	for _, node := range n.Body {
		nodeCode := node.Generate()
		if nodeCode != "" {
			// Indent the code
			lines := strings.Split(nodeCode, "\n")
			for _, line := range lines {
				if line != "" {
					code.WriteString("\t" + line + "\n")
				}
			}
		}
	}
	
	code.WriteString("\t}\n")
	
	return code.String()
}

// Helper function to capitalize first letter
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}