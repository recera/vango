package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register("basic", &BasicTemplate{})
}

// BasicTemplate generates a minimal starter template
type BasicTemplate struct{}

func (t *BasicTemplate) Name() string {
	return "basic"
}

func (t *BasicTemplate) Description() string {
	return "Minimal starter template"
}

func (t *BasicTemplate) Generate(config *ProjectConfig) error {
	// Force Tailwind for the basic template
	config.UseTailwind = true
	config.TailwindStrategy = "npm"
	config.DarkMode = true

	// Create main.go for WASM client with routing support
	if err := t.createMainFile(config); err != nil {
		return err
	}

	// Create layout wrapper for consistent navigation
	if err := t.createLayout(config); err != nil {
		return err
	}

	// Create index route with overview
	if err := t.createIndexRoute(config); err != nil {
		return err
	}

	// Create Layer 1 VEX demo page (fluent builder)
	if err := t.createLayer1Route(config); err != nil {
		return err
	}

	// Create Layer 2 VEX demo page (template syntax)
	if err := t.createLayer2Route(config); err != nil {
		return err
	}

	// Create server component demo
	if err := t.createServerRoute(config); err != nil {
		return err
	}

	// Create client component demo
	if err := t.createClientRoute(config); err != nil {
		return err
	}

	// Create features showcase page
	if err := t.createFeaturesRoute(config); err != nil {
		return err
	}

	// Create beautiful styles with dark mode
	if err := t.createStyles(config); err != nil {
		return err
	}

	// Create Tailwind config
	if err := t.createTailwindConfig(config); err != nil {
		return err
	}

	return nil
}

func (t *BasicTemplate) createMainFile(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"strings"
	"syscall/js"
	
	// Import routes package with alias
	routes "%s/app/routes"
	layer1 "%s/app/routes/vex"
	layer2 "%s/app/routes/templates"
	server "%s/app/routes/server"
	client "%s/app/routes/client"
	features "%s/app/routes/features"
	"github.com/recera/vango/pkg/vango/vdom"
)

// Retained handlers to prevent GC
var retainedHandlers []js.Func

func main() {
	// Initialize the Vango runtime
	js.Global().Get("console").Call("log", "🚀 Vango Showcase App Starting...")
	
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
	console.Call("log", "DOM ready, initializing Vango Showcase...")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Set up navigation handler
	setupNavigation()
	
	// Initial render
	renderCurrentRoute()
}

func renderCurrentRoute() {
	console := js.Global().Get("console")
	window := js.Global().Get("window")
	
	// Get current route
	path := window.Get("location").Get("pathname").String()
	console.Call("log", "Rendering route:", path)
	
	// Render the appropriate component with layout
	var pageContent *vdom.VNode
	switch {
	case path == "/" || path == "":
		pageContent = routes.Page()
	case path == "/vex/layer1":
		pageContent = layer1.Page()
	case path == "/templates/layer2":
		pageContent = layer2.Page()
	case path == "/server/demo":
		pageContent = server.Page()
	case path == "/client/demo":
		pageContent = client.Page()
	case path == "/features":
		pageContent = features.Page()
	default:
		// 404 page
		pageContent = routes.NotFoundPage()
	}
	
	// Wrap with layout
	vnode := routes.Layout(pageContent)
	renderVNode(vnode)
}

func setupNavigation() {
	window := js.Global().Get("window")
	document := js.Global().Get("document")
	
	// Handle browser back/forward buttons
	window.Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		renderCurrentRoute()
		return nil
	}))
	
	// Export navigation function
	js.Global().Set("navigateTo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			path := args[0].String()
			window.Get("history").Call("pushState", nil, "", path)
			renderCurrentRoute()
		}
		return nil
	}))
	
	// Intercept link clicks for client-side routing
	document.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			event := args[0]
			target := event.Get("target")
			
			// Check if it's a link
			for !target.IsNull() && !target.IsUndefined() {
				tagName := strings.ToLower(target.Get("tagName").String())
				if tagName == "a" {
					href := target.Get("href").String()
					// Check if it's an internal link
					if strings.HasPrefix(href, window.Get("location").Get("origin").String()) {
						event.Call("preventDefault")
						path := strings.TrimPrefix(href, window.Get("location").Get("origin").String())
						window.Get("history").Call("pushState", nil, "", path)
						renderCurrentRoute()
						return nil
					}
					break
				}
				target = target.Get("parentElement")
			}
		}
		return nil
	}))
	
	// Export dark mode toggle
	js.Global().Set("toggleDarkMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		toggleDarkMode()
		renderCurrentRoute() // Re-render to update dark mode state
		return nil
	}))
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
}

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering VNode with tag:", vnode.Tag)
	
	// Get the app root
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() || appRoot.IsUndefined() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Clear existing content
	appRoot.Set("innerHTML", "")
	
	// Render the actual VNode to DOM
	domNode := vnodeToDOM(vnode)
	if !domNode.IsNull() && !domNode.IsUndefined() {
		appRoot.Call("appendChild", domNode)
		console.Call("log", "✅ VNode rendered successfully!")
	}
}

