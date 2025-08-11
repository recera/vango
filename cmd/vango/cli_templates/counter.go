package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register("counter", &CounterTemplate{})
}

// CounterTemplate generates an interactive counter example
type CounterTemplate struct{}

func (t *CounterTemplate) Name() string {
	return "counter"
}

func (t *CounterTemplate) Description() string {
	return "Interactive counter example"
}

func (t *CounterTemplate) Generate(config *ProjectConfig) error {
	// Ensure app/routes directory exists
	if err := os.MkdirAll(filepath.Join(config.Directory, "app/routes"), 0755); err != nil {
		return fmt.Errorf("failed to create app/routes directory: %w", err)
	}
	
	// Create main.go for counter
	if err := t.createMainFile(config); err != nil {
		return err
	}
	
	// Create counter route
	if err := t.createCounterRoute(config); err != nil {
		return err
	}
	
	// Create counter styles
	if err := t.createStyles(config); err != nil {
		return err
	}
	
	return nil
}

func (t *CounterTemplate) createMainFile(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"syscall/js"
	
	// Import routes package with alias
	routes "%s/app/routes"
	"github.com/recera/vango/pkg/vango/vdom"
)

func main() {
	// Initialize the Vango runtime
	js.Global().Get("console").Call("log", "🚀 Counter app starting...")
	
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
	console.Call("log", "DOM ready, initializing Counter...")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Create a simple counter state
	count := 0
	
	// Render function
	render := func() {
		vnode := routes.Page(count)
		renderVNode(vnode)
	}
	
	// Initial render
	render()
	
	// Set up global handlers for increment/decrement
	js.Global().Set("increment", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		count++
		render()
		return nil
	}))
	
	js.Global().Set("decrement", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		count--
		render()
		return nil
	}))
	
	js.Global().Set("reset", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		count = 0
		render()
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
		
		// Re-render with new theme
		render()
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

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering Counter VNode...")
	
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
		console.Call("log", "✅ Counter rendered successfully!")
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
				case "onclick":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "onclick", v)
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
}`, config.Module)
	
	return WriteFile(filepath.Join(config.Directory, "app/main.go"), content)
}

func (t *CounterTemplate) createCounterRoute(config *ProjectConfig) error {
	content := `package routes

