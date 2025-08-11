package template

// AST node types for VEX templates

// TemplateAST is the root node of a parsed template
type TemplateAST struct {
	Nodes []Node
}

// Node is the interface for all AST nodes
type Node interface {
	Generate() string // Generate Go code from this node
}

// TextNode represents plain text content
type TextNode struct {
	Content string
}

// ElementNode represents an HTML element
type ElementNode struct {
	Tag         string
	Attributes  map[string]string
	Events      map[string]string
	Children    []Node
	SelfClosing bool
}

// ComponentNode represents a custom component
type ComponentNode struct {
	Name     string
	Props    map[string]string
	Children []Node
}

// ExpressionNode represents a template expression {{expr}}
type ExpressionNode struct {
	Content string
	IsProp  bool // true if it's a prop reference like {{.Name}}
}

// IfNode represents an if statement
type IfNode struct {
	Condition string
	Then      []Node
	ElseIf    []*ElseIfNode
	Else      []Node
}

// ElseIfNode represents an else-if clause
type ElseIfNode struct {
	Condition string
	Then      []Node
}

// ForNode represents a for loop
type ForNode struct {
	Variable string
	Iterator string
	Body     []Node
}

// EventAttr represents an event attribute like @click
type EventAttr struct {
	Name    string
	Handler string
}

// Attr represents a regular HTML attribute
type Attr struct {
	Name  string
	Value string
}