package routes

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
	
	// Import the app routes package - this assumes routes exist
	// The actual import will be added dynamically based on scanning
)

// Handler provides HTTP routing for compiled routes
type Handler struct {
	router   *server.Router
	routes   []RouteFile
	fallback http.Handler // Fallback to static file serving
}

// NewHandler creates a new route handler
func NewHandler(routesDir string, fallback http.Handler) (*Handler, error) {
	scanner, err := NewScanner(routesDir)
	if err != nil {
		return nil, err
	}

	routes, err := scanner.ScanRoutes()
	if err != nil {
		return nil, err
	}

	h := &Handler{
		router:   server.NewRouter(),
		routes:   routes,
		fallback: fallback,
	}

	// Register routes
	if err := h.registerRoutes(); err != nil {
		return nil, err
	}

	return h, nil
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to match a route
	if h.router != nil {
		// Check if this path matches any of our routes
		for _, route := range h.routes {
			if h.matchesRoute(r.URL.Path, route.URLPattern) {
				// Check if this is a client-only route
				// Only treat as client-only if explicitly marked as client-only (HasClient && !HasServer)
				// Routes with no markers default to server-side rendering
				if route.HasClient && !route.HasServer {
					// Client-only route - serve static files (index.html)
					// This allows the WASM app to handle client-side routing
					if h.fallback != nil {
						h.fallback.ServeHTTP(w, r)
					} else {
						http.NotFound(w, r)
					}
					return
				}
				
				// Server-side route (default) - use the router
				h.router.ServeHTTP(w, r)
				return
			}
		}
	}

	// No route matched, fall back to static file serving
	if h.fallback != nil {
		h.fallback.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// matchesRoute checks if a path matches a route pattern
func (h *Handler) matchesRoute(path, pattern string) bool {
	// Simple matching for now - exact match or with trailing slash
	if path == pattern {
		return true
	}
	if path == pattern+"/" || path+"/" == pattern {
		return true
	}
	
	// Handle parameter routes (simplified)
	if strings.Contains(pattern, ":") {
		// Convert pattern to segments
		pathSegments := strings.Split(strings.Trim(path, "/"), "/")
		patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
		
		if len(pathSegments) != len(patternSegments) {
			return false
		}
		
		for i, segment := range patternSegments {
			if strings.HasPrefix(segment, ":") {
				// Parameter segment, matches anything
				continue
			}
			if segment != pathSegments[i] {
				return false
			}
		}
		return true
	}
	
	return false
}

// registerRoutes registers all discovered routes
func (h *Handler) registerRoutes() error {
	// For now, we'll register placeholder handlers
	// In a full implementation, these would call the actual compiled route handlers
	
	for _, route := range h.routes {
		// Handle client-only routes
		// Routes are client-only ONLY if explicitly marked as client-only
		if route.HasClient && !route.HasServer {
			// Skip registering server handlers for client-only routes
			// The static file server will handle these by serving index.html
			log.Printf("📱 Client-only route (no server handler): %s -> %s", 
				route.URLPattern, route.HandlerName)
			continue
		}

		routeCopy := route // Capture for closure
		
		if route.IsAPI {
			// Register API route
			h.router.AddAPIRoute(route.URLPattern, func(ctx server.Ctx) (any, error) {
				// For demo, return a simple JSON response
				return map[string]interface{}{
					"message": fmt.Sprintf("API route: %s", routeCopy.URLPattern),
					"handler": routeCopy.HandlerName,
					"server":  routeCopy.HasServer,
				}, nil
			})
		} else {
			// Register server-side route (default for routes without explicit markers)
			h.router.AddRoute(route.URLPattern, func(ctx server.Ctx) (*vdom.VNode, error) {
				// Special handling for blog routes to demonstrate the template
				if routeCopy.URLPattern == "/" || routeCopy.URLPattern == "/blog" {
					// Return the blog index with sample data
					return h.renderBlogIndex()
				} else if strings.HasPrefix(routeCopy.URLPattern, "/blog/") {
					// Return a blog post page
					return h.renderBlogPost(ctx)
				}
				
				// For other routes, return a simple demo
				node := functional.Div(
					vdom.Props{"class": "universal-route"},
					functional.H1(nil, functional.Text(fmt.Sprintf("Route: %s", routeCopy.URLPattern))),
					functional.P(nil, functional.Text(fmt.Sprintf("Handler: %s", routeCopy.HandlerName))),
				)
				return node, nil
			})
		}
		
		log.Printf("📍 Registered server route: %s -> %s (server=%v, api=%v)", 
			route.URLPattern, route.HandlerName, route.HasServer, route.IsAPI)
	}

	// Set up 404 handler
	h.router.SetNotFound(func(ctx server.Ctx) (*vdom.VNode, error) {
		ctx.Status(404)
		node := functional.Div(
			vdom.Props{"class": "min-h-screen flex items-center justify-center"},
			functional.Div(
				vdom.Props{"class": "text-center"},
				functional.H1(
					vdom.Props{"class": "text-6xl font-bold text-gray-800"},
					functional.Text("404"),
				),
				functional.P(
					vdom.Props{"class": "text-xl text-gray-600 mt-4"},
					functional.Text("Page not found"),
				),
			),
		)
		return node, nil
	})

	return nil
}

// Refresh rescans and updates routes (for hot reload)
func (h *Handler) Refresh(routesDir string) error {
	scanner, err := NewScanner(routesDir)
	if err != nil {
		return err
	}

	routes, err := scanner.ScanRoutes()
	if err != nil {
		return err
	}

	// Create new router
	newRouter := server.NewRouter()
	h.routes = routes
	h.router = newRouter

	// Re-register routes
	return h.registerRoutes()
}

// renderBlogIndex renders a blog index page with sample posts
func (h *Handler) renderBlogIndex() (*vdom.VNode, error) {
	// Sample blog posts data
	posts := []map[string]interface{}{
		{
			"slug":        "getting-started-with-vango",
			"title":       "Getting Started with Vango",
			"date":        "2024-01-15",
			"author":      "Jane Developer",
			"excerpt":     "Learn how to build modern web applications with Vango, the Go-native frontend framework.",
			"readingTime": 5,
			"tags":        []string{"tutorial", "vango", "golang"},
		},
		{
			"slug":        "building-reactive-components",
			"title":       "Building Reactive Components in Go",
			"date":        "2024-01-10",
			"author":      "John Coder",
			"excerpt":     "Discover how to create reactive, state-driven components using Vango's powerful component system.",
			"readingTime": 8,
			"tags":        []string{"components", "reactive", "tutorial"},
		},
		{
			"slug":        "webassembly-performance-tips",
			"title":       "WebAssembly Performance Optimization",
			"date":        "2024-01-05",
			"author":      "Tech Expert",
			"excerpt":     "Optimize your Vango applications for maximum performance with these WebAssembly tips.",
			"readingTime": 10,
			"tags":        []string{"performance", "wasm", "optimization"},
		},
	}
	
	// Create the blog index structure
	return functional.Div(
		vdom.Props{"class": "min-h-screen bg-white dark:bg-gray-900 transition-colors duration-200"},
		
		// Navigation Header
		h.renderBlogHeader(),
		
		// Hero Section
		functional.Div(
			vdom.Props{"class": "relative overflow-hidden bg-gradient-to-br from-purple-600 via-blue-600 to-purple-700 dark:from-purple-900 dark:via-blue-900 dark:to-purple-800"},
			functional.Div(
				vdom.Props{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 text-center"},
				functional.H1(
					vdom.Props{"class": "text-5xl md:text-6xl font-bold text-white mb-6"},
					functional.Text("Welcome to Our Blog"),
				),
				functional.P(
					vdom.Props{"class": "text-xl text-purple-100 max-w-2xl mx-auto"},
					functional.Text("Discover insights, tutorials, and thoughts on modern web development with Go"),
				),
			),
		),
		
		// Blog Posts Grid
		functional.Div(
			vdom.Props{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"},
			functional.Div(
				vdom.Props{"class": "grid gap-8 md:grid-cols-2 lg:grid-cols-3"},
				h.createPostCards(posts)...,
			),
		),
		
		// Footer
		h.renderBlogFooter(),
	), nil
}

// renderBlogPost renders an individual blog post
func (h *Handler) renderBlogPost(ctx server.Ctx) (*vdom.VNode, error) {
	// Extract slug from URL
	path := ctx.Path()
	slug := strings.TrimPrefix(path, "/blog/")
	
	// Sample post data
	post := map[string]interface{}{
		"slug":        slug,
		"title":       "Getting Started with Vango",
		"date":        "2024-01-15",
		"author":      "Jane Developer",
		"content":     "This is where the full blog post content would go. In a real implementation, this would be loaded from markdown files or a database.",
		"readingTime": 5,
		"tags":        []string{"tutorial", "vango", "golang"},
	}
	
	return functional.Div(
		vdom.Props{"class": "min-h-screen bg-white dark:bg-gray-900"},
		
		// Header
		h.renderBlogHeader(),
		
		// Article
		functional.Article(
			vdom.Props{"class": "max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"},
			
			// Post Header
			functional.Header(
				vdom.Props{"class": "mb-8"},
				functional.H1(
					vdom.Props{"class": "text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4"},
					functional.Text(post["title"].(string)),
				),
				functional.Div(
					vdom.Props{"class": "flex items-center text-gray-600 dark:text-gray-400"},
					functional.Text(fmt.Sprintf("By %s • %s • %d min read", 
						post["author"].(string), 
						post["date"].(string), 
						post["readingTime"].(int))),
				),
			),
			
			// Post Content
			functional.Div(
				vdom.Props{"class": "prose prose-lg dark:prose-invert max-w-none"},
				functional.P(nil, functional.Text(post["content"].(string))),
			),
		),
		
		// Footer
		h.renderBlogFooter(),
	), nil
}

// renderBlogHeader renders the blog navigation header
func (h *Handler) renderBlogHeader() *vdom.VNode {
	return functional.Nav(
		vdom.Props{"class": "bg-white dark:bg-gray-900 shadow-sm border-b border-gray-200 dark:border-gray-800"},
		functional.Div(
			vdom.Props{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			functional.Div(
				vdom.Props{"class": "flex justify-between items-center h-16"},
				
				// Logo
				functional.A(
					vdom.Props{"href": "/", "class": "flex items-center"},
					functional.Span(
						vdom.Props{"class": "text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent"},
						functional.Text("Vango Blog"),
					),
				),
				
				// Navigation Links
				functional.Div(
					vdom.Props{"class": "hidden md:flex items-center space-x-8"},
					functional.A(
						vdom.Props{"href": "/", "class": "text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400"},
						functional.Text("Home"),
					),
					functional.A(
						vdom.Props{"href": "/about", "class": "text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400"},
						functional.Text("About"),
					),
					functional.A(
						vdom.Props{"href": "/archive", "class": "text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400"},
						functional.Text("Archive"),
					),
				),
				
				// Dark Mode Toggle Button
				functional.Button(
					vdom.Props{
						"class": "p-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300",
						"onclick": "toggleDarkMode()",
						"aria-label": "Toggle dark mode",
					},
					functional.Text("🌙"),
				),
			),
		),
	)
}

// renderBlogFooter renders the blog footer
func (h *Handler) renderBlogFooter() *vdom.VNode {
	return functional.Footer(
		vdom.Props{"class": "bg-gray-50 dark:bg-gray-800 mt-16"},
		functional.Div(
			vdom.Props{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"},
			functional.Div(
				vdom.Props{"class": "text-center text-gray-600 dark:text-gray-400"},
				functional.P(nil, functional.Text("© 2024 Vango Blog. Built with Vango 🚀")),
			),
		),
	)
}

// createPostCards creates blog post cards for the index
func (h *Handler) createPostCards(posts []map[string]interface{}) []*vdom.VNode {
	cards := make([]*vdom.VNode, 0, len(posts))
	
	for _, post := range posts {
		card := functional.Article(
			vdom.Props{"class": "bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden hover:shadow-xl transition-shadow duration-300"},
			
			// Card Header with gradient
			functional.Div(
				vdom.Props{"class": "h-2 bg-gradient-to-r from-purple-600 to-blue-600"},
			),
			
			// Card Body
			functional.Div(
				vdom.Props{"class": "p-6"},
				
				// Title
				functional.H2(
					vdom.Props{"class": "text-2xl font-bold text-gray-900 dark:text-white mb-2"},
					functional.A(
						vdom.Props{
							"href": fmt.Sprintf("/blog/%s", post["slug"].(string)),
							"class": "hover:text-purple-600 dark:hover:text-purple-400",
						},
						functional.Text(post["title"].(string)),
					),
				),
				
				// Meta
				functional.Div(
					vdom.Props{"class": "text-sm text-gray-600 dark:text-gray-400 mb-4"},
					functional.Text(fmt.Sprintf("%s • %d min read", 
						post["date"].(string), 
						post["readingTime"].(int))),
				),
				
				// Excerpt
				functional.P(
					vdom.Props{"class": "text-gray-700 dark:text-gray-300 mb-4"},
					functional.Text(post["excerpt"].(string)),
				),
				
				// Tags
				functional.Div(
					vdom.Props{"class": "flex flex-wrap gap-2"},
					h.createTags(post["tags"].([]string))...,
				),
			),
		)
		cards = append(cards, card)
	}
	
	return cards
}

// createTags creates tag elements
func (h *Handler) createTags(tags []string) []*vdom.VNode {
	tagNodes := make([]*vdom.VNode, 0, len(tags))
	for _, tag := range tags {
		tagNode := functional.Span(
			vdom.Props{"class": "px-3 py-1 bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 text-sm rounded-full"},
			functional.Text(tag),
		)
		tagNodes = append(tagNodes, tagNode)
	}
	return tagNodes
}