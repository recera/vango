package cli_templates

import (
	"fmt"
	"path/filepath"
)

func init() {
	Register("base", &BaseTemplate{})
}

// BaseTemplate is the configurable base template generator
type BaseTemplate struct{}

func (t *BaseTemplate) Name() string {
	return "base"
}

func (t *BaseTemplate) Description() string {
	return "Configurable base template"
}

func (t *BaseTemplate) Generate(config *ProjectConfig) error {
	// Generate based on routing strategy
	switch config.RoutingStrategy {
	case "file-based":
		if err := t.generateFileBased(config); err != nil {
			return err
		}
	case "programmatic":
		if err := t.generateProgrammatic(config); err != nil {
			return err
		}
	case "minimal":
		if err := t.generateMinimal(config); err != nil {
			return err
		}
	default:
		// Default to file-based
		config.RoutingStrategy = "file-based"
		if err := t.generateFileBased(config); err != nil {
			return err
		}
	}

	return nil
}

// createComponents creates reusable component files
func (t *BaseTemplate) createComponents(config *ProjectConfig) error {
	// Create Card component (Layer 1 VEX - Builder API)
	cardContent := `package components

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// CardProps defines properties for the Card component
type CardProps struct {
	Title       string
	Description string
	Footer      string
	OnClick     func()
}

// Card creates a reusable card component using Layer 1 VEX (Builder API)
func Card(props CardProps) *vdom.VNode {
	card := builder.Div().
		Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 hover:shadow-xl transition-shadow")
	
	if props.OnClick != nil {
		card.OnClick(props.OnClick)
		card.Class("cursor-pointer")
	}
	
	var children []*vdom.VNode
	
	if props.Title != "" {
		children = append(children, 
			builder.H3().
				Class("text-xl font-semibold text-gray-900 dark:text-white mb-3").
				Text(props.Title).
				Build(),
		)
	}
	
	if props.Description != "" {
		children = append(children, 
			builder.P().
				Class("text-gray-600 dark:text-gray-300 mb-4").
				Text(props.Description).
				Build(),
		)
	}
	
	if props.Footer != "" {
		children = append(children, 
			builder.Div().
				Class("text-sm text-gray-500 dark:text-gray-400 border-t pt-3 mt-auto").
				Text(props.Footer).
				Build(),
		)
	}
	
	return card.Children(children...).Build()
}
`

	if err := WriteFile(filepath.Join(config.Directory, "app/components/card.go"), cardContent); err != nil {
		return err
	}

	// Create Navigation component
	navContent := `package components

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// Navigation creates the shared navigation bar
func Navigation() *vdom.VNode {
	return builder.Nav().
		Class("bg-white dark:bg-gray-800 shadow-md").
		Children(
			builder.Div().
				Class("container mx-auto px-6 py-4").
				Children(
					builder.Div().
						Class("flex items-center justify-between").
						Children(
							// Logo
							builder.Button().
								Class("text-2xl font-bold text-gray-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors").
								OnClick(func() {
									// Navigation handled by JavaScript
								}).
								Attr("onclick", "navigateTo('/')").
								Text("🚀 Vango").
								Build(),
							
							// Navigation links
							builder.Div().
								Class("flex items-center space-x-6").
								Children(
									builder.Button().
										Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
										OnClick(func() {
											// Navigation handled by JavaScript
										}).
										Attr("onclick", "navigateTo('/')").
										Text("Home").
										Build(),
									builder.Button().
										Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
										OnClick(func() {
											// Navigation handled by JavaScript
										}).
										Attr("onclick", "navigateTo('/about')").
										Text("About").
										Build(),
									builder.Button().
										Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
										OnClick(func() {
											// Navigation handled by JavaScript
										}).
										Attr("onclick", "navigateTo('/counter')").
										Text("Counter").
										Build(),
									
									// Dark mode toggle
									builder.Button().
										Class("p-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors").
										OnClick(func() {
											// Dark mode toggle handled by JavaScript
										}).
										Attr("onclick", "toggleDarkMode()").
										Title("Toggle dark mode").
										Text("🌙").
										Build(),
								).Build(),
						).Build(),
				).Build(),
		).Build()
}
`

	if err := WriteFile(filepath.Join(config.Directory, "app/components/navigation.go"), navContent); err != nil {
		return err
	}

	// Create Footer component
	footerContent := `package components

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// Footer creates the shared footer
func Footer() *vdom.VNode {
	return builder.Footer().
		Class("bg-gray-800 text-white mt-16").
		Children(
			builder.Div().
				Class("container mx-auto px-6 py-8").
				Children(
					builder.Div().
						Class("text-center").
						Children(
							builder.P().
								Class("text-gray-400 mb-2").
								Text("Built with ❤️ using Vango - The Go Frontend Framework").
								Build(),
							builder.P().
								Class("text-sm text-gray-500").
								Children(
									builder.A().
										Href("https://github.com/recera/vango").
										Target("_blank").
										Class("hover:text-gray-300 transition-colors").
										Text("GitHub").
										Build(),
									builder.Span().Class("mx-2").Text("•").Build(),
									builder.A().
										Href("https://vango.dev/docs").
										Target("_blank").
										Class("hover:text-gray-300 transition-colors").
										Text("Documentation").
										Build(),
									builder.Span().Class("mx-2").Text("•").Build(),
									builder.A().
										Href("https://discord.gg/vango").
										Target("_blank").
										Class("hover:text-gray-300 transition-colors").
										Text("Community").
										Build(),
								).Build(),
						).Build(),
				).Build(),
		).Build()
}
`

	if err := WriteFile(filepath.Join(config.Directory, "app/components/footer.go"), footerContent); err != nil {
		return err
	}

	// Create FeatureItem component
	featureContent := `package components

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// FeatureItemProps defines properties for the FeatureItem component
type FeatureItemProps struct {
	Icon        string
	Title       string
	Description string
}

// FeatureItem creates a feature list item component
func FeatureItem(props FeatureItemProps) *vdom.VNode {
	return builder.Div().
		Class("flex space-x-4").
		Children(
			builder.Div().
				Class("flex-shrink-0 text-2xl").
				Text(props.Icon).
				Build(),
			
			builder.Div().
				Children(
					builder.H4().
						Class("font-semibold text-gray-800 dark:text-gray-200 mb-1").
						Text(props.Title).
						Build(),
					builder.P().
						Class("text-sm text-gray-600 dark:text-gray-400").
						Text(props.Description).
						Build(),
				).Build(),
		).Build()
}
`

	return WriteFile(filepath.Join(config.Directory, "app/components/feature_item.go"), featureContent)
}

