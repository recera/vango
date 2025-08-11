// Package cache implements a build artifact caching system for Vango.
// It caches compilation outputs to speed up incremental builds and hot reloads.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Cache represents a build cache for storing compiled artifacts
type Cache struct {
	mu       sync.RWMutex
	dir      string
	index    *Index
	maxSize  int64 // Maximum cache size in bytes
	maxAge   time.Duration
	strategy EvictionStrategy
	stats    *Stats
	stopCh   chan struct{} // Channel to signal cleanup goroutine to stop
}

// Index tracks all cached entries
type Index struct {
	Version string                  `json:"version"`
	Entries map[string]*CacheEntry  `json:"entries"`
	Updated time.Time               `json:"updated"`
}

// CacheEntry represents a single cached artifact
type CacheEntry struct {
	Key         string            `json:"key"`
	Hash        string            `json:"hash"`
	Path        string            `json:"path"`
	Size        int64             `json:"size"`
	Created     time.Time         `json:"created"`
	LastAccess  time.Time         `json:"last_access"`
	AccessCount int               `json:"access_count"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"`
}

// Stats tracks cache performance metrics
type Stats struct {
	mu          sync.RWMutex
	Hits        int64         `json:"hits"`
	Misses      int64         `json:"misses"`
	Evictions   int64         `json:"evictions"`
	TotalSize   int64         `json:"total_size"`
	EntryCount  int           `json:"entry_count"`
	SavedTime   time.Duration `json:"saved_time"`
}

// EvictionStrategy defines how cache entries are removed
type EvictionStrategy int

const (
	// LRU removes least recently used entries
	LRU EvictionStrategy = iota
	// LFU removes least frequently used entries
	LFU
	// FIFO removes oldest entries first
	FIFO
)

// Config holds cache configuration
type Config struct {
	Dir      string           // Cache directory (default: $HOME/.cache/vango)
	MaxSize  int64           // Maximum cache size in bytes (default: 1GB)
	MaxAge   time.Duration   // Maximum age for cache entries (default: 7 days)
	Strategy EvictionStrategy // Eviction strategy (default: LRU)
}

// DefaultConfig returns the default cache configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		Dir:      filepath.Join(homeDir, ".cache", "vango"),
		MaxSize:  1 << 30, // 1 GB
		MaxAge:   7 * 24 * time.Hour,
		Strategy: LRU,
	}
}

// New creates a new cache instance
func New(config Config) (*Cache, error) {
	if config.Dir == "" {
		config = DefaultConfig()
	}

	// Create cache directory
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &Cache{
		dir:      config.Dir,
		maxSize:  config.MaxSize,
		maxAge:   config.MaxAge,
		strategy: config.Strategy,
		stats:    &Stats{},
		stopCh:   make(chan struct{}),
		index: &Index{
			Version: "1.0",
			Entries: make(map[string]*CacheEntry),
			Updated: time.Now(),
		},
	}

	// Load existing index
	if err := cache.loadIndex(); err != nil {
		// Index doesn't exist or is corrupted, start fresh
		cache.index = &Index{
			Version: "1.0",
			Entries: make(map[string]*CacheEntry),
			Updated: time.Now(),
		}
	}

	// Clean up old entries on startup
	go cache.cleanup()

	return cache, nil
}

// Get retrieves a cached artifact
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, exists := c.index.Entries[key]
	c.mu.RUnlock()

	if !exists {
		c.recordMiss()
		return nil, false
	}

	// Check if entry is expired
	if c.isExpired(entry) {
		c.Delete(key)
		c.recordMiss()
		return nil, false
	}

	// Read cached file
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		// Cache file is missing or corrupted
		c.Delete(key)
		c.recordMiss()
		return nil, false
	}

	// Update access time and count
	c.mu.Lock()
	entry.LastAccess = time.Now()
	entry.AccessCount++
	c.mu.Unlock()

	c.recordHit()
	c.saveIndex() // Save index asynchronously

	return data, true
}