func vnodeToDOM(vnode *vdom.VNode) js.Value {
	document := js.Global().Get("document")
	
	if vnode == nil {
		return js.Null()
	}
	
	switch vnode.Kind {
	case vdom.KindText:
		// Handle text nodes
		return document.Call("createTextNode", vnode.Text)
		
	case vdom.KindElement:
		// Create element
		elem := document.Call("createElement", vnode.Tag)
		
		// Set properties
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				switch key {
				case "class", "className":
					if v, ok := value.(string); ok {
						elem.Set("className", v)
					}
				case "id":
					if v, ok := value.(string); ok {
						elem.Set("id", v)
					}
				case "href":
					if v, ok := value.(string); ok {
						elem.Set("href", v)
					}
				case "target":
					if v, ok := value.(string); ok {
						elem.Set("target", v)
					}
				case "style":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "style", v)
					}
				default:
					// Set as attribute for other properties
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", key, v)
					}
				}
			}
		}
		
		// Render children
		for _, child := range vnode.Kids {
			// Pass pointer to child since Kids stores values
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				elem.Call("appendChild", childNode)
			}
		}
		
		return elem
		
	case vdom.KindFragment:
		// Create document fragment
		fragment := document.Call("createDocumentFragment")
		for _, child := range vnode.Kids {
			// Pass pointer to child since Kids stores values
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}`, config.Module, config.Module, config.Module, config.Module, config.Module, config.Module)

	return WriteFile(filepath.Join(config.Directory, "app/main.go"), content)
}

func (t *BasicTemplate) createIndexRoute(config *ProjectConfig) error {
	content := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Page is the home page handler showcasing Vango features
func Page() *vdom.VNode {
	return functional.Div(nil,
		// Hero Section with gradient background
		functional.Section(functional.MergeProps(
			functional.Class("relative overflow-hidden bg-gradient-to-br from-purple-600 via-blue-600 to-indigo-700 text-white"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 relative z-10"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("text-center"),
				),
					functional.H1(functional.MergeProps(
						functional.Class("text-5xl md:text-7xl font-bold mb-6"),
					),
						functional.Text("Welcome to "),
						functional.Span(functional.MergeProps(
							functional.Class("bg-clip-text text-transparent bg-gradient-to-r from-yellow-400 to-orange-500"),
						), functional.Text("Vango")),
					),
					functional.P(functional.MergeProps(
						functional.Class("text-xl md:text-2xl mb-8 text-blue-100"),
					), functional.Text("The Go-Native Frontend Framework")),
					functional.P(functional.MergeProps(
						functional.Class("text-lg mb-12 text-blue-200 max-w-2xl mx-auto"),
					), functional.Text("Build blazing-fast web applications with Go, WebAssembly, and a modern developer experience")),
					
					// CTA Buttons
					functional.Div(functional.MergeProps(
						functional.Class("flex flex-wrap gap-4 justify-center"),
					),
						functional.A(functional.MergeProps(
							functional.Href("/vex/layer1"),
							functional.Class("px-8 py-4 bg-white text-purple-600 rounded-lg font-semibold hover:bg-gray-100 transition-colors shadow-lg"),
						), functional.Text("Explore VEX Syntax →")),
						functional.A(functional.MergeProps(
							functional.Href("/features"),
							functional.Class("px-8 py-4 bg-purple-800 text-white rounded-lg font-semibold hover:bg-purple-900 transition-colors shadow-lg"),
						), functional.Text("View Features")),
					),
				),
			),
			// Animated background pattern
			functional.Div(functional.MergeProps(
				functional.Class("absolute inset-0 opacity-10"),
				functional.StyleAttr(` + "`" + `
					background-image: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.4'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
				` + "`" + `),
			), nil),
		),
		
		// Main Content
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16"),
		),
			// What You're Seeing Section
			functional.Section(functional.MergeProps(
				functional.Class("mb-16"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold text-gray-900 dark:text-white mb-8"),
				), functional.Text("📚 What This Showcase Demonstrates")),
				
				functional.Div(functional.MergeProps(
					functional.Class("grid md:grid-cols-2 gap-6"),
				),
					showcaseCard(
						"🗂️ File-Based Routing",
						"Each page in this showcase is a separate Go file in app/routes/. The navigation automatically works based on the file structure.",
						"/vex/layer1",
						"See Layer 1 VEX",
					),
					showcaseCard(
						"🎨 VEX Syntax Layers",
						"Vango offers multiple ways to write components - from functional Go to fluent builders to template syntax.",
						"/templates/layer2",
						"See Layer 2 VEX",
					),
					showcaseCard(
						"🖥️ Server Components",
						"Components marked with //vango:server run only on the server, perfect for data fetching and secure operations.",
						"/server/demo",
						"Server Demo",
					),
					showcaseCard(
						"⚡ Client Components",
						"Client components run in the browser via WebAssembly, enabling rich interactivity without JavaScript.",
						"/client/demo",
						"Client Demo",
					),
				),
			),
			
			// Code Example Section
			functional.Section(functional.MergeProps(
				functional.Class("mb-16"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold text-gray-900 dark:text-white mb-8"),
				), functional.Text("💻 How This Page Was Built")),
				
				functional.Div(functional.MergeProps(
					functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
				),
					functional.Pre(functional.MergeProps(
						functional.Class("text-sm text-gray-300"),
					), functional.Text(` + "`" + `// This page uses Layer 1 VEX (Fluent Builder API)
func Page() *vdom.VNode {
    return functional.Div(nil,
        functional.H1(functional.MergeProps(
            functional.Class("text-5xl font-bold"),
        ), functional.Text("Welcome to Vango")),
        
        functional.P(functional.MergeProps(
            functional.Class("text-xl text-gray-600"),
        ), functional.Text("Build with Go, run everywhere")),
    )
}` + "`" + `)),
				),
				
				functional.P(functional.MergeProps(
					functional.Class("mt-4 text-gray-600 dark:text-gray-400"),
				), functional.Text("This entire page is written in Go, compiled to WebAssembly, and runs in your browser!")),
			),
			
			// Features Overview
			functional.Section(functional.MergeProps(
				functional.Class("mb-16"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold text-gray-900 dark:text-white mb-8"),
				), functional.Text("✨ Framework Features")),
				
				functional.Div(functional.MergeProps(
					functional.Class("grid md:grid-cols-3 gap-6"),
				),
					featureCard("🚀", "Pure Go", "Write your entire application in Go - frontend and backend"),
					featureCard("⚡", "WebAssembly", "Near-native performance with Go compiled to WASM"),
					featureCard("🔄", "Hot Reload", "See changes instantly during development"),
					featureCard("📦", "Type Safety", "Catch errors at compile time, not runtime"),
					featureCard("🎨", "Flexible Styling", "Use Tailwind CSS, scoped styles, or inline styles"),
					featureCard("🌐", "SSR + CSR", "Server-side rendering with client-side interactivity"),
				),
			),
			
			// Quick Start
			functional.Section(nil,
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold text-gray-900 dark:text-white mb-8"),
				), functional.Text("🚀 Quick Start")),
				
				functional.Div(functional.MergeProps(
					functional.Class("bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-6 rounded-r-lg"),
				),
					functional.H3(functional.MergeProps(
						functional.Class("text-xl font-semibold text-blue-900 dark:text-blue-100 mb-4"),
					), functional.Text("Start exploring the showcase:")),
					functional.Ol(functional.MergeProps(
						functional.Class("space-y-2 text-blue-800 dark:text-blue-200"),
					),
						functional.Li(nil, functional.Text("1. Click through the navigation to see different VEX syntax layers")),
						functional.Li(nil, functional.Text("2. Try the dark mode toggle in the top-right corner")),
						functional.Li(nil, functional.Text("3. View the source code for each page to learn Vango patterns")),
						functional.Li(nil, functional.Text("4. Check the Features page for a complete overview")),
					),
				),
			),
		),
	)
}

// Helper function for showcase cards
func showcaseCard(title, description, link, linkText string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 hover:shadow-xl transition-shadow"),
	),
		functional.H3(functional.MergeProps(
			functional.Class("text-xl font-semibold text-gray-900 dark:text-white mb-3"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-300 mb-4"),
		), functional.Text(description)),
		functional.A(functional.MergeProps(
			functional.Href(link),
			functional.Class("inline-flex items-center text-purple-600 dark:text-purple-400 font-medium hover:text-purple-700 dark:hover:text-purple-300"),
		), functional.Text(linkText + " →")),
	)
}

// Helper function for feature cards
func featureCard(icon, title, description string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md hover:shadow-lg transition-shadow"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-3xl mb-3"),
		), functional.Text(icon)),
		functional.H3(functional.MergeProps(
			functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-2"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-300 text-sm"),
		), functional.Text(description)),
	)
}`

	return WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), content)
}

