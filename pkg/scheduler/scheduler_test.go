package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recera/vango/pkg/vango/vdom"
)

func TestScheduler_CreateFiber(t *testing.T) {
	sched := NewScheduler()
	
	renderCalled := false
	render := func() *vdom.VNode {
		renderCalled = true
		return vdom.NewText("test")
	}
	
	fiber := sched.CreateFiber(render, nil)
	
	if fiber == nil {
		t.Fatal("CreateFiber returned nil")
	}
	
	if fiber.ID() == 0 {
		t.Error("Fiber ID should not be 0")
	}
	
	if fiber.Parent() != nil {
		t.Error("Parent should be nil")
	}
	
	if renderCalled {
		t.Error("Render should not be called during creation")
	}
	
	// Check fiber is tracked
	if sched.FiberCount() != 1 {
		t.Errorf("Expected 1 fiber, got %d", sched.FiberCount())
	}
}

func TestScheduler_MarkDirty(t *testing.T) {
	sched := NewScheduler()
	
	var renderCount atomic.Int32
	var patchCount atomic.Int32
	
	render := func() *vdom.VNode {
		renderCount.Add(1)
		return vdom.NewElement("div", nil, vdom.NewText("test"))
	}
	
	sched.SetPatchApplier(func(patches []vdom.Patch) {
		patchCount.Add(int32(len(patches)))
	})
	
	fiber := sched.CreateFiber(render, nil)
	
	// Start scheduler
	sched.Start()
	defer sched.Stop()
	
	// Mark fiber dirty
	sched.MarkDirty(fiber)
	
	// Wait for processing
	time.Sleep(50 * time.Millisecond)
	
	if renderCount.Load() != 1 {
		t.Errorf("Expected render to be called once, got %d", renderCount.Load())
	}
	
	// Mark dirty again
	sched.MarkDirty(fiber)
	time.Sleep(50 * time.Millisecond)
	
	if renderCount.Load() != 2 {
		t.Errorf("Expected render to be called twice, got %d", renderCount.Load())
	}
}

func TestScheduler_BatchProcessing(t *testing.T) {
	sched := NewScheduler()
	
	var totalRenders atomic.Int32
	renderFunc := func(id int) RenderFunc {
		return func() *vdom.VNode {
			totalRenders.Add(1)
			return vdom.NewText(string(rune('A' + id)))
		}
	}
	
	// Create multiple fibers
	fibers := make([]*Fiber, 10)
	for i := 0; i < 10; i++ {
		fibers[i] = sched.CreateFiber(renderFunc(i), nil)
	}
	
	// Start scheduler
	sched.Start()
	defer sched.Stop()
	
	// Mark all fibers dirty at once
	for _, f := range fibers {
		sched.MarkDirty(f)
	}
	
	// Wait for batch processing
	time.Sleep(100 * time.Millisecond)
	
	if totalRenders.Load() != 10 {
		t.Errorf("Expected 10 renders, got %d", totalRenders.Load())
	}
}

func TestScheduler_ErrorHandling(t *testing.T) {
	sched := NewScheduler()
	
	var errorHandled atomic.Bool
	var shouldContinue = true
	
	sched.SetDefaultErrorHandler(func(f *Fiber, err interface{}) bool {
		errorHandled.Store(true)
		return shouldContinue
	})
	
	// Create fiber that panics
	panicRender := func() *vdom.VNode {
		panic("test panic")
	}
	
	fiber := sched.CreateFiber(panicRender, nil)
	
	sched.Start()
	defer sched.Stop()
	
	// Trigger render
	sched.MarkDirty(fiber)
	time.Sleep(50 * time.Millisecond)
	
	if !errorHandled.Load() {
		t.Error("Error handler was not called")
	}
	
	// Fiber should still exist
	if sched.GetFiber(fiber.ID()) == nil {
		t.Error("Fiber was removed despite error handler returning true")
	}
	
	// Test with error handler returning false
	shouldContinue = false
	errorHandled.Store(false)
	
	fiber2 := sched.CreateFiber(panicRender, nil)
	sched.MarkDirty(fiber2)
	time.Sleep(50 * time.Millisecond)
	
	if !errorHandled.Load() {
		t.Error("Error handler was not called for second fiber")
	}
	
	// Fiber should be removed
	if sched.GetFiber(fiber2.ID()) != nil {
		t.Error("Fiber was not removed when error handler returned false")
	}
}