// Put stores an artifact in the cache
func (c *Cache) Put(key string, data []byte) error {
	// Calculate hash of data
	hash := c.hash(data)

	// Check if we already have this exact artifact
	c.mu.RLock()
	if existing, ok := c.index.Entries[key]; ok && existing.Hash == hash {
		c.mu.RUnlock()
		return nil // Already cached
	}
	c.mu.RUnlock()

	// Ensure we have space
	size := int64(len(data))
	if err := c.ensureSpace(size); err != nil {
		return fmt.Errorf("failed to ensure cache space: %w", err)
	}

	// Create cache file path
	filename := fmt.Sprintf("%s_%s", sanitizeKey(key), hash[:8])
	path := filepath.Join(c.dir, "artifacts", filename)

	// Ensure artifacts directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Write data to cache file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Create cache entry
	entry := &CacheEntry{
		Key:         key,
		Hash:        hash,
		Path:        path,
		Size:        size,
		Created:     time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 0,
		Metadata:    make(map[string]string),
	}

	// Update index
	c.mu.Lock()
	// Remove old entry if it exists
	if old, ok := c.index.Entries[key]; ok {
		c.removeFile(old.Path)
		c.stats.TotalSize -= old.Size
	}
	c.index.Entries[key] = entry
	c.index.Updated = time.Now()
	c.stats.TotalSize += size
	c.stats.EntryCount = len(c.index.Entries)
	c.mu.Unlock()

	// Save index
	return c.saveIndex()
}

// PutWithDeps stores an artifact with dependency tracking
func (c *Cache) PutWithDeps(key string, data []byte, deps []string) error {
	if err := c.Put(key, data); err != nil {
		return err
	}

	// Update dependencies
	c.mu.Lock()
	if entry, ok := c.index.Entries[key]; ok {
		entry.Dependencies = deps
	}
	c.mu.Unlock()

	return c.saveIndex()
}

// Delete removes an entry from the cache
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.index.Entries[key]
	if !ok {
		return nil // Already deleted
	}

	// Remove file
	c.removeFile(entry.Path)

	// Update index
	delete(c.index.Entries, key)
	c.stats.TotalSize -= entry.Size
	c.stats.EntryCount = len(c.index.Entries)
	c.index.Updated = time.Now()

	return c.saveIndexNoLock()
}

// InvalidateByDependency removes entries that depend on the given file
func (c *Cache) InvalidateByDependency(dep string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key, entry := range c.index.Entries {
		for _, d := range entry.Dependencies {
			if d == dep || strings.HasPrefix(d, dep) {
				// Remove this entry
				c.removeFile(entry.Path)
				delete(c.index.Entries, key)
				c.stats.TotalSize -= entry.Size
				count++
				break
			}
		}
	}

	c.stats.EntryCount = len(c.index.Entries)
	c.index.Updated = time.Now()
	c.saveIndexNoLock()

	return count
}

// Clear removes all cached entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all artifact files
	artifactsDir := filepath.Join(c.dir, "artifacts")
	if err := os.RemoveAll(artifactsDir); err != nil {
		return fmt.Errorf("failed to clear artifacts: %w", err)
	}

	// Reset index
	c.index = &Index{
		Version: "1.0",
		Entries: make(map[string]*CacheEntry),
		Updated: time.Now(),
	}

	// Reset stats
	c.stats = &Stats{}

	return c.saveIndexNoLock()
}

// GetStats returns cache statistics
func (c *Cache) GetStats() Stats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return *c.stats
}

