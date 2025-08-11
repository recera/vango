package routes

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

// RouteFile contains information about a discovered route file
type RouteFile struct {
	Path        string  // "app/routes/dashboard.go"
	URLPattern  string  // "/dashboard"
	IsAPI       bool    // true if in routes/api/
	HasServer   bool    // true if has //vango:server or vango_server build tag
	HasClient   bool    // true if has //vango:client or vango_client build tag
	Package     string  // "routes"
	HandlerName string  // "Page" or "ServerCounterPage"
	ImportPath  string  // "github.com/user/app/app/routes"
	Params      []Param // Parameters extracted from path
}

// Param represents a route parameter
type Param struct {
	Name    string // "slug", "id"
	Type    string // "string", "int", "int64", "uuid"
	Pattern string // Regex pattern for matching
}

// Scanner scans the routes directory for route files
type Scanner struct {
	routesDir  string
	moduleName string
}

// NewScanner creates a new route scanner
func NewScanner(routesDir string) (*Scanner, error) {
	// Get module name from go.mod
	moduleName, err := getModuleName()
	if err != nil {
		return nil, fmt.Errorf("failed to get module name: %w", err)
	}

	return &Scanner{
		routesDir:  routesDir,
		moduleName: moduleName,
	}, nil
}

// ScanRoutes scans the routes directory and returns all route files
func (s *Scanner) ScanRoutes() ([]RouteFile, error) {
	var routes []RouteFile

	// Check if routes directory exists
	if _, err := os.Stat(s.routesDir); os.IsNotExist(err) {
		// No routes directory, return empty list
		return routes, nil
	}

	err := filepath.WalkDir(s.routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and special files
		name := d.Name()
		if strings.HasSuffix(name, "_test.go") ||
			name == "_layout.go" ||
			name == "_middleware.go" ||
			name == "_404.go" ||
			name == "_500.go" {
			return nil
		}

		// Process route file
		route, err := s.processRouteFile(path)
		if err != nil {
			// Log error but continue scanning
			fmt.Printf("Warning: failed to process %s: %v\n", path, err)
			return nil
		}

		if route != nil {
			routes = append(routes, *route)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan routes: %w", err)
	}

	// Sort routes for deterministic output
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].URLPattern < routes[j].URLPattern
	})

	return routes, nil
}

// processRouteFile processes a single route file
func (s *Scanner) processRouteFile(filePath string) (*RouteFile, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Check for pragmas and build tags
	hasServer := s.hasServerPragma(string(content))
	hasClient := s.hasClientPragma(string(content))

	// Parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Get package name
	packageName := node.Name.Name

	// Find handler function
	handlerName := s.findHandlerFunction(node)
	if handlerName == "" {
		// Not a route file if no handler found
		return nil, nil
	}

	// Get relative path from routes directory
	relPath, err := filepath.Rel(s.routesDir, filePath)
	if err != nil {
		return nil, err
	}

	// Convert file path to URL pattern
	urlPattern := s.filePathToURLPattern(relPath)

	// Check if it's an API route
	isAPI := strings.HasPrefix(relPath, "api/") || strings.HasPrefix(relPath, "api"+string(filepath.Separator))

	// Extract parameters from URL pattern
	params := s.extractParams(urlPattern)

	// Build import path
	importPath := filepath.Join(s.moduleName, s.routesDir)
	if packageName != "routes" {
		// Handle subdirectories
		dir := filepath.Dir(relPath)
		if dir != "." {
			importPath = filepath.Join(importPath, dir)
		}
	}

	return &RouteFile{
		Path:        filePath,
		URLPattern:  urlPattern,
		IsAPI:       isAPI,
		HasServer:   hasServer,
		HasClient:   hasClient,
		Package:     packageName,
		HandlerName: handlerName,
		ImportPath:  strings.ReplaceAll(importPath, string(filepath.Separator), "/"),
		Params:      params,
	}, nil
}

// hasServerPragma checks if the file has server pragma or build tag
func (s *Scanner) hasServerPragma(content string) bool {
	// Check for //vango:server pragma
	if strings.Contains(content, "//vango:server") {
		return true
	}

	// Check for vango_server build tag
	if strings.Contains(content, "vango_server") && strings.Contains(content, "//go:build") {
		return true
	}
	if strings.Contains(content, "vango_server") && strings.Contains(content, "// +build") {
		return true
	}

	return false
}

// hasClientPragma checks if the file has client pragma or build tag
func (s *Scanner) hasClientPragma(content string) bool {
	// Check for //vango:client pragma
	if strings.Contains(content, "//vango:client") {
		return true
	}

	// Check for vango_client build tag
	if strings.Contains(content, "vango_client") && strings.Contains(content, "//go:build") {
		return true
	}
	if strings.Contains(content, "vango_client") && strings.Contains(content, "// +build") {
		return true
	}

	return false
}

// findHandlerFunction finds the main handler function in the AST
func (s *Scanner) findHandlerFunction(node *ast.File) string {
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// Check if it's exported
			if !ast.IsExported(fn.Name.Name) {
				continue
			}

			// Look for common handler names
			name := fn.Name.Name
			if name == "Page" || strings.HasSuffix(name, "Page") ||
				name == "Handler" || strings.HasSuffix(name, "Handler") ||
				name == "Get" || name == "Post" || name == "Put" || name == "Delete" {
				return name
			}
		}
	}
	return ""
}

// filePathToURLPattern converts a file path to a URL pattern
// NOTE: We standardize on bracket-style parameters expected by pkg/server.Router:
//
//	[slug], [id:int], [...rest]
func (s *Scanner) filePathToURLPattern(relPath string) string {
	// Remove .go extension
	path := strings.TrimSuffix(relPath, ".go")

	// Convert to URL path
	urlPath := "/" + strings.ReplaceAll(path, string(filepath.Separator), "/")

	// Handle index.go -> /
	if strings.HasSuffix(urlPath, "/index") {
		urlPath = strings.TrimSuffix(urlPath, "/index")
		if urlPath == "" {
			urlPath = "/"
		}
	}

	// Keep bracket params as-is so the runtime router can parse them
	return urlPath
}

// extractParams extracts parameters from a URL pattern
func (s *Scanner) extractParams(urlPattern string) []Param {
	var params []Param

	// Extract bracket params: [name] or [name:type]
	// Catch-all: [...rest]
	segs := strings.Split(strings.Trim(urlPattern, "/"), "/")
	re := regexp.MustCompile(`^\[(\.{3})?([^:\]]+)(?::([^\]]+))?\]$`)
	for _, seg := range segs {
		m := re.FindStringSubmatch(seg)
		if m == nil {
			continue
		}
		isCatchAll := m[1] == "..."
		name := m[2]
		ptype := m[3]
		if ptype == "" {
			ptype = "string"
		}
		if isCatchAll {
			params = append(params, Param{
				Name:    name,
				Type:    "[]string",
				Pattern: ".*",
			})
		} else {
			pattern := `[^/]+`
			switch ptype {
			case "int", "int64":
				pattern = `[0-9]+`
			case "uuid":
				pattern = `[0-9a-f-]{36}`
			}
			params = append(params, Param{
				Name:    name,
				Type:    ptype,
				Pattern: pattern,
			})
		}
	}

	return params
}

// getModuleName reads the module name from go.mod
func getModuleName() (string, error) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	// Parse module line
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("module name not found in go.mod")
}
