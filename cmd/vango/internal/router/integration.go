package router

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// RouteManifest contains all discovered routes and metadata
type RouteManifest struct {
	Routes      []RouteInfo `json:"routes"`
	HasAPI      bool        `json:"hasApi"`
	HasLayouts  bool        `json:"hasLayouts"`
	TotalRoutes int         `json:"totalRoutes"`
}

// ScanRoutes scans the routes directory and returns a manifest
func ScanRoutes(routesDir string) (*RouteManifest, error) {
	// Ensure routes directory exists
	if _, err := os.Stat(routesDir); os.IsNotExist(err) {
		// Create empty routes directory if it doesn't exist
		if err := os.MkdirAll(routesDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create routes directory: %w", err)
		}
		log.Printf("📁 Created routes directory at %s", routesDir)
		
		// Return empty manifest for new project
		return &RouteManifest{
			Routes:      []RouteInfo{},
			TotalRoutes: 0,
		}, nil
	}

	scanner := NewScanner(routesDir)
	routes, err := scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("failed to scan routes: %w", err)
	}

	// Check if scanner had any errors
	if errors := scanner.GetErrors(); len(errors) > 0 {
		for _, err := range errors {
			log.Printf("⚠️  Route scan warning: %v", err)
		}
	}

	// Build manifest
	manifest := &RouteManifest{
		Routes:      routes,
		TotalRoutes: len(routes),
	}

	// Check for special route types
	for _, route := range routes {
		if route.IsAPI {
			manifest.HasAPI = true
		}
		if route.HasLayout {
			manifest.HasLayouts = true
		}
	}

	return manifest, nil
}

// GenerateRouteTree generates all routing code from a manifest
func GenerateRouteTree(manifest *RouteManifest) error {
	if manifest == nil || len(manifest.Routes) == 0 {
		log.Println("📝 No routes to generate")
		return nil
	}

	// Convert RouteInfo to Route for code generator
	generator := NewCodeGenerator("app/routes", "router")
	
	for _, info := range manifest.Routes {
		route := Route{
			Path:           info.URLPath,
			FilePath:       info.FilePath,
			Package:        info.PackageName,
			ComponentName:  info.HandlerName,
			IsAPI:          info.IsAPI,
			HasLayout:      info.HasLayout,
			HasMiddleware:  info.HasMiddleware,
		}
		
		// Convert ParamInfo to RouteParam
		for _, param := range info.Params {
			route.Params = append(route.Params, RouteParam{
				Name:       param.Name,
				Type:       param.Type,
				IsCatchAll: info.IsCatchAll && param.Position == len(info.Params)-1,
			})
		}
		
		generator.routes = append(generator.routes, route)
	}

	// Generate all routing code
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("failed to generate routing code: %w", err)
	}

	log.Printf("✅ Generated routing code for %d routes", len(manifest.Routes))
	return nil
}

// GenerateClientRouteTable generates a JSON route table for the client
func GenerateClientRouteTable(manifest *RouteManifest) error {
	// Create router directory if it doesn't exist
	if err := os.MkdirAll("router", 0755); err != nil {
		return fmt.Errorf("failed to create router directory: %w", err)
	}

	// Build client-safe route table
	type ClientRoute struct {
		Path       string   `json:"path"`
		Component  string   `json:"component"`
		Params     []string `json:"params,omitempty"`
		IsAPI      bool     `json:"isApi,omitempty"`
		HasLayout  bool     `json:"hasLayout,omitempty"`
	}

	clientRoutes := []ClientRoute{}
	for _, route := range manifest.Routes {
		cr := ClientRoute{
			Path:      route.URLPath,
			Component: route.HandlerName,
			IsAPI:     route.IsAPI,
			HasLayout: route.HasLayout,
		}
		
		// Add param names
		for _, param := range route.Params {
			cr.Params = append(cr.Params, param.Name)
		}
		
		clientRoutes = append(clientRoutes, cr)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(map[string]interface{}{
		"routes": clientRoutes,
		"generated": true,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal route table: %w", err)
	}

	// Write to file
	tablePath := filepath.Join("router", "table.json")
	if err := os.WriteFile(tablePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write route table: %w", err)
	}

	return nil
}

// WatchRoutesDirectory sets up file watching for the routes directory
func WatchRoutesDirectory(routesDir string, onChange func()) error {
	// This would be called from the dev server's file watcher
	// The actual implementation is in the dev server
	return nil
}

// LoadManifest loads a previously saved route manifest
func LoadManifest(path string) (*RouteManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest RouteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// SaveManifest saves a route manifest to disk
func (m *RouteManifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}