func (t *BasicTemplate) createLayout(config *ProjectConfig) error {
	content := `package routes

import (
	"syscall/js"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Layout wraps all pages with consistent navigation and styling
func Layout(content *vdom.VNode) *vdom.VNode {
	// Check if we're in dark mode
	isDarkMode := false
	if js.Global().Truthy() {
		isDarkMode = js.Global().Get("document").Get("documentElement").Get("classList").Call("contains", "dark").Bool()
	}
	
	// Get current path for active nav styling
	currentPath := "/"
	if js.Global().Truthy() {
		currentPath = js.Global().Get("window").Get("location").Get("pathname").String()
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 transition-colors duration-300"),
	),
		// Navigation Header
		functional.Header(functional.MergeProps(
			functional.Class("bg-white/80 dark:bg-gray-900/80 backdrop-blur-md border-b border-gray-200 dark:border-gray-700 sticky top-0 z-50"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("flex justify-between items-center py-4"),
				),
					// Logo and brand
					functional.Div(functional.MergeProps(
						functional.Class("flex items-center space-x-8"),
					),
						functional.A(functional.MergeProps(
							functional.Href("/"),
							functional.Class("flex items-center space-x-2"),
						),
							functional.Span(functional.MergeProps(
								functional.Class("text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-500 bg-clip-text text-transparent"),
							), functional.Text("Vango")),
							functional.Span(functional.MergeProps(
								functional.Class("text-xs bg-purple-600 text-white px-2 py-1 rounded-full font-semibold"),
							), functional.Text("SHOWCASE")),
						),
						
						// Main navigation
						functional.Nav(functional.MergeProps(
							functional.Class("hidden md:flex space-x-6"),
						),
							navLink("/", "Home", currentPath == "/"),
							navLink("/vex/layer1", "Layer 1 VEX", currentPath == "/vex/layer1"),
							navLink("/templates/layer2", "Layer 2 VEX", currentPath == "/templates/layer2"),
							navLink("/server/demo", "Server", currentPath == "/server/demo"),
							navLink("/client/demo", "Client", currentPath == "/client/demo"),
							navLink("/features", "Features", currentPath == "/features"),
						),
					),
					
					// Dark mode toggle
					functional.Button(functional.MergeProps(
						functional.Class("p-2 rounded-lg bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"),
						vdom.Props{"onclick": "toggleDarkMode()"},
					),
						functional.Span(functional.MergeProps(
							functional.Class("text-xl"),
						), functional.Text(func() string {
							if isDarkMode {
								return "☀️"
							}
							return "🌙"
						}())),
					),
				),
			),
		),
		
		// Main content area
		functional.Main(functional.MergeProps(
			functional.Class("flex-1"),
		),
			content,
		),
		
		// Footer
		functional.Footer(functional.MergeProps(
			functional.Class("bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 mt-16"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("text-center text-gray-600 dark:text-gray-400"),
				),
					functional.P(nil,
						functional.Text("Built with "),
						functional.Span(functional.MergeProps(
							functional.Class("text-purple-600 dark:text-purple-400 font-semibold"),
						), functional.Text("Vango")),
						functional.Text(" - The Go-Native Frontend Framework"),
					),
					functional.P(functional.MergeProps(
						functional.Class("mt-2 text-sm"),
					),
						functional.Text("Explore the power of Go in the browser with WebAssembly"),
					),
				),
			),
		),
	)
}

// Helper function for navigation links
func navLink(href, text string, isActive bool) *vdom.VNode {
	class := "px-3 py-2 text-sm font-medium rounded-md transition-colors "
	if isActive {
		class += "bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300"
	} else {
		class += "text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800"
	}
	
	return functional.A(functional.MergeProps(
		functional.Href(href),
		functional.Class(class),
	), functional.Text(text))
}

// NotFoundPage renders a 404 error page
func NotFoundPage() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-4xl mx-auto px-4 py-16 text-center"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-6xl font-bold text-gray-300 dark:text-gray-700"),
		), functional.Text("404")),
		functional.H2(functional.MergeProps(
			functional.Class("text-2xl font-semibold text-gray-900 dark:text-gray-100 mt-4"),
		), functional.Text("Page Not Found")),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-400 mt-2"),
		), functional.Text("The page you're looking for doesn't exist.")),
		functional.A(functional.MergeProps(
			functional.Href("/"),
			functional.Class("inline-block mt-6 px-6 py-3 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors"),
		), functional.Text("Go Home")),
	)
}
`

	return WriteFile(filepath.Join(config.Directory, "app/routes/layout.go"), content)
}

