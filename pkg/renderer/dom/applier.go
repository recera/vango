//go:build js && wasm
// +build js,wasm

package dom

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/recera/vango/pkg/vango/vdom"
)

// DOMApplier applies VNode patches to the browser DOM
type DOMApplier struct {
	document      js.Value
	window        js.Value
	nodeMap       map[uint32]js.Value           // Maps node IDs to DOM elements
	eventHandlers map[uint32]map[string]js.Func // Maps node IDs to event handlers
	nodeCounter   uint32                        // For assigning IDs during hydration
}

// NewDOMApplier creates a new DOM applier
func NewDOMApplier() *DOMApplier {
	return &DOMApplier{
		document:      js.Global().Get("document"),
		window:        js.Global().Get("window"),
		nodeMap:       make(map[uint32]js.Value),
		eventHandlers: make(map[uint32]map[string]js.Func),
		nodeCounter:   1,
	}
}

// Apply applies patches to transform the DOM
func (a *DOMApplier) Apply(patches []vdom.Patch) error {
	for _, patch := range patches {
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Applying patch: %v", patch))
		if err := a.applyPatch(patch); err != nil {
			return fmt.Errorf("failed to apply patch %v: %w", patch, err)
		}
	}
	return nil
}

// applyPatch applies a single patch to the DOM
func (a *DOMApplier) applyPatch(patch vdom.Patch) error {
	switch patch.Op {
	case vdom.OpReplaceText:
		return a.replaceText(patch)
	case vdom.OpSetAttribute:
		return a.setAttribute(patch)
	case vdom.OpRemoveAttribute:
		return a.removeAttribute(patch)
	case vdom.OpRemoveNode:
		return a.removeNode(patch)
	case vdom.OpInsertNode:
		return a.insertNode(patch)
	case vdom.OpUpdateEvents:
		return a.updateEvents(patch)
	case vdom.OpMoveNode:
		return a.moveNode(patch)
	default:
		return fmt.Errorf("unknown patch operation: %v", patch.Op)
	}
}

// replaceText replaces the text content of a text node
func (a *DOMApplier) replaceText(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] ERROR: Text node %d not found in nodeMap", patch.NodeID))
		// Log what's in the nodeMap
		for id := range a.nodeMap {
			js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] nodeMap contains ID: %d", id))
		}
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Replacing text of node %d to: %q", patch.NodeID, patch.Value))
	node.Set("textContent", patch.Value)
	return nil
}

// setAttribute sets an attribute on an element
func (a *DOMApplier) setAttribute(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	// Special handling for certain attributes
	switch patch.Key {
	case "class":
		node.Set("className", patch.Value)
	case "for":
		node.Set("htmlFor", patch.Value)
	case "checked", "selected", "disabled", "readonly", "required":
		// Boolean attributes
		node.Set(patch.Key, patch.Value == "true")
	case "value":
		// For input elements, set the value property directly
		if node.Get("tagName").String() == "INPUT" ||
			node.Get("tagName").String() == "TEXTAREA" ||
			node.Get("tagName").String() == "SELECT" {
			node.Set("value", patch.Value)
		} else {
			node.Call("setAttribute", patch.Key, patch.Value)
		}
	default:
		node.Call("setAttribute", patch.Key, patch.Value)
	}

	return nil
}

// removeAttribute removes an attribute from an element
func (a *DOMApplier) removeAttribute(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	// Special handling for certain attributes
	switch patch.Key {
	case "class":
		node.Set("className", "")
	case "checked", "selected", "disabled", "readonly", "required":
		node.Set(patch.Key, false)
	default:
		node.Call("removeAttribute", patch.Key)
	}

	return nil
}

// removeNode removes a node from the DOM
func (a *DOMApplier) removeNode(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	parent := node.Get("parentNode")
	if !parent.IsNull() && !parent.IsUndefined() {
		parent.Call("removeChild", node)
	}

	// Clean up node map
	delete(a.nodeMap, patch.NodeID)

	return nil
}

