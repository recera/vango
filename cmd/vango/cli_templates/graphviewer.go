package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register("graphviewer", &GraphViewerTemplate{})
}

// GraphViewerTemplate generates a knowledge graph viewer application
type GraphViewerTemplate struct{}

func (t *GraphViewerTemplate) Name() string {
	return "graphviewer"
}

func (t *GraphViewerTemplate) Description() string {
	return "Interactive knowledge graph visualization with multiple graph examples"
}

func (t *GraphViewerTemplate) Generate(config *ProjectConfig) error {
	// Force Tailwind for graphviewer template
	config.UseTailwind = true
	// Use standalone to avoid relying on local Node/npm and ensure
	// the CLI auto-downloads a Tailwind binary when needed
	config.TailwindStrategy = "standalone"
	config.DarkMode = true

	// Create necessary directories
	dirs := []string{
		"app/data",
		"app/routes",
		"styles",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(config.Directory, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create main.go for graph viewer app
	if err := t.createMainFile(config); err != nil {
		return err
	}

	// Create graph data
	if err := t.createGraphData(config); err != nil {
		return err
	}

	// Create graph viewer route
	if err := t.createGraphRoute(config); err != nil {
		return err
	}

	// Create enhanced styles
	if err := t.createEnhancedStyles(config); err != nil {
		return err
	}

	// Create tailwind config (always for graphviewer)
	if err := t.createTailwindConfig(config); err != nil {
		return err
	}

	return nil
}

func (t *GraphViewerTemplate) createMainFile(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
    "fmt"
    "strings"
    "syscall/js"
    
    // Import our packages
    routes "%s/app/routes"
    "github.com/recera/vango/pkg/vango/vdom"
)

// Retained handlers to prevent GC
var retainedHandlers []js.Func

func main() {
	js.Global().Get("console").Call("log", "🚀 Knowledge Graph Viewer starting...")
	
	// Initialize app
	initApp()
	
	// Keep the WASM runtime alive
	select {}
}

func initApp() {
	document := js.Global().Get("document")
	
	// Wait for DOM ready
	if document.Get("readyState").String() != "loading" {
		onReady()
	} else {
		document.Call("addEventListener", "DOMContentLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			onReady()
			return nil
		}))
	}
}

func onReady() {
	console := js.Global().Get("console")
	console.Call("log", "DOM ready, initializing Graph Viewer...")
	document := js.Global().Get("document")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Get current route
	path := js.Global().Get("window").Get("location").Get("pathname").String()
	
	// Simple routing - determine which graph to show
	graphType := "programming" // default
	if strings.HasPrefix(path, "/graph/") {
		graphType = strings.TrimPrefix(path, "/graph/")
	}
	
	console.Call("log", "Loading graph type:", graphType)
	
	// Create the graph viewer VNode
	vnode := routes.GraphViewerWithType(graphType)
	
	// Mount to DOM
	app := document.Call("getElementById", "app")
	if !app.Truthy() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Clear the app container
	app.Set("innerHTML", "")
	
	// Create a DOM applier to render the VNode
	renderVNodeToDOM(vnode, app)
	
	// Setup navigation
	setupNavigation()
	
	// Setup event handlers after rendering
	setupEventHandlers()
}

// renderVNodeToDOM renders a VNode to a DOM element
func renderVNodeToDOM(vnode *vdom.VNode, container js.Value) {
	if vnode == nil {
		return
	}
	
	document := js.Global().Get("document")
	console := js.Global().Get("console")

    // Release previously retained handlers (avoid leaks on re-render)
    for _, f := range retainedHandlers {
        f.Release()
    }
    retainedHandlers = nil

    // Create DOM element from VNode
    domElement := createDOMFromVNode(vnode, document)
	if domElement.Truthy() {
		container.Call("appendChild", domElement)
		console.Call("log", "Rendered VNode to DOM")
	} else {
		console.Call("error", "Failed to create DOM from VNode")
	}
}