// createLayer1Route creates the Layer 1 VEX demo page
func (t *BasicTemplate) createLayer1Route(config *ProjectConfig) error {
	content := `package vex

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Page demonstrates Layer 1 VEX - Fluent Builder API
func Page() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
	),
		// Page Title
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
		), functional.Text("Layer 1 VEX: Fluent Builder API")),
		
		// Description
		functional.Div(functional.MergeProps(
			functional.Class("bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-6 rounded-r-lg mb-12"),
		),
			functional.P(functional.MergeProps(
				functional.Class("text-blue-900 dark:text-blue-100"),
			), functional.Text("Layer 1 VEX provides a fluent, chainable API for building components. This page itself is built using the fluent builder pattern.")),
		),
		
		// Code Example
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Code Example")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
			),
				functional.Pre(functional.MergeProps(
					functional.Class("text-sm text-gray-300"),
				), functional.Text(` + "`" + `// Layer 1 VEX - Fluent Builder Pattern
import "github.com/recera/vango/pkg/vex/functional"

func Component() *vdom.VNode {
    return functional.Div(functional.MergeProps(
        functional.Class("container mx-auto"),
        functional.ID("my-component"),
    ),
        functional.H1(functional.MergeProps(
            functional.Class("text-4xl font-bold"),
        ), functional.Text("Hello Vango")),
        
        functional.P(functional.MergeProps(
            functional.Class("text-gray-600"),
        ), functional.Text("Building UIs with Go")),
        
        functional.Button(functional.MergeProps(
            functional.Class("btn btn-primary"),
            vdom.Props{"onclick": "handleClick()"},
        ), functional.Text("Click Me")),
    )
}` + "`" + `)),
			),
		),
		
		// Live Demo
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Live Demo")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8"),
			),
				// Demo Card Component
				functional.Div(functional.MergeProps(
					functional.Class("space-y-6"),
				),
					// Card 1
					createDemoCard(
						"🎨 Styling",
						"Components can use Tailwind classes, inline styles, or a combination",
						"purple",
					),
					// Card 2
					createDemoCard(
						"🔧 Props",
						"Pass properties using MergeProps for clean, composable configuration",
						"blue",
					),
					// Card 3
					createDemoCard(
						"🎯 Events",
						"Event handlers are added as props and work seamlessly with WASM",
						"green",
					),
				),
			),
		),
		
		// Advantages Section
		functional.Section(nil,
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Advantages of Layer 1 VEX")),
			
			functional.Ul(functional.MergeProps(
				functional.Class("space-y-3 text-gray-700 dark:text-gray-300"),
			),
				listItem("✅", "Full type safety - catch errors at compile time"),
				listItem("✅", "Excellent IDE support with auto-completion"),
				listItem("✅", "Familiar Go syntax - no new language to learn"),
				listItem("✅", "Composable and reusable component functions"),
				listItem("✅", "Direct control over the virtual DOM structure"),
			),
		),
		
		// Next Steps
		functional.Div(functional.MergeProps(
			functional.Class("mt-12 p-6 bg-gradient-to-r from-purple-500 to-blue-500 rounded-lg text-white"),
		),
			functional.H3(functional.MergeProps(
				functional.Class("text-xl font-semibold mb-3"),
			), functional.Text("Ready for more?")),
			functional.P(functional.MergeProps(
				functional.Class("mb-4"),
			), functional.Text("Check out Layer 2 VEX for an even more expressive template syntax!")),
			functional.A(functional.MergeProps(
				functional.Href("/templates/layer2"),
				functional.Class("inline-block px-6 py-3 bg-white text-purple-600 rounded-lg font-semibold hover:bg-gray-100 transition-colors"),
			), functional.Text("Explore Layer 2 VEX →")),
		),
	)
}

// Helper function for demo cards
func createDemoCard(icon, title, description, color string) *vdom.VNode {
	bgClass := "bg-" + color + "-50 dark:bg-" + color + "-900/20"
	borderClass := "border-l-4 border-" + color + "-500"
	
	return functional.Div(functional.MergeProps(
		functional.Class(bgClass + " " + borderClass + " p-6 rounded-r-lg"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("flex items-start space-x-4"),
		),
			functional.Span(functional.MergeProps(
				functional.Class("text-2xl"),
			), functional.Text(icon)),
			functional.Div(nil,
				functional.H3(functional.MergeProps(
					functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-2"),
				), functional.Text(title)),
				functional.P(functional.MergeProps(
					functional.Class("text-gray-700 dark:text-gray-300"),
				), functional.Text(description)),
			),
		),
	)
}

// Helper function for list items
func listItem(icon, text string) *vdom.VNode {
	return functional.Li(functional.MergeProps(
		functional.Class("flex items-start space-x-2"),
	),
		functional.Span(nil, functional.Text(icon)),
		functional.Span(nil, functional.Text(text)),
	)
}
`

	// Create the vex directory
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes/vex"), 0755); err != nil {
		return fmt.Errorf("failed to create vex directory: %w", err)
	}

	return WriteFile(filepath.Join(config.Directory, "app/routes/vex/layer1.go"), content)
}

