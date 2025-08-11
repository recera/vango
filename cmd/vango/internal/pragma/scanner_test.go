package pragma

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPragmaScanner_ParsePragma(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *PragmaType
		options  []string
	}{
		{
			name:     "server pragma",
			input:    "//vango:server",
			expected: (*PragmaType)(stringPtr("server")),
			options:  []string{},
		},
		{
			name:     "client pragma",
			input:    "//vango:client",
			expected: (*PragmaType)(stringPtr("client")),
			options:  []string{},
		},
		{
			name:     "universal pragma",
			input:    "//vango:universal",
			expected: (*PragmaType)(stringPtr("universal")),
			options:  []string{},
		},
		{
			name:     "pragma with options",
			input:    "//vango:server noCache async",
			expected: (*PragmaType)(stringPtr("server")),
			options:  []string{"noCache", "async"},
		},
		{
			name:     "pragma with spaces",
			input:    "// vango:client",
			expected: (*PragmaType)(stringPtr("client")),
			options:  []string{},
		},
		{
			name:     "invalid pragma",
			input:    "//vango:invalid",
			expected: nil,
			options:  nil,
		},
		{
			name:     "non-vango comment",
			input:    "// regular comment",
			expected: nil,
			options:  nil,
		},
	}
	
	scanner, _ := NewScanner(ScannerConfig{})
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pragma := scanner.parsePragma(tt.input, token.Position{})
			
			if tt.expected == nil {
				if pragma != nil {
					t.Errorf("Expected nil pragma, got %v", pragma)
				}
				return
			}
			
			if pragma == nil {
				t.Errorf("Expected pragma type %v, got nil", *tt.expected)
				return
			}
			
			if string(pragma.Type) != string(*tt.expected) {
				t.Errorf("Expected pragma type %v, got %v", *tt.expected, pragma.Type)
			}
			
			if len(pragma.Options) != len(tt.options) {
				t.Errorf("Expected %d options, got %d", len(tt.options), len(pragma.Options))
			}
			
			for i, opt := range tt.options {
				if i >= len(pragma.Options) || pragma.Options[i] != opt {
					t.Errorf("Expected option %s at index %d, got %v", opt, i, pragma.Options)
				}
			}
		})
	}
}

func TestPragmaScanner_ScanFile(t *testing.T) {
	// Create test files
	tmpDir := t.TempDir()
	
	testFiles := map[string]string{
		"server.go": `//vango:server
package main

func ServerOnly() {
	println("Server code")
}`,
		"client.go": `//vango:client
package main

func ClientOnly() {
	println("Client code")
}`,
		"universal.go": `//vango:universal
package main

func Universal() {
	println("Universal code")
}`,
		"mixed.go": `package main

//vango:server
func ServerFunc() {}

//vango:client
func ClientFunc() {}`,
		"no_pragma.go": `package main

func Regular() {
	println("No pragma")
}`,
	}
	
	// Write test files
	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", name, err)
		}
	}
	
	// Create scanner
	scanner, err := NewScanner(ScannerConfig{
		RootDir: tmpDir,
		Verbose: false,
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	
	// Scan files
	manifest, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}
	
	// Verify results
	// Note: mixed.go has both pragmas but only the first one (server) counts
	if len(manifest.ServerFiles) != 2 { // server.go and mixed.go
		t.Errorf("Expected 2 server files, got %d: %v", len(manifest.ServerFiles), manifest.ServerFiles)
	}
	
	if len(manifest.ClientFiles) != 1 { // client.go only
		t.Errorf("Expected 1 client file, got %d: %v", len(manifest.ClientFiles), manifest.ClientFiles)
	}
	
	if len(manifest.SharedFiles) != 1 { // universal.go
		t.Errorf("Expected 1 shared file, got %d: %v", len(manifest.SharedFiles), manifest.SharedFiles)
	}
}

