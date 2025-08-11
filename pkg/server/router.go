package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	
	"github.com/recera/vango/pkg/renderer/html"
	"github.com/recera/vango/pkg/vango/vdom"
)

// HandlerFunc is the signature for route handlers
type HandlerFunc func(ctx Ctx) (*vdom.VNode, error)

// APIHandlerFunc is the signature for API route handlers
type APIHandlerFunc func(ctx Ctx) (any, error)

// MiddlewareFunc is the signature for middleware
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Middleware interface for before/after hooks
type Middleware interface {
	Before(ctx Ctx) error  // return Stop() to abort chain
	After(ctx Ctx) error   // always called if Before succeeded
}

// RouteNode represents a node in the radix tree
type RouteNode struct {
	segment   string
	param     bool
	catchAll  bool
	paramName string
	paramType string // "string", "int", "int64", "uuid"
	handler   HandlerFunc
	apiHandler APIHandlerFunc
	children  []*RouteNode
	middleware []Middleware
}

// Router manages all routes and middleware
type Router struct {
	root       *RouteNode
	notFound   HandlerFunc
	errorPage  HandlerFunc
	middleware []Middleware
	mu         sync.RWMutex
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		root: &RouteNode{
			children: make([]*RouteNode, 0),
		},
		middleware: make([]Middleware, 0),
	}
}

// AddRoute registers a page handler for a path
func (r *Router) AddRoute(path string, handler HandlerFunc, middleware ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	segments := splitPath(path)
	node := r.root
	
	for _, segment := range segments {
		child := r.findOrCreateChild(node, segment)
		node = child
	}
	
	node.handler = handler
	node.middleware = middleware
}

// AddAPIRoute registers an API handler for a path
func (r *Router) AddAPIRoute(path string, handler APIHandlerFunc, middleware ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	segments := splitPath(path)
	node := r.root
	
	for _, segment := range segments {
		child := r.findOrCreateChild(node, segment)
		node = child
	}
	
	node.apiHandler = handler
	node.middleware = middleware
}

// Use adds global middleware
func (r *Router) Use(middleware ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// SetNotFound sets the 404 handler
func (r *Router) SetNotFound(handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notFound = handler
}

// SetErrorPage sets the 500 error handler
func (r *Router) SetErrorPage(handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errorPage = handler
}

// Match finds a handler for the given path
func (r *Router) Match(path string) (HandlerFunc, map[string]string, []Middleware) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	segments := splitPath(path)
	params := make(map[string]string)
	
	node, matched := r.matchNode(r.root, segments, params)
	if !matched || (node.handler == nil && node.apiHandler == nil) {
		return r.notFound, params, r.middleware
	}
	
	// Collect middleware from root to matched node
	allMiddleware := append([]Middleware{}, r.middleware...)
	allMiddleware = append(allMiddleware, node.middleware...)
	
	// Wrap API handler as regular handler if needed
	if node.apiHandler != nil {
		return wrapAPIHandler(node.apiHandler), params, allMiddleware
	}
	
	return node.handler, params, allMiddleware
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := NewContext(w, req)
	
	// Find matching route
	handler, params, middleware := r.Match(req.URL.Path)
	
	// If no handler found, return 404
	if handler == nil {
		ctx.Status(http.StatusNotFound)
		ctx.Text(404, "Not Found")
		return
	}
	
	// Set route parameters
	ctx = WithParams(ctx, params)
	
	// Handle panics
	defer func() {
		if err := recover(); err != nil {
			ctx.Logger().Error("panic in handler", "error", err)
			r.handleError(ctx, fmt.Errorf("internal server error: %v", err))
		}
	}()
	
	// Execute middleware chain
	finalHandler := handler
	
	// Build middleware chain
	for i := len(middleware) - 1; i >= 0; i-- {
		mw := middleware[i]
		next := finalHandler
		finalHandler = func(c Ctx) (*vdom.VNode, error) {
			// Execute Before hook
			if err := mw.Before(c); err != nil {
				if err == ErrStop {
					return nil, nil // Middleware handled response
				}
				return nil, err
			}
			
			// Execute handler
			result, err := next(c)
			
			// Execute After hook
			if afterErr := mw.After(c); afterErr != nil {
				c.Logger().Error("error in After middleware", "error", afterErr)
			}
			
			return result, err
		}
	}
	
	// Execute final handler
	vnode, err := finalHandler(ctx)
	if err != nil {
		r.handleError(ctx, err)
		return
	}
	
	// If vnode is nil, assume middleware handled the response
	if vnode == nil {
		return
	}
	
	// Render VNode to HTML and send response
	htmlContent, err := html.RenderToString(vnode)
	if err != nil {
		r.handleError(ctx, fmt.Errorf("failed to render VNode: %w", err))
		return
	}
	
	// Set content type and write response
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	ctx.(*ctxImpl).w.WriteHeader(ctx.StatusCode())
	ctx.(*ctxImpl).w.Write([]byte(htmlContent))
}

