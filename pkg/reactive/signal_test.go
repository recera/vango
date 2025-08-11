package reactive

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango/vdom"
)

func TestState_GetSet(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState(42, sched)
	
	// Test initial value
	if got := state.Get(); got != 42 {
		t.Errorf("Expected initial value 42, got %d", got)
	}
	
	// Test set
	state.Set(100)
	if got := state.Get(); got != 100 {
		t.Errorf("Expected value 100 after Set, got %d", got)
	}
}

func TestState_DependencyTracking(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState("hello", sched)
	
	var renderCount atomic.Int32
	
	// Create a fiber that depends on the state
	fiber := sched.CreateFiber(func() *vdom.VNode {
		// Set current fiber for dependency tracking
		SetCurrentFiber(sched.GetFiber(1)) // Assuming fiber ID is 1
		defer SetCurrentFiber(nil)
		
		renderCount.Add(1)
		value := state.Get()
		return vdom.NewText(value)
	}, nil)
	
	// Set the fiber as current and call Get to establish dependency
	SetCurrentFiber(fiber)
	_ = state.Get()
	SetCurrentFiber(nil)
	
	// Start scheduler
	sched.Start()
	defer sched.Stop()
	
	// Initial render
	sched.MarkDirty(fiber)
	time.Sleep(50 * time.Millisecond)
	
	initialCount := renderCount.Load()
	if initialCount != 1 {
		t.Errorf("Expected 1 initial render, got %d", initialCount)
	}
	
	// Update state should trigger re-render
	state.Set("world")
	time.Sleep(50 * time.Millisecond)
	
	if renderCount.Load() != 2 {
		t.Errorf("Expected 2 renders after state update, got %d", renderCount.Load())
	}
}

func TestState_Update(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState(10, sched)
	
	// Test update function
	state.Update(func(v int) int {
		return v * 2
	})
	
	if got := state.Get(); got != 20 {
		t.Errorf("Expected value 20 after Update, got %d", got)
	}
}

func TestState_ConcurrentAccess(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState(0, sched)
	
	var wg sync.WaitGroup
	
	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			state.Set(val)
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = state.Get()
		}()
	}
	
	wg.Wait()
	
	// Just verify no panic occurred
	t.Log("Concurrent access completed without panic")
}

func TestComputed_Basic(t *testing.T) {
	sched := scheduler.NewScheduler()
	
	count := NewState(5, sched)
	
	// Create a fiber context for dependency tracking
	fiber := sched.CreateFiber(func() *vdom.VNode {
		return vdom.NewText("test")
	}, nil)
	
	SetCurrentFiber(fiber)
	double := NewComputed(func() int {
		return count.Get() * 2
	}, sched)
	SetCurrentFiber(nil)
	
	// Initial computed value
	if got := double.Get(); got != 10 {
		t.Errorf("Expected computed value 10, got %d", got)
	}
	
	// Update dependency
	count.Set(7)
	double.Invalidate() // For now, manual invalidation
	
	// Computed should be invalidated
	if got := double.Get(); got != 14 {
		t.Errorf("Expected computed value 14 after update, got %d", got)
	}
}

func TestComputed_Memoization(t *testing.T) {
	sched := scheduler.NewScheduler()
	
	var computeCount atomic.Int32
	expensive := NewComputed(func() int {
		computeCount.Add(1)
		time.Sleep(10 * time.Millisecond) // Simulate expensive computation
		return 42
	}, sched)
	
	// First call should compute
	_ = expensive.Get()
	if computeCount.Load() != 1 {
		t.Errorf("Expected 1 computation, got %d", computeCount.Load())
	}
	
	// Second call should use cached value
	_ = expensive.Get()
	if computeCount.Load() != 1 {
		t.Errorf("Expected still 1 computation (memoized), got %d", computeCount.Load())
	}
	
	// After invalidation, should recompute
	expensive.Invalidate()
	_ = expensive.Get()
	if computeCount.Load() != 2 {
		t.Errorf("Expected 2 computations after invalidation, got %d", computeCount.Load())
	}
}

