package bench

//phase 0 tests

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/renderer/html"
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango/vdom"
)

// BenchmarkRender1kNodes tests rendering performance for 1000 nodes
// Requirement: Must complete in <30ms per P-0-core-wasm.md
func BenchmarkRender1kNodes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := generate1kNodeTree()

		// Measure diff time for subsequent render
		prev := generate1kNodeTree()
		patches := vdom.Diff(prev, root)

		if len(patches) > 0 {
			b.Errorf("Expected no patches for identical trees, got %d", len(patches))
		}
	}
}

// TestRender1kNodesUnder30ms verifies the 30ms performance target
func TestRender1kNodesUnder30ms(t *testing.T) {
	iterations := 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Generate tree
		root := generate1kNodeTree()

		// Simulate a diff operation
		prev := generateSlightlyDifferent1kNodeTree()
		_ = vdom.Diff(prev, root)

		duration := time.Since(start)
		totalDuration += duration
	}

	avgDuration := totalDuration / time.Duration(iterations)

	if avgDuration > 30*time.Millisecond {
		t.Errorf("Render 1k nodes took %v (average), expected <30ms", avgDuration)
	} else {
		t.Logf("✓ Render 1k nodes: %v (average)", avgDuration)
	}
}

// BenchmarkDiff1kNodes benchmarks the diff algorithm with 1k nodes
func BenchmarkDiff1kNodes(b *testing.B) {
	prev := generate1kNodeTree()
	next := generateSlightlyDifferent1kNodeTree()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vdom.Diff(prev, next)
	}
}

// BenchmarkDiffLargeList benchmarks diff with a large list of items
func BenchmarkDiffLargeList(b *testing.B) {
	prev := generateListWithNItems(1000)
	next := generateShuffledList(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vdom.Diff(prev, next)
	}
}

// BenchmarkSSRStreaming benchmarks server-side rendering performance
func BenchmarkSSRStreaming(b *testing.B) {
	root := generate1kNodeTree()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		applier := html.NewHTMLApplier(&buf)
		_ = applier.Apply(nil, root)
	}
}

// TestSSRFirstByteUnder50ms verifies SSR first byte target
func TestSSRFirstByteUnder50ms(t *testing.T) {
	iterations := 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create a complex component tree
		root := generateComplexComponentTree()

		// Apply SSR
		var buf bytes.Buffer
		applier := html.NewHTMLApplier(&buf)
		_ = applier.Apply(nil, root)

		duration := time.Since(start)
		totalDuration += duration
	}

	avgDuration := totalDuration / time.Duration(iterations)

	if avgDuration > 50*time.Millisecond {
		t.Errorf("SSR first byte took %v (average), expected <50ms", avgDuration)
	} else {
		t.Logf("✓ SSR first byte: %v (average)", avgDuration)
	}
}

// BenchmarkSchedulerWith100Fibers tests scheduler performance with many fibers
func BenchmarkSchedulerWith100Fibers(b *testing.B) {
	sched := scheduler.NewScheduler()

	// Create 100 fibers
	fibers := make([]*scheduler.Fiber, 100)
	for i := 0; i < 100; i++ {
		idx := i
		fiber := sched.CreateFiber(func() *vdom.VNode {
			return vdom.NewElement("div", nil,
				vdom.NewText(fmt.Sprintf("Fiber %d", idx)))
		}, nil)
		fibers[i] = fiber
	}

	sched.Start()
	defer sched.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mark all fibers dirty
		for _, f := range fibers {
			sched.MarkDirty(f)
		}

		// Wait for processing
		time.Sleep(10 * time.Millisecond)
	}
}

// BenchmarkReactiveSignalPropagation tests signal propagation performance
func BenchmarkReactiveSignalPropagation(b *testing.B) {
	sched := scheduler.NewScheduler()

	// Create a chain of computed signals
	base := reactive.NewState(0, sched)
	computed1 := reactive.NewComputed(func() int {
		return base.Get() * 2
	}, sched)
	computed2 := reactive.NewComputed(func() int {
		return computed1.Get() + 10
	}, sched)
	computed3 := reactive.NewComputed(func() int {
		return computed2.Get() * 3
	}, sched)

	// Create fibers that depend on the signals
	for i := 0; i < 10; i++ {
		sched.CreateFiber(func() *vdom.VNode {
			value := computed3.Get()
			return vdom.NewText(fmt.Sprintf("%d", value))
		}, nil)
	}

	sched.Start()
	defer sched.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		base.Set(i)
		time.Sleep(time.Millisecond) // Allow propagation
	}
}

