package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Route represents a discovered route
type Route struct {
	Path           string       // URL path pattern (e.g., "/blog/[slug]")
	FilePath       string       // File system path (e.g., "app/routes/blog/[slug].go")
	Package        string       // Go package name
	ComponentName  string       // Component function name (usually "Page")
	Params         []RouteParam // Route parameters
	IsAPI          bool         // True if this is an API route
	HasLayout      bool         // True if directory has _layout.go
	HasMiddleware  bool         // True if directory has _middleware.go
	LayoutPath     string       // Path to layout file if exists
	MiddlewarePath string       // Path to middleware file if exists
}

// RouteParam represents a route parameter
type RouteParam struct {
	Name       string // Parameter name (e.g., "slug")
	Type       string // Parameter type (e.g., "string", "int", "uuid")
	IsCatchAll bool   // True for [...rest] parameters
}

// CodeGenerator generates routing code from discovered routes
type CodeGenerator struct {
	routes     []Route
	routesDir  string
	outputDir  string
	modulePath string
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(routesDir, outputDir string) *CodeGenerator {
	return &CodeGenerator{
		routesDir: routesDir,
		outputDir: outputDir,
		routes:    make([]Route, 0),
	}
}

// Generate scans the routes directory and generates all routing code
func (g *CodeGenerator) Generate() error {
	// Step 1: Scan routes directory if not already populated
	if len(g.routes) == 0 {
		if err := g.scanRoutes(); err != nil {
			return fmt.Errorf("failed to scan routes: %w", err)
		}
	}
	// Step 1.1: Detect module path for imports
	mod, err := detectModulePath()
	if err != nil {
		return fmt.Errorf("failed to detect module path: %w", err)
	}
	g.modulePath = mod

	// Step 2: Generate radix tree
	if err := g.generateRadixTree(); err != nil {
		return fmt.Errorf("failed to generate radix tree: %w", err)
	}

	// Step 3: Generate params structs
	if err := g.generateParams(); err != nil {
		return fmt.Errorf("failed to generate params: %w", err)
	}

	// Step 4: Generate path helpers
	if err := g.generatePaths(); err != nil {
		return fmt.Errorf("failed to generate paths: %w", err)
	}

	// Step 5: Generate router table JSON
	if err := g.generateRouterTable(); err != nil {
		return fmt.Errorf("failed to generate router table: %w", err)
	}

	return nil
}

// scanRoutes walks the routes directory and discovers all routes
func (g *CodeGenerator) scanRoutes() error {
	return filepath.WalkDir(g.routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and special files
		if strings.HasSuffix(path, "_test.go") ||
			strings.Contains(path, "_layout.go") ||
			strings.Contains(path, "_middleware.go") {
			return nil
		}

		// Skip special error pages (handled separately)
		basename := filepath.Base(path)
		if basename == "_404.go" || basename == "_500.go" {
			return nil
		}

		// Convert file path to URL path
		relPath, err := filepath.Rel(g.routesDir, path)
		if err != nil {
			return err
		}

		urlPath := g.filePathToURLPath(relPath)
		params := g.extractParams(urlPath)

		// Check for layout and middleware in the same directory
		dir := filepath.Dir(path)
		layoutPath := filepath.Join(dir, "_layout.go")
		middlewarePath := filepath.Join(dir, "_middleware.go")

		route := Route{
			Path:           urlPath,
			FilePath:       path,
			Package:        g.extractPackageName(relPath),
			ComponentName:  "Page",
			Params:         params,
			IsAPI:          strings.Contains(urlPath, "/api/"),
			HasLayout:      fileExists(layoutPath),
			HasMiddleware:  fileExists(middlewarePath),
			LayoutPath:     layoutPath,
			MiddlewarePath: middlewarePath,
		}

		g.routes = append(g.routes, route)
		return nil
	})
}

