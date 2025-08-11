package reactive

import (
	"sync"
	"sync/atomic"

	"github.com/recera/vango/pkg/scheduler"
)

// Scheduler interface for reactive system
type Scheduler interface {
	MarkDirty(fiber *scheduler.Fiber)
}

// debugLog is set by platform-specific code
var debugLog func(args ...interface{})

// SetDebugLog sets the debug logging function
func SetDebugLog(fn func(args ...interface{})) {
	debugLog = fn
}

// currentFiber is dynamically scoped to track dependencies
var currentFiber atomic.Pointer[scheduler.Fiber]

// SetCurrentFiber sets the current fiber for dependency tracking
// This should be called by the scheduler before rendering
func SetCurrentFiber(fiber *scheduler.Fiber) {
	currentFiber.Store(fiber)
}

// GetCurrentFiber returns the current fiber
func GetCurrentFiber() *scheduler.Fiber {
	return currentFiber.Load()
}

// Signal is the interface for reactive values
type Signal[T any] interface {
	Get() T
	Set(T)
	Subscribe(fiber *scheduler.Fiber)
	Unsubscribe(fiber *scheduler.Fiber)
}

// State represents a reactive state value
type State[T any] struct {
	value T
	mu    sync.RWMutex
	
	// Dependencies - fibers that depend on this signal
	deps      map[uint32]*scheduler.Fiber
	depsMu    sync.RWMutex
	scheduler Scheduler
}

// NewState creates a new reactive state
func NewState[T any](initial T, sched Scheduler) *State[T] {
	return &State[T]{
		value:     initial,
		deps:      make(map[uint32]*scheduler.Fiber),
		scheduler: sched,
	}
}

// Get returns the current value and tracks dependencies
func (s *State[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Track dependency if we're in a fiber context
	if fiber := GetCurrentFiber(); fiber != nil {
		s.Subscribe(fiber)
	}
	
	return s.value
}

// Set updates the value and marks dependent fibers as dirty
func (s *State[T]) Set(value T) {
	if debugLog != nil {
		debugLog("[State] Set called with value:", value)
	}
	
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
	
	// Mark all dependent fibers as dirty
	s.depsMu.RLock()
	deps := make([]*scheduler.Fiber, 0, len(s.deps))
	for _, fiber := range s.deps {
		deps = append(deps, fiber)
	}
	s.depsMu.RUnlock()
	
	if debugLog != nil {
		debugLog("[State] Found", len(deps), "dependent fibers")
	}
	
	// Mark fibers dirty outside the lock to avoid deadlock
	for _, fiber := range deps {
		if debugLog != nil {
			debugLog("[State] Marking fiber", fiber.ID(), "as dirty")
		}
		markDirtyOrBatch(s.scheduler, fiber)
	}
}

// Subscribe adds a fiber as a dependency
func (s *State[T]) Subscribe(fiber *scheduler.Fiber) {
	if fiber == nil {
		return
	}
	
	s.depsMu.Lock()
	defer s.depsMu.Unlock()
	
	s.deps[fiber.ID()] = fiber
	if debugLog != nil {
		debugLog("[State] Subscribed fiber", fiber.ID(), "to state, total deps:", len(s.deps))
	}
}

// Unsubscribe removes a fiber as a dependency
func (s *State[T]) Unsubscribe(fiber *scheduler.Fiber) {
	if fiber == nil {
		return
	}
	
	s.depsMu.Lock()
	defer s.depsMu.Unlock()
	
	delete(s.deps, fiber.ID())
}

// Update atomically reads, modifies, and writes the value
func (s *State[T]) Update(fn func(T) T) {
	s.mu.Lock()
	oldValue := s.value
	s.value = fn(oldValue)
	newValue := s.value
	s.mu.Unlock()
	
	if debugLog != nil {
		debugLog("[State] Update called, old:", oldValue, "new:", newValue)
	}
	
	// Mark all dependent fibers as dirty
	s.depsMu.RLock()
	deps := make([]*scheduler.Fiber, 0, len(s.deps))
	for _, fiber := range s.deps {
		deps = append(deps, fiber)
	}
	s.depsMu.RUnlock()
	
	if debugLog != nil {
		debugLog("[State] Found", len(deps), "dependent fibers to update")
	}
	
	// Mark fibers dirty outside the lock
	for _, fiber := range deps {
		if debugLog != nil {
			debugLog("[State] Marking fiber", fiber.ID(), "as dirty")
		}
		markDirtyOrBatch(s.scheduler, fiber)
	}
}

// Computed represents a memoized computed value
type Computed[T any] struct {
	compute   func() T
	value     T
	valid     bool
	mu        sync.RWMutex
	deps      []*State[any] // Signals this computed depends on
	scheduler Scheduler
	
	// Dependencies - fibers that depend on this computed
	fiberDeps   map[uint32]*scheduler.Fiber
	fiberDepsMu sync.RWMutex
}

// NewComputed creates a new computed value
func NewComputed[T any](compute func() T, sched Scheduler) *Computed[T] {
	return &Computed[T]{
		compute:   compute,
		scheduler: sched,
		fiberDeps: make(map[uint32]*scheduler.Fiber),
	}
}

// Get returns the computed value, recalculating if necessary
func (c *Computed[T]) Get() T {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Track dependency if we're in a fiber context
	if fiber := GetCurrentFiber(); fiber != nil {
		c.Subscribe(fiber)
	}
	
	if !c.valid {
		// Clear old dependencies
		c.deps = nil
		
		// Compute the value (this will track new dependencies)
		c.value = c.compute()
		c.valid = true
	}
	
	return c.value
}

// Invalidate marks the computed value as needing recalculation
func (c *Computed[T]) Invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
	
	// Mark all dependent fibers as dirty
	c.fiberDepsMu.RLock()
	deps := make([]*scheduler.Fiber, 0, len(c.fiberDeps))
	for _, fiber := range c.fiberDeps {
		deps = append(deps, fiber)
	}
	c.fiberDepsMu.RUnlock()
	
	// Mark fibers dirty outside the lock
	for _, fiber := range deps {
		markDirtyOrBatch(c.scheduler, fiber)
	}
}

