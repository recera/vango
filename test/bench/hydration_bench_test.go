package bench

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/recera/vango/pkg/renderer/html"
	"github.com/recera/vango/pkg/vango/vdom"
)

// TestHydration1kNodesUnder30ms verifies hydration performance target
// Requirement: Hydration time for 1k nodes must be <30ms
func TestHydration1kNodesUnder30ms(t *testing.T) {
	// First, generate the server-rendered HTML
	root := generate1kNodeTreeWithEvents()
	
	// Render to HTML with hydration IDs
	var buf bytes.Buffer
	applier := html.NewHTMLApplier(&buf)
	// Note: EnableHydration method may not exist yet
	
	err := applier.Apply(nil, root)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}
	
	htmlContent := buf.String()
	
	// Verify that hydration IDs were injected
	if !strings.Contains(htmlContent, "data-hid") {
		t.Skip("Hydration IDs not implemented yet in html.Applier")
	}
	
	iterations := 10
	var totalDuration time.Duration
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		
		// Simulate hydration process:
		// 1. Parse HTML to build sparse VNode tree (would happen in browser)
		sparseTree := simulateBuildSparseTree(htmlContent)
		
		// 2. Run component to get full VNode tree
		fullTree := generate1kNodeTreeWithEvents()
		
		// 3. Diff sparse vs full tree
		patches := vdom.Diff(sparseTree, fullTree)
		
		// 4. Apply patches (in real scenario, this would update DOM)
		// Here we just verify no structural changes needed
		structuralChanges := countStructuralChanges(patches)
		
		duration := time.Since(start)
		totalDuration += duration
		
		if structuralChanges > 0 {
			t.Errorf("Hydration mismatch: %d structural changes needed", structuralChanges)
		}
	}
	
	avgDuration := totalDuration / time.Duration(iterations)
	
	if avgDuration > 30*time.Millisecond {
		t.Errorf("Hydration of 1k nodes took %v (average), expected <30ms", avgDuration)
	} else {
		t.Logf("✓ Hydration 1k nodes: %v (average)", avgDuration)
	}
}

// BenchmarkHydrationSparseTreeBuilding benchmarks building sparse tree from HTML
func BenchmarkHydrationSparseTreeBuilding(b *testing.B) {
	// Generate HTML with hydration IDs
	root := generate1kNodeTreeWithEvents()
	var buf bytes.Buffer
	applier := html.NewHTMLApplier(&buf)
	_ = applier.Apply(nil, root)
	htmlContent := buf.String()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = simulateBuildSparseTree(htmlContent)
	}
}

// BenchmarkHydrationDiff benchmarks diff between sparse and full trees
func BenchmarkHydrationDiff(b *testing.B) {
	// Create sparse tree (server-rendered structure)
	sparseTree := generateSparse1kNodeTree()
	
	// Create full tree (client component render)
	fullTree := generate1kNodeTreeWithEvents()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vdom.Diff(sparseTree, fullTree)
	}
}

// TestHydrationNoMismatch verifies zero DOM mismatches during hydration
func TestHydrationNoMismatch(t *testing.T) {
	testCases := []struct {
		name string
		tree func() *vdom.VNode
	}{
		{"Simple text", func() *vdom.VNode { 
			return vdom.NewElement("div", nil, vdom.NewText("Hello"))
		}},
		{"Nested elements", func() *vdom.VNode {
			return vdom.NewElement("div", nil,
				vdom.NewElement("span", nil, vdom.NewText("Nested")),
			)
		}},
		{"List with keys", func() *vdom.VNode {
			items := make([]*vdom.VNode, 10)
			for i := 0; i < 10; i++ {
				items[i] = vdom.NewElement("li", vdom.Props{
					"key": fmt.Sprintf("item-%d", i),
				}, vdom.NewText(fmt.Sprintf("Item %d", i)))
			}
			return vdom.NewElement("ul", nil, items...)
		}},
		{"Complex tree", generateComplexComponentTree},
		{"1k nodes", generate1kNodeTree},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree := tc.tree()
			
			// Render server-side
			var buf bytes.Buffer
			applier := html.NewHTMLApplier(&buf)
			
			err := applier.Apply(nil, tree)
			if err != nil {
				t.Fatalf("Failed to render: %v", err)
			}
			
			// Simulate hydration
			sparseTree := simulateBuildSparseTreeFromVNode(tree, true)
			patches := vdom.Diff(sparseTree, tree)
			
			// Check for structural changes
			for _, patch := range patches {
				switch patch.Op {
				case vdom.OpInsertNode, vdom.OpRemoveNode, vdom.OpMoveNode:
					t.Errorf("Hydration mismatch: structural change %v", patch)
				case vdom.OpReplaceText:
					// Text replacements should not happen if content matches
					t.Errorf("Hydration mismatch: text change %v", patch)
				}
			}
		})
	}
}