// createDOMFromVNode recursively creates DOM elements from a VNode tree
func createDOMFromVNode(vnode *vdom.VNode, document js.Value) js.Value {
	if vnode == nil {
		return js.Undefined()
	}
	
	switch vnode.Kind {
	case vdom.KindText:
		return document.Call("createTextNode", vnode.Text)
		
	case vdom.KindElement:
		elem := document.Call("createElement", vnode.Tag)
		
        // Set attributes and handle events
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				// Handle event handlers specially
				if strings.HasPrefix(key, "on") {
					eventName := strings.ToLower(key[2:]) // onclick -> click
                    var jsFunc js.Func
                    if strHandler, ok := value.(string); ok {
                        funcName := strings.TrimSuffix(strHandler, "()")
                        jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                            if fn := js.Global().Get(funcName); fn.Truthy() { fn.Invoke() }
                            return nil
                        })
                    } else {
                        switch h := value.(type) {
                        case func():
                            jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} { h(); return nil })
                        case func(js.Value):
                            jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                                if len(args) > 0 { h(args[0]) } else { h(js.Undefined()) }
                                return nil
                            })
                        case func(float64, float64):
                            jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                                var x, y float64
                                if len(args) > 0 {
                                    ev := args[0]
                                    bx, by := 0.0, 0.0
                                    if this.Truthy() {
                                        rect := this.Call("getBoundingClientRect")
                                        bx = rect.Get("left").Float()
                                        by = rect.Get("top").Float()
                                    }
                                    x = ev.Get("clientX").Float() - bx
                                    y = ev.Get("clientY").Float() - by
                                }
                                h(x, y); return nil
                            })
                        case func(float64):
                            jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                                d := 0.0; if len(args) > 0 { d = args[0].Get("deltaY").Float() }
                                h(d); return nil
                            })
                        case func(string):
                            jsFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
                                s := ""
                                if len(args) > 0 {
                                    ev := args[0]
                                    switch eventName {
                                    case "input", "change":
                                        tgt := ev.Get("target"); if tgt.Truthy() { s = tgt.Get("value").String() }
                                    case "keydown", "keyup", "keypress":
                                        s = ev.Get("key").String()
                                    default:
                                        s = ev.Get("type").String()
                                    }
                                }
                                h(s); return nil
                            })
                        }
                    }
                    if jsFunc.Truthy() {
                        elem.Call("addEventListener", eventName, jsFunc)
                        // Store on the element to keep from GC in TinyGo
                        if elem.Truthy() {
                            elem.Set("__vangoHandler__"+eventName, jsFunc)
                        }
                        retainedHandlers = append(retainedHandlers, jsFunc)
                    }
					continue
				}
				
				// Skip refs for special handling
				if key == "ref" {
					continue
				}
				
				// Handle regular attributes
				switch key {
				case "class":
					elem.Set("className", value)
				case "style":
					elem.Set("style", value)
				default:
					elem.Call("setAttribute", key, value)
				}
			}
			
			// Handle ref callback if present
			if ref, ok := vnode.Props["ref"]; ok {
				if refFunc, ok := ref.(func(js.Value)); ok {
					refFunc(elem)
				}
			}
		}
		
		// Add children
		for _, child := range vnode.Kids {
			childDOM := createDOMFromVNode(&child, document)
			if childDOM.Truthy() {
				elem.Call("appendChild", childDOM)
			}
		}
		
		return elem
		
	default:
		return js.Undefined()
	}
}

func initDarkMode() {
	document := js.Global().Get("document")
	localStorage := js.Global().Get("localStorage")
	
	// Check localStorage first
	darkMode := localStorage.Call("getItem", "darkMode").String()
	
	if darkMode == "true" {
		document.Get("documentElement").Get("classList").Call("add", "dark")
	} else if darkMode == "false" {
		document.Get("documentElement").Get("classList").Call("remove", "dark")
	} else {
		// Check system preference if no stored preference
		if js.Global().Get("window").Get("matchMedia").Truthy() {
			prefersDark := js.Global().Get("window").Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Bool()
			if prefersDark {
				document.Get("documentElement").Get("classList").Call("add", "dark")
			} else {
				document.Get("documentElement").Get("classList").Call("remove", "dark")
			}
		}
	}
}

// Global function for dark mode toggle
func toggleDarkMode() {
	document := js.Global().Get("document")
	localStorage := js.Global().Get("localStorage")
	
	isDark := document.Get("documentElement").Get("classList").Call("contains", "dark").Bool()
	
	if isDark {
		document.Get("documentElement").Get("classList").Call("remove", "dark")
		localStorage.Call("setItem", "darkMode", "false")
	} else {
		document.Get("documentElement").Get("classList").Call("add", "dark")
		localStorage.Call("setItem", "darkMode", "true")
	}

    // Re-render to propagate theme into graph options immediately
    onReady()
}

func setupNavigation() {
	// Setup client-side navigation
	window := js.Global().Get("window")
	
	// Export navigation function to global scope
	js.Global().Set("navigateTo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			path := args[0].String()
			window.Get("history").Call("pushState", nil, "", path)
			onReady() // Re-render with new route
		}
		return nil
	}))
	
	// Export dark mode toggle
	js.Global().Set("toggleDarkMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		toggleDarkMode()
		return nil
	}))
}

func setupEventHandlers() {
	document := js.Global().Get("document")
	console := js.Global().Get("console")
	
	// Setup dark mode toggle button
	darkModeBtn := document.Call("getElementById", "dark-mode-toggle")
	if darkModeBtn.Truthy() {
		darkModeBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			console.Call("log", "Dark mode button clicked")
			toggleDarkMode()
			return nil
		}))
		console.Call("log", "Dark mode button handler attached")
	} else {
		console.Call("error", "Dark mode button not found")
	}
	
	// Setup graph tab buttons
	graphTypes := []string{"programming", "social", "tech", "vango"}
	for _, graphType := range graphTypes {
		tabID := fmt.Sprintf("tab-%%s", graphType)
		tabBtn := document.Call("getElementById", tabID)
		if tabBtn.Truthy() {
			// Create a closure to capture the graphType value
			(func(gt string) {
				tabBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					console.Call("log", "Tab clicked:", gt)
					path := fmt.Sprintf("/graph/%%s", gt)
					js.Global().Get("window").Get("history").Call("pushState", nil, "", path)
					onReady() // Re-render with new route
					return nil
				}))
			})(graphType)
			console.Call("log", "Tab handler attached for:", graphType)
		} else {
			console.Call("error", "Tab button not found:", tabID)
		}
	}
}
`, config.Module)

	return WriteFile(filepath.Join(config.Directory, "app/main.go"), content)
}

func (t *GraphViewerTemplate) createGraphData(config *ProjectConfig) error {
	content := `package data

import (
	"github.com/recera/vango/pkg/components/graphviewer"
)

