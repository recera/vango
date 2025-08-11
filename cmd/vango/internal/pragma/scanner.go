// Package pragma implements the Vango pragma scanner for build-time code splitting.
// It processes //vango:server and //vango:client directives to automatically configure
// build tags and separate server/client code during compilation.
package pragma

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// PragmaType represents the type of pragma directive
type PragmaType string

const (
	// PragmaServer indicates server-only code
	PragmaServer PragmaType = "server"
	// PragmaClient indicates client-only code
	PragmaClient PragmaType = "client"
	// PragmaUniversal indicates code that runs on both server and client
	PragmaUniversal PragmaType = "universal"
)

// Pragma represents a parsed pragma directive
type Pragma struct {
	Type     PragmaType `json:"type"`
	FilePath string     `json:"file_path"`
	Line     int        `json:"line"`
	Column   int        `json:"column"`
	Raw      string     `json:"raw"`
	Options  []string   `json:"options,omitempty"`
}

// Manifest represents the build manifest generated from pragma scanning
type Manifest struct {
	Version      string             `json:"version"`
	Timestamp    time.Time          `json:"timestamp"`
	ServerFiles  []string           `json:"server_files"`
	ClientFiles  []string           `json:"client_files"`
	SharedFiles  []string           `json:"shared_files"`
	Pragmas      map[string]Pragma  `json:"pragmas"`
	Dependencies map[string]string  `json:"dependencies"`
	Hash         string             `json:"hash"`
}

// Scanner handles pragma detection and processing
type Scanner struct {
	mu          sync.RWMutex
	rootDir     string
	pragmas     map[string][]Pragma
	fileHashes  map[string]string
	errors      []error
	verbose     bool
	
	// Configuration
	config ScannerConfig
	
	// Caching
	cache       *pragmaCache
}

// ScannerConfig holds scanner configuration
type ScannerConfig struct {
	// RootDir is the root directory to scan
	RootDir string
	
	// IncludePatterns are glob patterns for files to include
	IncludePatterns []string
	
	// ExcludePatterns are glob patterns for files to exclude
	ExcludePatterns []string
	
	// AutoInjectTags automatically injects build tags
	AutoInjectTags bool
	
	// PreservePragmas keeps pragma comments after tag injection
	PreservePragmas bool
	
	// Verbose enables detailed logging
	Verbose bool
	
	// CacheDir is the directory for caching scan results
	CacheDir string
}

// pragmaCache handles caching of scan results
type pragmaCache struct {
	mu       sync.RWMutex
	dir      string
	entries  map[string]*cacheEntry
}

type cacheEntry struct {
	Hash      string    `json:"hash"`
	Pragmas   []Pragma  `json:"pragmas"`
	Timestamp time.Time `json:"timestamp"`
}

// pragmaRegex matches //vango: directives
var pragmaRegex = regexp.MustCompile(`^//\s*vango:(\w+)(?:\s+(.*))?$`)

// NewScanner creates a new pragma scanner
func NewScanner(config ScannerConfig) (*Scanner, error) {
	if config.RootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		config.RootDir = cwd
	}
	
	// Set default patterns if not provided
	if len(config.IncludePatterns) == 0 {
		config.IncludePatterns = []string{"**/*.go"}
	}
	
	if len(config.ExcludePatterns) == 0 {
		config.ExcludePatterns = []string{
			"**/vendor/**",
			"**/node_modules/**",
			"**/.git/**",
			"**/testdata/**",
			"**/*_test.go",
		}
	}
	
	scanner := &Scanner{
		rootDir:     config.RootDir,
		pragmas:     make(map[string][]Pragma),
		fileHashes:  make(map[string]string),
		errors:      []error{},
		verbose:     config.Verbose,
		config:      config,
	}
	
	// Initialize cache if cache directory is provided
	if config.CacheDir != "" {
		scanner.cache = &pragmaCache{
			dir:     config.CacheDir,
			entries: make(map[string]*cacheEntry),
		}
		scanner.cache.load()
	}
	
	return scanner, nil
}