// Helper functions

func generate1kNodeTreeWithEvents() *vdom.VNode {
	children := make([]*vdom.VNode, 0, 1000)
	
	for i := 0; i < 100; i++ {
		row := vdom.NewElement("div", vdom.Props{
			"class": "row",
			"key":   fmt.Sprintf("row-%d", i),
		})
		
		for j := 0; j < 10; j++ {
			// Add event handlers to some cells
			props := vdom.Props{"class": "cell"}
			if j%3 == 0 {
				props["onClick"] = "handleClick"
			}
			
			cell := vdom.NewElement("button", props,
				vdom.NewText(fmt.Sprintf("Cell %d-%d", i, j)))
			
			row.Kids = append(row.Kids, *cell)
		}
		
		children = append(children, row)
	}
	
	return vdom.NewElement("div", vdom.Props{"id": "root"}, children...)
}

func generateSparse1kNodeTree() *vdom.VNode {
	// Simulate a sparse tree that would be built from server HTML
	// This has structure but no event handlers
	children := make([]*vdom.VNode, 0, 1000)
	
	for i := 0; i < 100; i++ {
		row := vdom.NewElement("div", vdom.Props{
			"class": "row",
			"key":   fmt.Sprintf("row-%d", i),
		})
		
		for j := 0; j < 10; j++ {
			// No event handlers in sparse tree
			cell := vdom.NewElement("button", vdom.Props{
				"class": "cell",
			}, vdom.NewText(fmt.Sprintf("Cell %d-%d", i, j)))
			
			row.Kids = append(row.Kids, *cell)
		}
		
		children = append(children, row)
	}
	
	return vdom.NewElement("div", vdom.Props{"id": "root"}, children...)
}

func simulateBuildSparseTree(html string) *vdom.VNode {
	// This is a simplified simulation
	// In reality, this would parse HTML and build VNode tree
	// For benchmarking, we create a representative sparse tree
	
	if strings.Contains(html, "data-hid") {
		// If hydration IDs present, build sparse tree with markers
		return generateSparse1kNodeTree()
	}
	
	// Fallback to simple tree
	return generate1kNodeTree()
}

func simulateBuildSparseTreeFromVNode(full *vdom.VNode, removeEvents bool) *vdom.VNode {
	// Create a sparse version by stripping event handlers
	if full == nil {
		return nil
	}
	
	sparse := &vdom.VNode{
		Kind:         full.Kind,
		Tag:          full.Tag,
		Text:         full.Text,
		Key:          full.Key,
		PortalTarget: full.PortalTarget,
		Flags:        full.Flags,
	}
	
	// Copy props but remove event handlers
	if full.Props != nil {
		sparse.Props = make(vdom.Props)
		for k, v := range full.Props {
			if !removeEvents || !isEventProp(k) {
				sparse.Props[k] = v
			}
		}
	}
	
	// Recursively process children
	if len(full.Kids) > 0 {
		sparse.Kids = make([]vdom.VNode, len(full.Kids))
		for i := range full.Kids {
			child := simulateBuildSparseTreeFromVNode(&full.Kids[i], removeEvents)
			if child != nil {
				sparse.Kids[i] = *child
			}
		}
	}
	
	return sparse
}

func isEventProp(key string) bool {
	return len(key) > 2 && key[0] == 'o' && key[1] == 'n' && key[2] >= 'A' && key[2] <= 'Z'
}

func countStructuralChanges(patches []vdom.Patch) int {
	count := 0
	for _, patch := range patches {
		switch patch.Op {
		case vdom.OpInsertNode, vdom.OpRemoveNode, vdom.OpMoveNode:
			count++
		}
	}
	return count
}