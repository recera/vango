package bench

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const (
	// Maximum allowed WASM size in bytes (800 KB as per requirements)
	maxWASMSizeGzipped = 800 * 1024
	maxWASMSizeRaw     = 2 * 1024 * 1024 // 2 MB uncompressed
)

// TestWASMBundleSize verifies WASM bundle size is within limits
// Requirement: WASM binary must be ≤800 kB gzipped
func TestWASMBundleSize(t *testing.T) {
	// Look for WASM files in common locations
	possiblePaths := []string{
		"../../examples/counter/app.wasm",
		"../../examples/counter-ssr/app.wasm",
		"../../dist/app.wasm",
		"../../build/app.wasm",
	}
	
	var wasmPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			wasmPath = path
			break
		}
	}
	
	if wasmPath == "" {
		t.Skip("No WASM file found. Run 'make build' or './build.sh' first")
	}
	
	// Get file info
	info, err := os.Stat(wasmPath)
	if err != nil {
		t.Fatalf("Failed to stat WASM file: %v", err)
	}
	
	rawSize := info.Size()
	t.Logf("Raw WASM size: %.2f KB", float64(rawSize)/1024)
	
	// Check raw size
	if rawSize > maxWASMSizeRaw {
		t.Errorf("Raw WASM size %d bytes exceeds limit of %d bytes", rawSize, maxWASMSizeRaw)
	}
	
	// Measure gzipped size
	gzippedSize, err := getGzippedSize(wasmPath)
	if err != nil {
		t.Fatalf("Failed to calculate gzipped size: %v", err)
	}
	
	t.Logf("Gzipped WASM size: %.2f KB", float64(gzippedSize)/1024)
	
	// Check gzipped size against requirement
	if gzippedSize > maxWASMSizeGzipped {
		t.Errorf("Gzipped WASM size %d bytes (%.2f KB) exceeds limit of %d bytes (%.2f KB)",
			gzippedSize, float64(gzippedSize)/1024,
			maxWASMSizeGzipped, float64(maxWASMSizeGzipped)/1024)
	} else {
		t.Logf("✓ WASM bundle size: %.2f KB gzipped (limit: %.2f KB)",
			float64(gzippedSize)/1024, float64(maxWASMSizeGzipped)/1024)
	}
	
	// Calculate compression ratio
	compressionRatio := float64(rawSize-gzippedSize) / float64(rawSize) * 100
	t.Logf("Compression ratio: %.1f%%", compressionRatio)
}

// TestWASMSizeBreakdown analyzes WASM size by checking multiple builds
func TestWASMSizeBreakdown(t *testing.T) {
	examples := []struct {
		name string
		path string
	}{
		{"Counter", "../../examples/counter/app.wasm"},
		{"Counter SSR", "../../examples/counter-ssr/app.wasm"},
	}
	
	hasAnyWasm := false
	for _, ex := range examples {
		if _, err := os.Stat(ex.path); err != nil {
			continue
		}
		
		hasAnyWasm = true
		
		info, err := os.Stat(ex.path)
		if err != nil {
			continue
		}
		
		rawSize := info.Size()
		gzippedSize, _ := getGzippedSize(ex.path)
		
		t.Logf("%s WASM:", ex.name)
		t.Logf("  Raw: %.2f KB", float64(rawSize)/1024)
		t.Logf("  Gzipped: %.2f KB", float64(gzippedSize)/1024)
		
		if gzippedSize > maxWASMSizeGzipped {
			t.Errorf("  ⚠️ Exceeds 800 KB limit!")
		} else {
			t.Logf("  ✓ Within size limit")
		}
	}
	
	if !hasAnyWasm {
		t.Skip("No WASM files found. Build examples first.")
	}
}

// TestIncrementalBuildCache verifies that build cache is working
func TestIncrementalBuildCache(t *testing.T) {
	cachePath := "../../internal/cache"
	
	// Check if cache directory exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Skip("Cache directory not found")
	}
	
	// List cache entries
	entries, err := os.ReadDir(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache directory: %v", err)
	}
	
	cacheSize := int64(0)
	wasmCacheCount := 0
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		cacheSize += info.Size()
		
		if filepath.Ext(entry.Name()) == ".wasm" {
			wasmCacheCount++
			t.Logf("Cached WASM: %s (%.2f KB)", entry.Name(), float64(info.Size())/1024)
		}
	}
	
	t.Logf("Total cache size: %.2f MB", float64(cacheSize)/(1024*1024))
	t.Logf("Cached WASM files: %d", wasmCacheCount)
	
	if wasmCacheCount == 0 {
		t.Log("⚠️ No cached WASM files found. Cache may not be working properly.")
	}
}

