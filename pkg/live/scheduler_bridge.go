//go:build !wasm
// +build !wasm

package live

import (
	"errors"
	"log"
	"sync"
	
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango"
	"github.com/recera/vango/pkg/vango/vdom"
)

// SchedulerBridge connects the scheduler to the Live Protocol
type SchedulerBridge struct {
	mu        sync.RWMutex
	scheduler *scheduler.Scheduler
	server    *Server
	sessions  map[string]*BridgedSession
}

// BridgedSession represents a session with scheduler integration
type BridgedSession struct {
	Session    *Session
	Scheduler  *scheduler.Scheduler
	Components map[string]*server.ComponentInstance
}

// NewSchedulerBridge creates a new scheduler bridge
func NewSchedulerBridge(liveServer *Server) *SchedulerBridge {
	return &SchedulerBridge{
		server:   liveServer,
		sessions: make(map[string]*BridgedSession),
	}
}

// CreateSessionScheduler creates a scheduler for a session
func (b *SchedulerBridge) CreateSessionScheduler(sessionID string) *scheduler.Scheduler {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Create a new scheduler for this session
	sched := scheduler.NewScheduler()
	
	// Get the live session
	session, exists := b.server.GetSession(sessionID)
	if !exists {
		log.Printf("[SchedulerBridge] Session %s not found", sessionID)
		return nil
	}
	
	// Create bridged session
	bridged := &BridgedSession{
		Session:    session,
		Scheduler:  sched,
		Components: make(map[string]*server.ComponentInstance),
	}
	
	b.sessions[sessionID] = bridged
	
	// Set up patch applier that sends patches via WebSocket
	sched.SetPatchApplier(func(patches []vdom.Patch) {
		if len(patches) == 0 {
			return
		}
		
		log.Printf("[SchedulerBridge] Sending %d patches for session %s", len(patches), sessionID)
		
		// Send patches to client via WebSocket
		if err := session.SendPatches(patches); err != nil {
			log.Printf("[SchedulerBridge] Failed to send patches: %v", err)
		}
	})
	
	// Set up error handler
	sched.SetDefaultErrorHandler(func(fiber *scheduler.Fiber, err interface{}) bool {
		log.Printf("[SchedulerBridge] Fiber error in session %s: %v", sessionID, err)
		// Continue scheduling
		return true
	})
	
	// Start the scheduler
	sched.Start()
	log.Printf("[SchedulerBridge] Started scheduler for session %s", sessionID)
	
	// Connect any existing components for this session
	b.connectExistingComponents(sessionID, bridged)
	
	return sched
}

// CreateServerComponent creates a server-driven component instance
func (b *SchedulerBridge) CreateServerComponent(
	sessionID string,
	componentID string,
	render func(ctx *vango.Context) *vdom.VNode,
) (*server.ComponentInstance, error) {
	// First check if we need to create a scheduler (without holding lock)
	b.mu.RLock()
	_, exists := b.sessions[sessionID]
	b.mu.RUnlock()
	
	if !exists {
		// Create scheduler if not exists (this will acquire its own lock)
		sched := b.CreateSessionScheduler(sessionID)
		if sched == nil {
			return nil, ErrSessionNotFound
		}
	}
	
	// Now acquire lock for the rest of the operation
	b.mu.Lock()
	defer b.mu.Unlock()
	
	bridged, exists := b.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}
	
	// Create component instance
	component := server.NewComponentInstance(componentID, sessionID, render)
	
	// Create context for the component
	ctx := vango.NewContext(vango.ModeServerDriven).
		WithScheduler(bridged.Scheduler).
		WithSessionID(sessionID)
	
	// CRITICAL: Set the component in the context so render function can access it
	ctx.Set("component", component)
	
	component.Context = ctx
	
	// Create a fiber for the component
	fiber := bridged.Scheduler.CreateFiber(func() *vdom.VNode {
		// Ensure component is in context for each render
		ctx.Set("component", component)
		
		// This render function will be called by the scheduler
		vnode := render(ctx)
		
		// Store the rendered VNode in the component
		component.LastVNode = vnode
		
		return vnode
	}, nil)
	
	component.Fiber = fiber
	
	// Store component in bridged session
	bridged.Components[componentID] = component
	
	// Register in global registry
	server.GetRegistry().Register(component)
	
	log.Printf("[SchedulerBridge] Created server component %s for session %s", componentID, sessionID)
	
	// Trigger initial render
	bridged.Scheduler.MarkDirty(fiber)
	
	return component, nil
}