// Helper functions to generate test trees

func generate1kNodeTree() *vdom.VNode {
	children := make([]*vdom.VNode, 0, 1000)

	for i := 0; i < 100; i++ {
		row := vdom.NewElement("div", vdom.Props{
			"class": "row",
			"key":   fmt.Sprintf("row-%d", i),
		})

		for j := 0; j < 10; j++ {
			cell := vdom.NewElement("span", vdom.Props{
				"class": "cell",
			}, vdom.NewText(fmt.Sprintf("Cell %d-%d", i, j)))

			row.Kids = append(row.Kids, *cell)
		}

		children = append(children, row)
	}

	return vdom.NewElement("div", vdom.Props{"id": "root"}, children...)
}

func generateSlightlyDifferent1kNodeTree() *vdom.VNode {
	children := make([]*vdom.VNode, 0, 1000)

	for i := 0; i < 100; i++ {
		row := vdom.NewElement("div", vdom.Props{
			"class": "row",
			"key":   fmt.Sprintf("row-%d", i),
		})

		for j := 0; j < 10; j++ {
			// Change some cells
			text := fmt.Sprintf("Cell %d-%d", i, j)
			if i%10 == 0 && j%3 == 0 {
				text = fmt.Sprintf("Modified %d-%d", i, j)
			}

			cell := vdom.NewElement("span", vdom.Props{
				"class": "cell",
			}, vdom.NewText(text))

			row.Kids = append(row.Kids, *cell)
		}

		children = append(children, row)
	}

	return vdom.NewElement("div", vdom.Props{"id": "root"}, children...)
}

func generateListWithNItems(n int) *vdom.VNode {
	items := make([]*vdom.VNode, n)
	for i := 0; i < n; i++ {
		items[i] = vdom.NewElement("li", vdom.Props{
			"key": fmt.Sprintf("item-%d", i),
		}, vdom.NewText(fmt.Sprintf("Item %d", i)))
	}

	return vdom.NewElement("ul", nil, items...)
}

func generateShuffledList(n int) *vdom.VNode {
	items := make([]*vdom.VNode, n)

	// Generate items in a different order (simple shuffle)
	for i := 0; i < n; i++ {
		idx := (i*7 + 3) % n // Simple deterministic shuffle
		items[i] = vdom.NewElement("li", vdom.Props{
			"key": fmt.Sprintf("item-%d", idx),
		}, vdom.NewText(fmt.Sprintf("Item %d", idx)))
	}

	return vdom.NewElement("ul", nil, items...)
}

func generateComplexComponentTree() *vdom.VNode {
	// Simulate a realistic component tree with nested components
	header := vdom.NewElement("header", nil,
		vdom.NewElement("nav", nil,
			vdom.NewElement("ul", nil,
				vdom.NewElement("li", nil, vdom.NewText("Home")),
				vdom.NewElement("li", nil, vdom.NewText("About")),
				vdom.NewElement("li", nil, vdom.NewText("Contact")),
			),
		),
	)

	sidebar := vdom.NewElement("aside", nil,
		vdom.NewElement("h3", nil, vdom.NewText("Categories")),
		vdom.NewElement("ul", nil,
			vdom.NewElement("li", nil, vdom.NewText("Technology")),
			vdom.NewElement("li", nil, vdom.NewText("Science")),
			vdom.NewElement("li", nil, vdom.NewText("Business")),
		),
	)

	// Main content with articles
	articles := make([]*vdom.VNode, 20)
	for i := 0; i < 20; i++ {
		articles[i] = vdom.NewElement("article", vdom.Props{"key": fmt.Sprintf("article-%d", i)},
			vdom.NewElement("h2", nil, vdom.NewText(fmt.Sprintf("Article %d", i))),
			vdom.NewElement("p", nil, vdom.NewText("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")),
			vdom.NewElement("footer", nil,
				vdom.NewElement("span", nil, vdom.NewText("Author Name")),
				vdom.NewElement("time", nil, vdom.NewText("2025-01-15")),
			),
		)
	}

	main := vdom.NewElement("main", nil, articles...)

	footer := vdom.NewElement("footer", nil,
		vdom.NewElement("p", nil, vdom.NewText("© 2025 Vango Framework")),
	)

	return vdom.NewElement("div", vdom.Props{"id": "app"},
		header,
		vdom.NewElement("div", vdom.Props{"class": "container"},
			sidebar,
			main,
		),
		footer,
	)
}