// Scan performs a full scan of the project directory
func (s *Scanner) Scan() (*Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear previous results
	s.pragmas = make(map[string][]Pragma)
	s.fileHashes = make(map[string]string)
	s.errors = []error{}
	
	// Walk the directory tree
	err := filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.errors = append(s.errors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue scanning
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if file should be processed
		if !s.shouldProcess(path) {
			return nil
		}
		
		// Process Go file
		if strings.HasSuffix(path, ".go") {
			if err := s.scanFile(path); err != nil {
				s.errors = append(s.errors, fmt.Errorf("error scanning %s: %w", path, err))
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	// Generate manifest
	manifest := s.generateManifest()
	
	// Auto-inject build tags if configured
	if s.config.AutoInjectTags {
		if err := s.injectBuildTags(); err != nil {
			return manifest, fmt.Errorf("failed to inject build tags: %w", err)
		}
	}
	
	return manifest, nil
}

// scanFile scans a single Go file for pragmas
func (s *Scanner) scanFile(path string) error {
	// Check cache first
	if s.cache != nil {
		hash, err := s.hashFile(path)
		if err == nil {
			if cached, ok := s.cache.get(path, hash); ok {
				s.pragmas[path] = cached
				s.fileHashes[path] = hash
				return nil
			}
		}
	}
	
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	// Calculate hash
	hash := s.hashContent(content)
	s.fileHashes[path] = hash
	
	// Parse file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}
	
	// Extract pragmas from comments
	var pragmas []Pragma
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if pragma := s.parsePragma(comment.Text, fset.Position(comment.Pos())); pragma != nil {
				pragma.FilePath = path
				pragmas = append(pragmas, *pragma)
			}
		}
	}
	
	// Store results
	if len(pragmas) > 0 {
		s.pragmas[path] = pragmas
		
		// Update cache
		if s.cache != nil {
			s.cache.put(path, hash, pragmas)
		}
	}
	
	return nil
}

// parsePragma parses a pragma directive from a comment
func (s *Scanner) parsePragma(text string, pos token.Position) *Pragma {
	// Trim comment markers
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)
	
	// Check for vango pragma
	if !strings.HasPrefix(text, "vango:") {
		return nil
	}
	
	// Parse pragma
	matches := pragmaRegex.FindStringSubmatch("//" + text)
	if len(matches) < 2 {
		return nil
	}
	
	pragmaType := matches[1]
	options := []string{}
	if len(matches) > 2 && matches[2] != "" {
		options = strings.Fields(matches[2])
	}
	
	// Map pragma type
	var pType PragmaType
	switch pragmaType {
	case "server":
		pType = PragmaServer
	case "client":
		pType = PragmaClient
	case "universal":
		pType = PragmaUniversal
	default:
		// Unknown pragma type, skip
		return nil
	}
	
	return &Pragma{
		Type:   pType,
		Line:   pos.Line,
		Column: pos.Column,
		Raw:    text,
		Options: options,
	}
}

// injectBuildTags injects build tags based on pragmas
func (s *Scanner) injectBuildTags() error {
	for path, pragmas := range s.pragmas {
		if len(pragmas) == 0 {
			continue
		}
		
		// Determine build tag based on pragma type
		pragma := pragmas[0] // Use first pragma in file
		var buildTag string
		switch pragma.Type {
		case PragmaServer:
			buildTag = "vango_server"
		case PragmaClient:
			buildTag = "vango_client"
		case PragmaUniversal:
			// No build tag needed for universal code
			continue
		}
		
		// Inject build tag
		if err := s.injectBuildTag(path, buildTag); err != nil {
			return fmt.Errorf("failed to inject build tag in %s: %w", path, err)
		}
	}
	
	return nil
}

