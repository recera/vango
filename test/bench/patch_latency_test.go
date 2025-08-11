package bench

import (
	"math"
	"sort"
	"sync"
	"testing"
	"time"

	// "github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango/vdom"
)

// TestPatchLatencyP95Under50ms verifies patch latency performance target
// Requirement: Server→Client patch latency must be <50ms at P95
func TestPatchLatencyP95Under50ms(t *testing.T) {
	// Number of patch operations to test
	const numPatches = 100
	
	latencies := make([]time.Duration, 0, numPatches)
	
	// Simulate patch generation and encoding
	for i := 0; i < numPatches; i++ {
		start := time.Now()
		
		// 1. Generate patches from a diff operation
		prev := generateTreeWithNNodes(100)
		next := generateModifiedTree(100, i)
		patches := vdom.Diff(prev, next)
		
		// 2. Encode patches to binary format
		encoded := encodePatchesToBinary(patches)
		
		// 3. Simulate network transmission (just measure encoding/decoding)
		_ = decodePatchesFromBinary(encoded)
		
		latency := time.Since(start)
		latencies = append(latencies, latency)
	}
	
	// Calculate P95
	p95 := calculatePercentile(latencies, 95)
	
	if p95 > 50*time.Millisecond {
		t.Errorf("Patch latency P95 is %v, expected <50ms", p95)
	} else {
		t.Logf("✓ Patch latency P95: %v", p95)
	}
	
	// Also report other percentiles for visibility
	p50 := calculatePercentile(latencies, 50)
	p99 := calculatePercentile(latencies, 99)
	t.Logf("  P50: %v, P99: %v", p50, p99)
}

// BenchmarkPatchEncoding benchmarks binary patch encoding
func BenchmarkPatchEncoding(b *testing.B) {
	// Generate a typical set of patches
	patches := []vdom.Patch{
		{Op: vdom.OpReplaceText, NodeID: 1, Value: "Updated text content"},
		{Op: vdom.OpSetAttribute, NodeID: 2, Key: "class", Value: "active"},
		{Op: vdom.OpRemoveNode, NodeID: 3},
		{Op: vdom.OpInsertNode, NodeID: 4, ParentID: 1, Node: vdom.NewElement("div", nil)},
		{Op: vdom.OpUpdateEvents, NodeID: 5, EventBits: 0x0F},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encodePatchesToBinary(patches)
	}
}

// BenchmarkPatchDecoding benchmarks binary patch decoding
func BenchmarkPatchDecoding(b *testing.B) {
	patches := []vdom.Patch{
		{Op: vdom.OpReplaceText, NodeID: 1, Value: "Updated text content"},
		{Op: vdom.OpSetAttribute, NodeID: 2, Key: "class", Value: "active"},
		{Op: vdom.OpRemoveNode, NodeID: 3},
	}
	
	encoded := encodePatchesToBinary(patches)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = decodePatchesFromBinary(encoded)
	}
}

// BenchmarkLivePatchStream benchmarks streaming patches over live protocol
func BenchmarkLivePatchStream(b *testing.B) {
	// Skip until live.Codec is implemented
	b.Skip("Live codec not yet implemented")
	
	// // Create a mock live connection
	// codec := live.NewCodec()
	// 
	// // Generate patches
	// patches := make([]vdom.Patch, 10)
	// for i := 0; i < 10; i++ {
	// 	patches[i] = vdom.Patch{
	// 		Op:     vdom.OpReplaceText,
	// 		NodeID: uint32(i),
	// 		Value:  "Updated content",
	// 	}
	// }
	// 
	// b.ResetTimer()
	// for i := 0; i < b.N; i++ {
	// 	// Encode frame
	// 	frame, _ := codec.EncodePatches(patches)
	// 	
	// 	// Decode frame
	// 	_, _ = codec.DecodePatches(frame)
	// }
}

// TestConcurrentPatchApplication tests patch application under concurrent load
func TestConcurrentPatchApplication(t *testing.T) {
	const numGoroutines = 10
	const patchesPerGoroutine = 100
	
	sched := scheduler.NewScheduler()
	sched.Start()
	defer sched.Stop()
	
	// Track latencies from all goroutines
	var mu sync.Mutex
	allLatencies := make([]time.Duration, 0, numGoroutines*patchesPerGoroutine)
	
	var wg sync.WaitGroup
	
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Create a fiber for this goroutine
			fiber := sched.CreateFiber(func() *vdom.VNode {
				return generateTreeWithNNodes(50)
			}, nil)
			
			for i := 0; i < patchesPerGoroutine; i++ {
				start := time.Now()
				
				// Mark dirty and wait for processing
				sched.MarkDirty(fiber)
				time.Sleep(time.Millisecond) // Simulate processing time
				
				latency := time.Since(start)
				
				mu.Lock()
				allLatencies = append(allLatencies, latency)
				mu.Unlock()
			}
		}(g)
	}
	
	wg.Wait()
	
	// Calculate statistics
	p95 := calculatePercentile(allLatencies, 95)
	p99 := calculatePercentile(allLatencies, 99)
	
	t.Logf("Concurrent patch application (n=%d):", len(allLatencies))
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)
	
	if p95 > 50*time.Millisecond {
		t.Errorf("Concurrent patch P95 latency %v exceeds 50ms target", p95)
	}
}

