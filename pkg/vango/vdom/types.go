package vdom

// VKind represents the type of virtual node
type VKind uint8

const (
	// KindElement represents a DOM element node
	KindElement VKind = iota
	// KindText represents a text node
	KindText
	// KindFragment represents a fragment (multiple children without parent)
	KindFragment
	// KindPortal represents a portal (render children elsewhere in DOM)
	KindPortal
)

// VNodeFlags are bitwise flags for VNode optimizations
type VNodeFlags uint8

const (
	// FlagStatic indicates this node and its children will never change
	FlagStatic VNodeFlags = 1 << iota
	// FlagHasKey indicates this node has a key for list reconciliation
	FlagHasKey
	// FlagHasRef indicates this node has a ref callback
	FlagHasRef
	// FlagHasEvents indicates this node has event listeners
	FlagHasEvents
	// FlagDirty indicates this node needs re-rendering
	FlagDirty
)

// Props represents the properties/attributes of a VNode
// Using map[string]any for stability as specified in the blueprint
type Props map[string]any

// VNode represents a virtual DOM node
// This struct is immutable - once created, it should never be modified
type VNode struct {
	// Kind determines the type of this node
	Kind VKind

	// Tag is the element tag name (e.g., "div", "span")
	// Only used when Kind == KindElement
	Tag string

	// Props contains all properties/attributes for this node
	// This includes event handlers, style, class, etc.
	Props Props

	// Kids contains child nodes
	// For KindText, this is nil
	Kids []VNode

	// Key is used for efficient list reconciliation
	// Empty string means no key
	Key string

	// Flags contains optimization hints
	Flags VNodeFlags

	// Text content (only used when Kind == KindText)
	Text string

	// Portal target (only used when Kind == KindPortal)
	PortalTarget string
}

// NewElement creates a new element VNode
func NewElement(tag string, props Props, children ...*VNode) *VNode {
	flags := VNodeFlags(0)
	
	// Check if props contain event handlers
	if props != nil {
		for k := range props {
			if len(k) > 2 && k[0] == 'o' && k[1] == 'n' {
				flags |= FlagHasEvents
				break
			}
		}
		
		// Check for key
		if _, hasKey := props["key"]; hasKey {
			flags |= FlagHasKey
		}
		
		// Check for ref
		if _, hasRef := props["ref"]; hasRef {
			flags |= FlagHasRef
		}
	}
	
	// Convert children pointers to values
	kids := make([]VNode, 0, len(children))
	for _, child := range children {
		if child != nil {
			kids = append(kids, *child)
		}
	}
	
	return &VNode{
		Kind:  KindElement,
		Tag:   tag,
		Props: props,
		Kids:  kids,
		Flags: flags,
	}
}

// NewText creates a new text VNode
func NewText(text string) *VNode {
	return &VNode{
		Kind: KindText,
		Text: text,
	}
}

// NewFragment creates a new fragment VNode
func NewFragment(children ...*VNode) *VNode {
	// Convert children pointers to values
	kids := make([]VNode, 0, len(children))
	for _, child := range children {
		if child != nil {
			kids = append(kids, *child)
		}
	}
	
	return &VNode{
		Kind: KindFragment,
		Kids: kids,
	}
}

// NewPortal creates a new portal VNode
func NewPortal(target string, children ...*VNode) *VNode {
	// Convert children pointers to values
	kids := make([]VNode, 0, len(children))
	for _, child := range children {
		if child != nil {
			kids = append(kids, *child)
		}
	}
	
	return &VNode{
		Kind:         KindPortal,
		PortalTarget: target,
		Kids:         kids,
	}
}

// IsElement returns true if this is an element node
func (v VNode) IsElement() bool {
	return v.Kind == KindElement
}

// IsText returns true if this is a text node
func (v VNode) IsText() bool {
	return v.Kind == KindText
}

// IsFragment returns true if this is a fragment node
func (v VNode) IsFragment() bool {
	return v.Kind == KindFragment
}

// IsPortal returns true if this is a portal node
func (v VNode) IsPortal() bool {
	return v.Kind == KindPortal
}

// HasFlag returns true if the specified flag is set
func (v VNode) HasFlag(flag VNodeFlags) bool {
	return v.Flags&flag != 0
}

// GetKey returns the key of this node, handling the Props map safely
func (v VNode) GetKey() string {
	if v.Props != nil {
		if key, ok := v.Props["key"].(string); ok {
			return key
		}
	}
	return v.Key
}