func TestPragmaScanner_InjectBuildTags(t *testing.T) {
	tmpDir := t.TempDir()
	
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := `//vango:server
package main

func ServerFunc() {
	println("Server")
}`
	
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Create scanner with auto-inject enabled
	scanner, err := NewScanner(ScannerConfig{
		RootDir:        tmpDir,
		AutoInjectTags: true,
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	
	// Scan and inject
	_, err = scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan and inject: %v", err)
	}
	
	// Read modified file
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	
	// Check for build tag
	if !strings.Contains(string(content), "//go:build vango_server") {
		t.Errorf("Build tag not injected. Content:\n%s", string(content))
	}
	
	// Check that package declaration is preserved
	if !strings.Contains(string(content), "package main") {
		t.Errorf("Package declaration lost. Content:\n%s", string(content))
	}
	
	// Check that function is preserved
	if !strings.Contains(string(content), "func ServerFunc()") {
		t.Errorf("Function lost. Content:\n%s", string(content))
	}
}

func TestPragmaScanner_ExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create directory structure
	dirs := []string{
		"src",
		"vendor/github.com/example",
		"testdata",
		"internal",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}
	
	// Create test files
	files := map[string]string{
		"src/main.go":                       "//vango:server\npackage main",
		"vendor/github.com/example/lib.go":  "//vango:client\npackage lib",
		"testdata/test.go":                  "//vango:server\npackage test",
		"internal/helper.go":                "//vango:client\npackage internal",
		"src/main_test.go":                  "//vango:server\npackage main",
	}
	
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}
	
	// Create scanner with exclude patterns
	scanner, err := NewScanner(ScannerConfig{
		RootDir: tmpDir,
		ExcludePatterns: []string{
			"**/vendor/**",
			"**/testdata/**",
			"**/*_test.go",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	
	// Scan
	manifest, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}
	
	// Verify only expected files were processed
	totalFiles := len(manifest.ServerFiles) + len(manifest.ClientFiles) + len(manifest.SharedFiles)
	if totalFiles != 2 { // Only src/main.go and internal/helper.go
		t.Errorf("Expected 2 files to be processed, got %d", totalFiles)
		t.Logf("Server files: %v", manifest.ServerFiles)
		t.Logf("Client files: %v", manifest.ClientFiles)
		t.Logf("Shared files: %v", manifest.SharedFiles)
	}
	
	// Check that vendor and testdata were excluded
	for _, file := range append(manifest.ServerFiles, manifest.ClientFiles...) {
		if strings.Contains(file, "vendor") {
			t.Errorf("Vendor file was not excluded: %s", file)
		}
		if strings.Contains(file, "testdata") {
			t.Errorf("Testdata file was not excluded: %s", file)
		}
		if strings.HasSuffix(file, "_test.go") {
			t.Errorf("Test file was not excluded: %s", file)
		}
	}
}

func TestPragmaScanner_ManifestSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a manifest
	original := &Manifest{
		Version:     "1.0",
		ServerFiles: []string{"server.go", "api.go"},
		ClientFiles: []string{"client.go", "ui.go"},
		SharedFiles: []string{"shared.go"},
		Pragmas: map[string]Pragma{
			"server.go": {Type: PragmaServer, FilePath: "server.go", Line: 1},
			"client.go": {Type: PragmaClient, FilePath: "client.go", Line: 1},
		},
		Dependencies: map[string]string{
			"server.go": "abc123",
			"client.go": "def456",
		},
		Hash: "manifest123",
	}
	
	// Save manifest
	manifestPath := filepath.Join(tmpDir, "vango.manifest.json")
	if err := original.Save(manifestPath); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}
	
	// Load manifest
	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}
	
	// Verify loaded data
	if loaded.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", loaded.Version, original.Version)
	}
	
	if len(loaded.ServerFiles) != len(original.ServerFiles) {
		t.Errorf("ServerFiles count mismatch: got %d, want %d", 
			len(loaded.ServerFiles), len(original.ServerFiles))
	}
	
	if len(loaded.ClientFiles) != len(original.ClientFiles) {
		t.Errorf("ClientFiles count mismatch: got %d, want %d",
			len(loaded.ClientFiles), len(original.ClientFiles))
	}
	
	if loaded.Hash != original.Hash {
		t.Errorf("Hash mismatch: got %s, want %s", loaded.Hash, original.Hash)
	}
}

