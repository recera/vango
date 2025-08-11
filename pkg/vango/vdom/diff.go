package vdom

import (
	"fmt"
)

// PatchOp represents the type of patch operation
type PatchOp uint8

const (
	// OpReplaceText replaces text node content
	OpReplaceText PatchOp = 0x01
	// OpSetAttribute sets or replaces an attribute
	OpSetAttribute PatchOp = 0x02
	// OpRemoveNode removes a node
	OpRemoveNode PatchOp = 0x03
	// OpInsertNode inserts a new node
	OpInsertNode PatchOp = 0x04
	// OpUpdateEvents updates event subscriptions
	OpUpdateEvents PatchOp = 0x05
	// OpRemoveAttribute removes an attribute
	OpRemoveAttribute PatchOp = 0x06
	// OpMoveNode moves a node to a new position
	OpMoveNode PatchOp = 0x07
)

// Patch represents a single DOM mutation
type Patch struct {
	Op        PatchOp
	NodeID    uint32
	ParentID  uint32 // For insert operations
	BeforeID  uint32 // For insert operations (0 means append)
	Key       string // Attribute key for set/remove attribute
	Value     string // Text content or attribute value
	Node      *VNode // For insert operations
	EventBits uint32 // For event updates
}

// String returns a human-readable representation of the patch
func (p Patch) String() string {
	switch p.Op {
	case OpReplaceText:
		return fmt.Sprintf("ReplaceText(node=%d, text=%q)", p.NodeID, p.Value)
	case OpSetAttribute:
		return fmt.Sprintf("SetAttribute(node=%d, key=%q, value=%q)", p.NodeID, p.Key, p.Value)
	case OpRemoveAttribute:
		return fmt.Sprintf("RemoveAttribute(node=%d, key=%q)", p.NodeID, p.Key)
	case OpRemoveNode:
		return fmt.Sprintf("RemoveNode(node=%d)", p.NodeID)
	case OpInsertNode:
		return fmt.Sprintf("InsertNode(parent=%d, before=%d)", p.ParentID, p.BeforeID)
	case OpUpdateEvents:
		return fmt.Sprintf("UpdateEvents(node=%d, bits=%x)", p.NodeID, p.EventBits)
	case OpMoveNode:
		return fmt.Sprintf("MoveNode(node=%d, parent=%d, before=%d)", p.NodeID, p.ParentID, p.BeforeID)
	default:
		return fmt.Sprintf("Unknown(op=%d)", p.Op)
	}
}

// DiffContext holds state during diffing
type DiffContext struct {
	patches     []Patch
	nodeCounter uint32
	nodeMap     map[*VNode]uint32
}

// newDiffContext creates a new diff context
func newDiffContext() *DiffContext {
	return &DiffContext{
		patches:     make([]Patch, 0, 16),
		nodeCounter: 1,
		nodeMap:     make(map[*VNode]uint32),
	}
}

// getNodeID gets or assigns a node ID
func (ctx *DiffContext) getNodeID(node *VNode) uint32 {
	if node == nil {
		return 0
	}
	if id, ok := ctx.nodeMap[node]; ok {
		return id
	}
	id := ctx.nodeCounter
	ctx.nodeCounter++
	ctx.nodeMap[node] = id
	return id
}

// addPatch adds a patch to the context
func (ctx *DiffContext) addPatch(patch Patch) {
	ctx.patches = append(ctx.patches, patch)
}

// Diff computes the patches needed to transform prev into next
func Diff(prev, next *VNode) []Patch {
	ctx := newDiffContext()
	diffNode(ctx, prev, next, 0)
	return ctx.patches
}

