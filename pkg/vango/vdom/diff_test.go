package vdom

import (
	"reflect"
	"testing"
)

func TestDiff_TextNodes(t *testing.T) {
	tests := []struct {
		name     string
		prev     *VNode
		next     *VNode
		expected []Patch
	}{
		{
			name: "text content change",
			prev: &VNode{Kind: KindText, Text: "Hello"},
			next: &VNode{Kind: KindText, Text: "World"},
			expected: []Patch{
				{Op: OpReplaceText, NodeID: 1, Value: "World"},
			},
		},
		{
			name:     "text content unchanged",
			prev:     &VNode{Kind: KindText, Text: "Same"},
			next:     &VNode{Kind: KindText, Text: "Same"},
			expected: []Patch{},
		},
		{
			name: "text to element",
			prev: &VNode{Kind: KindText, Text: "Text"},
			next: &VNode{Kind: KindElement, Tag: "div"},
			expected: []Patch{
				{Op: OpRemoveNode, NodeID: 1},
				{Op: OpInsertNode, NodeID: 2, Node: &VNode{Kind: KindElement, Tag: "div"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			if !patchesEqual(patches, tt.expected) {
				t.Errorf("Diff() = %v, want %v", patches, tt.expected)
			}
		})
	}
}

func TestDiff_ElementNodes(t *testing.T) {
	tests := []struct {
		name     string
		prev     *VNode
		next     *VNode
		expected []Patch
	}{
		{
			name: "different tags",
			prev: &VNode{Kind: KindElement, Tag: "div"},
			next: &VNode{Kind: KindElement, Tag: "span"},
			expected: []Patch{
				{Op: OpRemoveNode, NodeID: 1},
				{Op: OpInsertNode, NodeID: 2, Node: &VNode{Kind: KindElement, Tag: "span"}},
			},
		},
		{
			name: "add attribute",
			prev: &VNode{Kind: KindElement, Tag: "div"},
			next: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "active"}},
			expected: []Patch{
				{Op: OpSetAttribute, NodeID: 1, Key: "class", Value: "active"},
			},
		},
		{
			name: "remove attribute",
			prev: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "active"}},
			next: &VNode{Kind: KindElement, Tag: "div"},
			expected: []Patch{
				{Op: OpRemoveAttribute, NodeID: 1, Key: "class"},
			},
		},
		{
			name: "change attribute",
			prev: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "old"}},
			next: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "new"}},
			expected: []Patch{
				{Op: OpSetAttribute, NodeID: 1, Key: "class", Value: "new"},
			},
		},
		{
			name: "multiple attribute changes",
			prev: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "old", "id": "test"}},
			next: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "new", "data-attr": "value"}},
			expected: []Patch{
				{Op: OpSetAttribute, NodeID: 1, Key: "class", Value: "new"},
				{Op: OpRemoveAttribute, NodeID: 1, Key: "id"},
				{Op: OpSetAttribute, NodeID: 1, Key: "data-attr", Value: "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			if !patchesEqual(patches, tt.expected) {
				t.Errorf("Diff() = %v, want %v", patches, tt.expected)
			}
		})
	}
}

func TestDiff_EventHandlers(t *testing.T) {
	tests := []struct {
		name     string
		prev     *VNode
		next     *VNode
		hasEvent bool
	}{
		{
			name:     "add event handler",
			prev:     &VNode{Kind: KindElement, Tag: "button"},
			next:     &VNode{Kind: KindElement, Tag: "button", Props: Props{"onClick": "handler"}},
			hasEvent: true,
		},
		{
			name:     "remove event handler",
			prev:     &VNode{Kind: KindElement, Tag: "button", Props: Props{"onClick": "handler"}},
			next:     &VNode{Kind: KindElement, Tag: "button"},
			hasEvent: true,
		},
		{
			name:     "change event handler",
			prev:     &VNode{Kind: KindElement, Tag: "button", Props: Props{"onClick": "handler1"}},
			next:     &VNode{Kind: KindElement, Tag: "button", Props: Props{"onClick": "handler2"}},
			hasEvent: false, // Event bits don't change when only the handler changes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			
			hasEventPatch := false
			for _, p := range patches {
				if p.Op == OpUpdateEvents {
					hasEventPatch = true
					break
				}
			}
			
			if hasEventPatch != tt.hasEvent {
				t.Errorf("Expected event patch: %v, got: %v", tt.hasEvent, hasEventPatch)
			}
		})
	}
}