// GetProgrammingConceptsGraph returns a graph of programming concepts and their relationships
func GetProgrammingConceptsGraph() graphviewer.Data {
	return graphviewer.Data{
		Nodes: []graphviewer.Node{
			// Core concepts
			{ID: "prog", Label: "Programming", X: 0, Y: 0, Size: 15, Color: "#ff6b6b"},
			
			// Paradigms
			{ID: "oop", Label: "OOP", X: -150, Y: -100, Size: 12, Color: "#4ecdc4"},
			{ID: "func", Label: "Functional", X: -150, Y: 100, Size: 12, Color: "#4ecdc4"},
			{ID: "proc", Label: "Procedural", X: 150, Y: -100, Size: 12, Color: "#4ecdc4"},
			{ID: "logic", Label: "Logic", X: 150, Y: 100, Size: 12, Color: "#4ecdc4"},
			
			// OOP concepts
			{ID: "class", Label: "Classes", X: -250, Y: -150, Size: 10, Color: "#95e1d3"},
			{ID: "inherit", Label: "Inheritance", X: -300, Y: -100, Size: 10, Color: "#95e1d3"},
			{ID: "poly", Label: "Polymorphism", X: -250, Y: -50, Size: 10, Color: "#95e1d3"},
			{ID: "encap", Label: "Encapsulation", X: -200, Y: -150, Size: 10, Color: "#95e1d3"},
			
			// Functional concepts
			{ID: "lambda", Label: "Lambda", X: -250, Y: 150, Size: 10, Color: "#f38181"},
			{ID: "pure", Label: "Pure Functions", X: -300, Y: 100, Size: 10, Color: "#f38181"},
			{ID: "immut", Label: "Immutability", X: -250, Y: 50, Size: 10, Color: "#f38181"},
			{ID: "monad", Label: "Monads", X: -200, Y: 150, Size: 10, Color: "#f38181"},
			
			// Data structures
			{ID: "ds", Label: "Data Structures", X: 0, Y: -200, Size: 12, Color: "#feca57"},
			{ID: "array", Label: "Arrays", X: -50, Y: -250, Size: 9, Color: "#ff9ff3"},
			{ID: "list", Label: "Lists", X: 0, Y: -280, Size: 9, Color: "#ff9ff3"},
			{ID: "tree", Label: "Trees", X: 50, Y: -250, Size: 9, Color: "#ff9ff3"},
			{ID: "graph", Label: "Graphs", X: 100, Y: -200, Size: 9, Color: "#ff9ff3"},
			{ID: "hash", Label: "Hash Tables", X: -100, Y: -200, Size: 9, Color: "#ff9ff3"},
			
			// Algorithms
			{ID: "algo", Label: "Algorithms", X: 0, Y: 200, Size: 12, Color: "#54a0ff"},
			{ID: "sort", Label: "Sorting", X: -50, Y: 250, Size: 9, Color: "#48dbfb"},
			{ID: "search", Label: "Searching", X: 0, Y: 280, Size: 9, Color: "#48dbfb"},
			{ID: "dp", Label: "Dynamic Prog", X: 50, Y: 250, Size: 9, Color: "#48dbfb"},
			{ID: "greedy", Label: "Greedy", X: 100, Y: 200, Size: 9, Color: "#48dbfb"},
			{ID: "divide", Label: "Divide & Conquer", X: -100, Y: 200, Size: 9, Color: "#48dbfb"},
		},
		Edges: []graphviewer.Edge{
			// Main connections
			{Source: "prog", Target: "oop", Weight: 2},
			{Source: "prog", Target: "func", Weight: 2},
			{Source: "prog", Target: "proc", Weight: 2},
			{Source: "prog", Target: "logic", Weight: 2},
			{Source: "prog", Target: "ds", Weight: 2},
			{Source: "prog", Target: "algo", Weight: 2},
			
			// OOP connections
			{Source: "oop", Target: "class", Weight: 1},
			{Source: "oop", Target: "inherit", Weight: 1},
			{Source: "oop", Target: "poly", Weight: 1},
			{Source: "oop", Target: "encap", Weight: 1},
			{Source: "class", Target: "inherit", Weight: 0.5},
			{Source: "inherit", Target: "poly", Weight: 0.5},
			
			// Functional connections
			{Source: "func", Target: "lambda", Weight: 1},
			{Source: "func", Target: "pure", Weight: 1},
			{Source: "func", Target: "immut", Weight: 1},
			{Source: "func", Target: "monad", Weight: 1},
			{Source: "pure", Target: "immut", Weight: 0.5},
			{Source: "lambda", Target: "monad", Weight: 0.5},
			
			// Data structure connections
			{Source: "ds", Target: "array", Weight: 1},
			{Source: "ds", Target: "list", Weight: 1},
			{Source: "ds", Target: "tree", Weight: 1},
			{Source: "ds", Target: "graph", Weight: 1},
			{Source: "ds", Target: "hash", Weight: 1},
			{Source: "list", Target: "array", Weight: 0.3},
			{Source: "tree", Target: "graph", Weight: 0.5},
			
			// Algorithm connections
			{Source: "algo", Target: "sort", Weight: 1},
			{Source: "algo", Target: "search", Weight: 1},
			{Source: "algo", Target: "dp", Weight: 1},
			{Source: "algo", Target: "greedy", Weight: 1},
			{Source: "algo", Target: "divide", Weight: 1},
			{Source: "dp", Target: "divide", Weight: 0.3},
			{Source: "greedy", Target: "dp", Weight: 0.3},
			
			// Cross connections
			{Source: "ds", Target: "algo", Weight: 1.5},
			{Source: "tree", Target: "search", Weight: 0.5},
			{Source: "array", Target: "sort", Weight: 0.5},
			{Source: "hash", Target: "search", Weight: 0.5},
		},
	}
}

