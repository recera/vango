package server

import (
	"fmt"
	"sync"
	
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango"
	"github.com/recera/vango/pkg/vango/vdom"
)

// ComponentInstance represents a server-side component instance
type ComponentInstance struct {
	ID        string
	SessionID string
	Fiber     *scheduler.Fiber
	Context   *vango.Context
	
	// Component function
	RenderFunc func(ctx *vango.Context) *vdom.VNode // Made public for scheduler bridge
	
	// Current state
	state map[string]interface{}
	mu    sync.RWMutex
	
	// Event handlers registered by the component
	handlers map[uint32]func() // nodeID -> handler
	
	// Last rendered VNode tree
	LastVNode *vdom.VNode
}

// NewComponentInstance creates a new component instance
func NewComponentInstance(id, sessionID string, render func(ctx *vango.Context) *vdom.VNode) *ComponentInstance {
	return &ComponentInstance{
		ID:         id,
		SessionID:  sessionID,
		RenderFunc: render,
		state:      make(map[string]interface{}),
		handlers:   make(map[uint32]func()),
	}
}

// SetState updates component state
func (c *ComponentInstance) SetState(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state[key] = value
	
	// Mark fiber as dirty to trigger re-render
	if c.Fiber != nil {
		if sched := c.Context.Scheduler; sched != nil {
			sched.MarkDirty(c.Fiber)
		}
	}
}

// GetState retrieves component state
func (c *ComponentInstance) GetState(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.state[key]
	return val, ok
}

// RegisterHandler registers an event handler for a node
func (c *ComponentInstance) RegisterHandler(nodeID uint32, handler func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[nodeID] = handler
	
	// Also register the mapping in the global registry
	// so that events can find this component by node ID
	GetRegistry().MapNodeToComponent(nodeID, c)
}

// HandleEvent processes an event for this component
func (c *ComponentInstance) HandleEvent(nodeID uint32, eventType string) error {
	c.mu.RLock()
	handler, ok := c.handlers[nodeID]
	c.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("no handler for node %d", nodeID)
	}
	
	// Execute the handler
	handler()
	
	return nil
}

// Render renders the component and returns patches
func (c *ComponentInstance) Render() ([]vdom.Patch, error) {
	// Set up context for rendering
	ctx := vango.NewContext(vango.ModeServerDriven).
		WithSessionID(c.SessionID)
	
	// Store component instance in context for access during render
	ctx.Set("component", c)
	
	// Render the component
	newVNode := c.RenderFunc(ctx)
	
	// Diff against previous render
	var patches []vdom.Patch
	if c.LastVNode != nil {
		patches = vdom.Diff(c.LastVNode, newVNode)
	}
	
	// Update last VNode
	c.LastVNode = newVNode
	
	return patches, nil
}

// ComponentRegistry manages component instances
type ComponentRegistry struct {
	mu         sync.RWMutex
	instances  map[string]*ComponentInstance // instanceID -> instance
	bySession  map[string][]*ComponentInstance // sessionID -> instances
	byNodeID   map[uint32]*ComponentInstance // nodeID -> instance
}

// NewComponentRegistry creates a new registry
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		instances: make(map[string]*ComponentInstance),
		bySession: make(map[string][]*ComponentInstance),
		byNodeID:  make(map[uint32]*ComponentInstance),
	}
}

// Register adds a component instance to the registry
func (r *ComponentRegistry) Register(instance *ComponentInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.instances[instance.ID] = instance
	r.bySession[instance.SessionID] = append(r.bySession[instance.SessionID], instance)
}

// Unregister removes a component instance
func (r *ComponentRegistry) Unregister(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instance, ok := r.instances[instanceID]
	if !ok {
		return
	}
	
	delete(r.instances, instanceID)
	
	// Remove from session list
	if sessions, ok := r.bySession[instance.SessionID]; ok {
		for i, inst := range sessions {
			if inst.ID == instanceID {
				r.bySession[instance.SessionID] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		}
	}
	
	// Remove node mappings
	for nodeID, inst := range r.byNodeID {
		if inst.ID == instanceID {
			delete(r.byNodeID, nodeID)
		}
	}
}

// GetByID retrieves a component by instance ID
func (r *ComponentRegistry) GetByID(id string) (*ComponentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instance, ok := r.instances[id]
	return instance, ok
}

// GetByNodeID retrieves a component by node ID
func (r *ComponentRegistry) GetByNodeID(nodeID uint32) (*ComponentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	instance, ok := r.byNodeID[nodeID]
	return instance, ok
}

// GetBySession retrieves all components for a session
func (r *ComponentRegistry) GetBySession(sessionID string) []*ComponentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bySession[sessionID]
}

// MapNodeToComponent maps a node ID to a component instance
func (r *ComponentRegistry) MapNodeToComponent(nodeID uint32, instance *ComponentInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byNodeID[nodeID] = instance
}

// CleanupSession removes all components for a session
func (r *ComponentRegistry) CleanupSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instances := r.bySession[sessionID]
	for _, instance := range instances {
		delete(r.instances, instance.ID)
		
		// Remove node mappings
		for nodeID, inst := range r.byNodeID {
			if inst.ID == instance.ID {
				delete(r.byNodeID, nodeID)
			}
		}
	}
	
	delete(r.bySession, sessionID)
}

// Global registry instance
var globalRegistry = NewComponentRegistry()

// GetRegistry returns the global component registry
func GetRegistry() *ComponentRegistry {
	return globalRegistry
}