// createTemplateFile creates a Layer 2 VEX template example comment file
func (t *BaseTemplate) createTemplateFile(config *ProjectConfig) error {
	// Create a template example comment file (not actual template since processor not implemented yet)
	templateContent := `package components

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// TemplateExample shows what a Layer 2 VEX template would look like
// NOTE: This is a comment example. When the VEX template processor is implemented,
// this would be written in a .vex file with the following syntax:
/*
//vango:template
package routes
//vango:props { Features []Feature }

type Feature struct {
	Icon        string
	Title       string
	Description string
}

<div class="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
	<h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">
		Features Showcase (Layer 2 VEX Template)
	</h2>
	
	{{#if len(.Features) > 0}}
		<div class="grid md:grid-cols-2 gap-6">
			{{#for feature in .Features}}
				<div class="flex space-x-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
					<div class="text-3xl">{{feature.Icon}}</div>
					<div>
						<h3 class="font-semibold text-gray-800 dark:text-gray-200">
							{{feature.Title}}
						</h3>
						<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
							{{feature.Description}}
						</p>
					</div>
				</div>
			{{/for}}
		</div>
	{{else}}
		<p class="text-gray-500 dark:text-gray-400 text-center">
			No features to display
		</p>
	{{/if}}
</div>
*/

// For now, here's the equivalent using Layer 1 VEX Builder API:

type Feature struct {
	Icon        string
	Title       string
	Description string
}

type FeaturesProps struct {
	Features []Feature
}

func FeaturesShowcase(props FeaturesProps) *vdom.VNode {
	container := builder.Div().
		Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8")
	
	var children []*vdom.VNode
	
	// Title
	children = append(children,
		builder.H2().
			Class("text-2xl font-bold text-gray-900 dark:text-white mb-6").
			Text("Features Showcase (Layer 1 VEX - Builder API)").
			Build(),
	)
	
	// Features grid or empty message
	if len(props.Features) > 0 {
		var featureItems []*vdom.VNode
		for _, feature := range props.Features {
			featureItems = append(featureItems,
				builder.Div().
					Class("flex space-x-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg").
					Children(
						builder.Div().
							Class("text-3xl").
							Text(feature.Icon).
							Build(),
						builder.Div().
							Children(
								builder.H3().
									Class("font-semibold text-gray-800 dark:text-gray-200").
									Text(feature.Title).
									Build(),
								builder.P().
									Class("text-sm text-gray-600 dark:text-gray-400 mt-1").
									Text(feature.Description).
									Build(),
							).Build(),
					).Build(),
			)
		}
		
		children = append(children,
			builder.Div().
				Class("grid md:grid-cols-2 gap-6").
				Children(featureItems...).
				Build(),
		)
	} else {
		children = append(children,
			builder.P().
				Class("text-gray-500 dark:text-gray-400 text-center").
				Text("No features to display").
				Build(),
		)
	}
	
	// Info note
	children = append(children,
		builder.Div().
			Class("mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg").
			Children(
				builder.P().
					Class("text-sm text-blue-800 dark:text-blue-200").
					Children(
						builder.Strong().Text("Note: ").Build(),
						builder.Span().
							Text("When VEX template processing is implemented, this component can be written using HTML-like syntax with Go template directives.").
							Build(),
					).Build(),
			).Build(),
	)
	
	return container.Children(children...).Build()
}
`

	return WriteFile(filepath.Join(config.Directory, "app/components/template_example.go"), templateContent)
}