// insertNode inserts a new node into the DOM
func (a *DOMApplier) insertNode(patch vdom.Patch) error {
	if patch.Node == nil {
		return fmt.Errorf("insert patch missing node")
	}

	// Create the entire DOM tree with proper IDs and event handlers
	domNode, nextID := a.createDOMTree(patch.Node, patch.NodeID)

	js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Created DOM tree starting at ID %d, next ID: %d", patch.NodeID, nextID))

	// Find parent
	parent, ok := a.nodeMap[patch.ParentID]
	if !ok && patch.ParentID != 0 {
		return fmt.Errorf("parent node %d not found", patch.ParentID)
	}

	// If no parent specified, append to body
	if patch.ParentID == 0 {
		parent = a.document.Get("body")
	}

	// Insert the node
	if patch.BeforeID != 0 {
		before, ok := a.nodeMap[patch.BeforeID]
		if ok {
			parent.Call("insertBefore", domNode, before)
		} else {
			parent.Call("appendChild", domNode)
		}
	} else {
		parent.Call("appendChild", domNode)
	}

	return nil
}

// createDOMTree creates a DOM tree from a VNode tree, assigning IDs and attaching event handlers
func (a *DOMApplier) createDOMTree(vnode *vdom.VNode, startID uint32) (js.Value, uint32) {
	if vnode == nil {
		return js.Undefined(), startID
	}

	currentID := startID

	switch vnode.Kind {
	case vdom.KindText:
		textNode := a.document.Call("createTextNode", vnode.Text)
		a.nodeMap[currentID] = textNode
		return textNode, currentID + 1

	case vdom.KindElement:
		elem := a.document.Call("createElement", vnode.Tag)
		a.nodeMap[currentID] = elem

		// Set attributes
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				// Skip event handlers and special props
				if key == "key" || key == "ref" || (len(key) > 2 && key[0] == 'o' && key[1] == 'n') {
					continue
				}

				// Apply attribute directly
				elem.Call("setAttribute", key, fmt.Sprintf("%v", value))
			}

			// Attach event handlers
			// Attach event handlers and apply ref callbacks
			a.attachEventHandlers(currentID, elem, vnode.Props)
			if refVal, ok := vnode.Props["ref"]; ok {
				if refFn, ok := refVal.(func(js.Value)); ok {
					// Call ref with the element reference
					refFn(elem)
				} else if refFn2, ok := refVal.(func(vdom.ElementRef)); ok {
					refFn2(elem)
				}
			}
		}

		// Create and append children
		nextID := currentID + 1
		for _, child := range vnode.Kids {
			childDOM, newNextID := a.createDOMTree(&child, nextID)
			if !childDOM.IsUndefined() {
				elem.Call("appendChild", childDOM)
			}
			nextID = newNextID
		}

		return elem, nextID

	default:
		return js.Undefined(), currentID
	}
}

// createDOMNode creates a DOM node from a VNode
func (a *DOMApplier) createDOMNode(vnode *vdom.VNode) (js.Value, error) {
	dom, _ := a.createDOMTree(vnode, a.nodeCounter)
	a.nodeCounter++
	if dom.IsUndefined() {
		return dom, fmt.Errorf("failed to create DOM node")
	}
	return dom, nil
}

// createDOMNodeWithID creates a DOM node from a VNode with a specific ID
func (a *DOMApplier) createDOMNodeWithID(vnode *vdom.VNode, nodeID uint32) (js.Value, error) {
	switch vnode.Kind {
	case vdom.KindText:
		return a.document.Call("createTextNode", vnode.Text), nil

	case vdom.KindElement:
		elem := a.document.Call("createElement", vnode.Tag)

		// Set attributes
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				// Skip event handlers and special props
				if key == "key" || (len(key) > 2 && key[0] == 'o' && key[1] == 'n') {
					continue
				}

				// Apply attribute
				p := vdom.Patch{
					NodeID: 0, // Temporary, not used in setAttribute
					Key:    key,
					Value:  fmt.Sprintf("%v", value),
				}

				// Temporarily store the element in nodeMap
				tempID := uint32(0)
				a.nodeMap[tempID] = elem
				a.setAttribute(p)
				delete(a.nodeMap, tempID)
			}
		}

		// Attach event handlers and apply ref callbacks if we have a nodeID
		if nodeID != 0 {
			a.attachEventHandlers(nodeID, elem, vnode.Props)
			if vnode.Props != nil {
				if refVal, ok := vnode.Props["ref"]; ok {
					if refFn, ok := refVal.(func(js.Value)); ok {
						refFn(elem)
					} else if refFn2, ok := refVal.(func(vdom.ElementRef)); ok {
						refFn2(elem)
					}
				}
			}
		}

		// Create and append children
		for _, child := range vnode.Kids {
			childNode, err := a.createDOMNode(&child)
			if err != nil {
				return js.Undefined(), err
			}
			elem.Call("appendChild", childNode)
		}

		return elem, nil

	case vdom.KindFragment:
		// Fragments are handled by their parent
		return js.Undefined(), fmt.Errorf("fragments should not be created directly")

	case vdom.KindPortal:
		// Portals need special handling
		return js.Undefined(), fmt.Errorf("portal creation not yet implemented")

	default:
		return js.Undefined(), fmt.Errorf("unknown node kind: %v", vnode.Kind)
	}
}