// diffNode recursively diffs two nodes
func diffNode(ctx *DiffContext, prev, next *VNode, parentID uint32) {
	// Both nil - nothing to do
	if prev == nil && next == nil {
		return
	}

	// Node removed
	if prev != nil && next == nil {
		nodeID := ctx.getNodeID(prev)
		ctx.addPatch(Patch{
			Op:     OpRemoveNode,
			NodeID: nodeID,
		})
		return
	}

	// Node added
	if prev == nil && next != nil {
		nodeID := ctx.getNodeID(next)
		ctx.addPatch(Patch{
			Op:       OpInsertNode,
			NodeID:   nodeID,
			ParentID: parentID,
			Node:     next,
		})
		return
	}

	// Different node types - replace
	if prev.Kind != next.Kind || (prev.Kind == KindElement && prev.Tag != next.Tag) {
		nodeID := ctx.getNodeID(prev)
		ctx.addPatch(Patch{
			Op:     OpRemoveNode,
			NodeID: nodeID,
		})
		nodeID = ctx.getNodeID(next)
		ctx.addPatch(Patch{
			Op:       OpInsertNode,
			NodeID:   nodeID,
			ParentID: parentID,
			Node:     next,
		})
		return
	}

	nodeID := ctx.getNodeID(prev)

	// Update node ID mapping for next node
	ctx.nodeMap[next] = nodeID

	// Diff based on node type
	switch prev.Kind {
	case KindText:
		if prev.Text != next.Text {
			ctx.addPatch(Patch{
				Op:     OpReplaceText,
				NodeID: nodeID,
				Value:  next.Text,
			})
		}

	case KindElement:
		// Diff attributes/props
		diffProps(ctx, nodeID, prev.Props, next.Props)

		// Diff children
		diffChildren(ctx, nodeID, prev.Kids, next.Kids)

	case KindFragment:
		// Fragment only has children
		diffChildren(ctx, nodeID, prev.Kids, next.Kids)

	case KindPortal:
		// Portal has target and children
		if prev.PortalTarget != next.PortalTarget {
			// Portal target changed - need to re-render
			ctx.addPatch(Patch{
				Op:     OpRemoveNode,
				NodeID: nodeID,
			})
			nodeID = ctx.getNodeID(next)
			ctx.addPatch(Patch{
				Op:       OpInsertNode,
				NodeID:   nodeID,
				ParentID: parentID,
				Node:     next,
			})
		} else {
			diffChildren(ctx, nodeID, prev.Kids, next.Kids)
		}
	}
}

// diffProps diffs properties/attributes
func diffProps(ctx *DiffContext, nodeID uint32, prevProps, nextProps Props) {
	// Track event changes
	var prevEvents, nextEvents uint32

	// Remove props that are no longer present
	if prevProps != nil {
		for key, prevVal := range prevProps {
			if key == "key" || key == "ref" { // skip special props
				continue // Skip key property
			}

			// Track event listeners
			if isEventProp(key) {
				prevEvents |= getEventBit(key)
			}

			nextVal, exists := nextProps[key]
			if !exists {
				if isEventProp(key) {
					// Event removed - will be handled by event update
				} else {
					ctx.addPatch(Patch{
						Op:     OpRemoveAttribute,
						NodeID: nodeID,
						Key:    key,
					})
				}
			} else if !propsEqual(prevVal, nextVal) {
				if isEventProp(key) {
					// Event handler changed - still need to track it
					nextEvents |= getEventBit(key)
				} else {
					ctx.addPatch(Patch{
						Op:     OpSetAttribute,
						NodeID: nodeID,
						Key:    key,
						Value:  propToString(nextVal),
					})
				}
			} else if isEventProp(key) {
				// Event unchanged
				nextEvents |= getEventBit(key)
			}
		}
	}

	// Add new props
	if nextProps != nil {
		for key, nextVal := range nextProps {
			if key == "key" || key == "ref" { // skip special props
				continue // Skip key property
			}

			// Track event listeners (only if not already tracked above)
			if isEventProp(key) && (prevProps == nil || prevProps[key] == nil) {
				nextEvents |= getEventBit(key)
			}

			if prevProps == nil {
				if !isEventProp(key) {
					ctx.addPatch(Patch{
						Op:     OpSetAttribute,
						NodeID: nodeID,
						Key:    key,
						Value:  propToString(nextVal),
					})
				}
			} else if _, exists := prevProps[key]; !exists {
				if !isEventProp(key) {
					ctx.addPatch(Patch{
						Op:     OpSetAttribute,
						NodeID: nodeID,
						Key:    key,
						Value:  propToString(nextVal),
					})
				}
			}
		}
	}

	// Update events if changed
	if prevEvents != nextEvents {
		ctx.addPatch(Patch{
			Op:        OpUpdateEvents,
			NodeID:    nodeID,
			EventBits: nextEvents,
		})
	}
}

// diffChildren diffs child nodes with keyed and unkeyed reconciliation
func diffChildren(ctx *DiffContext, parentID uint32, prevKids, nextKids []VNode) {
	// Fast path: no children
	if len(prevKids) == 0 && len(nextKids) == 0 {
		return
	}

	// Fast path: all children removed
	if len(nextKids) == 0 {
		for i := range prevKids {
			diffNode(ctx, &prevKids[i], nil, parentID)
		}
		return
	}

	// Fast path: all children added
	if len(prevKids) == 0 {
		for i := range nextKids {
			diffNode(ctx, nil, &nextKids[i], parentID)
		}
		return
	}

	// Check if children have keys
	hasKeys := false
	for i := range nextKids {
		if nextKids[i].GetKey() != "" {
			hasKeys = true
			break
		}
	}

	if hasKeys {
		diffKeyedChildren(ctx, parentID, prevKids, nextKids)
	} else {
		diffUnkeyedChildren(ctx, parentID, prevKids, nextKids)
	}
}