// findOrCreateChild finds or creates a child node
func (r *Router) findOrCreateChild(parent *RouteNode, segment string) *RouteNode {
	// Check if it's a parameter segment
	if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
		paramDef := segment[1 : len(segment)-1]
		
		// Check for catch-all
		if strings.HasPrefix(paramDef, "...") {
			paramName := paramDef[3:]
			for _, child := range parent.children {
				if child.catchAll && child.paramName == paramName {
					return child
				}
			}
			node := &RouteNode{
				segment:   segment,
				catchAll:  true,
				paramName: paramName,
				paramType: "string",
				children:  make([]*RouteNode, 0),
			}
			parent.children = append(parent.children, node)
			return node
		}
		
		// Parse param type
		paramName, paramType := parseParamDef(paramDef)
		
		// Look for existing param node
		for _, child := range parent.children {
			if child.param && child.paramName == paramName {
				return child
			}
		}
		
		// Create new param node
		node := &RouteNode{
			segment:   segment,
			param:     true,
			paramName: paramName,
			paramType: paramType,
			children:  make([]*RouteNode, 0),
		}
		parent.children = append(parent.children, node)
		return node
	}
	
	// Static segment
	for _, child := range parent.children {
		if !child.param && !child.catchAll && child.segment == segment {
			return child
		}
	}
	
	// Create new static node
	node := &RouteNode{
		segment:  segment,
		children: make([]*RouteNode, 0),
	}
	parent.children = append(parent.children, node)
	return node
}

// matchNode attempts to match a path against the tree
func (r *Router) matchNode(node *RouteNode, segments []string, params map[string]string) (*RouteNode, bool) {
	// Base case: no more segments
	if len(segments) == 0 {
		return node, true
	}
	
	segment := segments[0]
	remaining := segments[1:]
	
	// Try static match first (highest priority)
	for _, child := range node.children {
		if !child.param && !child.catchAll && child.segment == segment {
			return r.matchNode(child, remaining, params)
		}
	}
	
	// Try parameter match
	for _, child := range node.children {
		if child.param {
			// Validate parameter type
			if validateParam(segment, child.paramType) {
				params[child.paramName] = segment
				if result, ok := r.matchNode(child, remaining, params); ok {
					return result, true
				}
				delete(params, child.paramName)
			}
		}
	}
	
	// Try catch-all match (lowest priority)
	for _, child := range node.children {
		if child.catchAll {
			// Collect all remaining segments
			params[child.paramName] = strings.Join(segments, "/")
			return child, true
		}
	}
	
	return nil, false
}

// handleError renders the error page
func (r *Router) handleError(ctx Ctx, err error) {
	ctx.Logger().Error("handler error", "error", err)
	ctx.Status(http.StatusInternalServerError)
	
	if r.errorPage != nil {
		if vnode, err := r.errorPage(ctx); err == nil && vnode != nil {
			// Render error page VNode
			if htmlContent, renderErr := html.RenderToString(vnode); renderErr == nil {
				ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
				ctx.(*ctxImpl).w.WriteHeader(http.StatusInternalServerError)
				ctx.(*ctxImpl).w.Write([]byte(htmlContent))
				return
			}
		}
	}
	
	// Fallback error response
	ctx.Text(500, "Internal Server Error")
}

// Helper functions

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func parseParamDef(def string) (name, paramType string) {
	parts := strings.Split(def, ":")
	name = parts[0]
	paramType = "string"
	
	if len(parts) > 1 {
		paramType = parts[1]
	}
	
	return name, paramType
}

func validateParam(value, paramType string) bool {
	switch paramType {
	case "int":
		// Basic int validation
		for _, r := range value {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(value) > 0
	case "int64":
		// Same as int for now
		return validateParam(value, "int")
	case "uuid":
		// Basic UUID format validation
		if len(value) != 36 {
			return false
		}
		// Check format: 8-4-4-4-12
		if value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
			return false
		}
		return true
	default:
		// String accepts anything except empty
		return len(value) > 0
	}
}

func wrapAPIHandler(handler APIHandlerFunc) HandlerFunc {
	return func(ctx Ctx) (*vdom.VNode, error) {
		result, err := handler(ctx)
		if err != nil {
			return nil, err
		}
		
		// Serialize to JSON
		if err := ctx.JSON(http.StatusOK, result); err != nil {
			return nil, err
		}
		
		// Return nil to indicate response was handled
		return nil, nil
	}
}

// RouteTable represents the serialized routing table
type RouteTable struct {
	Routes []RouteEntry `json:"routes"`
}

// RouteEntry represents a single route in the table
type RouteEntry struct {
	Path       string            `json:"path"`
	Component  string            `json:"component"`
	Params     []ParamDef        `json:"params,omitempty"`
	Middleware []string          `json:"middleware,omitempty"`
}

// ParamDef represents a route parameter definition
type ParamDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ExportTable exports the routing table as JSON
func (r *Router) ExportTable() (*RouteTable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	table := &RouteTable{
		Routes: make([]RouteEntry, 0),
	}
	
	// Walk the tree and collect routes
	r.collectRoutes(r.root, "", table)
	
	return table, nil
}

func (r *Router) collectRoutes(node *RouteNode, path string, table *RouteTable) {
	// Build current path
	currentPath := path
	if node.segment != "" {
		if path == "" {
			currentPath = node.segment
		} else {
			currentPath = path + "/" + node.segment
		}
	}
	
	// If this node has a handler, add it to the table
	if node.handler != nil || node.apiHandler != nil {
		entry := RouteEntry{
			Path:      currentPath,
			Component: currentPath, // TODO: Map to actual component name
			Params:    make([]ParamDef, 0),
		}
		
		// Extract params from path
		segments := splitPath(currentPath)
		for _, seg := range segments {
			if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
				paramDef := seg[1 : len(seg)-1]
				if !strings.HasPrefix(paramDef, "...") {
					name, paramType := parseParamDef(paramDef)
					entry.Params = append(entry.Params, ParamDef{
						Name: name,
						Type: paramType,
					})
				}
			}
		}
		
		table.Routes = append(table.Routes, entry)
	}
	
	// Recurse to children
	for _, child := range node.children {
		r.collectRoutes(child, currentPath, table)
	}
}