// HandleComponentEvent routes an event to a component
func (b *SchedulerBridge) HandleComponentEvent(sessionID string, nodeID uint32, eventType string) error {
	log.Printf("[SchedulerBridge] HandleComponentEvent: session=%s, nodeID=%d, type=%s", sessionID, nodeID, eventType)
	
	b.mu.RLock()
	bridged, exists := b.sessions[sessionID]
	b.mu.RUnlock()
	
	if !exists {
		log.Printf("[SchedulerBridge] Session not found: %s", sessionID)
		log.Printf("[SchedulerBridge] Available sessions: %v", b.getSessionIDs())
		return ErrSessionNotFound
	}
	
	// Find component by node ID
	registry := server.GetRegistry()
	component, found := registry.GetByNodeID(nodeID)
	if !found {
		log.Printf("[SchedulerBridge] No component found for node %d", nodeID)
		log.Printf("[SchedulerBridge] Looking for component in session %s", sessionID)
		
		// Try to find component by session
		for _, comp := range bridged.Components {
			log.Printf("[SchedulerBridge] Available component: %s", comp.ID)
			// Check if this component has the handler
			if err := comp.HandleEvent(nodeID, eventType); err == nil {
				log.Printf("[SchedulerBridge] Found handler in component %s", comp.ID)
				return nil
			}
		}
		
		return ErrComponentNotFound
	}
	
	// Handle the event
	if err := component.HandleEvent(nodeID, eventType); err != nil {
		log.Printf("[SchedulerBridge] Error handling event: %v", err)
		return err
	}
	
	// Use the bridged scheduler for this session
	_ = bridged // Mark as used
	
	// The event handler should have updated state and marked fiber dirty
	// The scheduler will handle re-rendering and patch generation
	
	return nil
}

// CleanupSession cleans up when a session ends
func (b *SchedulerBridge) CleanupSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	bridged, exists := b.sessions[sessionID]
	if !exists {
		return
	}
	
	// Stop the scheduler
	if bridged.Scheduler != nil {
		bridged.Scheduler.Stop()
	}
	
	// Clean up components
	server.GetRegistry().CleanupSession(sessionID)
	
	// Remove from sessions
	delete(b.sessions, sessionID)
	
	log.Printf("[SchedulerBridge] Cleaned up session %s", sessionID)
}

// GetBridgedSession returns a bridged session
func (b *SchedulerBridge) GetBridgedSession(sessionID string) (*BridgedSession, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	session, exists := b.sessions[sessionID]
	return session, exists
}

// getSessionIDs returns all active session IDs for debugging
func (b *SchedulerBridge) getSessionIDs() []string {
	var ids []string
	for id := range b.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Global bridge instance
var globalBridge *SchedulerBridge

// InitBridge initializes the global scheduler bridge
func InitBridge(server *Server) {
	globalBridge = NewSchedulerBridge(server)
}

// GetBridge returns the global scheduler bridge
func GetBridge() *SchedulerBridge {
	return globalBridge
}

// connectExistingComponents connects components that were created before the scheduler
func (b *SchedulerBridge) connectExistingComponents(sessionID string, bridged *BridgedSession) {
	registry := server.GetRegistry()
	components := registry.GetBySession(sessionID)
	
	for _, component := range components {
		if component.Fiber == nil {
			log.Printf("[SchedulerBridge] Connecting existing component %s to scheduler", component.ID)
			
			// Create context with scheduler
			ctx := vango.NewContext(vango.ModeServerDriven).
				WithScheduler(bridged.Scheduler).
				WithSessionID(sessionID)
			
			// Set component in context
			ctx.Set("component", component)
			component.Context = ctx
			
			// Create fiber for the component
			fiber := bridged.Scheduler.CreateFiber(func() *vdom.VNode {
				// Ensure component is in context
				ctx.Set("component", component)
				
				// Render the component
				vnode := component.RenderFunc(ctx)
				
				// Store the rendered VNode
				component.LastVNode = vnode
				
				return vnode
			}, nil)
			
			component.Fiber = fiber
			
			// Store in bridged session
			bridged.Components[component.ID] = component
			
			// Trigger initial render
			bridged.Scheduler.MarkDirty(fiber)
			
			log.Printf("[SchedulerBridge] Connected component %s with fiber", component.ID)
		}
	}
}

// Error definitions
var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrComponentNotFound  = errors.New("component not found")
	ErrSchedulerNotActive = errors.New("scheduler not active")
)