// createLayer2Route creates the Layer 2 VEX template syntax demo page
func (t *BasicTemplate) createLayer2Route(config *ProjectConfig) error {
	// This would normally use the template syntax, but for demo purposes we'll show what it compiles to
	content := `package templates

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Page demonstrates Layer 2 VEX - Template Syntax
// In real usage, this would be written as a .vex.go file with template syntax
func Page() *vdom.VNode {
	// Show what the template would compile to
	templateExample := ` + "`" + `//vango:template
//vango:props { Title string; Items []string }

<div class="container">
    <h1>{{.Title}}</h1>
    
    {{#if .Items}}
        <ul>
        {{#for item in .Items}}
            <li @click="handleClick(item)">{{item}}</li>
        {{/for}}
        </ul>
    {{#else}}
        <p>No items to display</p>
    {{/if}}
    
    <button @click="addItem()">Add Item</button>
</div>` + "`" + `
	
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
		), functional.Text("Layer 2 VEX: Template Syntax")),
		
		// Description
		functional.Div(functional.MergeProps(
			functional.Class("bg-green-50 dark:bg-green-900/20 border-l-4 border-green-500 p-6 rounded-r-lg mb-12"),
		),
			functional.P(functional.MergeProps(
				functional.Class("text-green-900 dark:text-green-100"),
			), functional.Text("Layer 2 VEX provides an HTML-like template syntax with Go expressions. Templates are compiled to Go code at build time for zero runtime overhead.")),
		),
		
		// Template Example
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Template Syntax Example")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
			),
				functional.Pre(functional.MergeProps(
					functional.Class("text-sm text-gray-300"),
				), functional.Text(templateExample)),
			),
		),
		
		// Compiled Output
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Compiles To")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
			),
				functional.Pre(functional.MergeProps(
					functional.Class("text-sm text-gray-300"),
				), functional.Text(` + "`" + `func Page(ctx vango.Ctx, props PageProps) *vdom.VNode {
    var children []vdom.VNode
    
    children = append(children, 
        functional.H1(nil, functional.Text(props.Title)))
    
    if len(props.Items) > 0 {
        var listItems []vdom.VNode
        for _, item := range props.Items {
            listItems = append(listItems, 
                functional.Li(functional.MergeProps(
                    vdom.Props{"onclick": fmt.Sprintf("handleClick('%s')", item)},
                ), functional.Text(item)))
        }
        children = append(children, 
            functional.Ul(nil, listItems...))
    } else {
        children = append(children, 
            functional.P(nil, functional.Text("No items to display")))
    }
    
    children = append(children, 
        functional.Button(functional.MergeProps(
            vdom.Props{"onclick": "addItem()"},
        ), functional.Text("Add Item")))
    
    return functional.Div(functional.MergeProps(
        functional.Class("container"),
    ), children...)
}` + "`" + `)),
			),
		),
		
		// Features
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Template Features")),
			
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-2 gap-6"),
			),
				featureBox("Conditionals", "Use {{#if}}, {{#elseif}}, and {{#else}} for conditional rendering"),
				featureBox("Loops", "Iterate with {{#for item in items}} syntax"),
				featureBox("Events", "Bind events with @ prefix like @click, @input, @submit"),
				featureBox("Props", "Declare component props with //vango:props directive"),
				featureBox("Expressions", "Embed Go expressions with {{expression}} syntax"),
				featureBox("Components", "Use custom components with <MyComponent /> syntax"),
			),
		),
		
		// Benefits
		functional.Section(nil,
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Why Use Templates?")),
			
			functional.Ul(functional.MergeProps(
				functional.Class("space-y-3 text-gray-700 dark:text-gray-300"),
			),
				listItem("📝", "Familiar HTML-like syntax for designers and frontend developers"),
				listItem("⚡", "Zero runtime overhead - templates compile to Go code"),
				listItem("🔍", "Compile-time validation catches template errors early"),
				listItem("🎨", "Clean separation of markup and logic"),
				listItem("🔧", "Full Go expression support within templates"),
			),
		),
	)
}

func featureBox(title, description string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md"),
	),
		functional.H3(functional.MergeProps(
			functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-2"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-400 text-sm"),
		), functional.Text(description)),
	)
}
`

	// Create the templates directory
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes/templates"), 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	return WriteFile(filepath.Join(config.Directory, "app/routes/templates/layer2.go"), content)
}