func TestComputed_ChainedDependencies(t *testing.T) {
	sched := scheduler.NewScheduler()
	
	// Create a chain: a -> b -> c
	a := NewState(1, sched)
	
	b := NewComputed(func() int {
		SetCurrentFiber(&scheduler.Fiber{}) // Mock fiber for dependency tracking
		defer SetCurrentFiber(nil)
		return a.Get() + 1
	}, sched)
	
	c := NewComputed(func() int {
		SetCurrentFiber(&scheduler.Fiber{}) // Mock fiber for dependency tracking
		defer SetCurrentFiber(nil)
		return b.Get() * 2
	}, sched)
	
	// Initial values
	if got := c.Get(); got != 4 { // (1 + 1) * 2 = 4
		t.Errorf("Expected computed value 4, got %d", got)
	}
	
	// Update root
	a.Set(5)
	b.Invalidate() // Manual invalidation for this test
	c.Invalidate()
	
	if got := c.Get(); got != 12 { // (5 + 1) * 2 = 12
		t.Errorf("Expected computed value 12 after update, got %d", got)
	}
}

func TestBatch(t *testing.T) {
	sched := scheduler.NewScheduler()
	
	// Track how many times fibers are marked dirty
	var markDirtyCount atomic.Int32
	
	// Create states with tracking scheduler
	trackingSched := &trackingScheduler{
		Scheduler: sched,
		counter:   &markDirtyCount,
	}
	
	state1 := NewState(1, trackingSched)
	state2 := NewState(2, trackingSched)
	state3 := NewState(3, trackingSched)
	
	// Create a fiber that depends on all states
	fiber := sched.CreateFiber(func() *vdom.VNode {
		sum := state1.Get() + state2.Get() + state3.Get()
		return vdom.NewText(string(rune(sum)))
	}, nil)
	
	// Establish dependencies
	SetCurrentFiber(fiber)
	_ = state1.Get()
	_ = state2.Get()
	_ = state3.Get()
	SetCurrentFiber(nil)
	
	// Without batch - each Set should mark fiber dirty
	markDirtyCount.Store(0)
	state1.Set(10)
	state2.Set(20)
	state3.Set(30)
	
	withoutBatchCount := markDirtyCount.Load()
	if withoutBatchCount != 3 {
		t.Errorf("Expected 3 MarkDirty calls without batch, got %d", withoutBatchCount)
	}
	
	// With batch - should only mark dirty once at the end
	markDirtyCount.Store(0)
	RunBatch(trackingSched, func() {
		state1.Set(100)
		state2.Set(200)
		state3.Set(300)
	})
	
	withBatchCount := markDirtyCount.Load()
	if withBatchCount != 1 {
		t.Errorf("Expected 1 MarkDirty call with batch, got %d", withBatchCount)
	}
}

// trackingScheduler wraps a scheduler to count MarkDirty calls
type trackingScheduler struct {
	*scheduler.Scheduler
	counter *atomic.Int32
}

func (t *trackingScheduler) MarkDirty(fiber *scheduler.Fiber) {
	t.counter.Add(1)
	t.Scheduler.MarkDirty(fiber)
}

func TestSignal_Unsubscribe(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState("test", sched)
	
	fiber := &scheduler.Fiber{}
	
	// Subscribe
	state.Subscribe(fiber)
	if len(state.deps) != 1 {
		t.Errorf("Expected 1 dependency after subscribe, got %d", len(state.deps))
	}
	
	// Unsubscribe
	state.Unsubscribe(fiber)
	if len(state.deps) != 0 {
		t.Errorf("Expected 0 dependencies after unsubscribe, got %d", len(state.deps))
	}
}

func TestSignal_NilFiber(t *testing.T) {
	sched := scheduler.NewScheduler()
	state := NewState(42, sched)
	
	// Should not panic with nil fiber
	state.Subscribe(nil)
	state.Unsubscribe(nil)
	
	// Get without current fiber should work
	SetCurrentFiber(nil)
	val := state.Get()
	if val != 42 {
		t.Errorf("Expected value 42, got %d", val)
	}
}

func BenchmarkState_Get(b *testing.B) {
	sched := scheduler.NewScheduler()
	state := NewState(42, sched)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.Get()
	}
}

func BenchmarkState_Set(b *testing.B) {
	sched := scheduler.NewScheduler()
	state := NewState(0, sched)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Set(i)
	}
}

func BenchmarkComputed_Get(b *testing.B) {
	sched := scheduler.NewScheduler()
	base := NewState(10, sched)
	computed := NewComputed(func() int {
		return base.Get() * 2
	}, sched)
	
	// Prime the cache
	_ = computed.Get()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computed.Get()
	}
}