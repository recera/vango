package routes

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"runtime"
	"strings"
	"text/template"

	"github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango/vdom"
)

// DynamicLoader loads and executes route handlers dynamically
type DynamicLoader struct {
	buildDir      string
	routes        []RouteFile
	router        *server.Router
	liveServer    *live.Server
	bridge        *live.SchedulerBridge
	registry      *server.ComponentRegistry
	useReflection bool // Use reflection instead of plugins (more portable)
}

// NewDynamicLoader creates a new dynamic loader
func NewDynamicLoader(routesDir string, liveServer *live.Server) (*DynamicLoader, error) {
	scanner, err := NewScanner(routesDir)
	if err != nil {
		return nil, err
	}

	routes, err := scanner.ScanRoutes()
	if err != nil {
		return nil, err
	}

	// Create build directory
	buildDir := filepath.Join(".vango", "build", "dynamic")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}

	return &DynamicLoader{
		buildDir:      buildDir,
		routes:        routes,
		router:        server.NewRouter(),
		liveServer:    liveServer,
		bridge:        live.NewSchedulerBridge(liveServer),
		registry:      server.NewComponentRegistry(),
		useReflection: true, // Default to reflection for better portability
	}, nil
}

// LoadRoutes dynamically loads all route handlers
func (l *DynamicLoader) LoadRoutes() error {
	// Determine loading strategy based on platform
	if runtime.GOOS == "windows" || l.useReflection {
		return l.loadWithReflection()
	}
	return l.loadWithPlugin()
}

// loadWithReflection uses build-time code generation and reflection
func (l *DynamicLoader) loadWithReflection() error {
	// Generate wrapper code that imports the actual route handlers
	if err := l.generateWrapperCode(); err != nil {
		return fmt.Errorf("failed to generate wrapper code: %w", err)
	}

	// For each route, create a handler that connects to the live protocol
	for _, route := range l.routes {
		routeCopy := route // Capture for closure

		if route.HasServer {
			// Server-driven component (inject minimal client)
			l.router.AddRoute(route.URLPattern, func(ctx server.Ctx) (*vdom.VNode, error) {
				// Get or create session
				sessionID := l.getSessionID(ctx)

				// Create scheduler for this session if not exists
				if _, exists := l.liveServer.GetSession(sessionID); !exists {
					// Create a new session
					// This would normally happen when WebSocket connects
					log.Printf("Creating session for %s", sessionID)
				}

				// Render server-driven shell and inject client
				vnode, err := l.renderServerComponent(ctx, routeCopy, sessionID)
				if err != nil {
					return nil, err
				}
				// Ensure live minimal client is injected (defensive)
				vnode = server.InjectServerDrivenClient(vnode, sessionID)
				return vnode, nil
			})
		} else if route.IsAPI {
			// API route
			l.router.AddAPIRoute(route.URLPattern, func(ctx server.Ctx) (any, error) {
				// TODO: Call actual API handler
				return map[string]interface{}{
					"message": fmt.Sprintf("API route: %s", routeCopy.URLPattern),
					"handler": routeCopy.HandlerName,
				}, nil
			})
		} else if route.HasClient && !route.HasServer {
			// Client-only components: do not register here in dev live loader.
			// Let the dev server serve index.html and let the WASM client handle CSR routing.
			log.Printf("Client-only route (no server handler): %s -> %s",
				route.URLPattern, route.HandlerName)
		} else {
			// Default: SSR/Universal route (no explicit markers means server-side)
			l.router.AddRoute(route.URLPattern, func(ctx server.Ctx) (*vdom.VNode, error) {
				// For now, render using the actual handler
				// In a full implementation, this would call the actual compiled handler
				vnode, err := l.renderUniversalComponent(ctx, routeCopy)
				if err != nil {
					return nil, err
				}
				return vnode, nil
			})
		}

		log.Printf("Loaded route: %s -> %s (server=%v)",
			route.URLPattern, route.HandlerName, route.HasServer)
	}

	return nil
}