// GetSocialNetworkGraph returns a social network graph
func GetSocialNetworkGraph() graphviewer.Data {
	return graphviewer.Data{
		Nodes: []graphviewer.Node{
			// Central figures
			{ID: "alice", Label: "Alice", X: 0, Y: 0, Size: 14, Color: "#e74c3c"},
			{ID: "bob", Label: "Bob", X: 100, Y: 50, Size: 12, Color: "#3498db"},
			{ID: "carol", Label: "Carol", X: -100, Y: 50, Size: 12, Color: "#e67e22"},
			{ID: "david", Label: "David", X: 50, Y: -100, Size: 11, Color: "#2ecc71"},
			{ID: "eve", Label: "Eve", X: -50, Y: -100, Size: 11, Color: "#9b59b6"},
			
			// Alice's network
			{ID: "frank", Label: "Frank", X: -150, Y: -50, Size: 9, Color: "#1abc9c"},
			{ID: "grace", Label: "Grace", X: -200, Y: 0, Size: 9, Color: "#f39c12"},
			{ID: "henry", Label: "Henry", X: -150, Y: 100, Size: 9, Color: "#d35400"},
			
			// Bob's network
			{ID: "irene", Label: "Irene", X: 200, Y: 50, Size: 9, Color: "#c0392b"},
			{ID: "jack", Label: "Jack", X: 150, Y: 150, Size: 9, Color: "#7f8c8d"},
			{ID: "karen", Label: "Karen", X: 100, Y: 150, Size: 9, Color: "#34495e"},
			
			// Carol's network
			{ID: "liam", Label: "Liam", X: -100, Y: 150, Size: 9, Color: "#16a085"},
			{ID: "mary", Label: "Mary", X: -200, Y: 100, Size: 9, Color: "#27ae60"},
			
			// Community leaders
			{ID: "nathan", Label: "Nathan", X: 0, Y: 200, Size: 13, Color: "#e74c3c"},
			{ID: "olivia", Label: "Olivia", X: 0, Y: -200, Size: 13, Color: "#8e44ad"},
			
			// Extended network
			{ID: "paul", Label: "Paul", X: 150, Y: -150, Size: 8, Color: "#95a5a6"},
			{ID: "quinn", Label: "Quinn", X: -150, Y: -150, Size: 8, Color: "#2c3e50"},
			{ID: "rachel", Label: "Rachel", X: 250, Y: 0, Size: 8, Color: "#f1c40f"},
			{ID: "steve", Label: "Steve", X: -250, Y: 0, Size: 8, Color: "#e74c3c"},
			{ID: "tina", Label: "Tina", X: 100, Y: -200, Size: 8, Color: "#3498db"},
		},
		Edges: []graphviewer.Edge{
			// Core friendships
			{Source: "alice", Target: "bob", Weight: 2, Color: "#2ecc71"},
			{Source: "alice", Target: "carol", Weight: 2, Color: "#2ecc71"},
			{Source: "alice", Target: "david", Weight: 1.5},
			{Source: "alice", Target: "eve", Weight: 1.5},
			{Source: "bob", Target: "carol", Weight: 1},
			{Source: "bob", Target: "david", Weight: 1},
			{Source: "carol", Target: "eve", Weight: 1},
			{Source: "david", Target: "eve", Weight: 1},
			
			// Alice's connections
			{Source: "alice", Target: "frank", Weight: 1},
			{Source: "alice", Target: "grace", Weight: 1},
			{Source: "frank", Target: "grace", Weight: 0.5},
			{Source: "frank", Target: "henry", Weight: 0.5},
			{Source: "carol", Target: "henry", Weight: 1},
			
			// Bob's connections
			{Source: "bob", Target: "irene", Weight: 1},
			{Source: "bob", Target: "jack", Weight: 1},
			{Source: "bob", Target: "karen", Weight: 0.5},
			{Source: "irene", Target: "jack", Weight: 0.5},
			{Source: "jack", Target: "karen", Weight: 1},
			
			// Carol's connections
			{Source: "carol", Target: "liam", Weight: 1},
			{Source: "carol", Target: "mary", Weight: 1},
			{Source: "liam", Target: "mary", Weight: 0.5},
			{Source: "henry", Target: "mary", Weight: 0.5},
			
			// Community connections
			{Source: "nathan", Target: "liam", Weight: 1},
			{Source: "nathan", Target: "jack", Weight: 1},
			{Source: "nathan", Target: "karen", Weight: 1},
			{Source: "olivia", Target: "david", Weight: 1},
			{Source: "olivia", Target: "eve", Weight: 1},
			{Source: "olivia", Target: "paul", Weight: 1},
			{Source: "olivia", Target: "tina", Weight: 1},
			
			// Extended network
			{Source: "paul", Target: "david", Weight: 0.5},
			{Source: "quinn", Target: "eve", Weight: 0.5},
			{Source: "quinn", Target: "frank", Weight: 0.5},
			{Source: "rachel", Target: "irene", Weight: 0.5},
			{Source: "rachel", Target: "bob", Weight: 0.3},
			{Source: "steve", Target: "grace", Weight: 0.5},
			{Source: "steve", Target: "mary", Weight: 0.5},
			{Source: "tina", Target: "paul", Weight: 0.5},
			
			// Weak ties
			{Source: "nathan", Target: "olivia", Weight: 0.3, Color: "#95a5a6"},
		},
	}
}

