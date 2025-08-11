//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"
	
	routes "github.com/recera/vango/app/routes"
	"github.com/recera/vango/pkg/vango/vdom"
)

var (
	document js.Value
	window   js.Value
	console  js.Value
	counterValue int
)

func main() {
	// Initialize global JS references
	document = js.Global().Get("document")
	window = js.Global().Get("window")
	console = js.Global().Get("console")
	
	console.Call("log", "🚀 Vango WASM client starting...")
	
	// Initialize the app
	initApp()
	
	// Keep the WASM runtime alive
	select {}
}

func initApp() {
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
	console.Call("log", "DOM ready, initializing app...")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Get app root element
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Render the initial page based on path
	path := window.Get("location").Get("pathname").String()
	renderPage(path)
	
	// Set up navigation handler
	js.Global().Set("navigateTo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newPath := args[0].String()
			window.Get("history").Call("pushState", nil, "", newPath)
			renderPage(newPath)
		}
		return nil
	}))
	
	// Set up dark mode toggle
	js.Global().Set("toggleDarkMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		localStorage := window.Get("localStorage")
		
		isDark := document.Get("documentElement").Get("classList").Call("contains", "dark").Bool()
		
		if isDark {
			document.Get("documentElement").Get("classList").Call("remove", "dark")
			localStorage.Call("setItem", "darkMode", "false")
		} else {
			document.Get("documentElement").Get("classList").Call("add", "dark")
			localStorage.Call("setItem", "darkMode", "true")
		}
		
		// Re-render current page to update dark mode state
		path := window.Get("location").Get("pathname").String()
		renderPage(path)
		return nil
	}))
	
	// Handle browser back/forward buttons
	window.Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		path := window.Get("location").Get("pathname").String()
		renderPage(path)
		return nil
	}))
	
	// Set up counter functionality
	js.Global().Set("updateCounter", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			delta := args[0].Int()
			if delta == 0 {
				counterValue = 0 // Reset
			} else {
				counterValue += delta
			}
			
			// Update the counter display
			counterElem := document.Call("getElementById", "counter-value")
			if !counterElem.IsNull() && !counterElem.IsUndefined() {
				counterElem.Set("textContent", fmt.Sprintf("%d", counterValue))
			}
		}
		return nil
	}))
	
	console.Call("log", "✅ Vango client initialized")
}

func renderPage(path string) {
	console.Call("log", "Rendering page:", path)
	
	var page *vdom.VNode
	
	switch path {
	case "/", "/index.html":
		page = routes.IndexPage()
	case "/about":
		page = routes.AboutPage()
	case "/counter":
		page = routes.CounterPage()
	default:
		// 404 page
		page = routes.IndexPage() // Fallback to index for now
	}
	
	if page != nil {
		renderVNode(page)
	}
}

func renderVNode(vnode *vdom.VNode) {
	console.Call("log", "Rendering VNode to DOM")
	
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Convert VNode to DOM and replace app content
	domNode := vnodeToDOM(vnode)
	if !domNode.IsNull() && !domNode.IsUndefined() {
		// Clear existing content
		appRoot.Set("innerHTML", "")
		// Append new content
		appRoot.Call("appendChild", domNode)
		
		// Restore counter value if on counter page
		if window.Get("location").Get("pathname").String() == "/counter" {
			counterElem := document.Call("getElementById", "counter-value")
			if !counterElem.IsNull() && !counterElem.IsUndefined() {
				counterElem.Set("textContent", fmt.Sprintf("%d", counterValue))
			}
		}
	}
}

func vnodeToDOM(vnode *vdom.VNode) js.Value {
	if vnode == nil {
		return js.Null()
	}
	
	switch vnode.Kind {
	case vdom.KindText:
		// Create text node
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
				case "title":
					if v, ok := value.(string); ok {
						elem.Set("title", v)
					}
				case "style":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "style", v)
					}
				case "onclick":
					// Handle onclick attribute
					if v, ok := value.(string); ok {
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

func initDarkMode() {
	localStorage := window.Get("localStorage")
	
	// Check localStorage first
	darkMode := localStorage.Call("getItem", "darkMode").String()
	
	if darkMode == "true" {
		document.Get("documentElement").Get("classList").Call("add", "dark")
	} else if darkMode == "false" {
		document.Get("documentElement").Get("classList").Call("remove", "dark")
	} else {
		// Check system preference
		if window.Get("matchMedia").Truthy() {
			prefersDark := window.Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Bool()
			if prefersDark {
				document.Get("documentElement").Get("classList").Call("add", "dark")
			}
		}
	}
}