func TestPragmaScanner_PreservePragmas(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	
	originalContent := `//vango:server
// This is a doc comment
package main

func ServerFunc() {}`
	
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Test with PreservePragmas = true
	scanner, _ := NewScanner(ScannerConfig{
		RootDir:         tmpDir,
		AutoInjectTags:  true,
		PreservePragmas: true,
	})
	
	scanner.Scan()
	
	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "//vango:server") {
		t.Errorf("Pragma was not preserved when PreservePragmas=true")
	}
	
	// Reset file
	os.WriteFile(testFile, []byte(originalContent), 0644)
	
	// Test with PreservePragmas = false
	scanner, _ = NewScanner(ScannerConfig{
		RootDir:         tmpDir,
		AutoInjectTags:  true,
		PreservePragmas: false,
	})
	
	scanner.Scan()
	
	content, _ = os.ReadFile(testFile)
	if strings.Count(string(content), "//vango:server") > 0 {
		// Should be removed since pragma was replaced with build tag
		t.Errorf("Pragma was not removed when PreservePragmas=false")
	}
	
	// But build tag should be there
	if !strings.Contains(string(content), "//go:build vango_server") {
		t.Errorf("Build tag was not added")
	}
}

func TestPragmaScanner_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	
	testFile := filepath.Join(tmpDir, "test.go")
	content := `//vango:server
package main

func Server() {}`
	
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// First scan with cache
	scanner1, _ := NewScanner(ScannerConfig{
		RootDir:  tmpDir,
		CacheDir: cacheDir,
	})
	
	manifest1, err := scanner1.Scan()
	if err != nil {
		t.Fatalf("First scan failed: %v", err)
	}
	
	// Wait for async cache save
	time.Sleep(100 * time.Millisecond)
	
	// Second scan should use cache
	scanner2, _ := NewScanner(ScannerConfig{
		RootDir:  tmpDir,
		CacheDir: cacheDir,
	})
	
	manifest2, err := scanner2.Scan()
	if err != nil {
		t.Fatalf("Second scan failed: %v", err)
	}
	
	// Manifests should be identical
	if manifest1.Hash != manifest2.Hash {
		t.Errorf("Manifest hashes don't match: %s vs %s", manifest1.Hash, manifest2.Hash)
	}
	
	// Verify cache file exists
	cacheFile := filepath.Join(cacheDir, "pragma-cache.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Errorf("Cache file was not created")
	}
}

func TestPragmaScanner_ConcurrentScanning(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create multiple test files
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("file%d.go", i)
		content := fmt.Sprintf(`//vango:server
package main

func Func%d() {}`, i)
		
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}
	
	// Run multiple scans concurrently
	errors := make(chan error, 5)
	manifests := make(chan *Manifest, 5)
	
	for i := 0; i < 5; i++ {
		go func() {
			scanner, err := NewScanner(ScannerConfig{
				RootDir: tmpDir,
			})
			if err != nil {
				errors <- err
				return
			}
			
			manifest, err := scanner.Scan()
			if err != nil {
				errors <- err
				return
			}
			
			manifests <- manifest
			errors <- nil
		}()
	}
	
	// Collect results
	var manifestList []*Manifest
	for i := 0; i < 5; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Concurrent scan failed: %v", err)
		} else {
			select {
			case m := <-manifests:
				manifestList = append(manifestList, m)
			default:
			}
		}
	}
	
	// All manifests should be identical
	if len(manifestList) > 1 {
		firstHash := manifestList[0].Hash
		for i, m := range manifestList[1:] {
			if m.Hash != firstHash {
				t.Errorf("Manifest %d has different hash: %s vs %s", i+1, m.Hash, firstHash)
			}
		}
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}