// GetTechStackGraph returns a technology stack dependencies graph
func GetTechStackGraph() graphviewer.Data {
	return graphviewer.Data{
		Nodes: []graphviewer.Node{
			// Core layers
			{ID: "app", Label: "Application", X: 0, Y: -200, Size: 14, Color: "#6c5ce7"},
			
			// Frontend
			{ID: "frontend", Label: "Frontend", X: -150, Y: -100, Size: 12, Color: "#74b9ff"},
			{ID: "vango", Label: "Vango", X: -250, Y: -50, Size: 11, Color: "#00b894"},
			{ID: "wasm", Label: "WebAssembly", X: -200, Y: 0, Size: 10, Color: "#00cec9"},
			{ID: "tinygo", Label: "TinyGo", X: -150, Y: 50, Size: 10, Color: "#00cec9"},
			{ID: "tailwind", Label: "Tailwind CSS", X: -300, Y: -100, Size: 9, Color: "#fd79a8"},
			{ID: "canvas", Label: "Canvas API", X: -100, Y: 0, Size: 9, Color: "#fdcb6e"},
			
			// Backend
			{ID: "backend", Label: "Backend", X: 150, Y: -100, Size: 12, Color: "#a29bfe"},
			{ID: "golang", Label: "Go", X: 250, Y: -50, Size: 11, Color: "#00b894"},
			{ID: "api", Label: "REST API", X: 200, Y: 0, Size: 10, Color: "#ffeaa7"},
			{ID: "graphql", Label: "GraphQL", X: 150, Y: 0, Size: 10, Color: "#fab1a0"},
			{ID: "websocket", Label: "WebSocket", X: 100, Y: 50, Size: 10, Color: "#ff7675"},
			
			// Database
			{ID: "database", Label: "Database", X: 0, Y: 100, Size: 12, Color: "#e17055"},
			{ID: "postgres", Label: "PostgreSQL", X: -100, Y: 150, Size: 10, Color: "#0984e3"},
			{ID: "redis", Label: "Redis", X: 0, Y: 180, Size: 10, Color: "#d63031"},
			{ID: "mongo", Label: "MongoDB", X: 100, Y: 150, Size: 10, Color: "#00b894"},
			
			// Infrastructure
			{ID: "infra", Label: "Infrastructure", X: 0, Y: -50, Size: 12, Color: "#636e72"},
			{ID: "docker", Label: "Docker", X: -50, Y: 250, Size: 10, Color: "#0984e3"},
			{ID: "k8s", Label: "Kubernetes", X: 50, Y: 250, Size: 10, Color: "#5f3dc4"},
			{ID: "nginx", Label: "Nginx", X: 0, Y: 50, Size: 9, Color: "#00b894"},
			{ID: "cdn", Label: "CDN", X: -200, Y: -150, Size: 9, Color: "#e84393"},
			
			// Monitoring
			{ID: "monitor", Label: "Monitoring", X: 300, Y: 50, Size: 11, Color: "#f39c12"},
			{ID: "prometheus", Label: "Prometheus", X: 350, Y: 100, Size: 9, Color: "#e67e22"},
			{ID: "grafana", Label: "Grafana", X: 300, Y: 150, Size: 9, Color: "#27ae60"},
			{ID: "logging", Label: "Logging", X: 250, Y: 100, Size: 9, Color: "#3498db"},
		},
		Edges: []graphviewer.Edge{
			// Main architecture
			{Source: "app", Target: "frontend", Weight: 2},
			{Source: "app", Target: "backend", Weight: 2},
			{Source: "app", Target: "infra", Weight: 1.5},
			
			// Frontend dependencies
			{Source: "frontend", Target: "vango", Weight: 2},
			{Source: "vango", Target: "wasm", Weight: 2},
			{Source: "vango", Target: "tinygo", Weight: 2},
			{Source: "tinygo", Target: "wasm", Weight: 1.5},
			{Source: "frontend", Target: "tailwind", Weight: 1},
			{Source: "vango", Target: "canvas", Weight: 1},
			{Source: "frontend", Target: "cdn", Weight: 0.5},
			
			// Backend dependencies
			{Source: "backend", Target: "golang", Weight: 2},
			{Source: "backend", Target: "api", Weight: 1.5},
			{Source: "backend", Target: "graphql", Weight: 1},
			{Source: "backend", Target: "websocket", Weight: 1.5},
			{Source: "api", Target: "golang", Weight: 0.5},
			{Source: "graphql", Target: "golang", Weight: 0.5},
			{Source: "websocket", Target: "golang", Weight: 0.5},
			
			// Database connections
			{Source: "backend", Target: "database", Weight: 2},
			{Source: "database", Target: "postgres", Weight: 1.5},
			{Source: "database", Target: "redis", Weight: 1},
			{Source: "database", Target: "mongo", Weight: 1},
			
			// Infrastructure connections
			{Source: "infra", Target: "docker", Weight: 1.5},
			{Source: "infra", Target: "k8s", Weight: 1.5},
			{Source: "infra", Target: "nginx", Weight: 1},
			{Source: "docker", Target: "k8s", Weight: 1},
			{Source: "nginx", Target: "backend", Weight: 0.5},
			{Source: "nginx", Target: "frontend", Weight: 0.5},
			
			// Monitoring connections
			{Source: "backend", Target: "monitor", Weight: 1},
			{Source: "monitor", Target: "prometheus", Weight: 1.5},
			{Source: "monitor", Target: "grafana", Weight: 1.5},
			{Source: "monitor", Target: "logging", Weight: 1.5},
			{Source: "prometheus", Target: "grafana", Weight: 1},
			{Source: "infra", Target: "monitor", Weight: 0.5},
			
			// Cross-layer connections
			{Source: "websocket", Target: "frontend", Weight: 1, Color: "#ff7675"},
			{Source: "vango", Target: "websocket", Weight: 0.5, Color: "#ff7675"},
		},
	}
}