func (t *BasicTemplate) createAboutRoute(config *ProjectConfig) error {
	content := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// AboutPage handles the /about route (universal component)
func AboutPage() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("about-page"),
		functional.StyleAttr("padding: 2rem; max-width: 800px; margin: 0 auto;"),
	),
		functional.H1(functional.MergeProps(
			functional.StyleAttr("color: #333; margin-bottom: 1rem;"),
		), functional.Text("About Vango")),
		
		functional.Div(functional.MergeProps(
			functional.StyleAttr(` + "`" + `
				background: white;
				padding: 2rem;
				border-radius: 0.75rem;
				box-shadow: 0 2px 10px rgba(0,0,0,0.1);
				margin-bottom: 2rem;
			` + "`" + `),
		),
			functional.H2(functional.MergeProps(
				functional.StyleAttr("color: #667eea; margin-bottom: 1rem;"),
			), functional.Text("What is Vango?")),
			
			functional.P(functional.MergeProps(
				functional.StyleAttr("color: #666; line-height: 1.8; margin-bottom: 1rem;"),
			), functional.Text(
				"Vango is a modern web framework that brings the power and simplicity of Go to frontend development. " +
				"By compiling to WebAssembly, Vango enables you to write entire web applications in Go without " +
				"sacrificing performance or developer experience.",
			)),
			
			functional.H3(functional.MergeProps(
				functional.StyleAttr("color: #333; margin: 1.5rem 0 1rem 0;"),
			), functional.Text("Why Choose Vango?")),
			
			functional.Ul(functional.MergeProps(
				functional.StyleAttr("color: #666; line-height: 2; padding-left: 1.5rem;"),
			),
				functional.Li(nil, functional.Text("🚀 "), functional.Strong(nil, functional.Text("Go Native:")), functional.Text(" Write everything in Go, from backend to frontend")),
				functional.Li(nil, functional.Text("⚡ "), functional.Strong(nil, functional.Text("Performance:")), functional.Text(" Near-native speed with WebAssembly")),
				functional.Li(nil, functional.Text("🔒 "), functional.Strong(nil, functional.Text("Type Safety:")), functional.Text(" Catch errors at compile time, not runtime")),
				functional.Li(nil, functional.Text("🔄 "), functional.Strong(nil, functional.Text("Hot Reload:")), functional.Text(" See changes instantly during development")),
				functional.Li(nil, functional.Text("📦 "), functional.Strong(nil, functional.Text("No Build Step:")), functional.Text(" No webpack, no babel, just Go")),
			),
		),
		
		functional.Div(functional.MergeProps(
			functional.StyleAttr("text-align: center;"),
		),
			functional.A(functional.MergeProps(
				functional.Href("/"),
				functional.StyleAttr(` + "`" + `
					display: inline-block;
					padding: 0.75rem 1.5rem;
					background: #f8f9fa;
					color: #667eea;
					text-decoration: none;
					border-radius: 0.5rem;
					font-weight: 500;
					border: 1px solid #e9ecef;
				` + "`" + `),
			), functional.Text("← Back to Home")),
		),
	)
}`

	return WriteFile(filepath.Join(config.Directory, "app/routes/about.go"), content)
}

// createServerRoute creates the server component demo page
func (t *BasicTemplate) createServerRoute(config *ProjectConfig) error {
	content := `package server

import (
	"time"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// //vango:server
// Page demonstrates server-side components
// This component runs ONLY on the server
func Page() *vdom.VNode {
	// Simulate server-side data fetching
	currentTime := time.Now().Format("15:04:05 MST")
	serverInfo := "Linux vango-server 5.15.0"
	
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
		), functional.Text("Server Components")),
		
		// Description
		functional.Div(functional.MergeProps(
			functional.Class("bg-yellow-50 dark:bg-yellow-900/20 border-l-4 border-yellow-500 p-6 rounded-r-lg mb-12"),
		),
			functional.P(functional.MergeProps(
				functional.Class("text-yellow-900 dark:text-yellow-100"),
			), functional.Text("Server components run exclusively on the server. They're perfect for secure operations, database queries, and initial data fetching.")),
		),
		
		// Server Info Display
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Server-Side Data")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 space-y-4"),
			),
				infoRow("⏰ Server Time", currentTime),
				infoRow("🖥️ Server OS", serverInfo),
				infoRow("🔒 Secure Data", "This data never reaches the client"),
				infoRow("🗄️ Database Access", "Direct database queries possible"),
			),
		),
		
		// Code Example
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("How It Works")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
			),
				functional.Pre(functional.MergeProps(
					functional.Class("text-sm text-gray-300"),
				), functional.Text(` + "`" + `// Mark component as server-only with directive
//vango:server

func Page() *vdom.VNode {
    // This code runs on the server
    secretKey := os.Getenv("SECRET_API_KEY")
    data := fetchFromDatabase()
    
    // The rendered HTML is sent to client
    // But the code never reaches the browser
    return functional.Div(nil,
        functional.Text(data),
    )
}` + "`" + `)),
			),
		),
		
		// Use Cases
		functional.Section(nil,
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("When to Use Server Components")),
			
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-2 gap-6"),
			),
				useCaseCard("🔐 Security", "Handle sensitive data that should never reach the client"),
				useCaseCard("🗄️ Database", "Direct database queries without API endpoints"),
				useCaseCard("🚀 Performance", "Heavy computations done on powerful servers"),
				useCaseCard("📊 SEO", "Content rendered on server for search engines"),
			),
		),
	)
}

func infoRow(label, value string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("flex justify-between items-center py-3 border-b border-gray-200 dark:border-gray-700 last:border-0"),
	),
		functional.Span(functional.MergeProps(
			functional.Class("font-medium text-gray-700 dark:text-gray-300"),
		), functional.Text(label)),
		functional.Span(functional.MergeProps(
			functional.Class("text-gray-900 dark:text-white font-mono"),
		), functional.Text(value)),
	)
}

func useCaseCard(icon, title, description string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-2xl mb-3"),
		), functional.Text(icon)),
		functional.H3(functional.MergeProps(
			functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-2"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-400 text-sm"),
		), functional.Text(description)),
	)
}
`

	// Create the server directory
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes/server"), 0755); err != nil {
		return fmt.Errorf("failed to create server directory: %w", err)
	}

	return WriteFile(filepath.Join(config.Directory, "app/routes/server/demo.go"), content)
}

// createClientRoute creates the client component demo page
func (t *BasicTemplate) createClientRoute(config *ProjectConfig) error {
	content := `package client