// generateFileBased creates file-based routing structure
func (t *BaseTemplate) generateFileBased(config *ProjectConfig) error {
	// Create app/main.go for client-side bootstrap
	mainContent := `//go:build wasm
// +build wasm

package main

import (
	"fmt"
	"syscall/js"
	routes "%s/app/routes"
	"github.com/recera/vango/pkg/vango/vdom"
)

func main() {
	// Initialize Vango runtime
	js.Global().Get("console").Call("log", "🚀 Vango app starting...")
	
	// Initialize the app
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
	console.Call("log", "DOM ready, initializing app...")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Get app root element
	appRoot := js.Global().Get("document").Call("getElementById", "app")
	if appRoot.IsNull() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Render the initial page based on path
	path := js.Global().Get("window").Get("location").Get("pathname").String()
	var page *vdom.VNode
	
	switch path {
	case "/about":
		page = routes.AboutPage()
	case "/counter":
		page = routes.CounterPage()
	default:
		page = routes.IndexPage()
	}
	
	if page != nil {
		renderVNode(page)
	}
	
	// Set up navigation handler
	js.Global().Set("navigateTo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newPath := args[0].String()
			js.Global().Get("window").Get("history").Call("pushState", nil, "", newPath)
			
			// Re-render based on new path
			var newPage *vdom.VNode
			switch newPath {
			case "/about":
				newPage = routes.AboutPage()
			case "/counter":
				newPage = routes.CounterPage()
			default:
				newPage = routes.IndexPage()
			}
			
			if newPage != nil {
				renderVNode(newPage)
			}
		}
		return nil
	}))
	
	// Set up dark mode toggle
	js.Global().Set("toggleDarkMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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
		return nil
	}))
	
	// Handle browser back/forward buttons
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		path := js.Global().Get("window").Get("location").Get("pathname").String()
		var page *vdom.VNode
		
		switch path {
		case "/about":
			page = routes.AboutPage()
		case "/counter":
			page = routes.CounterPage()
		default:
			page = routes.IndexPage()
		}
		
		if page != nil {
			renderVNode(page)
		}
		return nil
	}))
	
	// Set up counter functionality
	counterValue := 0
	js.Global().Set("updateCounter", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			delta := args[0].Int()
			if delta == 0 {
				counterValue = 0 // Reset
			} else {
				counterValue += delta
			}
			
			// Update the counter display directly
			counterElem := js.Global().Get("document").Call("getElementById", "counter-value")
			if !counterElem.IsNull() && !counterElem.IsUndefined() {
				counterElem.Set("textContent", fmt.Sprintf("%%d", counterValue))
			}
		}
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
		// Check system preference
		prefersDark := js.Global().Get("window").
			Call("matchMedia", "(prefers-color-scheme: dark)").
			Get("matches").Bool()
		
		if prefersDark {
			document.Get("documentElement").Get("classList").Call("add", "dark")
			localStorage.Call("setItem", "darkMode", "true")
		}
	}
}

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering VNode...")
	
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
				case "style":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "style", v)
					}
				case "onclick":
					// Handle click events - can be either a function or a string
					switch v := value.(type) {
					case func():
						elem.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
							v()
							return nil
						}))
					case string:
						// For string onclick handlers (e.g., "navigateTo('/')")
						elem.Call("setAttribute", "onclick", v)
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
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}
`

	mainContent = fmt.Sprintf(mainContent, config.Module)
	if err := WriteFile(filepath.Join(config.Directory, "app/main.go"), mainContent); err != nil {
		return err
	}

	// Create app/routes/index.go (showcasing Layer 1 VEX - Builder API)
	indexContent := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
	components "%s/app/components"
)