// GetVangoArchitectureGraph returns the Vango framework architecture graph
func GetVangoArchitectureGraph() graphviewer.Data {
	return graphviewer.Data{
		Nodes: []graphviewer.Node{
			// Core
			{ID: "vango", Label: "Vango Core", X: 0, Y: 0, Size: 16, Color: "#6c5ce7"},
			
			// Main components
			{ID: "vdom", Label: "Virtual DOM", X: -150, Y: -100, Size: 13, Color: "#0984e3"},
			{ID: "reactive", Label: "Reactive System", X: 150, Y: -100, Size: 13, Color: "#00b894"},
			{ID: "scheduler", Label: "Scheduler", X: 0, Y: -150, Size: 12, Color: "#fdcb6e"},
			{ID: "router", Label: "Router", X: -100, Y: 100, Size: 12, Color: "#e17055"},
			{ID: "live", Label: "Live Protocol", X: 100, Y: 100, Size: 12, Color: "#a29bfe"},
			
			// VDOM components
			{ID: "vnode", Label: "VNode", X: -250, Y: -150, Size: 10, Color: "#74b9ff"},
			{ID: "diff", Label: "Diff Algorithm", X: -200, Y: -50, Size: 10, Color: "#74b9ff"},
			{ID: "patch", Label: "Patch System", X: -150, Y: 0, Size: 10, Color: "#74b9ff"},
			
			// Reactive components
			{ID: "signal", Label: "Signals", X: 250, Y: -150, Size: 10, Color: "#55efc4"},
			{ID: "computed", Label: "Computed", X: 200, Y: -50, Size: 10, Color: "#55efc4"},
			{ID: "effect", Label: "Effects", X: 150, Y: 0, Size: 10, Color: "#55efc4"},
			
			// Scheduler components
			{ID: "fiber", Label: "Fibers", X: -50, Y: -250, Size: 10, Color: "#ffeaa7"},
			{ID: "queue", Label: "Task Queue", X: 50, Y: -250, Size: 10, Color: "#ffeaa7"},
			
			// Rendering
			{ID: "ssr", Label: "SSR", X: -200, Y: 50, Size: 11, Color: "#ff7675"},
			{ID: "hydrate", Label: "Hydration", X: -100, Y: 50, Size: 11, Color: "#ff7675"},
			{ID: "wasm", Label: "WASM Runtime", X: 0, Y: 200, Size: 11, Color: "#fd79a8"},
			
			// Builder/Syntax
			{ID: "vex", Label: "VEX Syntax", X: 200, Y: 50, Size: 11, Color: "#e84393"},
			{ID: "builder", Label: "Builder API", X: 250, Y: 0, Size: 10, Color: "#d63031"},
			{ID: "template", Label: "Templates", X: 300, Y: 50, Size: 10, Color: "#d63031"},
			
			// Dev tools
			{ID: "cli", Label: "CLI", X: -300, Y: 0, Size: 11, Color: "#636e72"},
			{ID: "dev", Label: "Dev Server", X: -350, Y: 50, Size: 10, Color: "#2d3436"},
			{ID: "hmr", Label: "Hot Reload", X: -300, Y: 100, Size: 10, Color: "#2d3436"},
		},
		Edges: []graphviewer.Edge{
			// Core connections
			{Source: "vango", Target: "vdom", Weight: 2},
			{Source: "vango", Target: "reactive", Weight: 2},
			{Source: "vango", Target: "scheduler", Weight: 2},
			{Source: "vango", Target: "router", Weight: 1.5},
			{Source: "vango", Target: "live", Weight: 1.5},
			
			// VDOM internal
			{Source: "vdom", Target: "vnode", Weight: 2},
			{Source: "vdom", Target: "diff", Weight: 2},
			{Source: "vdom", Target: "patch", Weight: 2},
			{Source: "vnode", Target: "diff", Weight: 1},
			{Source: "diff", Target: "patch", Weight: 1.5},
			
			// Reactive internal
			{Source: "reactive", Target: "signal", Weight: 2},
			{Source: "reactive", Target: "computed", Weight: 2},
			{Source: "reactive", Target: "effect", Weight: 2},
			{Source: "signal", Target: "computed", Weight: 1},
			{Source: "signal", Target: "effect", Weight: 1},
			{Source: "computed", Target: "effect", Weight: 0.5},
			
			// Scheduler internal
			{Source: "scheduler", Target: "fiber", Weight: 2},
			{Source: "scheduler", Target: "queue", Weight: 2},
			{Source: "fiber", Target: "queue", Weight: 1},
			
			// Cross-component connections
			{Source: "reactive", Target: "scheduler", Weight: 1.5, Color: "#9b59b6"},
			{Source: "scheduler", Target: "vdom", Weight: 1.5, Color: "#9b59b6"},
			{Source: "vdom", Target: "reactive", Weight: 1, Color: "#9b59b6"},
			
			// Rendering connections
			{Source: "vdom", Target: "ssr", Weight: 1.5},
			{Source: "vdom", Target: "hydrate", Weight: 1.5},
			{Source: "ssr", Target: "hydrate", Weight: 1},
			{Source: "hydrate", Target: "wasm", Weight: 1.5},
			{Source: "patch", Target: "hydrate", Weight: 1},
			
			// Live protocol
			{Source: "live", Target: "wasm", Weight: 1.5},
			{Source: "live", Target: "patch", Weight: 1},
			{Source: "live", Target: "router", Weight: 0.5},
			
			// VEX/Builder
			{Source: "vex", Target: "vdom", Weight: 1.5},
			{Source: "vex", Target: "builder", Weight: 1.5},
			{Source: "vex", Target: "template", Weight: 1.5},
			{Source: "builder", Target: "vnode", Weight: 1},
			{Source: "template", Target: "vnode", Weight: 1},
			
			// Dev tools
			{Source: "cli", Target: "dev", Weight: 1.5},
			{Source: "cli", Target: "vango", Weight: 1},
			{Source: "dev", Target: "hmr", Weight: 1.5},
			{Source: "hmr", Target: "wasm", Weight: 1},
			{Source: "hmr", Target: "live", Weight: 1},
		},
	}
}
`

	return WriteFile(filepath.Join(config.Directory, "app/data/graphs.go"), content)
}

func (t *GraphViewerTemplate) createGraphRoute(config *ProjectConfig) error {
	content := fmt.Sprintf(`package routes

