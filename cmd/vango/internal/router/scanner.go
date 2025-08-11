package router

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RouteInfo contains information about a discovered route
type RouteInfo struct {
	FilePath      string      // Original file path
	URLPath       string      // Generated URL path (e.g., /blog/[slug])
	HandlerName   string      // Name of the handler function
	PackageName   string      // Package name
	Params        []ParamInfo // Extracted parameters
	HasLayout     bool        // Whether this route has a layout
	HasMiddleware bool        // Whether this route has middleware
	LayoutFunc    string      // Name of layout function if exists
	IsAPI         bool        // Whether this is an API route
	IsCatchAll    bool        // Whether this route has catch-all param
}

// ParamInfo contains information about a route parameter
type ParamInfo struct {
	Name     string // Parameter name (e.g., "slug", "id")
	Type     string // Parameter type (e.g., "string", "int", "uuid")
	Position int    // Position in URL
	Pattern  string // Regex pattern for matching
}

// Scanner scans the routes directory for route files
type Scanner struct {
	routesDir string
	routes    []RouteInfo
	errors    []error
}

// NewScanner creates a new route scanner
func NewScanner(routesDir string) *Scanner {
	return &Scanner{
		routesDir: routesDir,
		routes:    make([]RouteInfo, 0),
		errors:    make([]error, 0),
	}
}

// Scan scans the routes directory and returns discovered routes
func (s *Scanner) Scan() ([]RouteInfo, error) {
	// Walk the routes directory
	err := filepath.WalkDir(s.routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and test files
		if d.IsDir() || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip special files
		name := d.Name()
		if name == "_layout.go" || name == "_middleware.go" ||
			name == "_404.go" || name == "_500.go" {
			// These are special files, handle separately
			return s.handleSpecialFile(path, name)
		}

		// Process route file
		route, err := s.processRouteFile(path)
		if err != nil {
			s.errors = append(s.errors, fmt.Errorf("%s: %w", path, err))
			return nil // Continue scanning
		}

		if route != nil {
			s.routes = append(s.routes, *route)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Check for duplicate routes
	if err := s.checkDuplicates(); err != nil {
		return nil, err
	}

	// Sort routes for deterministic output
	sort.Slice(s.routes, func(i, j int) bool {
		return s.routes[i].URLPath < s.routes[j].URLPath
	})

	return s.routes, nil
}

// processRouteFile processes a single route file
func (s *Scanner) processRouteFile(filePath string) (*RouteInfo, error) {
	// Get relative path from routes directory
	relPath, err := filepath.Rel(s.routesDir, filePath)
	if err != nil {
		return nil, err
	}

	// Convert file path to URL path
	urlPath := s.filePathToURLPath(relPath)

	// Extract parameters from path
	params := s.extractParams(urlPath)

	// Parse the Go file to find handler function
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Find the Page handler function
	var handlerName string
	var packageName string
	packageName = node.Name.Name

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// Look for Page, Handler, or Get/Post/Put/Delete functions
			if fn.Name.Name == "Page" || fn.Name.Name == "Handler" ||
				fn.Name.Name == "Get" || fn.Name.Name == "Post" {
				handlerName = fn.Name.Name
				break
			}
		}
	}

	if handlerName == "" {
		// Not a route file
		return nil, nil
	}

	// Check if this is an API route
	isAPI := strings.HasPrefix(relPath, "api/")

	// Check for layout and middleware in the same directory
	dir := filepath.Dir(filePath)
	hasLayout := s.fileExists(filepath.Join(dir, "_layout.go"))
	hasMiddleware := s.fileExists(filepath.Join(dir, "_middleware.go"))

	return &RouteInfo{
		FilePath:      filePath,
		URLPath:       urlPath,
		HandlerName:   handlerName,
		PackageName:   packageName,
		Params:        params,
		HasLayout:     hasLayout,
		HasMiddleware: hasMiddleware,
		IsAPI:         isAPI,
		IsCatchAll:    len(params) > 0 && strings.HasPrefix(params[len(params)-1].Name, "..."),
	}, nil
}

// filePathToURLPath converts a file path to a URL path
func (s *Scanner) filePathToURLPath(relPath string) string {
	// Remove .go extension
	path := strings.TrimSuffix(relPath, ".go")

	// Convert path separators to URL separators
	path = filepath.ToSlash(path)

	// Handle index.go -> /
	if path == "index" {
		return "/"
	}

	// Remove /index suffix
	path = strings.TrimSuffix(path, "/index")

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// extractParams extracts parameters from a URL path
func (s *Scanner) extractParams(urlPath string) []ParamInfo {
	params := []ParamInfo{}

	// Regular expression to match [param] or [param:type]
	paramRegex := regexp.MustCompile(`\[([^:\]]+)(?::([^]]+))?\]`)

	segments := strings.Split(urlPath, "/")
	for i, segment := range segments {
		matches := paramRegex.FindStringSubmatch(segment)
		if matches != nil {
			paramName := matches[1]
			paramType := matches[2]

			// Default type is string
			if paramType == "" {
				paramType = "string"
			}

			// Handle catch-all params
			isCatchAll := strings.HasPrefix(paramName, "...")
			if isCatchAll {
				paramName = strings.TrimPrefix(paramName, "...")
			}

			param := ParamInfo{
				Name:     paramName,
				Type:     paramType,
				Position: i,
				Pattern:  s.getParamPattern(paramType, isCatchAll),
			}

			params = append(params, param)
		}
	}

	return params
}

// getParamPattern returns the regex pattern for a parameter type
func (s *Scanner) getParamPattern(paramType string, isCatchAll bool) string {
	if isCatchAll {
		return ".*" // Match everything for catch-all
	}

	switch paramType {
	case "int":
		return `[0-9]+`
	case "int64":
		return `[0-9]+`
	case "uuid":
		return `[0-9a-f-]{36}`
	default:
		return `[^/]+` // Default: match anything except slash
	}
}

// handleSpecialFile handles special files like _layout.go, _middleware.go
func (s *Scanner) handleSpecialFile(path, name string) error {
	// TODO: Process layout and middleware files
	// For now, just note their existence
	return nil
}

// checkDuplicates checks for duplicate routes
func (s *Scanner) checkDuplicates() error {
	seen := make(map[string]string) // urlPath -> filePath

	for _, route := range s.routes {
		// Normalize path for comparison (remove trailing slashes)
		normalizedPath := strings.TrimSuffix(route.URLPath, "/")
		if normalizedPath == "" {
			normalizedPath = "/"
		}

		if existingFile, exists := seen[normalizedPath]; exists {
			return fmt.Errorf("duplicate route '%s' found in:\n  - %s\n  - %s",
				normalizedPath, existingFile, route.FilePath)
		}

		seen[normalizedPath] = route.FilePath
	}

	return nil
}

// fileExists checks if a file exists
func (s *Scanner) fileExists(path string) bool {
	// Convert to absolute for reliability, but existence check uses os.Stat
	abs := path
	if p, err := filepath.Abs(path); err == nil {
		abs = p
	}
	info, err := os.Stat(abs)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetErrors returns any errors encountered during scanning
func (s *Scanner) GetErrors() []error {
	return s.errors
}