// loadWithPlugin uses Go plugins (Unix only)
func (l *DynamicLoader) loadWithPlugin() error {
	// Generate plugin code
	if err := l.generatePluginCode(); err != nil {
		return fmt.Errorf("failed to generate plugin code: %w", err)
	}

	// Build the plugin
	pluginPath := filepath.Join(l.buildDir, "routes.so")
	cmd := exec.Command("go", "build",
		"-buildmode=plugin",
		"-tags", "vango_server",
		"-o", pluginPath,
		filepath.Join(l.buildDir, "plugin_wrapper.go"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build plugin: %w", err)
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Get the handlers map
	handlersSymbol, err := p.Lookup("Handlers")
	if err != nil {
		return fmt.Errorf("failed to find Handlers in plugin: %w", err)
	}

	handlers, ok := handlersSymbol.(*map[string]server.HandlerFunc)
	if !ok {
		return fmt.Errorf("Handlers has unexpected type")
	}

	// Register all handlers
	for pattern, handler := range *handlers {
		l.router.AddRoute(pattern, handler)
		log.Printf("Loaded route from plugin: %s", pattern)
	}

	return nil
}

// generateWrapperCode generates wrapper code for reflection-based loading
func (l *DynamicLoader) generateWrapperCode() error {
	tmplStr := `// Code generated by vango dev; DO NOT EDIT.
package main

import (
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango/vdom"
{{range .Imports}}
	{{.Alias}} "{{.Path}}"
{{end}}
)

// RouteHandlers contains all route handlers
var RouteHandlers = map[string]server.HandlerFunc{
{{range .Routes}}
	"{{.URLPattern}}": wrap{{.SafeName}}Handler,
{{end}}
}

{{range .Routes}}
// wrap{{.SafeName}}Handler wraps the {{.HandlerName}} function
func wrap{{.SafeName}}Handler(ctx server.Ctx) (*vdom.VNode, error) {
	{{if .HasServer}}
	// Server-driven component
	return {{.ImportAlias}}.{{.HandlerName}}(ctx)
	{{else}}
	// Universal component - call with no context
	node := {{.ImportAlias}}.{{.HandlerName}}()
	return &node, nil
	{{end}}
}
{{end}}
`

	// Prepare template data
	type Import struct {
		Alias string
		Path  string
	}

	type RouteData struct {
		URLPattern  string
		SafeName    string // Safe name for Go function
		HandlerName string
		ImportAlias string
		HasServer   bool
	}

	imports := []Import{}
	routeData := []RouteData{}
	importMap := make(map[string]string)

	// Process routes
	for i, route := range l.routes {
		// Skip client-only routes
		if route.HasClient && !route.HasServer {
			continue
		}

		// Generate import alias
		alias, exists := importMap[route.ImportPath]
		if !exists {
			alias = fmt.Sprintf("routes%d", i)
			importMap[route.ImportPath] = alias
			imports = append(imports, Import{
				Alias: alias,
				Path:  route.ImportPath,
			})
		}

		// Generate safe name for function
		safeName := strings.ReplaceAll(route.URLPattern, "/", "_")
		safeName = strings.ReplaceAll(safeName, ":", "_")
		safeName = strings.ReplaceAll(safeName, "*", "_star_")
		safeName = strings.Title(safeName)

		routeData = append(routeData, RouteData{
			URLPattern:  route.URLPattern,
			SafeName:    safeName,
			HandlerName: route.HandlerName,
			ImportAlias: alias,
			HasServer:   route.HasServer,
		})
	}

	// Render template
	tmpl, err := template.New("wrapper").Parse(tmplStr)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Imports": imports,
		"Routes":  routeData,
	})
	if err != nil {
		return err
	}

	// Write to file
	outputPath := filepath.Join(l.buildDir, "wrapper_gen.go")
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// generatePluginCode generates plugin wrapper code
func (l *DynamicLoader) generatePluginCode() error {
	// Similar to generateWrapperCode but exports Handlers variable
	// Implementation omitted for brevity
	return l.generateWrapperCode() // Reuse for now
}

// renderServerComponent renders a server-driven component
func (l *DynamicLoader) renderServerComponent(ctx server.Ctx, route RouteFile, sessionID string) (*vdom.VNode, error) {
	// Create the HTML structure with hydration points
	html := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "html",
		Kids: []vdom.VNode{
			{
				Kind: vdom.KindElement,
				Tag:  "head",
				Kids: []vdom.VNode{
					{
						Kind: vdom.KindElement,
						Tag:  "title",
						Kids: []vdom.VNode{{Kind: vdom.KindText, Text: "Server Component"}},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "meta",
						Props: vdom.Props{
							"name":    "vango-session",
							"content": sessionID,
						},
					},
				},
			},
			{
				Kind: vdom.KindElement,
				Tag:  "body",
				Kids: []vdom.VNode{
					{
						Kind: vdom.KindElement,
						Tag:  "div",
						Props: vdom.Props{
							"id":       "app",
							"data-hid": "app",
						},
						Kids: []vdom.VNode{
							{
								Kind:  vdom.KindElement,
								Tag:   "div",
								Props: vdom.Props{"class": "loading"},
								Kids: []vdom.VNode{
									{Kind: vdom.KindText, Text: "Loading server component..."},
								},
							},
						},
					},
				},
			},
		},
	}

	// Inject the server-driven client script
	html = server.InjectServerDrivenClient(html, sessionID)

	return html, nil
}

// renderUniversalComponent renders a universal component
func (l *DynamicLoader) renderUniversalComponent(ctx server.Ctx, route RouteFile) (*vdom.VNode, error) {
	// For SSR routes, we need to render the full HTML page with the WASM app
	// The WASM app will hydrate and take over client-side routing
	
	// Import the actual route handler if it's the index page
	// For demonstration, we'll inline the IndexPage content
	if route.URLPattern == "/" && route.HandlerName == "IndexPage" {
		// Import and call the actual IndexPage function
		// Note: This requires compile-time linking in production
		// For now, return the HTML shell that loads the WASM app
		return l.renderSSRShell(ctx, route)
	}
	
	// For other routes, render SSR shell
	return l.renderSSRShell(ctx, route)
}

