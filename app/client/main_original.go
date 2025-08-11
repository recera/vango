//go:build vango_client
// +build vango_client

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall/js"

	"github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango/vdom"
)

// Global references
var (
	sched    *scheduler.Scheduler
	liveConn *live.Client
	document js.Value
	window   js.Value
	console  js.Value
)

func main() {
	// Initialize global JS references
	document = js.Global().Get("document")
	window = js.Global().Get("window")
	console = js.Global().Get("console")
	
	console.Call("log", "🚀 Vango WASM client starting...")
	
	// Set up debug logging for reactive system
	reactive.SetDebugLog(func(args ...interface{}) {
		formatted := fmt.Sprint(args...)
		console.Call("debug", formatted)
	})
	
	// Initialize scheduler
	sched = scheduler.New(scheduler.Config{
		MaxFibers:     100,
		BatchInterval: 16, // ~60fps
	})
	
	// Start scheduler
	go sched.Run()
	
	// Initialize live connection for server updates
	setupLiveConnection()
	
	// Load and parse route table
	if err := loadRouteTable(); err != nil {
		console.Call("error", "Failed to load route table:", err.Error())
	}
	
	// Set up client-side navigation
	setupClientNavigation()
	
	// Hydrate the initial page
	hydrateInitialPage()
	
	console.Call("log", "✅ Vango client initialized")
	
	// Keep the program running
	select {}
}

// setupLiveConnection establishes WebSocket connection to server
func setupLiveConnection() {
	// Determine WebSocket URL
	protocol := "ws:"
	if window.Get("location").Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := window.Get("location").Get("host").String()
	wsURL := fmt.Sprintf("%s//%s/vango/live/", protocol, host)
	
	console.Call("log", "Connecting to live server:", wsURL)
	
	// Create live client
	liveConn = live.NewClient(wsURL, sched)
	
	// Set up reconnection handler
	liveConn.OnConnect(func() {
		console.Call("log", "✅ Connected to live server")
	})
	
	liveConn.OnDisconnect(func() {
		console.Call("warn", "⚠️ Disconnected from live server, will retry...")
	})
	
	// Handle patch messages from server
	liveConn.OnPatch(func(patches []vdom.Patch) {
		console.Call("debug", fmt.Sprintf("Received %d patches from server", len(patches)))
		// Apply patches to DOM
		applyPatches(patches)
	})
	
	// Connect
	if err := liveConn.Connect(); err != nil {
		console.Call("error", "Failed to connect to live server:", err.Error())
	}
}

// RouteEntry represents a client-side route
type RouteEntry struct {
	Path      string   `json:"path"`
	Component string   `json:"component"`
	Params    []string `json:"params"`
	IsAPI     bool     `json:"isApi"`
	HasLayout bool     `json:"hasLayout"`
}

var routes []RouteEntry