import (
	"fmt"
	"syscall/js"

	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Page is the counter page handler (universal component)
// Takes count as parameter to render current state
func Page(count int) *vdom.VNode {
	// Check if we're in dark mode
	isDarkMode := false
	if js.Global().Truthy() {
		isDarkMode = js.Global().Get("document").Get("documentElement").Get("classList").Call("contains", "dark").Bool()
	}
	
	bgColor := "white"
	textColor := "#333"
	containerShadow := "0 2px 8px rgba(0, 0, 0, 0.1)"
	btnBg := "#6b7280"
	btnHoverBg := "#4b5563"
	
	if isDarkMode {
		bgColor = "#1f2937"
		textColor = "#f3f4f6"
		containerShadow = "0 2px 8px rgba(0, 0, 0, 0.3)"
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen"),
		functional.StyleAttr(` + "`" + `
			background: linear-gradient(to bottom right, #f9fafb, #e5e7eb);
			padding: 2rem;
		` + "`" + `),
		vdom.Props{"class": func() string {
			if isDarkMode {
				return "min-h-screen dark"
			}
			return "min-h-screen"
		}()},
	),
		// Dark mode toggle button
		functional.Button(functional.MergeProps(
			functional.StyleAttr(` + "`" + `
				position: fixed;
				top: 1rem;
				right: 1rem;
				padding: 0.5rem 1rem;
				background: rgba(255, 255, 255, 0.9);
				border: 1px solid #e5e7eb;
				border-radius: 0.5rem;
				cursor: pointer;
				font-size: 1.25rem;
				transition: all 0.2s;
				z-index: 1000;
			` + "`" + `),
			vdom.Props{"onclick": "toggleDarkMode()"},
		), functional.Text(func() string {
			if isDarkMode {
				return "☀️"
			}
			return "🌙"
		}())),
		
		functional.Div(functional.MergeProps(
			functional.Class("counter-container"),
			functional.StyleAttr(fmt.Sprintf(` + "`" + `
				max-width: 400px;
				margin: 4rem auto;
				text-align: center;
				padding: 2rem;
				border-radius: 12px;
				box-shadow: %s;
				background: %s;
				color: %s;
			` + "`" + `, containerShadow, bgColor, textColor)),
		),
			functional.H1(functional.MergeProps(
				functional.StyleAttr(fmt.Sprintf("margin-bottom: 2rem; color: %s;", textColor)),
			), functional.Text("Counter Example")),
			
			functional.Div(functional.MergeProps(
				functional.Class("counter-display"),
				functional.StyleAttr(` + "`" + `
					font-size: 4rem;
					font-weight: bold;
					margin: 2rem 0;
					color: #667eea;
					padding: 1rem;
					background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
					-webkit-background-clip: text;
					-webkit-text-fill-color: transparent;
					background-clip: text;
				` + "`" + `),
			), functional.Text(fmt.Sprintf("%d", count))),
			
			functional.Div(functional.MergeProps(
				functional.Class("counter-buttons"),
				functional.StyleAttr("display: flex; gap: 1rem; justify-content: center; margin-top: 2rem;"),
			),
				functional.Button(functional.MergeProps(
					functional.Class("btn btn-secondary"),
					functional.StyleAttr(fmt.Sprintf(` + "`" + `
						padding: 0.75rem 1.5rem;
						font-size: 1.25rem;
						background: %s;
						color: white;
						border: none;
						border-radius: 0.5rem;
						cursor: pointer;
						font-weight: 600;
						transition: all 0.2s;
						box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
					` + "`" + `, btnBg)),
					vdom.Props{
						"onclick": "decrement()",
						"onmouseover": fmt.Sprintf("this.style.background='%s'", btnHoverBg),
						"onmouseout": fmt.Sprintf("this.style.background='%s'", btnBg),
					},
				), functional.Text("-")),
				
				functional.Button(functional.MergeProps(
					functional.Class("btn btn-reset"),
					functional.StyleAttr(fmt.Sprintf(` + "`" + `
						padding: 0.75rem 1.5rem;
						font-size: 1rem;
						background: %s;
						color: white;
						border: none;
						border-radius: 0.5rem;
						cursor: pointer;
						font-weight: 600;
						transition: all 0.2s;
						box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
					` + "`" + `, btnBg)),
					vdom.Props{
						"onclick": "reset()",
						"onmouseover": fmt.Sprintf("this.style.background='%s'", btnHoverBg),
						"onmouseout": fmt.Sprintf("this.style.background='%s'", btnBg),
					},
				), functional.Text("Reset")),
				
				functional.Button(functional.MergeProps(
					functional.Class("btn btn-primary"),
					functional.StyleAttr(` + "`" + `
						padding: 0.75rem 1.5rem;
						font-size: 1.25rem;
						background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
						color: white;
						border: none;
						border-radius: 0.5rem;
						cursor: pointer;
						font-weight: 600;
						transition: all 0.2s;
						box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
					` + "`" + `),
					vdom.Props{
						"onclick": "increment()",
						"onmouseover": "this.style.transform='translateY(-2px)'; this.style.boxShadow='0 4px 8px rgba(0, 0, 0, 0.2)'",
						"onmouseout": "this.style.transform='translateY(0)'; this.style.boxShadow='0 2px 4px rgba(0, 0, 0, 0.1)'",
					},
				), functional.Text("+")),
			),
			
			functional.P(functional.MergeProps(
				functional.StyleAttr(fmt.Sprintf(` + "`" + `
					margin-top: 2rem;
					color: %s;
					font-size: 0.9rem;
					opacity: 0.8;
				` + "`" + `, textColor)),
			), functional.Text("Click the buttons to change the counter value")),
		),
	)
}`
	
	return WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), content)
}

func (t *CounterTemplate) createStyles(config *ProjectConfig) error {
	content := `/* Counter styles with dark mode support */
* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

html {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

body {
	min-height: 100vh;
	transition: background-color 0.3s ease;
}

/* Light mode styles */
body {
	background: linear-gradient(to bottom right, #f9fafb, #e5e7eb);
}

/* Dark mode styles */
.dark {
	background: linear-gradient(to bottom right, #111827, #1f2937) !important;
}

.dark .counter-container {
	background: #1f2937 !important;
	color: #f3f4f6 !important;
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3) !important;
}

.counter-container {
	max-width: 400px;
	margin: 4rem auto;
	text-align: center;
	padding: 2rem;
	border-radius: 12px;
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	background: white;
	transition: all 0.3s ease;
}

.counter-display {
	font-size: 4rem;
	font-weight: bold;
	margin: 2rem 0;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}

.counter-buttons {
	display: flex;
	gap: 1rem;
	justify-content: center;
	margin-top: 2rem;
}

.btn {
	padding: 0.75rem 1.5rem;
	font-size: 1.25rem;
	border: none;
	border-radius: 0.5rem;
	cursor: pointer;
	font-weight: 600;
	transition: all 0.2s;
	box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.btn:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.btn:active {
	transform: translateY(0);
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.btn-secondary {
	background-color: #6b7280;
	color: white;
}

.btn-secondary:hover {
	background-color: #4b5563;
}

.btn-reset {
	background-color: #6b7280;
	color: white;
	font-size: 1rem !important;
}

.btn-reset:hover {
	background-color: #4b5563;
}

/* Dark mode toggle button */
.dark-mode-toggle {
	position: fixed;
	top: 1rem;
	right: 1rem;
	padding: 0.5rem 1rem;
	background: rgba(255, 255, 255, 0.9);
	border: 1px solid #e5e7eb;
	border-radius: 0.5rem;
	cursor: pointer;
	font-size: 1.25rem;
	transition: all 0.2s;
	z-index: 1000;
}

.dark .dark-mode-toggle {
	background: rgba(31, 41, 55, 0.9);
	border-color: #374151;
}

.dark-mode-toggle:hover {
	transform: scale(1.05);
}

/* Min height for full screen */
.min-h-screen {
	min-height: 100vh;
}`
	
	return WriteFile(filepath.Join(config.Directory, "styles/counter.css"), content)
}