import (
	"fmt"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// //vango:client
// Page demonstrates client-side components
// This component runs in the browser via WebAssembly
func Page() *vdom.VNode {
	// This would normally have reactive state
	// For demo, we show the structure
	
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
		), functional.Text("Client Components")),
		
		// Description
		functional.Div(functional.MergeProps(
			functional.Class("bg-indigo-50 dark:bg-indigo-900/20 border-l-4 border-indigo-500 p-6 rounded-r-lg mb-12"),
		),
			functional.P(functional.MergeProps(
				functional.Class("text-indigo-900 dark:text-indigo-100"),
			), functional.Text("Client components run in the browser via WebAssembly. They enable rich interactivity, real-time updates, and stateful UIs without JavaScript.")),
		),
		
		// Interactive Demo
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Interactive Features")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8"),
			),
				// Counter Example
				functional.Div(functional.MergeProps(
					functional.Class("text-center mb-8"),
				),
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
					), functional.Text("Client-Side Counter")),
					functional.Div(functional.MergeProps(
						functional.Class("flex items-center justify-center space-x-4"),
					),
						functional.Button(functional.MergeProps(
							functional.Class("px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700"),
							vdom.Props{"onclick": "decrement()"},
						), functional.Text("-")),
						functional.Span(functional.MergeProps(
							functional.Class("text-3xl font-bold text-gray-900 dark:text-white px-8"),
						), functional.Text("0")),
						functional.Button(functional.MergeProps(
							functional.Class("px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700"),
							vdom.Props{"onclick": "increment()"},
						), functional.Text("+")),
					),
					functional.P(functional.MergeProps(
						functional.Class("mt-4 text-sm text-gray-600 dark:text-gray-400"),
					), functional.Text("State managed entirely in the browser")),
				),
				
				// Form Example
				functional.Div(nil,
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
					), functional.Text("Client-Side Form")),
					functional.Div(functional.MergeProps(
						functional.Class("space-y-4"),
					),
						functional.Input(functional.MergeProps(
							functional.Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700"),
							functional.Type("text"),
							functional.Placeholder("Type something..."),
						)),
						functional.P(functional.MergeProps(
							functional.Class("text-gray-700 dark:text-gray-300"),
						), functional.Text("You typed: (updates in real-time)")),
					),
				),
			),
		),
		
		// Code Example
		functional.Section(functional.MergeProps(
			functional.Class("mb-12"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Client Component Code")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gray-900 rounded-lg p-6 overflow-x-auto"),
			),
				functional.Pre(functional.MergeProps(
					functional.Class("text-sm text-gray-300"),
				), functional.Text(fmt.Sprintf(` + "`" + `//vango:client

func Page() *vdom.VNode {
    // Create reactive state
    count := reactive.Signal(0)
    text := reactive.Signal("")
    
    return functional.Div(nil,
        functional.Button(functional.MergeProps(
            vdom.Props{
                "onclick": func() { 
                    count.Set(count.Get() + 1) 
                },
            },
        ), functional.Text("Click me")),
        
        functional.Text(fmt.Sprintf("Count: %%d", count.Get())),
    )
}` + "`" + `, "%d"))),
			),
		),
		
		// Benefits
		functional.Section(nil,
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-6"),
			), functional.Text("Benefits of Client Components")),
			
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-3 gap-6"),
			),
				benefitCard("⚡", "Instant Updates", "No server round-trips for UI updates"),
				benefitCard("🎮", "Rich Interactions", "Complex UIs with drag & drop, animations"),
				benefitCard("📱", "Offline Support", "Works without server connection"),
				benefitCard("🔄", "Real-time", "WebSocket integration for live data"),
				benefitCard("💾", "Local State", "Persist data in browser storage"),
				benefitCard("🚀", "Zero JavaScript", "Pure Go compiled to WebAssembly"),
			),
		),
	)
}

func benefitCard(icon, title, description string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-gradient-to-br from-indigo-50 to-purple-50 dark:from-indigo-900/20 dark:to-purple-900/20 p-6 rounded-lg"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-2xl mb-2"),
		), functional.Text(icon)),
		functional.H3(functional.MergeProps(
			functional.Class("font-semibold text-gray-900 dark:text-white mb-1"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-sm text-gray-600 dark:text-gray-400"),
		), functional.Text(description)),
	)
}
`

	// Create the client directory
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes/client"), 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	return WriteFile(filepath.Join(config.Directory, "app/routes/client/demo.go"), content)
}

// createFeaturesRoute creates the features showcase page
func (t *BasicTemplate) createFeaturesRoute(config *ProjectConfig) error {
	content := `package features

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Page showcases all Vango features
func Page() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8 text-center"),
		), functional.Text("Vango Features Overview")),
		
		// Core Features
		functional.Section(functional.MergeProps(
			functional.Class("mb-16"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-3xl font-semibold text-gray-900 dark:text-white mb-8 text-center"),
			), functional.Text("🎯 Core Features")),
			
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-3 gap-8"),
			),
				featureCard(
					"🚀",
					"Pure Go Frontend",
					"Write your entire application in Go. No JavaScript required.",
					[]string{"Type-safe", "Familiar syntax", "Great tooling"},
				),
				featureCard(
					"⚡",
					"WebAssembly Performance",
					"Near-native performance with Go compiled to WASM.",
					[]string{"Fast execution", "Small bundles", "Efficient memory"},
				),
				featureCard(
					"🔄",
					"Hot Module Reload",
					"See changes instantly without losing application state.",
					[]string{"Fast iteration", "Preserves state", "Auto-refresh"},
				),
			),
		),
		
		// Developer Experience
		functional.Section(functional.MergeProps(
			functional.Class("mb-16"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-3xl font-semibold text-gray-900 dark:text-white mb-8 text-center"),
			), functional.Text("💻 Developer Experience")),
			
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-2 gap-8"),
			),
				dxCard(
					"📁 File-Based Routing",
					"Routes are automatically generated from your file structure.",
					"app/routes/about.go → /about",
				),
				dxCard(
					"🎨 VEX Syntax Layers",
					"Choose your preferred syntax: functional, fluent, or templates.",
					"Layer 0: Functional | Layer 1: Fluent | Layer 2: Templates",
				),
				dxCard(
					"🔧 Powerful CLI",
					"Single tool for development, building, and deployment.",
					"vango dev | vango build | vango deploy",
				),
				dxCard(
					"📦 Zero Config",
					"Works out of the box with sensible defaults.",
					"Optional vango.json for customization",
				),
			),
		),
		
		// Architecture
		functional.Section(functional.MergeProps(
			functional.Class("mb-16"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-3xl font-semibold text-gray-900 dark:text-white mb-8 text-center"),
			), functional.Text("🏗️ Architecture")),
			
			functional.Div(functional.MergeProps(
				functional.Class("bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 rounded-lg p-8"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("grid md:grid-cols-2 gap-8"),
				),
					archCard("Virtual DOM", "Efficient diffing algorithm for minimal DOM updates"),
					archCard("Reactive State", "Signal-based reactivity system for automatic UI updates"),
					archCard("SSR + Hydration", "Server-side rendering with seamless client hydration"),
					archCard("Live Protocol", "WebSocket-based live updates from server to client"),
				),
			),
		),
		
		// Call to Action
		functional.Section(nil,
			functional.Div(functional.MergeProps(
				functional.Class("bg-gradient-to-r from-purple-600 to-blue-600 rounded-lg p-12 text-center text-white"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold mb-4"),
				), functional.Text("Ready to Build with Vango?")),
				functional.P(functional.MergeProps(
					functional.Class("text-xl mb-8"),
				), functional.Text("Start building modern web applications with the power of Go")),
				functional.Div(functional.MergeProps(
					functional.Class("flex justify-center space-x-4"),
				),
					functional.A(functional.MergeProps(
						functional.Href("https://github.com/recera/vango"),
						functional.Target("_blank"),
						functional.Class("px-8 py-4 bg-white text-purple-600 rounded-lg font-semibold hover:bg-gray-100 transition-colors"),
					), functional.Text("Get Started")),
					functional.A(functional.MergeProps(
						functional.Href("https://vango.dev/docs"),
						functional.Target("_blank"),
						functional.Class("px-8 py-4 bg-purple-800 text-white rounded-lg font-semibold hover:bg-purple-900 transition-colors"),
					), functional.Text("Read Docs")),
				),
			),
		),
	)
}