// IndexPage is the home page showcasing Layer 1 VEX (Builder API)
func IndexPage() *vdom.VNode {
	// Using the Builder API (Layer 1 VEX)
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			// Navigation bar
			builder.Nav().
				Class("bg-white dark:bg-gray-800 shadow-md").
				Children(
					builder.Div().
						Class("container mx-auto px-6 py-4").
						Children(
							builder.Div().
								Class("flex items-center justify-between").
								Children(
									// Logo
									builder.H1().
										Class("text-2xl font-bold text-gray-900 dark:text-white").
										Text("🚀 Vango").
										Build(),
									
									// Navigation links
									builder.Div().
										Class("flex items-center space-x-6").
										Children(
											builder.A().
												Href("/").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("Home").
												Build(),
											builder.A().
												Href("/about").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("About").
												Build(),
											builder.A().
												Href("/counter").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("Counter").
												Build(),
											
											// Dark mode toggle
											builder.Button().
												Class("p-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors").
												OnClick(func() {
													// This will be handled by JavaScript
												}).
												Attr("onclick", "toggleDarkMode()").
												Title("Toggle dark mode").
												Text("🌙").
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
			
			// Main content
			builder.Main().
				Class("container mx-auto px-6 py-12").
				Children(
					// Hero section
					builder.Section().
						Class("text-center mb-16").
						Children(
							builder.H1().
								Class("text-5xl font-bold text-gray-900 dark:text-white mb-6").
								Text("Welcome to Vango").
								Build(),
							builder.P().
								Class("text-xl text-gray-600 dark:text-gray-300 max-w-3xl mx-auto mb-8").
								Text("Build blazing-fast web applications with Go and WebAssembly. Experience the power of server-driven components and reactive state management.").
								Build(),
							
							// CTA buttons
							builder.Div().
								Class("flex justify-center space-x-4").
								Children(
									builder.Button().
										Class("px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium shadow-lg").
										Attr("onclick", "navigateTo('/about')").
										Text("Learn More").
										Build(),
									builder.A().
										Href("https://github.com/recera/vango").
										Target("_blank").
										Class("px-6 py-3 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors font-medium shadow-lg").
										Text("View on GitHub").
										Build(),
								).Build(),
						).Build(),
					
					// Features section using custom Card components
					builder.Section().
						Children(
							builder.H2().
								Class("text-3xl font-bold text-center text-gray-900 dark:text-white mb-12").
								Text("Key Features").
								Build(),
							
							builder.Div().
								Class("grid md:grid-cols-3 gap-8").
								Children(
									// Feature cards using custom component
									components.Card(components.CardProps{
										Title:       "⚡ Blazing Fast",
										Description: "Compiled Go to WebAssembly delivers near-native performance in the browser.",
										Footer:      "Powered by TinyGo",
									}),
									components.Card(components.CardProps{
										Title:       "🔄 Live Updates",
										Description: "Server-driven components with real-time updates over WebSocket connection.",
										Footer:      "No JavaScript required",
									}),
									components.Card(components.CardProps{
										Title:       "🎨 Flexible Syntax",
										Description: "Choose between functional, builder, or template syntax based on your preference.",
										Footer:      "Three layers of VEX",
									}),
								).Build(),
						).Build(),
					
					// Code example section
					builder.Section().
						Class("mt-16").
						Children(
							builder.H2().
								Class("text-3xl font-bold text-center text-gray-900 dark:text-white mb-8").
								Text("Simple to Use").
								Build(),
							
							builder.Div().
								Class("bg-gray-900 rounded-lg p-6 max-w-3xl mx-auto").
								Children(
									builder.Pre().
										Class("text-green-400 overflow-x-auto").
										Children(
											builder.Code().
												Text(` + "`" + `// Using Layer 1 VEX - Builder API
func MyComponent() *vdom.VNode {
    return builder.Div().
        Class("p-4 bg-white rounded").
        Children(
            builder.H1().Text("Hello Vango!").Build(),
            builder.P().Text("Building UIs in Go").Build(),
        ).Build()
}` + "`" + `).
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
			
			// Footer
			builder.Footer().
				Class("bg-gray-800 text-white mt-16").
				Children(
					builder.Div().
						Class("container mx-auto px-6 py-8 text-center").
						Children(
							builder.P().
								Class("text-gray-400").
								Text("Built with ❤️ using Vango - The Go Frontend Framework").
								Build(),
						).Build(),
				).Build(),
		).Build()
}
`

	indexContent = fmt.Sprintf(indexContent, config.Module)

	if err := WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), indexContent); err != nil {
		return err
	}

	// Create app/routes/about.go (Layer 2 VEX - Template Macro will be created separately)
	aboutContent := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
	components "%s/app/components"
)

// AboutPage demonstrates more complex layouts and composition
func AboutPage() *vdom.VNode {
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			// Reusing navigation from components
			components.Navigation(),
			
			// Main content
			builder.Main().
				Class("container mx-auto px-6 py-12").
				Children(
					builder.H1().
						Class("text-4xl font-bold text-gray-900 dark:text-white mb-8").
						Text("About Vango").
						Build(),
					
					builder.Div().
						Class("grid lg:grid-cols-2 gap-12").
						Children(
							// Left column - content
							builder.Div().
								Children(
									builder.Section().
										Class("mb-8").
										Children(
											builder.H2().
												Class("text-2xl font-semibold text-gray-800 dark:text-gray-200 mb-4").
												Text("The Vision").
												Build(),
											builder.P().
												Class("text-gray-600 dark:text-gray-400 leading-relaxed mb-4").
												Text("Vango reimagines web development by bringing the power and simplicity of Go to the frontend. No more context switching between languages, no more complex build pipelines.").
												Build(),
											builder.P().
												Class("text-gray-600 dark:text-gray-400 leading-relaxed").
												Text("Write your entire application in Go, from backend to frontend, with type safety and excellent performance guaranteed.").
												Build(),
										).Build(),
									
									builder.Section().
										Class("mb-8").
										Children(
											builder.H2().
												Class("text-2xl font-semibold text-gray-800 dark:text-gray-200 mb-4").
												Text("Three Modes of Rendering").
												Build(),
											
											// Rendering modes list
											builder.Div().
												Class("space-y-4").
												Children(
													components.FeatureItem(components.FeatureItemProps{
														Icon:        "🌐",
														Title:       "Universal (SSR + Hydration)",
														Description: "Server-side rendering for SEO with client-side hydration for interactivity.",
													}),
													components.FeatureItem(components.FeatureItemProps{
														Icon:        "🔄",
														Title:       "Server-Driven",
														Description: "All state on the server, UI updates via WebSocket patches.",
													}),
													components.FeatureItem(components.FeatureItemProps{
														Icon:        "💻",
														Title:       "Client-Only (CSR)",
														Description: "Full client-side rendering for offline-capable applications.",
													}),
												).Build(),
										).Build(),
								).Build(),
							
							// Right column - features showcase
							builder.Div().
								Children(
									// VEX Syntax Examples
									builder.Section().
										Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6").
										Children(
											builder.H3().
												Class("text-xl font-semibold text-gray-800 dark:text-gray-200 mb-4").
												Text("VEX Syntax Layers").
												Build(),
											
											// Layer examples
											builder.Div().
												Class("space-y-4").
												Children(
													// Functional example
													builder.Div().
														Children(
															builder.H4().
																Class("font-medium text-gray-700 dark:text-gray-300 mb-2").
																Text("Layer 0: Functional").
																Build(),
															builder.Pre().
																Class("bg-gray-100 dark:bg-gray-900 p-3 rounded text-sm overflow-x-auto").
																Children(
																	builder.Code().
																		Class("text-blue-600 dark:text-blue-400").
																		Text("vango.Div(nil, vango.Text(\"Hello\"))").
																		Build(),
																).Build(),
														).Build(),
													
													// Builder example
													builder.Div().
														Children(
															builder.H4().
																Class("font-medium text-gray-700 dark:text-gray-300 mb-2").
																Text("Layer 1: Builder").
																Build(),
															builder.Pre().
																Class("bg-gray-100 dark:bg-gray-900 p-3 rounded text-sm overflow-x-auto").
																Children(
																	builder.Code().
																		Class("text-green-600 dark:text-green-400").
																		Text("builder.Div().Text(\"Hello\").Build()").
																		Build(),
																).Build(),
														).Build(),
													
													// Template example
													builder.Div().
														Children(
															builder.H4().
																Class("font-medium text-gray-700 dark:text-gray-300 mb-2").
																Text("Layer 2: Template").
																Build(),
															builder.Pre().
																Class("bg-gray-100 dark:bg-gray-900 p-3 rounded text-sm overflow-x-auto").
																Children(
																	builder.Code().
																		Class("text-purple-600 dark:text-purple-400").
																		Text("<div>{{\"Hello\"}}</div>").
																		Build(),
																).Build(),
														).Build(),
												).Build(),
										).Build(),
									
									// Call to action
									builder.Div().
										Class("mt-8 text-center").
										Children(
											builder.Button().
												Class("inline-block px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg hover:from-blue-700 hover:to-purple-700 transition-all shadow-lg font-medium").
												Attr("onclick", "navigateTo('/counter')").
												Text("Try the Interactive Demo →").
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
			
			// Footer
			components.Footer(),
		).Build()
}
`

	aboutContent = fmt.Sprintf(aboutContent, config.Module)
	if err := WriteFile(filepath.Join(config.Directory, "app/routes/about.go"), aboutContent); err != nil {
		return err
	}

	// Create app/routes/counter.go (Universal component for now, server-driven when live protocol is ready)
	counterContent := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
	components "%s/app/components"
)

// CounterPage demonstrates an interactive counter component
// Note: This will be server-driven when live protocol is implemented
func CounterPage() *vdom.VNode {
	// For now, this starts at 0
	// When server-driven mode is available, this will maintain state on server
	
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			// Navigation
			components.Navigation(),
			
			// Main content
			builder.Main().
				Class("container mx-auto px-6 py-12").
				Children(
					builder.Div().
						Class("max-w-2xl mx-auto").
						Children(
							// Header
							builder.Div().
								Class("text-center mb-8").
								Children(
									builder.H1().
										Class("text-4xl font-bold text-gray-900 dark:text-white mb-4").
										Text("Interactive Counter Demo").
										Build(),
									builder.Div().
										Class("inline-flex items-center px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full text-sm font-medium").
										Children(
											builder.Span().Class("mr-2").Text("🔵").Build(),
											builder.Span().Text("Client Mode - Interactive demo").Build(),
										).Build(),
								).Build(),
							
							// Counter display
							builder.Div().
								Class("bg-white dark:bg-gray-800 rounded-lg shadow-xl p-8 mb-8").
								Children(
									builder.Div().
										Class("text-center mb-8").
										Children(
											builder.Div().
												ID("counter-value").
												Class("text-7xl font-bold text-blue-600 dark:text-blue-400 transition-all").
												Text("0").
												Build(),
										).Build(),
									
									// Control buttons
									builder.Div().
										Class("flex justify-center space-x-4").
										Children(
											builder.Button().
												Class("px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-medium shadow-md").
												Attr("onclick", "updateCounter(-1)").
												Text("− Decrement").
												Build(),
											
											builder.Button().
												Class("px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors font-medium shadow-md").
												Attr("onclick", "updateCounter(0)").
												Text("↺ Reset").
												Build(),
											
											builder.Button().
												Class("px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors font-medium shadow-md").
												Attr("onclick", "updateCounter(1)").
												Text("+ Increment").
												Build(),
										).Build(),
								).Build(),
							
							// Info box
							builder.Div().
								Class("bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-4 rounded").
								Children(
									builder.P().
										Class("text-sm text-blue-800 dark:text-blue-200").
										Children(
											builder.Strong().Text("Demo Note: ").Build(),
											builder.Span().
												Text("This is a demo of Vango's interactive capabilities. When the live protocol is fully implemented, this will support server-driven state management with WebSocket updates.").
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
			
			// Footer
			components.Footer(),
		).Build()
}
`

	counterContent = fmt.Sprintf(counterContent, config.Module)
	if err := WriteFile(filepath.Join(config.Directory, "app/routes/counter.go"), counterContent); err != nil {
		return err
	}

	// Create components directory and files
	if err := t.createComponents(config); err != nil {
		return err
	}

	// Create template file for Layer 2 VEX example
	if err := t.createTemplateFile(config); err != nil {
		return err
	}

	// Create _404.go
	notFoundContent := `package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
	components "%s/app/components"
)

// NotFoundPage renders the 404 page
func NotFoundPage() *vdom.VNode {
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			components.Navigation(),
			
			builder.Main().
				Class("container mx-auto px-6 py-12").
				Children(
					builder.Div().
						Class("text-center py-16").
						Children(
							builder.H1().
								Class("text-8xl font-bold text-gray-300 dark:text-gray-700 mb-4").
								Text("404").
								Build(),
							
							builder.P().
								Class("text-xl text-gray-600 dark:text-gray-400 mb-8").
								Text("Oops! The page you're looking for doesn't exist.").
								Build(),
							
							builder.Button().
								Class("inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors shadow-lg").
								Attr("onclick", "navigateTo('/')").
								Text("← Back to Home").
								Build(),
						).Build(),
				).Build(),
			
			components.Footer(),
		).Build()
}
`

	notFoundContent = fmt.Sprintf(notFoundContent, config.Module)
	return WriteFile(filepath.Join(config.Directory, "app/routes/_404.go"), notFoundContent)
}

// generateProgrammatic creates programmatic routing structure
func (t *BaseTemplate) generateProgrammatic(config *ProjectConfig) error {
	// Create app/main.go
	mainContent := `//go:build wasm
// +build wasm

package main

import (
	"syscall/js"
	handlers "%s/app/handlers"
	"%s/server"
	"github.com/recera/vango/pkg/vango/vdom"
)

func main() {
	// Initialize Vango runtime
	js.Global().Get("console").Call("log", "🚀 Vango app starting...")
	
	// Initialize router
	router := server.NewRouter()
	
	// Initialize the app
	initApp(router)
	
	// Keep the WASM runtime alive
	select {}
}

func initApp(router *server.Router) {
	document := js.Global().Get("document")
	
	// Wait for DOM ready
	if document.Get("readyState").String() != "loading" {
		onReady(router)
	} else {
		document.Call("addEventListener", "DOMContentLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			onReady(router)
			return nil
		}))
	}
}

func onReady(router *server.Router) {
	console := js.Global().Get("console")
	console.Call("log", "DOM ready, initializing app...")
	
	// Get app root element
	appRoot := js.Global().Get("document").Call("getElementById", "app")
	if appRoot.IsNull() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Get current path
	path := js.Global().Get("window").Get("location").Get("pathname").String()
	
	// Render the appropriate handler
	handler := router.GetHandler(path)
	if handler != nil {
		page := handler()
		renderVNode(page)
	} else {
		// Render 404
		renderVNode(handlers.NotFound())
	}
}

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering VNode...")
	
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
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}
`

	mainContent = fmt.Sprintf(mainContent, config.Module, config.Module)
	if err := WriteFile(filepath.Join(config.Directory, "app/main.go"), mainContent); err != nil {
		return err
	}

	// Create app/handlers/home.go
	homeContent := `package handlers

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Home renders the home page
func Home() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("container mx-auto px-4 py-8"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-4"),
		), functional.Text("Welcome to Vango")),
		
		functional.P(functional.MergeProps(
			functional.Class("text-lg text-gray-600 dark:text-gray-300 mb-8"),
		), functional.Text("This app uses programmatic routing.")),
		
		functional.Div(functional.MergeProps(
			functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-2xl font-semibold text-gray-900 dark:text-white mb-4"),
			), functional.Text("Routing")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-gray-600 dark:text-gray-300 mb-4"),
			), functional.Text("Routes are defined in server/routes.go")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-gray-600 dark:text-gray-300"),
			), functional.Text("Handlers are in app/handlers/")),
		),
	)
}