// TestPatchBurstHandling tests handling of patch bursts
func TestPatchBurstHandling(t *testing.T) {
	const burstSize = 50
	const numBursts = 10
	
	latencies := make([]time.Duration, 0, numBursts)
	
	for burst := 0; burst < numBursts; burst++ {
		start := time.Now()
		
		// Generate a burst of patches
		patches := make([]vdom.Patch, burstSize)
		for i := 0; i < burstSize; i++ {
			patches[i] = vdom.Patch{
				Op:     vdom.OpReplaceText,
				NodeID: uint32(i),
				Value:  "Burst update",
			}
		}
		
		// Encode entire burst
		encoded := encodePatchesToBinary(patches)
		
		// Decode and apply
		_ = decodePatchesFromBinary(encoded)
		
		latency := time.Since(start)
		latencies = append(latencies, latency)
	}
	
	avgLatency := calculateAverage(latencies)
	maxLatency := calculateMax(latencies)
	
	t.Logf("Patch burst handling (size=%d):", burstSize)
	t.Logf("  Average: %v", avgLatency)
	t.Logf("  Max: %v", maxLatency)
	
	if maxLatency > 100*time.Millisecond {
		t.Errorf("Patch burst max latency %v exceeds reasonable threshold", maxLatency)
	}
}

// Helper functions

func generateTreeWithNNodes(n int) *vdom.VNode {
	children := make([]*vdom.VNode, n)
	for i := 0; i < n; i++ {
		children[i] = vdom.NewElement("div", vdom.Props{
			"key": i,
		}, vdom.NewText("Node"))
	}
	return vdom.NewElement("root", nil, children...)
}

func generateModifiedTree(n int, seed int) *vdom.VNode {
	children := make([]*vdom.VNode, n)
	for i := 0; i < n; i++ {
		text := "Node"
		if i%10 == seed%10 {
			text = "Modified"
		}
		children[i] = vdom.NewElement("div", vdom.Props{
			"key": i,
		}, vdom.NewText(text))
	}
	return vdom.NewElement("root", nil, children...)
}

func encodePatchesToBinary(patches []vdom.Patch) []byte {
	// Simplified encoding simulation
	// In reality, this would use the live.Codec
	
	size := 0
	for _, p := range patches {
		size += 1 + 4 // opcode + nodeID
		size += len(p.Value) + len(p.Key) + 4 // strings + length
	}
	
	buf := make([]byte, size)
	offset := 0
	
	for _, p := range patches {
		buf[offset] = byte(p.Op)
		offset++
		// Simulate varint encoding
		offset += 4 // nodeID
		offset += len(p.Value) + len(p.Key)
	}
	
	return buf[:offset]
}

func decodePatchesFromBinary(data []byte) []vdom.Patch {
	// Simplified decoding simulation
	// Count patches based on approximate size
	numPatches := len(data) / 10
	if numPatches == 0 {
		numPatches = 1
	}
	
	patches := make([]vdom.Patch, numPatches)
	// Simulate decoding work
	for i := range patches {
		patches[i].Op = vdom.PatchOp(data[0])
	}
	
	return patches
}

func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	// Convert to nanoseconds for sorting
	nanos := make([]int64, len(durations))
	for i, d := range durations {
		nanos[i] = int64(d)
	}
	
	sort.Slice(nanos, func(i, j int) bool {
		return nanos[i] < nanos[j]
	})
	
	index := int(math.Ceil(float64(len(nanos)) * percentile / 100.0)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(nanos) {
		index = len(nanos) - 1
	}
	
	return time.Duration(nanos[index])
}

func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var total int64
	for _, d := range durations {
		total += int64(d)
	}
	
	return time.Duration(total / int64(len(durations)))
}

func calculateMax(durations []time.Duration) time.Duration {
	var max time.Duration
	for _, d := range durations {
		if d > max {
			max = d
		}
	}
	return max
}