// updateEvents updates event listeners on a node
func (a *DOMApplier) updateEvents(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	// This is a simplified version - in reality, we'd need to:
	// 1. Track which events are currently attached
	// 2. Remove old event listeners
	// 3. Add new event listeners based on EventBits

	// For now, we'll store the event bits as a data attribute
	node.Call("setAttribute", "data-events", fmt.Sprintf("%d", patch.EventBits))

	return nil
}

// attachEventHandlers attaches event handlers from VNode props to a DOM element
func (a *DOMApplier) attachEventHandlers(nodeID uint32, elem js.Value, props vdom.Props) {
	if props == nil {
		return
	}

	// Debug logging
	js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Attaching handlers for node %d", nodeID))

	// Clean up existing handlers for this node
	if handlers, exists := a.eventHandlers[nodeID]; exists {
		for eventName, fn := range handlers {
			elem.Call("removeEventListener", eventName, fn)
			fn.Release()
		}
	}

	// Create new handler map
	handlers := make(map[string]js.Func)

	// Attach new handlers
	for key, value := range props {
		if len(key) > 2 && key[0] == 'o' && key[1] == 'n' {
			// Convert onClick to click, onChange to change, etc.
			eventName := strings.ToLower(key[2:])

			js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Found event %s on node %d", eventName, nodeID))

			// Support multiple handler signatures
			var jsFunc js.Func
			switch h := value.(type) {
			case func():
				jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					h()
					return nil
				})
			case func(js.Value):
				jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					if len(args) > 0 {
						h(args[0])
					} else {
						h(js.Undefined())
					}
					return nil
				})
			case func(x, y float64):
				jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					var x, y float64
					if len(args) > 0 {
						ev := args[0]
						// Convert to element-relative coords using bounding box
						bx := 0.0
						by := 0.0
						if this.Truthy() {
							rect := this.Call("getBoundingClientRect")
							bx = rect.Get("left").Float()
							by = rect.Get("top").Float()
						}
						x = ev.Get("clientX").Float() - bx
						y = ev.Get("clientY").Float() - by
					}
					h(x, y)
					return nil
				})
			case func(deltaY float64):
				jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					var d float64
					if len(args) > 0 {
						d = args[0].Get("deltaY").Float()
					}
					h(d)
					return nil
				})
			case func(string):
				jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					var s string
					if len(args) > 0 {
						ev := args[0]
						switch eventName {
						case "input", "change":
							tgt := ev.Get("target")
							if tgt.Truthy() {
								s = tgt.Get("value").String()
							}
						case "keydown", "keyup", "keypress":
							s = ev.Get("key").String()
						default:
							s = ev.Get("type").String()
						}
					}
					h(s)
					return nil
				})
			default:
				// Fallback: ignore unsupported types
				continue
			}

			// Add event listener
			elem.Call("addEventListener", eventName, jsFunc)
			handlers[eventName] = jsFunc
		}
	}

	// Store handlers for cleanup
	if len(handlers) > 0 {
		a.eventHandlers[nodeID] = handlers
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Stored %d handlers for node %d", len(handlers), nodeID))
	}
}