// filePathToURLPath converts a file path to a URL path
func (g *CodeGenerator) filePathToURLPath(filePath string) string {
	// Remove .go extension
	path := strings.TrimSuffix(filePath, ".go")

	// Replace file separators with URL separators
	path = filepath.ToSlash(path)

	// Handle index.go -> /
	if path == "index" {
		return "/"
	}

	// Remove "index" from paths
	path = strings.ReplaceAll(path, "/index", "")

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// extractParams extracts route parameters from a URL path
func (g *CodeGenerator) extractParams(path string) []RouteParam {
	params := make([]RouteParam, 0)

	// Regular expression to match [param] or [param:type] or [...param]
	re := regexp.MustCompile(`\[([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		paramDef := match[1]

		// Check for catch-all parameter
		if strings.HasPrefix(paramDef, "...") {
			params = append(params, RouteParam{
				Name:       paramDef[3:],
				Type:       "string",
				IsCatchAll: true,
			})
			continue
		}

		// Parse param:type
		parts := strings.Split(paramDef, ":")
		param := RouteParam{
			Name: parts[0],
			Type: "string", // default
		}

		if len(parts) > 1 {
			param.Type = parts[1]
		}

		params = append(params, param)
	}

	return params
}

// extractPackageName extracts the Go package name from a file path
func (g *CodeGenerator) extractPackageName(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "/" {
		return "routes"
	}

	// Use the last directory component as package name
	parts := strings.Split(filepath.ToSlash(dir), "/")
	return parts[len(parts)-1]
}

// generateRadixTree generates the radix tree matcher
func (g *CodeGenerator) generateRadixTree() error {
	// Use the improved generator in radix_codegen.go
	return g.generateRadixTreeV2()
}

// LEGACY TEMPLATE (kept for reference):
/* (legacy generator body omitted) */

// generateParams generates typed parameter structs
func (g *CodeGenerator) generateParams() error {
	tmpl := `// Code generated by vango; DO NOT EDIT.

package router

import (
	"strconv"
)

{{ range .Routes }}
{{ if .Params }}
// {{ .StructName }}Params represents parameters for {{ .Path }}
type {{ .StructName }}Params struct {
{{ range .Params }}	{{ .FieldName }} {{ .GoType }} ` + "`param:\"{{ .Name }}\"`" + `
{{ end }}}

// Parse{{ .StructName }}Params parses parameters from string map
func Parse{{ .StructName }}Params(params map[string]string) ({{ .StructName }}Params, error) {
	result := {{ .StructName }}Params{}
	
{{ range .Params }}{{ if eq .GoType "int" }}	if val, ok := params["{{ .Name }}"]; ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return result, err
		}
		result.{{ .FieldName }} = i
	}
{{ else if eq .GoType "int64" }}	if val, ok := params["{{ .Name }}"]; ok {
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return result, err
		}
		result.{{ .FieldName }} = i
	}
{{ else }}	if val, ok := params["{{ .Name }}"]; ok {
		result.{{ .FieldName }} = val
	}
{{ end }}{{ end }}	
	return result, nil
}
{{ end }}
{{ end }}
`

	// Group routes by unique parameter sets
	type routeData struct {
		Path       string
		StructName string
		Params     []struct {
			Name      string
			FieldName string
			GoType    string
		}
	}

	routes := make([]routeData, 0)
	seen := make(map[string]bool)

	for _, route := range g.routes {
		if len(route.Params) == 0 {
			continue
		}

		// Generate struct name from path
		structName := g.pathToStructName(route.Path)

		// Skip if already generated
		if seen[structName] {
			continue
		}
		seen[structName] = true

		rd := routeData{
			Path:       route.Path,
			StructName: structName,
		}

		for _, param := range route.Params {
			rd.Params = append(rd.Params, struct {
				Name      string
				FieldName string
				GoType    string
			}{
				Name:      param.Name,
				FieldName: strings.Title(param.Name),
				GoType:    g.paramTypeToGoType(param.Type),
			})
		}

		routes = append(routes, rd)
	}

	// Create output directory
	if err := os.MkdirAll("router", 0755); err != nil {
		return err
	}

	// Generate code
	t := template.Must(template.New("params").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]any{"Routes": routes}); err != nil {
		return err
	}

	// Format code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write file
	return os.WriteFile("router/params.go", formatted, 0644)
}

// generatePaths generates type-safe path helper functions
func (g *CodeGenerator) generatePaths() error {
	tmpl := `// Code generated by vango; DO NOT EDIT.

package router

import (
	"fmt"
	"strings"
)

{{ range .Routes }}
{{ if .HasParams }}
// {{ .FuncName }} generates a path for {{ .Path }}
func {{ .FuncName }}({{ .ParamArgs }}) string {
	path := "{{ .PathTemplate }}"
{{ range .Replacements }}	path = strings.Replace(path, "{{ .Pattern }}", {{ .Value }}, 1)
{{ end }}	return path
}
{{ else }}
// {{ .FuncName }} returns the path for {{ .Path }}
func {{ .FuncName }}() string {
	return "{{ .Path }}"
}
{{ end }}
{{ end }}
`

	// Generate path helpers
	type routeData struct {
		Path         string
		FuncName     string
		HasParams    bool
		ParamArgs    string
		PathTemplate string
		Replacements []struct {
			Pattern string
			Value   string
		}
	}

	routes := make([]routeData, 0)

	for _, route := range g.routes {
		rd := routeData{
			Path:         route.Path,
			FuncName:     g.pathToFuncName(route.Path),
			HasParams:    len(route.Params) > 0,
			PathTemplate: route.Path,
		}

		if len(route.Params) > 0 {
			// Build parameter arguments
			args := make([]string, 0)
			for _, param := range route.Params {
				goType := g.paramTypeToGoType(param.Type)
				args = append(args, fmt.Sprintf("%s %s", param.Name, goType))
			}
			rd.ParamArgs = strings.Join(args, ", ")

			// Build replacements
			for _, param := range route.Params {
				pattern := fmt.Sprintf("[%s]", param.Name)
				if param.Type != "string" {
					pattern = fmt.Sprintf("[%s:%s]", param.Name, param.Type)
				}

				value := param.Name
				if param.Type == "int" || param.Type == "int64" {
					value = fmt.Sprintf("fmt.Sprintf(\"%%d\", %s)", param.Name)
				}

				rd.Replacements = append(rd.Replacements, struct {
					Pattern string
					Value   string
				}{
					Pattern: pattern,
					Value:   value,
				})
			}
		}

		routes = append(routes, rd)
	}

	// Sort routes for consistent output
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	// Generate code
	t := template.Must(template.New("paths").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]any{"Routes": routes}); err != nil {
		return err
	}

	// Format code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write file
	return os.WriteFile("router/paths.go", formatted, 0644)
}

// generateRouterTable generates the router table JSON
func (g *CodeGenerator) generateRouterTable() error {
	type paramDef struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	type routeEntry struct {
		Path       string     `json:"path"`
		Component  string     `json:"component"`
		Params     []paramDef `json:"params,omitempty"`
		Middleware []string   `json:"middleware,omitempty"`
	}

	table := struct {
		Routes []routeEntry `json:"routes"`
	}{
		Routes: make([]routeEntry, 0),
	}

	for _, route := range g.routes {
		entry := routeEntry{
			Path:      route.Path,
			Component: route.ComponentName,
			Params:    make([]paramDef, 0),
		}

		for _, param := range route.Params {
			if !param.IsCatchAll {
				entry.Params = append(entry.Params, paramDef{
					Name: param.Name,
					Type: param.Type,
				})
			}
		}

		if route.HasMiddleware {
			entry.Middleware = append(entry.Middleware, route.MiddlewarePath)
		}

		table.Routes = append(table.Routes, entry)
	}

	// Sort routes for consistent output
	sort.Slice(table.Routes, func(i, j int) bool {
		return table.Routes[i].Path < table.Routes[j].Path
	})

	// Marshal to JSON
	data, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return err
	}

	// Write file
	return os.WriteFile("router/table.json", data, 0644)
}

// Helper functions

func (g *CodeGenerator) pathToStructName(path string) string {
	// Convert path like "/blog/[slug]" to "BlogSlug"
	parts := strings.Split(path, "/")
	name := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Remove brackets from params
		part = strings.Trim(part, "[]")
		// Remove param type suffix
		if idx := strings.Index(part, ":"); idx > 0 {
			part = part[:idx]
		}
		// Title case
		name += strings.Title(part)
	}

	if name == "" {
		name = "Index"
	}

	return name
}

func (g *CodeGenerator) pathToFuncName(path string) string {
	if path == "/" {
		return "Index"
	}
	return g.pathToStructName(path)
}

func (g *CodeGenerator) paramTypeToGoType(paramType string) string {
	switch paramType {
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "uuid":
		return "string" // For now, treat UUID as string
	default:
		return "string"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