// loadRouteTable loads the generated route table
func loadRouteTable() error {
	// Fetch route table
	resp := make(chan js.Value, 1)
	errChan := make(chan error, 1)
	
	promise := window.Call("fetch", "/router/table.json")
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		return response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp <- args[0]
			return nil
		}))
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errChan <- fmt.Errorf("failed to fetch route table: %v", args[0])
		return nil
	}))
	
	select {
	case data := <-resp:
		// Parse route table
		jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
		var table struct {
			Routes []RouteEntry `json:"routes"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &table); err != nil {
			return fmt.Errorf("failed to parse route table: %w", err)
		}
		routes = table.Routes
		console.Call("log", fmt.Sprintf("Loaded %d routes", len(routes)))
		return nil
	case err := <-errChan:
		return err
	}
}

// setupClientNavigation sets up client-side routing
func setupClientNavigation() {
	// Intercept link clicks
	document.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		target := event.Get("target")
		
		// Check if target is a link
		for !target.IsNull() && !target.IsUndefined() {
			tagName := target.Get("tagName")
			if !tagName.IsUndefined() && tagName.String() == "A" {
				href := target.Get("href").String()
				// Check if it's an internal link
				if isInternalLink(href) {
					event.Call("preventDefault")
					navigateTo(href)
				}
				break
			}
			// Check parent
			target = target.Get("parentElement")
		}
		
		return nil
	}))
	
	// Handle browser back/forward
	window.Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		currentPath := window.Get("location").Get("pathname").String()
		loadRoute(currentPath)
		return nil
	}))
}

// isInternalLink checks if a URL is an internal link
func isInternalLink(href string) bool {
	// Check if it's a full URL
	if len(href) > 0 && (href[0] == '/' || href[0] == '#') {
		return true
	}
	
	// Check if it's the same origin
	currentOrigin := window.Get("location").Get("origin").String()
	return len(href) >= len(currentOrigin) && href[:len(currentOrigin)] == currentOrigin
}

// navigateTo navigates to a new route
func navigateTo(path string) {
	// Update browser history
	window.Get("history").Call("pushState", nil, "", path)
	
	// Load the new route
	loadRoute(path)
}

// loadRoute loads a route component
func loadRoute(path string) {
	console.Call("log", "Loading route:", path)
	
	// Find matching route
	var matchedRoute *RouteEntry
	params := make(map[string]string)
	
	for _, route := range routes {
		if matched, extractedParams := matchRoute(path, route.Path); matched {
			matchedRoute = &route
			params = extractedParams
			break
		}
	}
	
	if matchedRoute == nil {
		console.Call("warn", "No route found for:", path)
		// TODO: Render 404 page
		return
	}
	
	console.Call("log", "Matched route:", matchedRoute.Component, "with params:", params)
	
	// TODO: Load and render the component
	// This would involve looking up the component function and rendering it
	renderRoute(matchedRoute, params)
}

// matchRoute checks if a path matches a route pattern
func matchRoute(path, pattern string) (bool, map[string]string) {
	// Simple exact match for now
	// TODO: Implement proper parameter matching
	if path == pattern {
		return true, make(map[string]string)
	}
	return false, nil
}

// renderRoute renders a matched route
func renderRoute(route *RouteEntry, params map[string]string) {
	// TODO: Look up component function and render
	// For now, just log
	console.Call("log", "Would render component:", route.Component)
	
	// Create a placeholder VNode
	vnode := vdom.NewElement("div", vdom.Props{
		"class": "route-placeholder",
	}, 
		vdom.NewElement("h2", nil, vdom.NewText("Route: "+route.Path)),
		vdom.NewElement("p", nil, vdom.NewText("Component: "+route.Component)),
	)
	
	// Mount to app root
	mountVNode(vnode)
}

// hydrateInitialPage hydrates the server-rendered HTML
func hydrateInitialPage() {
	console.Call("log", "Hydrating initial page...")
	
	// Find app root
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() || appRoot.IsUndefined() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Get current path
	currentPath := window.Get("location").Get("pathname").String()
	
	// Load the route for hydration
	loadRoute(currentPath)
	
	console.Call("log", "✅ Hydration complete")
}

// mountVNode mounts a VNode to the DOM
func mountVNode(vnode *vdom.VNode) {
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() || appRoot.IsUndefined() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Clear existing content
	appRoot.Set("innerHTML", "")
	
	// Render VNode to DOM
	domNode := renderVNodeToDOM(vnode)
	if !domNode.IsNull() && !domNode.IsUndefined() {
		appRoot.Call("appendChild", domNode)
	}
}

// renderVNodeToDOM renders a VNode to a DOM element
func renderVNodeToDOM(vnode *vdom.VNode) js.Value {
	if vnode == nil {
		return js.Null()
	}
	
	switch vnode.Kind {
	case vdom.KindText:
		return document.Call("createTextNode", vnode.Text)
		
	case vdom.KindElement:
		elem := document.Call("createElement", vnode.Tag)
		
		// Set properties
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				if key == "className" || key == "class" {
					elem.Get("classList").Call("add", fmt.Sprint(value))
				} else if strings.HasPrefix(key, "on") {
					// Event handler - skip for now
					// TODO: Implement event binding
				} else {
					elem.Call("setAttribute", key, fmt.Sprint(value))
				}
			}
		}
		
		// Render children
		for _, child := range vnode.Kids {
			childNode := renderVNodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				elem.Call("appendChild", childNode)
			}
		}
		
		return elem
		
	case vdom.KindFragment:
		// Create document fragment
		fragment := document.Call("createDocumentFragment")
		for _, child := range vnode.Kids {
			childNode := renderVNodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}

// applyPatches applies patches from the server to the DOM
func applyPatches(patches []vdom.Patch) {
	// TODO: Implement patch application
	console.Call("debug", "Would apply patches:", len(patches))
	
	for _, patch := range patches {
		switch patch.Type {
		case vdom.PatchReplace:
			console.Call("debug", "Replace patch at path:", patch.Path)
		case vdom.PatchProps:
			console.Call("debug", "Props patch at path:", patch.Path)
		case vdom.PatchText:
			console.Call("debug", "Text patch at path:", patch.Path)
		case vdom.PatchReorder:
			console.Call("debug", "Reorder patch at path:", patch.Path)
		}
	}
}