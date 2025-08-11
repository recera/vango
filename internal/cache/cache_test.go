package cache

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCache_GetPut(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:     tmpDir,
		MaxSize: 1 << 20, // 1 MB
		MaxAge:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	// Test Put
	key := "test-key"
	data := []byte("test data content")
	
	if err := cache.Put(key, data); err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}
	
	// Test Get
	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Data not found in cache")
	}
	
	if !bytes.Equal(retrieved, data) {
		t.Errorf("Retrieved data doesn't match: got %s, want %s", retrieved, data)
	}
	
	// Test cache hit
	stats := cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	
	// Test Get non-existent key
	_, found = cache.Get("non-existent")
	if found {
		t.Error("Found non-existent key")
	}
	
	// Test cache miss
	stats = cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestCache_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	key := "delete-test"
	data := []byte("data to delete")
	
	// Put data
	if err := cache.Put(key, data); err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}
	
	// Verify it exists
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Data not found after put")
	}
	
	// Delete
	if err := cache.Delete(key); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}
	
	// Verify it's gone
	_, found = cache.Get(key)
	if found {
		t.Error("Data found after delete")
	}
	
	// Delete again should not error
	if err := cache.Delete(key); err != nil {
		t.Errorf("Delete of non-existent key failed: %v", err)
	}
}

func TestCache_Eviction_LRU(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:      tmpDir,
		MaxSize:  100, // Very small cache
		Strategy: LRU,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	// Add entries that will exceed cache size
	data1 := bytes.Repeat([]byte("a"), 40)
	data2 := bytes.Repeat([]byte("b"), 40)
	data3 := bytes.Repeat([]byte("c"), 40)
	
	cache.Put("key1", data1)
	time.Sleep(10 * time.Millisecond)
	cache.Put("key2", data2)
	time.Sleep(10 * time.Millisecond)
	
	// Access key1 to make it more recent than key2
	cache.Get("key1")
	time.Sleep(10 * time.Millisecond)
	
	// This should evict key2 (least recently used)
	cache.Put("key3", data3)
	
	// key1 and key3 should exist
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	_, found3 := cache.Get("key3")
	
	if !found1 {
		t.Error("key1 was evicted but shouldn't have been")
	}
	if found2 {
		t.Error("key2 was not evicted but should have been")
	}
	if !found3 {
		t.Error("key3 not found")
	}
	
	stats := cache.GetStats()
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}

func TestCache_Eviction_LFU(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:      tmpDir,
		MaxSize:  100,
		Strategy: LFU,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	data1 := bytes.Repeat([]byte("a"), 40)
	data2 := bytes.Repeat([]byte("b"), 40)
	data3 := bytes.Repeat([]byte("c"), 40)
	
	cache.Put("key1", data1)
	cache.Put("key2", data2)
	
	// Access key1 multiple times to increase frequency
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key1")
	
	// Access key2 only once
	cache.Get("key2")
	
	// This should evict key2 (least frequently used)
	cache.Put("key3", data3)
	
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	_, found3 := cache.Get("key3")
	
	if !found1 {
		t.Error("key1 was evicted but shouldn't have been")
	}
	if found2 {
		t.Error("key2 was not evicted but should have been")
	}
	if !found3 {
		t.Error("key3 not found")
	}
}

func TestCache_Eviction_FIFO(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:      tmpDir,
		MaxSize:  100,
		Strategy: FIFO,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	data1 := bytes.Repeat([]byte("a"), 40)
	data2 := bytes.Repeat([]byte("b"), 40)
	data3 := bytes.Repeat([]byte("c"), 40)
	
	cache.Put("key1", data1)
	time.Sleep(10 * time.Millisecond)
	cache.Put("key2", data2)
	time.Sleep(10 * time.Millisecond)
	
	// Access patterns don't matter for FIFO
	cache.Get("key1")
	cache.Get("key1")
	
	// This should evict key1 (first in)
	cache.Put("key3", data3)
	
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	_, found3 := cache.Get("key3")
	
	if found1 {
		t.Error("key1 was not evicted but should have been")
	}
	if !found2 {
		t.Error("key2 was evicted but shouldn't have been")
	}
	if !found3 {
		t.Error("key3 not found")
	}
}

func TestCache_Expiration(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:    tmpDir,
		MaxAge: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	key := "expiring-key"
	data := []byte("expiring data")
	
	cache.Put(key, data)
	
	// Should exist immediately
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Data not found immediately after put")
	}
	
	// Wait for expiration
	time.Sleep(60 * time.Millisecond)
	
	// Should be expired
	_, found = cache.Get(key)
	if found {
		t.Error("Expired data was still found")
	}
}

func TestCache_Dependencies(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	// Put entries with dependencies
	cache.PutWithDeps("entry1", []byte("data1"), []string{"file1.go", "file2.go"})
	cache.PutWithDeps("entry2", []byte("data2"), []string{"file2.go", "file3.go"})
	cache.PutWithDeps("entry3", []byte("data3"), []string{"file3.go"})
	
	// Verify all exist
	_, found1 := cache.Get("entry1")
	_, found2 := cache.Get("entry2")
	_, found3 := cache.Get("entry3")
	
	if !found1 || !found2 || !found3 {
		t.Fatal("Not all entries were cached")
	}
	
	// Invalidate by dependency
	count := cache.InvalidateByDependency("file2.go")
	if count != 2 {
		t.Errorf("Expected 2 entries invalidated, got %d", count)
	}
	
	// Check what remains
	_, found1 = cache.Get("entry1")
	_, found2 = cache.Get("entry2")
	_, found3 = cache.Get("entry3")
	
	if found1 {
		t.Error("entry1 should have been invalidated")
	}
	if found2 {
		t.Error("entry2 should have been invalidated")
	}
	if !found3 {
		t.Error("entry3 should still exist")
	}
}

func TestCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	// Add multiple entries
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		data := []byte(fmt.Sprintf("data%d", i))
		cache.Put(key, data)
	}
	
	stats := cache.GetStats()
	if stats.EntryCount != 10 {
		t.Errorf("Expected 10 entries, got %d", stats.EntryCount)
	}
	
	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}
	
	// Verify all entries are gone
	stats = cache.GetStats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.EntryCount)
	}
	
	// Verify files are deleted
	artifactsDir := filepath.Join(tmpDir, "artifacts")
	if _, err := os.Stat(artifactsDir); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(artifactsDir)
		if len(entries) > 0 {
			t.Errorf("Artifacts directory still has %d files", len(entries))
		}
	}
}

func TestCache_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:     tmpDir,
		MaxSize: 10 << 20, // 10 MB
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100
	
	// Concurrent puts and gets
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				data := []byte(fmt.Sprintf("data-%d-%d", id, j))
				
				// Put
				if err := cache.Put(key, data); err != nil {
					t.Errorf("Failed to put: %v", err)
				}
				
				// Get
				retrieved, found := cache.Get(key)
				if !found {
					t.Errorf("Key not found: %s", key)
				}
				if !bytes.Equal(retrieved, data) {
					t.Errorf("Data mismatch for key %s", key)
				}
				
				// Occasionally delete
				if j%10 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify cache is still consistent
	stats := cache.GetStats()
	if stats.EntryCount < 0 {
		t.Errorf("Invalid entry count: %d", stats.EntryCount)
	}
	if stats.TotalSize < 0 {
		t.Errorf("Invalid total size: %d", stats.TotalSize)
	}
}

func TestCache_KeyGeneration(t *testing.T) {
	// Test basic key generation
	key1 := Key("input1", "input2", "input3")
	key2 := Key("input1", "input2", "input3")
	key3 := Key("input1", "input2", "different")
	
	if key1 != key2 {
		t.Error("Same inputs produced different keys")
	}
	
	if key1 == key3 {
		t.Error("Different inputs produced same key")
	}
	
	// Test file-based key generation
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)
	
	fileKey1, err := KeyFromFiles(file1, file2)
	if err != nil {
		t.Fatalf("Failed to generate key from files: %v", err)
	}
	
	fileKey2, err := KeyFromFiles(file1, file2)
	if err != nil {
		t.Fatalf("Failed to generate key from files: %v", err)
	}
	
	if fileKey1 != fileKey2 {
		t.Error("Same files produced different keys")
	}
	
	// Modify file
	os.WriteFile(file1, []byte("modified"), 0644)
	
	fileKey3, err := KeyFromFiles(file1, file2)
	if err != nil {
		t.Fatalf("Failed to generate key from files: %v", err)
	}
	
	if fileKey1 == fileKey3 {
		t.Error("Modified file produced same key")
	}
}

func TestCache_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create cache and add data
	cache1, err := New(Config{
		Dir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	cache1.Put("persistent-key", []byte("persistent-data"))
	
	// Simulate cache restart
	cache2, err := New(Config{
		Dir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create second cache: %v", err)
	}
	
	// Data should still be available
	data, found := cache2.Get("persistent-key")
	if !found {
		t.Fatal("Persistent data not found after restart")
	}
	
	if string(data) != "persistent-data" {
		t.Errorf("Persistent data corrupted: got %s", data)
	}
}

func TestCache_LargeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := New(Config{
		Dir:     tmpDir,
		MaxSize: 10 << 20, // 10 MB
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	// Create a 1MB file
	largeData := bytes.Repeat([]byte("x"), 1<<20)
	
	if err := cache.Put("large-file", largeData); err != nil {
		t.Fatalf("Failed to cache large file: %v", err)
	}
	
	retrieved, found := cache.Get("large-file")
	if !found {
		t.Fatal("Large file not found in cache")
	}
	
	if len(retrieved) != len(largeData) {
		t.Errorf("Large file size mismatch: got %d, want %d", len(retrieved), len(largeData))
	}
	
	if !bytes.Equal(retrieved, largeData) {
		t.Error("Large file data corrupted")
	}
}

func BenchmarkCache_Put(b *testing.B) {
	tmpDir := b.TempDir()
	cache, _ := New(Config{
		Dir:     tmpDir,
		MaxSize: 100 << 20, // 100 MB
	})
	
	data := bytes.Repeat([]byte("benchmark"), 1024) // 10KB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Put(key, data)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	tmpDir := b.TempDir()
	cache, _ := New(Config{
		Dir:     tmpDir,
		MaxSize: 100 << 20,
	})
	
	// Pre-populate cache
	data := bytes.Repeat([]byte("benchmark"), 1024)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Put(key, data)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkCache_KeyGeneration(b *testing.B) {
	inputs := []string{"input1", "input2", "input3", "input4", "input5"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Key(inputs...)
	}
}