// renderSSRShell renders the HTML shell for SSR routes
func (l *DynamicLoader) renderSSRShell(ctx server.Ctx, route RouteFile) (*vdom.VNode, error) {
	// Create the HTML structure that loads the WASM app
	// The WASM app will hydrate and render the actual content
	html := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "html",
		Props: vdom.Props{
			"lang": "en",
		},
		Kids: []vdom.VNode{
			{
				Kind: vdom.KindElement,
				Tag:  "head",
				Kids: []vdom.VNode{
					{
						Kind: vdom.KindElement,
						Tag:  "meta",
						Props: vdom.Props{
							"charset": "UTF-8",
						},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "meta",
						Props: vdom.Props{
							"name":    "viewport",
							"content": "width=device-width, initial-scale=1.0",
						},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "title",
						Kids: []vdom.VNode{{Kind: vdom.KindText, Text: "Vango App"}},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "link",
						Props: vdom.Props{
							"rel":  "stylesheet",
							"href": "/styles/base.css",
						},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "link",
						Props: vdom.Props{
							"rel":  "stylesheet",
							"href": "/styles.css",
						},
					},
					{
						Kind: vdom.KindElement,
						Tag:  "script",
						Props: vdom.Props{
							"src":   "/vango/bootstrap.js",
							"defer": "defer",
						},
					},
				},
			},
			{
				Kind: vdom.KindElement,
				Tag:  "body",
				Kids: []vdom.VNode{
					{
						Kind: vdom.KindElement,
						Tag:  "div",
						Props: vdom.Props{
							"id": "app",
						},
						Kids: []vdom.VNode{
							// Initial loading state
							{
								Kind: vdom.KindElement,
								Tag:  "div",
								Props: vdom.Props{
									"class": "min-h-screen flex items-center justify-center",
								},
								Kids: []vdom.VNode{
									{
										Kind: vdom.KindElement,
										Tag:  "div",
										Props: vdom.Props{
											"class": "text-center",
										},
										Kids: []vdom.VNode{
											{
												Kind: vdom.KindElement,
												Tag:  "div",
												Props: vdom.Props{
													"class": "text-4xl mb-4",
												},
												Kids: []vdom.VNode{{Kind: vdom.KindText, Text: "⚡"}},
											},
											{
												Kind: vdom.KindElement,
												Tag:  "p",
												Props: vdom.Props{
													"class": "text-gray-600",
												},
												Kids: []vdom.VNode{{Kind: vdom.KindText, Text: "Loading Vango..."}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	return html, nil
}

// getSessionID gets or creates a session ID for the request
func (l *DynamicLoader) getSessionID(ctx server.Ctx) string {
	// Check for existing session ID in header or cookie
	sessionID := ctx.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		// Check cookie
		if cookie, err := ctx.Request().Cookie("vango-session"); err == nil {
			sessionID = cookie.Value
		}
	}

	if sessionID == "" {
		// Generate new session ID
		sessionID = fmt.Sprintf("session_%d", generateUniqueID())

		// Set cookie
		ctx.SetHeader("Set-Cookie", fmt.Sprintf("vango-session=%s; Path=/; HttpOnly", sessionID))
	}

	return sessionID
}

// generateUniqueID generates a unique ID
func generateUniqueID() uint32 {
	// In production, use a proper UUID generator
	return uint32(os.Getpid()) ^ uint32(runtime.NumGoroutine())
}

// GetRouter returns the router with loaded routes
func (l *DynamicLoader) GetRouter() *server.Router {
	return l.router
}

// ServeHTTP implements http.Handler
func (l *DynamicLoader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.router.ServeHTTP(w, r)
}

// FindHandlerByReflection finds a handler function using reflection
func (l *DynamicLoader) FindHandlerByReflection(packagePath, handlerName string) (reflect.Value, error) {
	// This is a simplified approach
	// In a real implementation, we'd need to:
	// 1. Build the package if needed
	// 2. Load the symbols
	// 3. Find the function by name

	// For now, return an error indicating this needs implementation
	return reflect.Value{}, fmt.Errorf("reflection loading not yet implemented for %s.%s", packagePath, handlerName)
}

// analyzeRouteSignature analyzes a route handler's signature
func analyzeRouteSignature(filePath string, handlerName string) (needsContext bool, returnsError bool, err error) {
	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false, false, err
	}

	// Find the handler function
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == handlerName {
			// Check parameters
			if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
				needsContext = true
			}

			// Check return types
			if fn.Type.Results != nil && len(fn.Type.Results.List) > 1 {
				returnsError = true
			}

			return needsContext, returnsError, nil
		}
	}

	return false, false, fmt.Errorf("handler %s not found", handlerName)
}