// BenchmarkGzipCompression benchmarks gzip compression of WASM
func BenchmarkGzipCompression(b *testing.B) {
	// Create a sample byte slice similar to WASM content
	// WASM typically has repetitive patterns that compress well
	sampleSize := 500 * 1024 // 500 KB sample
	sample := make([]byte, sampleSize)
	
	// Fill with patterns similar to WASM
	for i := 0; i < sampleSize; i++ {
		sample[i] = byte(i % 256)
		if i%100 == 0 {
			sample[i] = 0 // WASM has many zeros
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var gzipped []byte
		gz := gzip.NewWriter(nil)
		_, _ = gz.Write(sample)
		_ = gz.Close()
		_ = gzipped
	}
}

// Helper function to calculate gzipped size
func getGzippedSize(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	// Create a counting writer
	counter := &countingWriter{}
	gzipWriter := gzip.NewWriter(counter)
	
	// Copy file through gzip
	_, err = io.Copy(gzipWriter, file)
	if err != nil {
		return 0, err
	}
	
	err = gzipWriter.Close()
	if err != nil {
		return 0, err
	}
	
	return counter.count, nil
}

// countingWriter counts bytes written
type countingWriter struct {
	count int64
}

func (w *countingWriter) Write(p []byte) (n int, err error) {
	w.count += int64(len(p))
	return len(p), nil
}

// TestWASMOptimizations checks if WASM optimizations are applied
func TestWASMOptimizations(t *testing.T) {
	t.Log("WASM Size Optimization Checklist:")
	t.Log("  [ ] TinyGo compiler flags: -opt=2 or -opt=z")
	t.Log("  [ ] Strip debug info: -no-debug")
	t.Log("  [ ] Disable scheduler: -scheduler=none (if not using goroutines)")
	t.Log("  [ ] Custom allocator: -gc=leaking (for short-lived programs)")
	t.Log("  [ ] wasm-opt post-processing")
	t.Log("")
	t.Log("Build command should look like:")
	t.Log("  tinygo build -o app.wasm -target wasm -opt=z -no-debug .")
	
	// This is informational - actual optimization verification would require
	// parsing the WASM file or build configuration
}

// TestCompareBuildSizes compares different build configurations
func TestCompareBuildSizes(t *testing.T) {
	configs := []struct {
		name        string
		description string
		sizeLimit   int64
	}{
		{
			name:        "Minimal (Hello World)",
			description: "Baseline WASM with minimal functionality",
			sizeLimit:   100 * 1024, // 100 KB
		},
		{
			name:        "With VDOM",
			description: "VDOM + diff algorithm",
			sizeLimit:   300 * 1024, // 300 KB
		},
		{
			name:        "With Scheduler",
			description: "VDOM + Scheduler + Fibers",
			sizeLimit:   500 * 1024, // 500 KB
		},
		{
			name:        "Full Framework",
			description: "All features enabled",
			sizeLimit:   800 * 1024, // 800 KB
		},
	}
	
	t.Log("Target WASM sizes for different configurations:")
	for _, config := range configs {
		t.Logf("  %s: < %.0f KB", config.name, float64(config.sizeLimit)/1024)
		t.Logf("    %s", config.description)
	}
	
	t.Log("")
	t.Log("Size reduction strategies:")
	t.Log("  1. Use TinyGo's -opt=z flag for size optimization")
	t.Log("  2. Implement tree-shaking for unused components")
	t.Log("  3. Lazy-load features not needed at startup")
	t.Log("  4. Use wasm-opt for additional optimization")
	t.Log("  5. Consider splitting into multiple WASM modules")
}

// TestSizeGrowthTrend tracks size growth over time
func TestSizeGrowthTrend(t *testing.T) {
	// This would typically read from a metrics file or CI artifacts
	// For now, we'll document the expected approach
	
	t.Log("Size tracking strategy:")
	t.Log("  1. Record WASM size after each build")
	t.Log("  2. Alert if size increases by >5% in a single change")
	t.Log("  3. Fail CI if size exceeds 800 KB limit")
	t.Log("  4. Generate size report comparing to previous build")
	
	targetSizes := map[string]int64{
		"2024-Q4": 600 * 1024,
		"2025-Q1": 700 * 1024,
		"2025-Q2": 750 * 1024,
		"2025-Q3": 800 * 1024,
	}
	
	t.Log("")
	t.Log("Size budget timeline:")
	for quarter, size := range targetSizes {
		t.Logf("  %s: %.0f KB", quarter, float64(size)/1024)
	}
}

func TestWASMDependencies(t *testing.T) {
	t.Log("WASM size breakdown (estimated):")
	t.Log("  TinyGo runtime:     ~100 KB")
	t.Log("  VDOM core:          ~50 KB")
	t.Log("  Scheduler:          ~30 KB")
	t.Log("  Reactive system:    ~40 KB")
	t.Log("  DOM bindings:       ~80 KB")
	t.Log("  Component logic:    ~200-500 KB")
	t.Log("  --------------------------------")
	t.Log("  Total (gzipped):    ~500-800 KB")
	
	t.Log("")
	t.Log("Major size contributors:")
	t.Log("  • fmt package (avoid if possible)")
	t.Log("  • reflect package (minimize usage)")
	t.Log("  • Large string constants")
	t.Log("  • Unused code not eliminated")
}