// injectBuildTag injects a build tag into a Go file
func (s *Scanner) injectBuildTag(path string, tag string) error {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	// Check if build tag already exists
	if bytes.Contains(content, []byte("//go:build "+tag)) ||
		bytes.Contains(content, []byte("// +build "+tag)) {
		// Tag already present
		return nil
	}
	
	// Parse file to find package declaration
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return err
	}
	
	// Build new content with build tag
	var buf bytes.Buffer
	
	// Write build constraint
	fmt.Fprintf(&buf, "//go:build %s\n\n", tag)
	
	// Preserve existing comments before package declaration
	if file.Doc != nil {
		for _, comment := range file.Doc.List {
			// Skip existing vango pragmas if not preserving
			if !s.config.PreservePragmas && strings.Contains(comment.Text, "vango:") {
				continue
			}
			fmt.Fprintln(&buf, comment.Text)
		}
		fmt.Fprintln(&buf)
	}
	
	// Write the rest of the file
	// Find package position
	packagePos := fset.Position(file.Package)
	lines := bytes.Split(content, []byte("\n"))
	for i := packagePos.Line - 1; i < len(lines); i++ {
		buf.Write(lines[i])
		if i < len(lines)-1 {
			buf.WriteByte('\n')
		}
	}
	
	// Format the result
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, use unformatted content
		formatted = buf.Bytes()
	}
	
	// Write back to file
	return os.WriteFile(path, formatted, 0644)
}

// shouldProcess determines if a file should be processed
func (s *Scanner) shouldProcess(path string) bool {
	// Convert to relative path
	relPath, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		return false
	}
	
	// Check exclude patterns first
	for _, pattern := range s.config.ExcludePatterns {
		matched, _ := filepath.Match(pattern, relPath)
		if matched {
			return false
		}
		// Also check with ** glob patterns
		if strings.Contains(pattern, "**") {
			if matchDoubleGlob(pattern, relPath) {
				return false
			}
		}
	}
	
	// Check include patterns
	for _, pattern := range s.config.IncludePatterns {
		matched, _ := filepath.Match(pattern, relPath)
		if matched {
			return true
		}
		// Also check with ** glob patterns
		if strings.Contains(pattern, "**") {
			if matchDoubleGlob(pattern, relPath) {
				return true
			}
		}
	}
	
	return false
}

// matchDoubleGlob matches ** glob patterns
func matchDoubleGlob(pattern, path string) bool {
	// Handle ** glob pattern which matches any number of directories
	if strings.Contains(pattern, "**") {
		// Handle patterns like **/vendor/** or **/testdata/**
		if strings.HasPrefix(pattern, "**/") && strings.HasSuffix(pattern, "/**") {
			// Extract the directory name to check
			dirName := strings.TrimPrefix(pattern, "**/")
			dirName = strings.TrimSuffix(dirName, "/**")
			// Check if path contains this directory
			return strings.Contains(path, "/"+dirName+"/") || strings.HasPrefix(path, dirName+"/")
		}
		
		// Handle patterns like **/*.go
		if strings.HasPrefix(pattern, "**/") {
			suffix := strings.TrimPrefix(pattern, "**/")
			// Check if the path ends with the suffix pattern
			if strings.HasPrefix(suffix, "*") {
				// For patterns like *.go, check extension
				return strings.HasSuffix(path, strings.TrimPrefix(suffix, "*"))
			}
			// For patterns like specific files
			return strings.HasSuffix(path, suffix)
		}
		
		// Handle patterns like **/*_test.go
		if strings.HasPrefix(pattern, "**/*") {
			suffix := strings.TrimPrefix(pattern, "**/*")
			// Check if filename matches pattern
			filename := filepath.Base(path)
			matched, _ := filepath.Match("*"+suffix, filename)
			return matched
		}
	}
	
	// Fallback to simple pattern matching
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// generateManifest generates a build manifest from scan results
func (s *Scanner) generateManifest() *Manifest {
	manifest := &Manifest{
		Version:      "1.0",
		Timestamp:    time.Now(),
		ServerFiles:  []string{},
		ClientFiles:  []string{},
		SharedFiles:  []string{},
		Pragmas:      make(map[string]Pragma),
		Dependencies: make(map[string]string),
	}
	
	// Categorize files (sorted for deterministic output)
	var pragmaPaths []string
	for path := range s.pragmas {
		pragmaPaths = append(pragmaPaths, path)
	}
	sort.Strings(pragmaPaths)
	
	for _, path := range pragmaPaths {
		pragmas := s.pragmas[path]
		if len(pragmas) == 0 {
			continue
		}
		
		relPath, _ := filepath.Rel(s.rootDir, path)
		pragma := pragmas[0] // Use first pragma
		
		switch pragma.Type {
		case PragmaServer:
			manifest.ServerFiles = append(manifest.ServerFiles, relPath)
		case PragmaClient:
			manifest.ClientFiles = append(manifest.ClientFiles, relPath)
		case PragmaUniversal:
			manifest.SharedFiles = append(manifest.SharedFiles, relPath)
		}
		
		manifest.Pragmas[relPath] = pragma
	}
	
	// Add file hashes as dependencies (sorted for deterministic output)
	var paths []string
	for path := range s.fileHashes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		relPath, _ := filepath.Rel(s.rootDir, path)
		manifest.Dependencies[relPath] = s.fileHashes[path]
	}
	
	// Calculate manifest hash
	manifest.Hash = s.calculateManifestHash(manifest)
	
	return manifest
}

