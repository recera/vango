package routes

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// LiveHandler provides HTTP routing with live protocol support
type LiveHandler struct {
	loader     *DynamicLoader
	liveServer *live.Server
	bridge     *live.SchedulerBridge
	fallback   http.Handler
}

// NewLiveHandler creates a new route handler with live protocol support
func NewLiveHandler(routesDir string, liveServer *live.Server, fallback http.Handler) (*LiveHandler, error) {
	// Create dynamic loader
	loader, err := NewDynamicLoader(routesDir, liveServer)
	if err != nil {
		return nil, err
	}

	// Load routes
	if err := loader.LoadRoutes(); err != nil {
		return nil, err
	}

	return &LiveHandler{
		loader:     loader,
		liveServer: liveServer,
		bridge:     live.NewSchedulerBridge(liveServer),
		fallback:   fallback,
	}, nil
}

// ServeHTTP implements http.Handler
func (h *LiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create server context
	ctx := server.NewContext(w, r)
	
	// Try to match a route
	handler, params, middleware := h.loader.router.Match(r.URL.Path)
	
	if handler != nil {
		// Set params on context
		server.WithParams(ctx, params)
		
		// Execute middleware chain
		for _, mw := range middleware {
			if err := mw.Before(ctx); err != nil {
				if err == server.ErrStop {
					return // Middleware stopped the chain
				}
				// Handle error
				ctx.Status(500)
				ctx.Text(500, "Middleware error")
				return
			}
		}
		
		// Execute handler
		vnode, err := handler(ctx)
		if err != nil {
			log.Printf("Handler error: %v", err)
			ctx.Status(500)
			ctx.Text(500, "Internal Server Error")
			return
		}
		
		// Render the VNode to HTML
		if vnode != nil {
			// Check if this is a server-driven component
			if h.isServerDriven(r.URL.Path) {
				// Inject live protocol client
				sessionID := h.getSessionID(ctx)
				vnode = server.InjectServerDrivenClient(vnode, sessionID)
				
				// Create component instance for this session
				h.createServerComponent(sessionID, r.URL.Path, handler)
			}
			
			// Render to HTML
			// For now, use a simple HTML rendering approach
			htmlStr := renderVNodeToHTML(vnode)
			
			// Set content type and write response
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte("<!DOCTYPE html>\n" + htmlStr))
		}
		
		// Execute after middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			middleware[i].After(ctx)
		}
		
		return
	}
	
	// No route matched, fall back to static file serving
	if h.fallback != nil {
		h.fallback.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// isServerDriven checks if a route is server-driven
func (h *LiveHandler) isServerDriven(path string) bool {
	// Check if the route is marked as server-driven
	for _, route := range h.loader.routes {
		if h.matchesRoute(path, route.URLPattern) && route.HasServer {
			return true
		}
	}
	return false
}

// matchesRoute checks if a path matches a route pattern
func (h *LiveHandler) matchesRoute(path, pattern string) bool {
	// Simple matching for now
	if path == pattern {
		return true
	}
	if path == pattern+"/" || path+"/" == pattern {
		return true
	}
	
	// Handle parameter routes
	if strings.Contains(pattern, ":") {
		pathSegments := strings.Split(strings.Trim(path, "/"), "/")
		patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
		
		if len(pathSegments) != len(patternSegments) {
			return false
		}
		
		for i, segment := range patternSegments {
			if strings.HasPrefix(segment, ":") {
				continue // Parameter segment matches anything
			}
			if segment != pathSegments[i] {
				return false
			}
		}
		return true
	}
	
	return false
}

// getSessionID gets or creates a session ID
func (h *LiveHandler) getSessionID(ctx server.Ctx) string {
	// Check for existing session ID
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

// createServerComponent creates a server component instance for a session
func (h *LiveHandler) createServerComponent(sessionID, path string, handler server.HandlerFunc) {
	// Find the route info
	var route *RouteFile
	for _, r := range h.loader.routes {
		if h.matchesRoute(path, r.URLPattern) {
			route = &r
			break
		}
	}
	
	if route == nil || !route.HasServer {
		return
	}
	
	// Create component ID
	componentID := fmt.Sprintf("%s_%s", sessionID, strings.ReplaceAll(path, "/", "_"))
	
	// Create render function that calls the handler
	renderFunc := func(ctx *vango.Context) *vdom.VNode {
		// Create a mock server context for the handler
		// In production, this would properly integrate with the context
		mockCtx := &mockServerCtx{
			path:   path,
			method: "GET",
		}
		
		// Call the handler
		vnode, err := handler(mockCtx)
		if err != nil {
			log.Printf("Handler error in render: %v", err)
			return functional.Div(nil,
				functional.Text("Error rendering component"),
			)
		}
		
		return vnode
	}
	
	// Create the server component via the bridge
	component, err := h.bridge.CreateServerComponent(sessionID, componentID, renderFunc)
	if err != nil {
		log.Printf("Failed to create server component: %v", err)
		return
	}
	
	// Register the component
	h.loader.registry.Register(component)
	
	log.Printf("Created server component: %s for session %s", componentID, sessionID)
}

// mockServerCtx is a mock implementation of server.Ctx for testing
type mockServerCtx struct {
	path       string
	method     string
	params     map[string]string
	statusCode int
}

func (m *mockServerCtx) Request() *http.Request                { return nil }
func (m *mockServerCtx) Path() string                          { return m.path }
func (m *mockServerCtx) Method() string                        { return m.method }
func (m *mockServerCtx) Query() url.Values                     { return nil }
func (m *mockServerCtx) Param(key string) string               { return m.params[key] }
func (m *mockServerCtx) Status(code int)                       { m.statusCode = code }
func (m *mockServerCtx) StatusCode() int                       { return m.statusCode }
func (m *mockServerCtx) Header() http.Header                   { return http.Header{} }
func (m *mockServerCtx) SetHeader(key, val string)             {}
func (m *mockServerCtx) Redirect(url string, code int)         {}
func (m *mockServerCtx) JSON(code int, v any) error            { return nil }
func (m *mockServerCtx) Text(code int, msg string) error       { return nil }
func (m *mockServerCtx) Session() server.Session               { return nil }
func (m *mockServerCtx) Done() <-chan struct{}                 { return nil }
func (m *mockServerCtx) Logger() *slog.Logger                  { return slog.Default() }

// renderVNodeToHTML renders a VNode tree to HTML string
func renderVNodeToHTML(node *vdom.VNode) string {
	if node == nil {
		return ""
	}
	
	switch node.Kind {
	case vdom.KindText:
		return escapeHTML(node.Text)
	case vdom.KindElement:
		var html strings.Builder
		html.WriteString("<")
		html.WriteString(node.Tag)
		
		// Add attributes
		for key, val := range node.Props {
			html.WriteString(" ")
			html.WriteString(key)
			html.WriteString(`="`)
			html.WriteString(escapeHTMLAttr(fmt.Sprintf("%v", val)))
			html.WriteString(`"`)
		}
		
		// Self-closing tags
		if isSelfClosing(node.Tag) && len(node.Kids) == 0 {
			html.WriteString(" />")
			return html.String()
		}
		
		html.WriteString(">")
		
		// Render children
		for _, child := range node.Kids {
			html.WriteString(renderVNodeToHTML(&child))
		}
		
		html.WriteString("</")
		html.WriteString(node.Tag)
		html.WriteString(">")
		
		return html.String()
	default:
		return ""
	}
}

// escapeHTML escapes HTML text content
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// escapeHTMLAttr escapes HTML attribute values
func escapeHTMLAttr(s string) string {
	s = escapeHTML(s)
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// isSelfClosing checks if a tag is self-closing
func isSelfClosing(tag string) bool {
	selfClosing := []string{
		"area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr",
	}
	for _, t := range selfClosing {
		if t == tag {
			return true
		}
	}
	return false
}

// Refresh rescans and updates routes (for hot reload)
func (h *LiveHandler) Refresh(routesDir string) error {
	// Create new loader
	loader, err := NewDynamicLoader(routesDir, h.liveServer)
	if err != nil {
		return err
	}

	// Load routes
	if err := loader.LoadRoutes(); err != nil {
		return err
	}

	// Replace the loader
	h.loader = loader
	
	log.Println("✅ Routes refreshed with live support")
	
	return nil
}