// moveNode moves a node to a new position
func (a *DOMApplier) moveNode(patch vdom.Patch) error {
	node, ok := a.nodeMap[patch.NodeID]
	if !ok {
		return fmt.Errorf("node %d not found", patch.NodeID)
	}

	parent, ok := a.nodeMap[patch.ParentID]
	if !ok && patch.ParentID != 0 {
		return fmt.Errorf("parent node %d not found", patch.ParentID)
	}

	// If no parent specified, use body
	if patch.ParentID == 0 {
		parent = a.document.Get("body")
	}

	// Remove from current position
	currentParent := node.Get("parentNode")
	if !currentParent.IsNull() && !currentParent.IsUndefined() {
		currentParent.Call("removeChild", node)
	}

	// Insert at new position
	if patch.BeforeID != 0 {
		before, ok := a.nodeMap[patch.BeforeID]
		if ok {
			parent.Call("insertBefore", node, before)
		} else {
			parent.Call("appendChild", node)
		}
	} else {
		parent.Call("appendChild", node)
	}

	return nil
}

// HydrateFromDOM builds the node map from existing DOM elements with data-hid attributes
func (a *DOMApplier) HydrateFromDOM() error {
	// Find all elements with data-hid attribute
	elements := a.document.Call("querySelectorAll", "[data-hid]")
	length := elements.Get("length").Int()

	for i := 0; i < length; i++ {
		elem := elements.Index(i)
		hidStr := elem.Call("getAttribute", "data-hid").String()

		// Parse the hydration ID
		var hid uint32
		if _, err := fmt.Sscanf(hidStr, "h%d", &hid); err != nil {
			return fmt.Errorf("invalid hydration ID: %s", hidStr)
		}

		// Store in node map
		a.nodeMap[hid] = elem
	}

	return nil
}

// HydrateFullTree walks the VNode tree and DOM tree together to build complete nodeMap
func (a *DOMApplier) HydrateFullTree(vnode *vdom.VNode, domNode js.Value, nodeID uint32) uint32 {
	if vnode == nil {
		return nodeID
	}

	switch vnode.Kind {
	case vdom.KindElement:
		// Store this element in nodeMap
		a.nodeMap[nodeID] = domNode
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Hydrated element %s with ID %d", vnode.Tag, nodeID))

		// Process children
		currentID := nodeID + 1
		childNodes := domNode.Get("childNodes")
		childLength := childNodes.Get("length").Int()

		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Element %s has %d DOM children and %d VNode children", vnode.Tag, childLength, len(vnode.Kids)))

		// Match VNode children with DOM children
		vnodeIdx := 0
		for i := 0; i < childLength && vnodeIdx < len(vnode.Kids); i++ {
			childDOM := childNodes.Index(i)
			nodeType := childDOM.Get("nodeType").Int()

			// Skip non-element/text nodes (like comments)
			if nodeType != 1 && nodeType != 3 { // 1=ELEMENT_NODE, 3=TEXT_NODE
				js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Child %d: nodeType=%d", i, nodeType))
				continue
			}

			if vnodeIdx >= len(vnode.Kids) {
				js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] WARNING: More DOM children than VNode children at index %d", vnodeIdx))
				break
			}

			currentID = a.HydrateFullTree(&vnode.Kids[vnodeIdx], childDOM, currentID)
			vnodeIdx++
		}

		// Check if we processed all VNode children
		if vnodeIdx < len(vnode.Kids) {
			js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] WARNING: Only processed %d of %d VNode children for %s", vnodeIdx, len(vnode.Kids), vnode.Tag))
		}

		return currentID

	case vdom.KindText:
		// Store text node in nodeMap
		a.nodeMap[nodeID] = domNode
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Hydrated text node with ID %d, content: %q", nodeID, vnode.Text))
		return nodeID + 1

	default:
		return nodeID
	}
}

// AttachHandlersForVNode attaches event handlers to hydrated elements
func (a *DOMApplier) AttachHandlersForVNode(nodeID uint32, vnode *vdom.VNode) {
	js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] AttachHandlersForVNode called for node %d", nodeID))

	if elem, ok := a.nodeMap[nodeID]; ok && vnode.Props != nil {
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] Found element in nodeMap for node %d", nodeID))
		a.attachEventHandlers(nodeID, elem, vnode.Props)
	} else {
		js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] No element in nodeMap for node %d, nodeMap size: %d", nodeID, len(a.nodeMap)))
		// Log what's actually in the nodeMap
		for id := range a.nodeMap {
			js.Global().Get("console").Call("log", fmt.Sprintf("[DOM] nodeMap contains ID: %d", id))
		}
	}
}

// GetNodeMap returns the current node map (for debugging)
func (a *DOMApplier) GetNodeMap() map[uint32]js.Value {
	return a.nodeMap
}