// NotFound renders the 404 page
func NotFound() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("text-center py-16"),
	),
		functional.H1(functional.MergeProps(
			functional.Class("text-6xl font-bold text-gray-300 dark:text-gray-700"),
		), functional.Text("404")),
		
		functional.P(functional.MergeProps(
			functional.Class("text-xl text-gray-600 dark:text-gray-400 mt-4"),
		), functional.Text("Page not found")),
	)
}
`

	if err := WriteFile(filepath.Join(config.Directory, "app/handlers/home.go"), homeContent); err != nil {
		return err
	}

	// Create server/routes.go
	routesContent := `package server

import (
	"github.com/recera/vango/pkg/vango/vdom"
	handlers "%s/app/handlers"
)

// Router manages application routes
type Router struct {
	routes map[string]func() *vdom.VNode
}

// NewRouter creates a new router with defined routes
func NewRouter() *Router {
	r := &Router{
		routes: make(map[string]func() *vdom.VNode),
	}
	
	// Define routes
	r.routes["/"] = handlers.Home
	// Add more routes here:
	// r.routes["/about"] = handlers.About
	// r.routes["/contact"] = handlers.Contact
	
	return r
}

// GetHandler returns the handler for a given path
func (r *Router) GetHandler(path string) func() *vdom.VNode {
	if handler, ok := r.routes[path]; ok {
		return handler
	}
	return nil
}
`

	routesContent = fmt.Sprintf(routesContent, config.Module)
	return WriteFile(filepath.Join(config.Directory, "server/routes.go"), routesContent)
}

// generateMinimal creates minimal structure
func (t *BaseTemplate) generateMinimal(config *ProjectConfig) error {
	// Create simple app/main.go
	mainContent := `//go:build wasm