import (
	"fmt"
	"syscall/js"
	
	"%s/app/data"
	"github.com/recera/vango/pkg/components/graphviewer"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// GraphViewer renders the main graph viewer interface
func GraphViewer() *vdom.VNode {
	return GraphViewerWithType("programming")
}

// GraphViewerWithType renders a specific graph type
func GraphViewerWithType(graphType string) *vdom.VNode {
	// Get the appropriate graph data
	var graphData graphviewer.Data
	var title string
	var description string
	
	switch graphType {
	case "social":
		graphData = data.GetSocialNetworkGraph()
		title = "Social Network Graph"
		description = "Interactive visualization of social connections and relationships"
	case "tech":
		graphData = data.GetTechStackGraph()
		title = "Technology Stack Dependencies"
		description = "Explore the interconnected world of modern tech stacks"
	case "vango":
		graphData = data.GetVangoArchitectureGraph()
		title = "Vango Architecture"
		description = "Deep dive into the Vango framework components and their relationships"
	default:
		graphData = data.GetProgrammingConceptsGraph()
		title = "Programming Concepts Graph"
		description = "Explore the relationships between programming paradigms, data structures, and algorithms"
	}
	
	// Determine if we're in dark mode
	isDarkMode := js.Global().Get("document").Get("documentElement").Get("classList").Call("contains", "dark").Bool()
	
	// Graph viewer options - adjust colors based on dark mode
	var bgColor, nodeColor, edgeColor, labelColor string
	if isDarkMode {
		bgColor = "#0f172a"    // dark blue-gray
		nodeColor = "#60a5fa"  // blue
		edgeColor = "#475569"  // gray
		labelColor = "#e2e8f0" // light gray
	} else {
		bgColor = "#ffffff"    // white
		nodeColor = "#3b82f6"  // blue-500
		edgeColor = "#9ca3af"  // gray-400
		labelColor = "#1f2937" // gray-800
	}
	
	options := &graphviewer.Options{
		// Physics
		Repulsion:       2500,
		SpringLength:    100,
		SpringStiffness: 0.05,
		Damping:         0.85,
		Gravity:         0.02, // Centering force to prevent drift
		
		// Rendering
		BackgroundColor: bgColor,
		NodeColor:       nodeColor,
		EdgeColor:       edgeColor,
		LabelColor:      labelColor,
		
		// Viewport
		MinScale: 0.2,
		MaxScale: 5.0,
		
		// Callbacks
		OnSelectNode: func(id string) {
			fmt.Printf("Selected node: %%s\n", id)
		},
		OnDblClickNode: func(id string) {
			fmt.Printf("Double-clicked node: %%s\n", id)
		},
		OnHoverNode: func(id string) {
			fmt.Printf("Hovering node: %%s\n", id)
		},
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-gradient-to-br from-gray-50 to-blue-50 dark:from-slate-900 dark:via-purple-900 dark:to-slate-900 text-gray-900 dark:text-gray-100 transition-colors duration-300"),
	),
		// Header
		functional.Header(functional.MergeProps(
			functional.Class("bg-white/80 dark:bg-black/30 backdrop-blur-md border-b border-gray-200 dark:border-white/10"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("flex justify-between items-center"),
				),
					// Title
					functional.Div(nil,
						functional.H1(functional.MergeProps(
							functional.Class("text-2xl font-bold text-gray-900 dark:text-white"),
						), functional.Text("Knowledge Graph Viewer")),
						functional.P(functional.MergeProps(
							functional.Class("text-sm text-gray-600 dark:text-gray-400 mt-1"),
						), functional.Text("Interactive graph visualization powered by Vango")),
					),
					
					// Dark mode toggle
					functional.Button(functional.MergeProps(
						functional.Class("p-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"),
						functional.ID("dark-mode-toggle"),
					),
						functional.Span(functional.MergeProps(
							functional.Class("text-xl"),
						), functional.Text("🌙")),
					),
				),
			),
		),
		
		// Graph selector tabs
		functional.Div(functional.MergeProps(
			functional.Class("bg-gray-100 dark:bg-black/20 backdrop-blur-sm border-b border-gray-200 dark:border-white/10"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("flex space-x-8"),
				),
					createTab("Programming Concepts", "programming", graphType),
					createTab("Social Network", "social", graphType),
					createTab("Tech Stack", "tech", graphType),
					createTab("Vango Architecture", "vango", graphType),
				),
			),
		),
		
		// Graph info
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("bg-white dark:bg-white/5 backdrop-blur-sm rounded-lg p-4 mb-6 shadow-sm dark:shadow-none"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-xl font-semibold text-gray-900 dark:text-white mb-2"),
				), functional.Text(title)),
				functional.P(functional.MergeProps(
					functional.Class("text-gray-600 dark:text-gray-400"),
				), functional.Text(description)),
				
				// Instructions
				functional.Div(functional.MergeProps(
					functional.Class("mt-4 flex flex-wrap gap-4 text-sm text-gray-500"),
				),
					functional.Span(functional.MergeProps(
						functional.Class("flex items-center gap-2"),
					),
						functional.Text("🖱️ Drag to pan"),
					),
					functional.Span(functional.MergeProps(
						functional.Class("flex items-center gap-2"),
					),
						functional.Text("⚡ Drag nodes to reposition"),
					),
					functional.Span(functional.MergeProps(
						functional.Class("flex items-center gap-2"),
					),
						functional.Text("🔍 Scroll to zoom"),
					),
					functional.Span(functional.MergeProps(
						functional.Class("flex items-center gap-2"),
					),
						functional.Text("👆 Click to select"),
					),
				),
			),
		),
		
		// Graph viewer container
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-8"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("bg-white dark:bg-black/40 backdrop-blur-sm rounded-lg p-2 shadow-lg dark:shadow-2xl"),
				functional.StyleAttr("height: 600px;"),
			),
				// The graph viewer component
				graphviewer.Viewer(graphData, options),
			),
		),
		
		// Stats panel
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-8"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
			),
				createStatCard("Nodes", fmt.Sprintf("%%d", len(graphData.Nodes)), "🔵"),
				createStatCard("Edges", fmt.Sprintf("%%d", len(graphData.Edges)), "🔗"),
				createStatCard("Graph Type", getGraphTypeLabel(graphType), "📊"),
				createStatCard("Physics", "Enabled", "⚡"),
			),
		),
	)
}

