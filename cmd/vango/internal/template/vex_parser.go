package template

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// VexParser is a recursive descent parser for VEX templates
type VexParser struct {
	input    string
	pos      int
	line     int
	col      int
	filename string
}

// NewVexParser creates a new VEX template parser
func NewVexParser(filename, input string) *VexParser {
	return &VexParser{
		input:    input,
		pos:      0,
		line:     1,
		col:      1,
		filename: filename,
	}
}

// Parse parses the entire template
func (p *VexParser) Parse() (*TemplateAST, error) {
	nodes, err := p.parseTemplateNodes()
	if err != nil {
		return nil, err
	}
	
	return &TemplateAST{
		Nodes: nodes,
	}, nil
}

// parseTemplateNodes parses a sequence of template nodes
func (p *VexParser) parseTemplateNodes() ([]Node, error) {
	var nodes []Node
	
	for p.pos < len(p.input) {
		// Check for control structures
		if p.peek("{{#if") {
			node, err := p.parseIfStatement()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if p.peek("{{#for") {
			node, err := p.parseForStatement()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if p.peek("{{/") {
			// End of a block
			break
		} else if p.peek("{{#else") {
			// End of current block in if statement
			break
		} else if p.peek("{{") {
			// Expression
			node, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if p.peek("</") {
			// This is a closing tag for the parent element
			break
		} else if p.peek("<") {
			// HTML element or component
			node, err := p.parseElement()
			if err != nil {
				return nil, err
			}
			if node != nil {
				nodes = append(nodes, node)
			}
		} else {
			// Plain text
			node := p.parseText()
			if node != nil && node.Content != "" {
				nodes = append(nodes, node)
			}
		}
	}
	
	return nodes, nil
}

// parseIfStatement parses an if statement
func (p *VexParser) parseIfStatement() (*IfNode, error) {
	// Consume {{#if
	if !p.consume("{{#if") {
		return nil, p.error("expected {{#if")
	}
	
	p.skipWhitespace()
	
	// Parse condition
	condition := p.parseUntil("}}")
	if condition == "" {
		return nil, p.error("expected condition in if statement")
	}
	// Fix prop references in condition
	condition = p.fixPropReferences(condition)
	
	if !p.consume("}}") {
		return nil, p.error("expected }}")
	}
	
	// Parse then branch
	thenNodes, err := p.parseTemplateNodes()
	if err != nil {
		return nil, err
	}
	
	// Parse else-if clauses
	var elseIfClauses []*ElseIfNode
	for p.peek("{{#elseif") {
		if !p.consume("{{#elseif") {
			break
		}
		
		p.skipWhitespace()
		elseIfCondition := p.parseUntil("}}")
		elseIfCondition = p.fixPropReferences(elseIfCondition)
		
		if !p.consume("}}") {
			return nil, p.error("expected }}")
		}
		
		elseIfNodes, err := p.parseTemplateNodes()
		if err != nil {
			return nil, err
		}
		
		elseIfClauses = append(elseIfClauses, &ElseIfNode{
			Condition: elseIfCondition,
			Then:      elseIfNodes,
		})
	}
	
	// Parse else clause
	var elseNodes []Node
	if p.peek("{{#else}}") {
		if !p.consume("{{#else}}") {
			return nil, p.error("expected {{#else}}")
		}
		
		elseNodes, err = p.parseTemplateNodes()
		if err != nil {
			return nil, err
		}
	}
	
	// Consume {{/if}}
	if !p.consume("{{/if}}") {
		return nil, p.error("expected {{/if}}")
	}
	
	return &IfNode{
		Condition: strings.TrimSpace(condition),
		Then:      thenNodes,
		ElseIf:    elseIfClauses,
		Else:      elseNodes,
	}, nil
}

// parseForStatement parses a for loop
func (p *VexParser) parseForStatement() (*ForNode, error) {
	// Consume {{#for
	if !p.consume("{{#for") {
		return nil, p.error("expected {{#for")
	}
	
	p.skipWhitespace()
	
	// Parse variable name
	variable := p.parseIdentifier()
	if variable == "" {
		return nil, p.error("expected variable name in for statement")
	}
	
	p.skipWhitespace()
	
	// Consume "in"
	if !p.consume("in") {
		return nil, p.error("expected 'in' in for statement")
	}
	
	p.skipWhitespace()
	
	// Parse iterator expression
	iterator := p.parseUntil("}}")
	if iterator == "" {
		return nil, p.error("expected iterator expression in for statement")
	}
	// Fix prop references in iterator
	iterator = p.fixPropReferences(iterator)
	
	if !p.consume("}}") {
		return nil, p.error("expected }}")
	}
	
	// Parse body
	body, err := p.parseTemplateNodes()
	if err != nil {
		return nil, err
	}
	
	// Consume {{/for}}
	if !p.consume("{{/for}}") {
		return nil, p.error("expected {{/for}}")
	}
	
	return &ForNode{
		Variable: variable,
		Iterator: strings.TrimSpace(iterator),
		Body:     body,
	}, nil
}

// parseExpression parses a template expression {{expr}}
func (p *VexParser) parseExpression() (*ExpressionNode, error) {
	if !p.consume("{{") {
		return nil, p.error("expected {{")
	}
	
	content := p.parseUntil("}}")
	
	if !p.consume("}}") {
		return nil, p.error("expected }}")
	}
	
	// Check if it's a prop reference
	content = strings.TrimSpace(content)
	isProp := strings.HasPrefix(content, ".")
	// Fix prop references to use "props." prefix
	if isProp {
		content = "props" + content
	}
	
	return &ExpressionNode{
		Content: content,
		IsProp:  isProp,
	}, nil
}

// parseElement parses an HTML element or component
func (p *VexParser) parseElement() (Node, error) {
	if !p.consume("<") {
		return nil, p.error("expected <")
	}
	
	// Parse tag name
	tagName := p.parseTagName()
	if tagName == "" {
		return nil, p.error("expected tag name")
	}
	
	// Check if it's a component (starts with uppercase)
	isComponent := unicode.IsUpper(rune(tagName[0]))
	
	// Parse attributes
	attributes, events := p.parseAttributes()
	
	// Check for self-closing
	p.skipWhitespace()
	if p.consume("/>") {
		if isComponent {
			return &ComponentNode{
				Name:  tagName,
				Props: attributes,
			}, nil
		}
		return &ElementNode{
			Tag:         tagName,
			Attributes:  attributes,
			Events:      events,
			SelfClosing: true,
		}, nil
	}
	
	// Consume >
	if !p.consume(">") {
		return nil, p.error("expected >")
	}
	
	// Parse children
	children, err := p.parseTemplateNodes()
	if err != nil {
		return nil, err
	}
	
	// Parse closing tag
	if !p.consume("</") {
		return nil, p.error("expected closing tag")
	}
	
	closingTag := p.parseTagName()
	if closingTag != tagName {
		return nil, p.error(fmt.Sprintf("mismatched tags: <%s> and </%s>", tagName, closingTag))
	}
	
	if !p.consume(">") {
		return nil, p.error("expected >")
	}
	
	if isComponent {
		return &ComponentNode{
			Name:     tagName,
			Props:    attributes,
			Children: children,
		}, nil
	}
	
	return &ElementNode{
		Tag:        tagName,
		Attributes: attributes,
		Events:     events,
		Children:   children,
	}, nil
}

// parseAttributes parses HTML attributes and event handlers
func (p *VexParser) parseAttributes() (map[string]string, map[string]string) {
	attributes := make(map[string]string)
	events := make(map[string]string)
	
	for {
		p.skipWhitespace()
		
		// Check if we've reached the end of attributes
		if p.peek(">") || p.peek("/>") {
			break
		}
		
		// Check for event handler
		if p.peek("@") {
			p.consume("@")
			eventName := p.parseAttributeName()
			
			p.skipWhitespace()
			if !p.consume("=") {
				break
			}
			p.skipWhitespace()
			
			// Parse quoted value
			if !p.consume("\"") {
				break
			}
			value := p.parseUntil("\"")
			p.consume("\"")
			
			events[eventName] = value
		} else {
			// Regular attribute
			attrName := p.parseAttributeName()
			if attrName == "" {
				break
			}
			
			p.skipWhitespace()
			if !p.consume("=") {
				// Boolean attribute
				attributes[attrName] = "true"
				continue
			}
			p.skipWhitespace()
			
			// Parse quoted value
			if !p.consume("\"") {
				break
			}
			value := p.parseUntil("\"")
			p.consume("\"")
			
			attributes[attrName] = value
		}
	}
	
	return attributes, events
}

// parseText parses plain text until the next template construct
func (p *VexParser) parseText() *TextNode {
	start := p.pos
	
	for p.pos < len(p.input) {
		if p.peek("{{") || p.peek("<") {
			break
		}
		p.advance()
	}
	
	if p.pos > start {
		return &TextNode{
			Content: p.input[start:p.pos],
		}
	}
	
	return nil
}

// Helper methods

func (p *VexParser) peek(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}
	return p.input[p.pos:p.pos+len(s)] == s
}

func (p *VexParser) consume(s string) bool {
	if p.peek(s) {
		for i := 0; i < len(s); i++ {
			p.advance()
		}
		return true
	}
	return false
}

func (p *VexParser) advance() {
	if p.pos < len(p.input) {
		if p.input[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}
}

func (p *VexParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.advance()
	}
}

func (p *VexParser) parseUntil(delimiter string) string {
	start := p.pos
	
	for p.pos < len(p.input) {
		if p.peek(delimiter) {
			return p.input[start:p.pos]
		}
		p.advance()
	}
	
	return p.input[start:p.pos]
}

func (p *VexParser) parseIdentifier() string {
	start := p.pos
	
	// First character must be letter or underscore
	if p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && ch != '_' {
			return ""
		}
		p.advance()
	}
	
	// Subsequent characters can be letters, digits, or underscore
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		p.advance()
	}
	
	return p.input[start:p.pos]
}

func (p *VexParser) parseTagName() string {
	start := p.pos
	
	// Parse tag name (letters, digits, hyphens)
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' {
			break
		}
		p.advance()
	}
	
	return p.input[start:p.pos]
}

func (p *VexParser) parseAttributeName() string {
	start := p.pos
	
	// Parse attribute name (letters, digits, hyphens, colons)
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' && ch != ':' {
			break
		}
		p.advance()
	}
	
	return p.input[start:p.pos]
}

func (p *VexParser) error(msg string) error {
	return fmt.Errorf("%s:%d:%d: %s", p.filename, p.line, p.col, msg)
}

// fixPropReferences replaces .PropName with props.PropName in expressions
func (p *VexParser) fixPropReferences(expr string) string {
	// Use regex to replace .Word with props.Word
	// This handles simple cases like .Items, .ShowCompleted
	re := regexp.MustCompile(`\.(\w+)`)
	return re.ReplaceAllString(expr, "props.$1")
}