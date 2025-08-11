package scheduler

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/recera/vango/pkg/vango/vdom"
)

// RenderFunc is the function type for component render functions
type RenderFunc func() *vdom.VNode

// ErrorHandler handles panics during rendering
// Returns true to continue scheduling, false to unmount the fiber
type ErrorHandler func(fiber *Fiber, err interface{}) bool

// Fiber represents a lightweight component execution context
type Fiber struct {
	id     uint32
	parent *Fiber
	vnode  *vdom.VNode // last rendered tree
	
	// Component render function
	render RenderFunc
	
	// Scheduling state
	ch    chan struct{} // wake-up signal
	dirty atomic.Bool   // atomic for thread safety
	
	// Error handling
	onError ErrorHandler
	
	// User data
	userData interface{}
}

// debugLog is set by platform-specific code
var debugLog func(args ...interface{})

// SetDebugLog sets the debug logging function
func SetDebugLog(fn func(args ...interface{})) {
	debugLog = fn
}

// Scheduler manages fiber execution
type Scheduler struct {
	mu         sync.Mutex
	fibers     map[uint32]*Fiber
	nextID     uint32
	dirtyQueue []*Fiber
	globalWake chan *Fiber
	running    atomic.Bool
	
	// Callbacks
	applyPatches func(patches []vdom.Patch)
	defaultError ErrorHandler
}

// NewScheduler creates a new scheduler instance
func NewScheduler() *Scheduler {
	return &Scheduler{
		fibers:     make(map[uint32]*Fiber),
		nextID:     1,
		dirtyQueue: make([]*Fiber, 0, 1024),
		globalWake: make(chan *Fiber, 1024), // buffered for performance
	}
}

// SetPatchApplier sets the function that applies patches to the DOM
func (s *Scheduler) SetPatchApplier(applier func(patches []vdom.Patch)) {
	s.applyPatches = applier
}

// SetDefaultErrorHandler sets the default error handler for fibers
func (s *Scheduler) SetDefaultErrorHandler(handler ErrorHandler) {
	s.defaultError = handler
}

// CreateFiber creates a new fiber for a component
func (s *Scheduler) CreateFiber(render RenderFunc, parent *Fiber) *Fiber {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	id := s.nextID
	s.nextID++
	
	fiber := &Fiber{
		id:     id,
		parent: parent,
		render: render,
		ch:     make(chan struct{}, 1), // buffered to avoid blocking
	}
	
	// Use default error handler if none specified
	if s.defaultError != nil {
		fiber.onError = s.defaultError
	}
	
	s.fibers[id] = fiber
	return fiber
}

// RemoveFiber removes a fiber from the scheduler
func (s *Scheduler) RemoveFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.fibers, fiber.id)
}