// Key generates a cache key from inputs
func Key(inputs ...string) string {
	h := sha256.New()
	for _, input := range inputs {
		h.Write([]byte(input))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// KeyFromFiles generates a cache key from file contents
func KeyFromFiles(files ...string) (string, error) {
	h := sha256.New()
	
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", file, err)
		}
		h.Write(data)
		
		// Include file modification time for extra safety
		info, err := os.Stat(file)
		if err == nil {
			h.Write([]byte(info.ModTime().String()))
		}
	}
	
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Private methods

func (c *Cache) loadIndex() error {
	indexPath := filepath.Join(c.dir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}

	c.index = &index

	// Calculate total size
	var totalSize int64
	for _, entry := range c.index.Entries {
		totalSize += entry.Size
	}
	c.stats.TotalSize = totalSize
	c.stats.EntryCount = len(c.index.Entries)

	return nil
}

func (c *Cache) saveIndex() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.saveIndexNoLock()
}

// saveIndexNoLock saves the index without acquiring a lock
// Caller must hold at least a read lock
func (c *Cache) saveIndexNoLock() error {
	data, err := json.MarshalIndent(c.index, "", "  ")
	if err != nil {
		return err
	}

	indexPath := filepath.Join(c.dir, "index.json")
	return os.WriteFile(indexPath, data, 0644)
}

func (c *Cache) isExpired(entry *CacheEntry) bool {
	// If maxAge is 0 or negative, entries never expire
	if c.maxAge <= 0 {
		return false
	}
	return time.Since(entry.Created) > c.maxAge
}

func (c *Cache) ensureSpace(needed int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If maxSize is 0 or negative, no limit
	if c.maxSize <= 0 {
		return nil
	}

	// Check if we need to evict entries
	for c.stats.TotalSize+needed > c.maxSize && len(c.index.Entries) > 0 {
		// Find entry to evict based on strategy
		var evictKey string
		var evictEntry *CacheEntry

		switch c.strategy {
		case LRU:
			// Find least recently used
			var oldestAccess time.Time
			for key, entry := range c.index.Entries {
				if evictEntry == nil || entry.LastAccess.Before(oldestAccess) {
					evictKey = key
					evictEntry = entry
					oldestAccess = entry.LastAccess
				}
			}

		case LFU:
			// Find least frequently used
			minCount := int(^uint(0) >> 1) // Max int
			for key, entry := range c.index.Entries {
				if entry.AccessCount < minCount {
					evictKey = key
					evictEntry = entry
					minCount = entry.AccessCount
				}
			}

		case FIFO:
			// Find oldest entry
			var oldestCreated time.Time
			for key, entry := range c.index.Entries {
				if evictEntry == nil || entry.Created.Before(oldestCreated) {
					evictKey = key
					evictEntry = entry
					oldestCreated = entry.Created
				}
			}
		}

		if evictEntry == nil {
			break
		}

		// Remove the entry
		c.removeFile(evictEntry.Path)
		delete(c.index.Entries, evictKey)
		c.stats.TotalSize -= evictEntry.Size
		c.stats.Evictions++
	}

	c.stats.EntryCount = len(c.index.Entries)
	return nil
}

func (c *Cache) cleanup() {
	// Run cleanup every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			
			// Remove expired entries
			for key, entry := range c.index.Entries {
				if c.isExpired(entry) {
					c.removeFile(entry.Path)
					delete(c.index.Entries, key)
					c.stats.TotalSize -= entry.Size
				}
			}
			
			c.stats.EntryCount = len(c.index.Entries)
			c.index.Updated = time.Now()
			c.mu.Unlock()
			
			c.saveIndex()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) removeFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		// Log error but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to remove cache file %s: %v\n", path, err)
	}
}

// Close stops the cleanup goroutine and saves the index
func (c *Cache) Close() error {
	// Signal cleanup to stop
	close(c.stopCh)
	
	// Save final index
	return c.saveIndex()
}

func (c *Cache) hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func (c *Cache) recordHit() {
	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()
}

func (c *Cache) recordMiss() {
	c.stats.mu.Lock()
	c.stats.Misses++
	c.stats.mu.Unlock()
}

func sanitizeKey(key string) string {
	// Replace problematic characters for filesystem
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	sanitized := replacer.Replace(key)
	
	// Limit length
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	return sanitized
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}