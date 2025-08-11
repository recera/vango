package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recera/vango/pkg/vango/vdom"
)

func TestRouter_AddRoute(t *testing.T) {
	router := NewRouter()
	
	// Test adding a simple route
	handler := func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("test")), nil
	}
	
	router.AddRoute("/test", handler)
	
	// Verify route was added by trying to match it
	matchedHandler, params, _ := router.Match("/test")
	
	if matchedHandler == nil {
		t.Error("Route /test was not added or could not be matched")
	}
	
	if len(params) != 0 {
		t.Errorf("Expected no params for /test route, got %v", params)
	}
	
	// Test that the handler executes correctly
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewContext(w, req)
	
	vnode, err := matchedHandler(ctx)
	if err != nil {
		t.Errorf("Handler returned error: %v", err)
	}
	
	if vnode == nil {
		t.Error("Handler returned nil VNode")
	}
}

func TestRouter_Match(t *testing.T) {
	router := NewRouter()
	
	// Create dummy handlers for each route
	homeHandler := func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("home")), nil
	}
	aboutHandler := func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("about")), nil
	}
	blogHandler := func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("blog")), nil
	}
	userPostsHandler := func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("user posts")), nil
	}
	
	// Add routes with different patterns
	router.AddRoute("/", homeHandler)
	router.AddRoute("/about", aboutHandler)
	router.AddRoute("/blog/[slug]", blogHandler)
	router.AddRoute("/user/[id]/posts", userPostsHandler)
	
	tests := []struct {
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		{"/", true, map[string]string{}},
		{"/about", true, map[string]string{}},
		{"/blog/hello-world", true, map[string]string{"slug": "hello-world"}},
		{"/user/123/posts", true, map[string]string{"id": "123"}},
		{"/notfound", false, nil},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler, params, _ := router.Match(tt.path)
			
			if tt.wantMatch {
				if handler == nil {
					t.Errorf("Expected match for %s but got nil", tt.path)
					return
				}
				
				// Check params match
				if tt.wantParams != nil {
					for key, want := range tt.wantParams {
						got, exists := params[key]
						if !exists {
							t.Errorf("Missing param %s", key)
						} else if got != want {
							t.Errorf("Param %s: want %s, got %s", key, want, got)
						}
					}
					
					// Check no extra params
					for key := range params {
						if _, expected := tt.wantParams[key]; !expected {
							t.Errorf("Unexpected param %s with value %s", key, params[key])
						}
					}
				}
			} else {
				// Should match the not found handler or nil
				// Router returns notFound handler which might be nil
				// Just verify params are empty for not found routes
				if len(params) > 0 {
					t.Errorf("Expected no params for not found route, got %v", params)
				}
			}
		})
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	router := NewRouter()
	
	// Add a test route
	router.AddRoute("/test", func(ctx Ctx) (*vdom.VNode, error) {
		return vdom.NewElement("div", nil, vdom.NewText("Test Page")), nil
	})
	
	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	// Check that HTML was rendered
	body := w.Body.String()
	if !contains(body, "<div>Test Page</div>") {
		t.Errorf("Expected HTML output, got: %s", body)
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter()
	
	// Request a non-existent route
	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}