// Subscribe adds a fiber as a dependency
func (c *Computed[T]) Subscribe(fiber *scheduler.Fiber) {
	if fiber == nil {
		return
	}
	
	c.fiberDepsMu.Lock()
	defer c.fiberDepsMu.Unlock()
	
	c.fiberDeps[fiber.ID()] = fiber
}

// Unsubscribe removes a fiber as a dependency
func (c *Computed[T]) Unsubscribe(fiber *scheduler.Fiber) {
	if fiber == nil {
		return
	}
	
	c.fiberDepsMu.Lock()
	defer c.fiberDepsMu.Unlock()
	
	delete(c.fiberDeps, fiber.ID())
}

// batchContext holds the current batch state
var batchContext atomic.Pointer[Batch]

// Batch allows multiple state updates without triggering re-renders until the batch completes
type Batch struct {
	scheduler   Scheduler
	dirtyFibers map[uint32]*scheduler.Fiber
	mu          sync.Mutex
	active      bool
}

// NewBatch creates a new batch context
func NewBatch(sched Scheduler) *Batch {
	return &Batch{
		scheduler:   sched,
		dirtyFibers: make(map[uint32]*scheduler.Fiber),
		active:      true,
	}
}

// Add adds a fiber to the batch
func (b *Batch) Add(fiber *scheduler.Fiber) {
	if !b.active || fiber == nil {
		return
	}
	
	b.mu.Lock()
	b.dirtyFibers[fiber.ID()] = fiber
	b.mu.Unlock()
}

// Commit commits all batched updates
func (b *Batch) Commit() {
	b.mu.Lock()
	b.active = false
	fibers := make([]*scheduler.Fiber, 0, len(b.dirtyFibers))
	for _, fiber := range b.dirtyFibers {
		fibers = append(fibers, fiber)
	}
	b.dirtyFibers = nil
	b.mu.Unlock()
	
	// Mark all collected fibers as dirty
	for _, fiber := range fibers {
		b.scheduler.MarkDirty(fiber)
	}
}

// RunBatch executes a function within a batch context
func RunBatch(sched Scheduler, fn func()) {
	batch := NewBatch(sched)
	oldBatch := batchContext.Swap(batch)
	
	defer func() {
		batchContext.Store(oldBatch)
		batch.Commit()
	}()
	
	fn()
}

// markDirtyOrBatch marks a fiber dirty or adds to current batch
func markDirtyOrBatch(sched Scheduler, fiber *scheduler.Fiber) {
	if batch := batchContext.Load(); batch != nil && batch.active {
		if debugLog != nil {
			debugLog("[State] Adding fiber to batch")
		}
		batch.Add(fiber)
	} else if sched != nil {
		if debugLog != nil {
			debugLog("[State] Calling scheduler.MarkDirty for fiber", fiber.ID())
		}
		sched.MarkDirty(fiber)
	} else {
		if debugLog != nil {
			debugLog("[State] ERROR: No scheduler available!")
		}
	}
}

// Helper functions for easier API

// CreateState is a convenience function to create a new state
func CreateState[T any](initial T) *State[T] {
	// This would normally get the scheduler from context
	// For now, we'll require passing it explicitly
	return NewState(initial, nil)
}

// CreateComputed is a convenience function to create a new computed value
func CreateComputed[T any](compute func() T) *Computed[T] {
	// This would normally get the scheduler from context
	// For now, we'll require passing it explicitly
	return NewComputed(compute, nil)
}