// MarkDirty marks a fiber as needing re-render
func (s *Scheduler) MarkDirty(fiber *Fiber) {
	if fiber == nil {
		return
	}
	
	if debugLog != nil {
		debugLog("[Scheduler] MarkDirty called for fiber", fiber.ID())
	}
	
	// Set dirty flag atomically
	if fiber.dirty.CompareAndSwap(false, true) {
		if debugLog != nil {
			debugLog("[Scheduler] Fiber", fiber.ID(), "marked dirty")
		}
		// Only send to wake channel if scheduler is running
		if s.running.Load() {
			if debugLog != nil {
				debugLog("[Scheduler] Scheduler is running, sending fiber to wake channel")
			}
			select {
			case s.globalWake <- fiber:
				if debugLog != nil {
					debugLog("[Scheduler] Fiber", fiber.ID(), "sent to wake channel")
				}
			default:
				// Channel full, fiber will be picked up in next batch
				if debugLog != nil {
					debugLog("[Scheduler] Wake channel full for fiber", fiber.ID())
				}
			}
		} else {
			if debugLog != nil {
				debugLog("[Scheduler] WARNING: Scheduler not running!")
			}
		}
	} else {
		if debugLog != nil {
			debugLog("[Scheduler] Fiber", fiber.ID(), "already dirty")
		}
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start() {
	if s.running.CompareAndSwap(false, true) {
		if debugLog != nil {
			debugLog("[Scheduler] Starting scheduler loop")
		}
		go s.loop()
	} else {
		if debugLog != nil {
			debugLog("[Scheduler] Scheduler already running")
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.running.Store(false)
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}

// loop is the main scheduler event loop
func (s *Scheduler) loop() {
	if debugLog != nil {
		debugLog("[Scheduler] Loop started")
	}
	for s.running.Load() {
		// Wait for at least one dirty fiber
		var fiber *Fiber
		select {
		case fiber = <-s.globalWake:
			if fiber == nil {
				continue
			}
			if debugLog != nil {
				debugLog("[Scheduler] Received fiber", fiber.ID(), "from wake channel")
			}
		default:
			// No dirty fibers, check if we should stop
			if !s.running.Load() {
				return
			}
			// Block waiting for work
			if debugLog != nil {
				debugLog("[Scheduler] Waiting for work...")
			}
			fiber = <-s.globalWake
			if fiber == nil {
				continue
			}
			if debugLog != nil {
				debugLog("[Scheduler] Received fiber", fiber.ID(), "from wake channel (blocked)")
			}
		}
		
		// Collect all currently dirty fibers to batch process
		batch := []*Fiber{fiber}
		
		// Drain the channel to get all pending dirty fibers
	drainLoop:
		for {
			select {
			case f := <-s.globalWake:
				if f != nil {
					batch = append(batch, f)
				}
			default:
				break drainLoop
			}
		}
		
		// Process the batch
		if debugLog != nil {
			debugLog("[Scheduler] Processing batch of", len(batch), "fibers")
		}
		for _, f := range batch {
			s.processFiber(f)
		}
	}
	if debugLog != nil {
		debugLog("[Scheduler] Loop ended")
	}
}

// processFiber renders a single fiber and applies patches
func (s *Scheduler) processFiber(fiber *Fiber) {
	if debugLog != nil {
		debugLog("[Scheduler] Processing fiber", fiber.ID())
	}
	
	// Check if still dirty (might have been processed in a previous batch)
	if !fiber.dirty.CompareAndSwap(true, false) {
		if debugLog != nil {
			debugLog("[Scheduler] Fiber", fiber.ID(), "no longer dirty, skipping")
		}
		return
	}
	
	// Wrap render in panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.handleFiberError(fiber, r)
			}
		}()
		
		if debugLog != nil {
			debugLog("[Scheduler] Rendering fiber", fiber.ID())
		}
		
		// Render the component
		next := fiber.render()
		
		// Diff against previous render
		patches := vdom.Diff(fiber.vnode, next)
		
		if debugLog != nil {
			debugLog("[Scheduler] Diff produced", len(patches), "patches for fiber", fiber.ID())
		}
		
		// Apply patches if we have an applier
		if s.applyPatches != nil && len(patches) > 0 {
			if debugLog != nil {
				debugLog("[Scheduler] Applying", len(patches), "patches")
			}
			s.applyPatches(patches)
		}
		
		// Update the fiber's vnode
		fiber.vnode = next
	}()
}

// handleFiberError handles a panic during fiber rendering
func (s *Scheduler) handleFiberError(fiber *Fiber, err interface{}) {
	// Create error with stack trace
	errorMsg := fmt.Sprintf("Fiber %d panic: %v\n%s", fiber.id, err, debug.Stack())
	
	// Call error handler
	shouldContinue := false
	if fiber.onError != nil {
		shouldContinue = fiber.onError(fiber, errorMsg)
	}
	
	// If error handler says not to continue, remove the fiber
	if !shouldContinue {
		s.RemoveFiber(fiber)
	}
}

// GetFiber returns a fiber by ID
func (s *Scheduler) GetFiber(id uint32) *Fiber {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fibers[id]
}

// FiberCount returns the number of active fibers
func (s *Scheduler) FiberCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.fibers)
}

// SetUserData sets custom data on a fiber
func (f *Fiber) SetUserData(data interface{}) {
	f.userData = data
}

// GetUserData gets custom data from a fiber
func (f *Fiber) GetUserData() interface{} {
	return f.userData
}

// ID returns the fiber's unique ID
func (f *Fiber) ID() uint32 {
	return f.id
}

// Parent returns the fiber's parent
func (f *Fiber) Parent() *Fiber {
	return f.parent
}

// VNode returns the fiber's last rendered VNode
func (f *Fiber) VNode() *vdom.VNode {
	return f.vnode
}

// SetVNode sets the current VNode for the fiber (used during hydration)
func (f *Fiber) SetVNode(vnode *vdom.VNode) {
	f.vnode = vnode
}

// SetErrorHandler sets a custom error handler for this fiber
func (f *Fiber) SetErrorHandler(handler ErrorHandler) {
	f.onError = handler
}