func featureCard(icon, title, description string, features []string) *vdom.VNode {
	var featureItems []*vdom.VNode
	for _, feature := range features {
		featureItems = append(featureItems, functional.Li(functional.MergeProps(
			functional.Class("flex items-center space-x-2"),
		),
			functional.Span(functional.MergeProps(
				functional.Class("text-green-500"),
			), functional.Text("✓")),
			functional.Span(nil, functional.Text(feature)),
		))
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 hover:shadow-xl transition-shadow"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-4xl mb-4 text-center"),
		), functional.Text(icon)),
		functional.H3(functional.MergeProps(
			functional.Class("text-xl font-bold text-gray-900 dark:text-white mb-3 text-center"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-300 mb-4"),
		), functional.Text(description)),
		functional.Ul(functional.MergeProps(
			functional.Class("space-y-2 text-sm text-gray-700 dark:text-gray-400"),
		), featureItems...),
	)
}

func dxCard(title, description, example string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md"),
	),
		functional.H3(functional.MergeProps(
			functional.Class("text-lg font-bold text-gray-900 dark:text-white mb-2"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-300 mb-3"),
		), functional.Text(description)),
		functional.Code(functional.MergeProps(
			functional.Class("block bg-gray-100 dark:bg-gray-900 p-2 rounded text-sm text-gray-800 dark:text-gray-200"),
		), functional.Text(example)),
	)
}

func archCard(title, description string) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("bg-white/80 dark:bg-gray-800/80 backdrop-blur rounded-lg p-6"),
	),
		functional.H3(functional.MergeProps(
			functional.Class("font-bold text-gray-900 dark:text-white mb-2"),
		), functional.Text(title)),
		functional.P(functional.MergeProps(
			functional.Class("text-gray-600 dark:text-gray-300 text-sm"),
		), functional.Text(description)),
	)
}
`

	// Create the features directory
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes/features"), 0755); err != nil {
		return fmt.Errorf("failed to create features directory: %w", err)
	}

	return WriteFile(filepath.Join(config.Directory, "app/routes/features/index.go"), content)
}

// createStyles creates custom styles for the template
func (t *BasicTemplate) createStyles(config *ProjectConfig) error {
	content := `/* Custom styles for Vango showcase app */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
	html {
		@apply antialiased scroll-smooth;
	}
	
	body {
		@apply bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100;
	}
}

@layer components {
	/* Custom button styles */
	.btn {
		@apply px-4 py-2 rounded-lg font-medium transition-all duration-200;
	}
	
	.btn-primary {
		@apply bg-purple-600 text-white hover:bg-purple-700 active:scale-95;
	}
	
	.btn-secondary {
		@apply bg-gray-600 text-white hover:bg-gray-700 active:scale-95;
	}
	
	/* Card styles */
	.card {
		@apply bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6;
	}
	
	/* Code block styles */
	.code-block {
		@apply bg-gray-900 text-gray-300 rounded-lg p-4 overflow-x-auto;
	}
}

/* Smooth transitions for dark mode */
* {
	transition-property: background-color, border-color;
	transition-duration: 300ms;
	transition-timing-function: ease-in-out;
}

/* Gradient text animation */
@keyframes gradient-shift {
	0%, 100% { background-position: 0% 50%; }
	50% { background-position: 100% 50%; }
}

.animate-gradient {
	background-size: 200% 200%;
	animation: gradient-shift 3s ease infinite;
}

/* Custom scrollbar */
::-webkit-scrollbar {
	width: 8px;
	height: 8px;
}

::-webkit-scrollbar-track {
	@apply bg-gray-100 dark:bg-gray-800;
}

::-webkit-scrollbar-thumb {
	@apply bg-gray-400 dark:bg-gray-600 rounded-full;
}

::-webkit-scrollbar-thumb:hover {
	@apply bg-gray-500 dark:bg-gray-500;
}
`

	return WriteFile(filepath.Join(config.Directory, "styles/app.css"), content)
}

// createTailwindConfig creates the Tailwind configuration
func (t *BasicTemplate) createTailwindConfig(config *ProjectConfig) error {
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
        'vango-purple': '#667eea',
        'vango-blue': '#764ba2',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.5s ease-in-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [],
}`

	return WriteFile(filepath.Join(config.Directory, "tailwind.config.js"), content)
}