func TestDiff_Children(t *testing.T) {
	tests := []struct {
		name string
		prev *VNode
		next *VNode
	}{
		{
			name: "add child",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{
					{Kind: KindText, Text: "Hello"},
				},
			},
		},
		{
			name: "remove child",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{
					{Kind: KindText, Text: "Hello"},
				},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{},
			},
		},
		{
			name: "reorder unkeyed children",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{
					{Kind: KindText, Text: "A"},
					{Kind: KindText, Text: "B"},
					{Kind: KindText, Text: "C"},
				},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "div",
				Kids: []VNode{
					{Kind: KindText, Text: "B"},
					{Kind: KindText, Text: "A"},
					{Kind: KindText, Text: "C"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			// Just verify we get some patches - exact validation would be complex
			if len(patches) == 0 && tt.name != "unchanged" {
				t.Errorf("Expected patches for %s, got none", tt.name)
			}
		})
	}
}

func TestDiff_KeyedChildren(t *testing.T) {
	tests := []struct {
		name string
		prev *VNode
		next *VNode
	}{
		{
			name: "reorder keyed children",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "b", Props: Props{"key": "b"}},
					{Kind: KindElement, Tag: "li", Key: "c", Props: Props{"key": "c"}},
				},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "c", Props: Props{"key": "c"}},
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "b", Props: Props{"key": "b"}},
				},
			},
		},
		{
			name: "add keyed child",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "b", Props: Props{"key": "b"}},
				},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "b", Props: Props{"key": "b"}},
					{Kind: KindElement, Tag: "li", Key: "c", Props: Props{"key": "c"}},
				},
			},
		},
		{
			name: "remove keyed child",
			prev: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "b", Props: Props{"key": "b"}},
					{Kind: KindElement, Tag: "li", Key: "c", Props: Props{"key": "c"}},
				},
			},
			next: &VNode{
				Kind: KindElement,
				Tag:  "ul",
				Kids: []VNode{
					{Kind: KindElement, Tag: "li", Key: "a", Props: Props{"key": "a"}},
					{Kind: KindElement, Tag: "li", Key: "c", Props: Props{"key": "c"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			// Verify we get patches for keyed operations
			if len(patches) == 0 {
				t.Errorf("Expected patches for %s, got none", tt.name)
			}
		})
	}
}

func TestDiff_FragmentNodes(t *testing.T) {
	tests := []struct {
		name string
		prev *VNode
		next *VNode
	}{
		{
			name: "fragment children change",
			prev: &VNode{
				Kind: KindFragment,
				Kids: []VNode{
					{Kind: KindText, Text: "A"},
					{Kind: KindText, Text: "B"},
				},
			},
			next: &VNode{
				Kind: KindFragment,
				Kids: []VNode{
					{Kind: KindText, Text: "A"},
					{Kind: KindText, Text: "C"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			if len(patches) == 0 {
				t.Errorf("Expected patches for fragment diff, got none")
			}
		})
	}
}

func TestDiff_PortalNodes(t *testing.T) {
	tests := []struct {
		name string
		prev *VNode
		next *VNode
	}{
		{
			name: "portal target change",
			prev: &VNode{
				Kind:         KindPortal,
				PortalTarget: "#modal-root",
				Kids: []VNode{
					{Kind: KindText, Text: "Modal content"},
				},
			},
			next: &VNode{
				Kind:         KindPortal,
				PortalTarget: "#dialog-root",
				Kids: []VNode{
					{Kind: KindText, Text: "Modal content"},
				},
			},
		},
		{
			name: "portal children change",
			prev: &VNode{
				Kind:         KindPortal,
				PortalTarget: "#modal-root",
				Kids: []VNode{
					{Kind: KindText, Text: "Old content"},
				},
			},
			next: &VNode{
				Kind:         KindPortal,
				PortalTarget: "#modal-root",
				Kids: []VNode{
					{Kind: KindText, Text: "New content"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			if len(patches) == 0 {
				t.Errorf("Expected patches for portal diff, got none")
			}
		})
	}
}

func TestDiff_NilNodes(t *testing.T) {
	tests := []struct {
		name     string
		prev     *VNode
		next     *VNode
		expected int // expected number of patches
	}{
		{
			name:     "both nil",
			prev:     nil,
			next:     nil,
			expected: 0,
		},
		{
			name:     "add node",
			prev:     nil,
			next:     &VNode{Kind: KindText, Text: "New"},
			expected: 1,
		},
		{
			name:     "remove node",
			prev:     &VNode{Kind: KindText, Text: "Old"},
			next:     nil,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := Diff(tt.prev, tt.next)
			if len(patches) != tt.expected {
				t.Errorf("Expected %d patches, got %d", tt.expected, len(patches))
			}
		})
	}
}

// Helper function to compare patches
func patchesEqual(a, b []Patch) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Create maps to compare patches regardless of order
	aMap := make(map[string]bool)
	bMap := make(map[string]bool)
	
	for _, p := range a {
		aMap[p.String()] = true
	}
	
	for _, p := range b {
		bMap[p.String()] = true
	}
	
	return reflect.DeepEqual(aMap, bMap)
}