// diffUnkeyedChildren performs simple index-based diffing
func diffUnkeyedChildren(ctx *DiffContext, parentID uint32, prevKids, nextKids []VNode) {
	minLen := len(prevKids)
	if len(nextKids) < minLen {
		minLen = len(nextKids)
	}

	// Diff common children
	for i := 0; i < minLen; i++ {
		diffNode(ctx, &prevKids[i], &nextKids[i], parentID)
	}

	// Remove extra old children
	for i := minLen; i < len(prevKids); i++ {
		diffNode(ctx, &prevKids[i], nil, parentID)
	}

	// Add extra new children
	for i := minLen; i < len(nextKids); i++ {
		diffNode(ctx, nil, &nextKids[i], parentID)
	}
}

// diffKeyedChildren performs keyed reconciliation for efficient list updates
func diffKeyedChildren(ctx *DiffContext, parentID uint32, prevKids, nextKids []VNode) {
	// Build maps of keyed children
	prevKeyed := make(map[string]int)
	nextKeyed := make(map[string]int)

	for i, child := range prevKids {
		if key := child.GetKey(); key != "" {
			prevKeyed[key] = i
		}
	}

	for i, child := range nextKids {
		if key := child.GetKey(); key != "" {
			nextKeyed[key] = i
		}
	}

	// Track which old children have been matched
	matched := make([]bool, len(prevKids))

	// Track nodes that need to be moved
	var moves []struct {
		nodeID   uint32
		newIndex int
	}

	// Process new children
	for nextIdx, nextChild := range nextKids {
		key := nextChild.GetKey()

		if key != "" {
			// Keyed child - look for match in old children
			if prevIdx, found := prevKeyed[key]; found {
				// Found matching key
				matched[prevIdx] = true

				// Get the node ID before diffing (which updates the mapping)
				nodeID := ctx.getNodeID(&prevKids[prevIdx])

				// Diff the nodes
				diffNode(ctx, &prevKids[prevIdx], &nextChild, parentID)

				// Check if node needs to be moved
				if prevIdx != nextIdx {
					moves = append(moves, struct {
						nodeID   uint32
						newIndex int
					}{nodeID, nextIdx})
				}
			} else {
				// New keyed child
				diffNode(ctx, nil, &nextChild, parentID)
			}
		} else {
			// Unkeyed child - match by position
			if nextIdx < len(prevKids) && prevKids[nextIdx].GetKey() == "" && !matched[nextIdx] {
				matched[nextIdx] = true
				diffNode(ctx, &prevKids[nextIdx], &nextChild, parentID)
			} else {
				// New unkeyed child
				diffNode(ctx, nil, &nextChild, parentID)
			}
		}
	}

	// Remove unmatched old children
	for i, wasMatched := range matched {
		if !wasMatched {
			diffNode(ctx, &prevKids[i], nil, parentID)
		}
	}

	// Apply moves
	for _, move := range moves {
		var beforeID uint32
		if move.newIndex+1 < len(nextKids) {
			// Get the ID of the node that should come after this one
			beforeID = ctx.getNodeID(&nextKids[move.newIndex+1])
		}

		ctx.addPatch(Patch{
			Op:       OpMoveNode,
			NodeID:   move.nodeID,
			ParentID: parentID,
			BeforeID: beforeID,
		})
	}
}

// Helper functions

func isEventProp(key string) bool {
	return len(key) > 2 && key[0] == 'o' && key[1] == 'n'
}

func getEventBit(eventName string) uint32 {
	// Map event names to bit positions
	// This is a simplified version - real implementation would have all events
	switch eventName {
	case "onClick", "onclick":
		return 1 << 0
	case "onChange", "onchange":
		return 1 << 1
	case "onInput", "oninput":
		return 1 << 2
	case "onSubmit", "onsubmit":
		return 1 << 3
	case "onFocus", "onfocus":
		return 1 << 4
	case "onBlur", "onblur":
		return 1 << 5
	case "onKeyDown", "onkeydown":
		return 1 << 6
	case "onKeyUp", "onkeyup":
		return 1 << 7
	case "onMouseDown", "onmousedown":
		return 1 << 8
	case "onMouseUp", "onmouseup":
		return 1 << 9
	case "onMouseMove", "onmousemove":
		return 1 << 10
	case "onMouseEnter", "onmouseenter":
		return 1 << 11
	case "onMouseLeave", "onmouseleave":
		return 1 << 12
	default:
		return 1 << 31 // Unknown event
	}
}

func propsEqual(a, b any) bool {
	// Simple equality check - could be enhanced
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func propToString(v any) string {
	return fmt.Sprintf("%v", v)
}