// +build wasm

package main

import (
	"syscall/js"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

func main() {
	// Initialize Vango runtime
	js.Global().Get("console").Call("log", "🚀 Vango minimal app starting...")
	
	// Wait for DOM and render
	document := js.Global().Get("document")
	if document.Get("readyState").String() == "loading" {
		document.Call("addEventListener", "DOMContentLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			render()
			return nil
		}))
	} else {
		render()
	}
	
	// Keep the WASM runtime alive
	select {}
}

func render() {
	// Get app root
	appRoot := js.Global().Get("document").Call("getElementById", "app")
	if appRoot.IsNull() {
		js.Global().Get("console").Call("error", "Could not find #app element")
		return
	}
	
	// Create and render the app
	app := createApp()
	renderVNode(app)
}

func createApp() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-center"),
		),
			functional.H1(functional.MergeProps(
				functional.Class("text-5xl font-bold text-gray-900 dark:text-white mb-4"),
			), functional.Text("Hello Vango!")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-xl text-gray-600 dark:text-gray-300"),
			), functional.Text("Minimal Vango app with Go and WebAssembly")),
			
			functional.Div(functional.MergeProps(
				functional.Class("mt-8 text-gray-500 dark:text-gray-400"),
			), functional.Text("Edit app/main.go to modify this page")),
		),
	)
}

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering VNode...")
	
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
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}
`

	return WriteFile(filepath.Join(config.Directory, "app/main.go"), mainContent)
}