// calculateManifestHash calculates a hash for the manifest
func (s *Scanner) calculateManifestHash(manifest *Manifest) string {
	h := sha256.New()
	
	// Hash all file paths and their hashes (already sorted in slices)
	for _, files := range [][]string{manifest.ServerFiles, manifest.ClientFiles, manifest.SharedFiles} {
		for _, file := range files {
			h.Write([]byte(file))
			if hash, ok := manifest.Dependencies[file]; ok {
				h.Write([]byte(hash))
			}
		}
	}
	
	// Also hash all dependencies in sorted order for completeness
	var depKeys []string
	for k := range manifest.Dependencies {
		depKeys = append(depKeys, k)
	}
	sort.Strings(depKeys)
	for _, k := range depKeys {
		h.Write([]byte(k))
		h.Write([]byte(manifest.Dependencies[k]))
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// hashFile calculates the hash of a file
func (s *Scanner) hashFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return s.hashContent(content), nil
}

// hashContent calculates the hash of content
func (s *Scanner) hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// GetErrors returns any errors encountered during scanning
func (s *Scanner) GetErrors() []error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errors
}

// SaveManifest saves the manifest to a file
func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}

// LoadManifest loads a manifest from a file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	
	return &manifest, nil
}

// Cache implementation

func (c *pragmaCache) load() error {
	if c.dir == "" {
		return nil
	}
	
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	
	// Load cache index
	indexPath := filepath.Join(c.dir, "pragma-cache.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache yet
		}
		return err
	}
	
	return json.Unmarshal(data, &c.entries)
}

func (c *pragmaCache) save() error {
	if c.dir == "" {
		return nil
	}
	
	c.mu.RLock()
	data, err := json.MarshalIndent(c.entries, "", "  ")
	c.mu.RUnlock()
	
	if err != nil {
		return err
	}
	
	indexPath := filepath.Join(c.dir, "pragma-cache.json")
	return os.WriteFile(indexPath, data, 0644)
}

func (c *pragmaCache) get(path, hash string) ([]Pragma, bool) {
	if c == nil {
		return nil, false
	}
	
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, ok := c.entries[path]
	if !ok || entry.Hash != hash {
		return nil, false
	}
	
	return entry.Pragmas, true
}

func (c *pragmaCache) put(path, hash string, pragmas []Pragma) {
	if c == nil {
		return
	}
	
	c.mu.Lock()
	c.entries[path] = &cacheEntry{
		Hash:      hash,
		Pragmas:   pragmas,
		Timestamp: time.Now(),
	}
	c.mu.Unlock()
	
	// Save cache asynchronously
	go c.save()
}

// ScanFile scans a single file for pragmas (utility function)
func ScanFile(path string) ([]Pragma, error) {
	scanner, err := NewScanner(ScannerConfig{})
	if err != nil {
		return nil, err
	}
	
	if err := scanner.scanFile(path); err != nil {
		return nil, err
	}
	
	return scanner.pragmas[path], nil
}

// InjectBuildTag injects a build tag into a file (utility function)
func InjectBuildTag(path string, tag string) error {
	scanner, err := NewScanner(ScannerConfig{})
	if err != nil {
		return err
	}
	
	return scanner.injectBuildTag(path, tag)
}