func TestScheduler_ConcurrentMarkDirty(t *testing.T) {
	sched := NewScheduler()
	
	var renderCount atomic.Int32
	render := func() *vdom.VNode {
		renderCount.Add(1)
		return vdom.NewText("concurrent")
	}
	
	fiber := sched.CreateFiber(render, nil)
	
	sched.Start()
	defer sched.Stop()
	
	// Concurrently mark dirty from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sched.MarkDirty(fiber)
		}()
	}
	
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	
	// Should have rendered at least once, but not necessarily 100 times
	// due to deduplication
	if renderCount.Load() == 0 {
		t.Error("Fiber was not rendered despite being marked dirty")
	}
	
	t.Logf("Fiber rendered %d times out of 100 dirty marks", renderCount.Load())
}

func TestScheduler_RemoveFiber(t *testing.T) {
	sched := NewScheduler()
	
	fiber1 := sched.CreateFiber(func() *vdom.VNode { return nil }, nil)
	fiber2 := sched.CreateFiber(func() *vdom.VNode { return nil }, fiber1)
	
	if sched.FiberCount() != 2 {
		t.Errorf("Expected 2 fibers, got %d", sched.FiberCount())
	}
	
	// Remove fiber1
	sched.RemoveFiber(fiber1)
	
	if sched.FiberCount() != 1 {
		t.Errorf("Expected 1 fiber after removal, got %d", sched.FiberCount())
	}
	
	if sched.GetFiber(fiber1.ID()) != nil {
		t.Error("Fiber1 should not be found after removal")
	}
	
	if sched.GetFiber(fiber2.ID()) == nil {
		t.Error("Fiber2 should still exist")
	}
}

func TestScheduler_StopStart(t *testing.T) {
	sched := NewScheduler()
	
	if sched.IsRunning() {
		t.Error("Scheduler should not be running initially")
	}
	
	sched.Start()
	time.Sleep(10 * time.Millisecond)
	
	if !sched.IsRunning() {
		t.Error("Scheduler should be running after Start")
	}
	
	sched.Stop()
	time.Sleep(10 * time.Millisecond)
	
	if sched.IsRunning() {
		t.Error("Scheduler should not be running after Stop")
	}
	
	// Verify no processing happens when stopped
	var renderCount atomic.Int32
	fiber := sched.CreateFiber(func() *vdom.VNode {
		renderCount.Add(1)
		return nil
	}, nil)
	
	sched.MarkDirty(fiber)
	time.Sleep(50 * time.Millisecond)
	
	if renderCount.Load() != 0 {
		t.Error("Fiber should not be rendered when scheduler is stopped")
	}
}

func TestFiber_UserData(t *testing.T) {
	sched := NewScheduler()
	fiber := sched.CreateFiber(func() *vdom.VNode { return nil }, nil)
	
	// Test setting and getting user data
	type customData struct {
		value string
	}
	
	data := &customData{value: "test"}
	fiber.SetUserData(data)
	
	retrieved := fiber.GetUserData()
	if retrieved == nil {
		t.Fatal("User data should not be nil")
	}
	
	retrievedData, ok := retrieved.(*customData)
	if !ok {
		t.Fatal("User data type assertion failed")
	}
	
	if retrievedData.value != "test" {
		t.Errorf("Expected user data value 'test', got '%s'", retrievedData.value)
	}
}

func TestScheduler_NilFiber(t *testing.T) {
	sched := NewScheduler()
	
	// Should not panic
	sched.MarkDirty(nil)
	sched.RemoveFiber(nil)
}

func BenchmarkScheduler_MarkDirty(b *testing.B) {
	sched := NewScheduler()
	fiber := sched.CreateFiber(func() *vdom.VNode {
		return vdom.NewText("bench")
	}, nil)
	
	sched.Start()
	defer sched.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sched.MarkDirty(fiber)
	}
}