func createTab(label, value, current string) *vdom.VNode {
	isActive := value == current
	class := "py-3 px-1 border-b-2 text-sm font-medium transition-colors "
	if isActive {
		class += "border-blue-500 dark:border-purple-500 text-blue-600 dark:text-purple-400"
	} else {
		class += "border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-700"
	}
	
	return functional.Button(functional.MergeProps(
		functional.Class(class),
		functional.ID(fmt.Sprintf("tab-%%s", value)),
		vdom.Props{"data-graph": value},
	), functional.Text(label))
}

func createStatCard(label, value, icon string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-white/5 backdrop-blur-sm rounded-lg p-4 shadow-sm dark:shadow-none"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("flex items-center justify-between"),
		),
			functional.Div(nil,
				functional.P(functional.MergeProps(
					functional.Class("text-xs text-gray-600 dark:text-gray-500 uppercase tracking-wide"),
				), functional.Text(label)),
				functional.P(functional.MergeProps(
					functional.Class("text-lg font-semibold text-gray-900 dark:text-white mt-1"),
				), functional.Text(value)),
			),
			functional.Span(functional.MergeProps(
				functional.Class("text-2xl"),
			), functional.Text(icon)),
		),
	)
}

func getGraphTypeLabel(graphType string) string {
	switch graphType {
	case "social":
		return "Social"
	case "tech":
		return "Tech Stack"
	case "vango":
		return "Architecture"
	default:
		return "Concepts"
	}
}
`, config.Module)

	return WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), content)
}

func (t *GraphViewerTemplate) createEnhancedStyles(config *ProjectConfig) error {
	content := `/* Enhanced styles for graph viewer */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
	html {
		@apply antialiased;
	}
	
	body {
		@apply bg-slate-900 text-gray-100;
	}
}

@layer components {
	.graph-container {
		@apply relative w-full h-full rounded-lg overflow-hidden;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	}
	
	.control-panel {
		@apply absolute top-4 right-4 bg-black/50 backdrop-blur-md rounded-lg p-4 space-y-2;
	}
	
	.control-btn {
		@apply w-full px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg transition-all duration-200 text-sm font-medium;
	}
	
	.stat-card {
		@apply bg-gradient-to-br from-white/10 to-white/5 backdrop-blur-sm rounded-lg p-4 border border-white/10;
	}
}

/* Canvas styles */
canvas {
	display: block;
	width: 100% !important;
	height: 100% !important;
	border-radius: 0.5rem;
}

/* Custom scrollbar */
::-webkit-scrollbar {
	width: 8px;
	height: 8px;
}

::-webkit-scrollbar-track {
	@apply bg-slate-800;
}

::-webkit-scrollbar-thumb {
	@apply bg-slate-600 rounded-full;
}

::-webkit-scrollbar-thumb:hover {
	@apply bg-slate-500;
}
`

	// Write the graph-specific stylesheet
	if err := WriteFile(filepath.Join(config.Directory, "styles/graph.css"), content); err != nil {
		return err
	}

	// Ensure Tailwind input includes our graph.css so it gets compiled into styles.css
	// We generate a template-specific input.css here; the common generator will
	// skip overwriting if this already exists.
	input := `@tailwind base;
@tailwind components;
@tailwind utilities;

@import "./graph.css";
`

	return WriteFile(filepath.Join(config.Directory, "styles/input.css"), input)
}

func (t *GraphViewerTemplate) createTailwindConfig(config *ProjectConfig) error {
	content := `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{go,html,js}",
    "./public/**/*.html",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'graph-bg': '#0f172a',
        'graph-node': '#60a5fa',
        'graph-edge': '#475569',
        'graph-label': '#e2e8f0',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
  plugins: [],
}`

	return WriteFile(filepath.Join(config.Directory, "tailwind.config.js"), content)
}
