package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
)

var (
	// ErrStop is a sentinel error used by middleware to stop the chain
	ErrStop = errors.New("vango: stop middleware chain")
)

// Stop returns the sentinel error to halt middleware chain execution
func Stop() error {
	return ErrStop
}

// Ctx is the canonical interface passed through routing, middleware, and page handlers
type Ctx interface {
	// === Request ===
	Request() *http.Request      // raw request pointer (read-only)
	Path() string                 // path without query string
	Method() string               // GET, POST, etc.
	Query() url.Values            // parsed query params
	Param(key string) string      // route param, panics if missing

	// === Response ===
	Status(code int)                  // set HTTP status (default 200)
	StatusCode() int                  // current status
	Header() http.Header              // writeable headers
	SetHeader(key, val string)        // convenience
	Redirect(url string, code int)    // sets 30x + Location header
	JSON(code int, v any) error       // serialise & write JSON
	Text(code int, msg string) error  // write text/plain

	// === Session ===
	Session() Session             // cookie-backed session helpers

	// === Internal ===
	Done() <-chan struct{}        // cancellation signal (ctx.Context style)
	Logger() *slog.Logger         // structured logger
}

// Session provides cookie-backed session management
type Session interface {
	IsAuthenticated() bool
	UserID() string
	Get(key string) (string, bool)
	Set(key, val string)
	Delete(key string)
}

// ctxImpl is the internal implementation of Ctx
type ctxImpl struct {
	req           *http.Request
	w             http.ResponseWriter
	params        map[string]string
	statusCode    int
	logger        *slog.Logger
	session       *sessionImpl
	done          chan struct{}
	headerWritten bool
	mu            sync.RWMutex
}

// sessionImpl implements the Session interface
type sessionImpl struct {
	data       map[string]string
	isAuth     bool
	userID     string
	modified   bool
	mu         sync.RWMutex
}

// NewContext creates a new context for handling a request
func NewContext(w http.ResponseWriter, r *http.Request) Ctx {
	logger := slog.Default().With(
		"path", r.URL.Path,
		"method", r.Method,
	)
	
	return &ctxImpl{
		req:        r,
		w:          w,
		params:     make(map[string]string),
		statusCode: http.StatusOK,
		logger:     logger,
		session:    newSession(r),
		done:       make(chan struct{}),
	}
}

// WithParams returns a new context with route parameters set
func WithParams(ctx Ctx, params map[string]string) Ctx {
	if impl, ok := ctx.(*ctxImpl); ok {
		impl.mu.Lock()
		impl.params = params
		impl.mu.Unlock()
	}
	return ctx
}

// === Request Methods ===

func (c *ctxImpl) Request() *http.Request {
	return c.req
}

func (c *ctxImpl) Path() string {
	return c.req.URL.Path
}

func (c *ctxImpl) Method() string {
	return c.req.Method
}

func (c *ctxImpl) Query() url.Values {
	return c.req.URL.Query()
}

func (c *ctxImpl) Param(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	val, ok := c.params[key]
	if !ok {
		panic("vango: route parameter '" + key + "' not found")
	}
	return val
}

// === Response Methods ===

func (c *ctxImpl) Status(code int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.headerWritten {
		c.logger.Warn("attempted to set status after headers written", "code", code)
		return
	}
	c.statusCode = code
}

func (c *ctxImpl) StatusCode() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.statusCode
}

func (c *ctxImpl) Header() http.Header {
	return c.w.Header()
}

func (c *ctxImpl) SetHeader(key, val string) {
	c.w.Header().Set(key, val)
}

func (c *ctxImpl) Redirect(url string, code int) {
	c.mu.Lock()
	c.headerWritten = true
	c.statusCode = code
	c.mu.Unlock()
	
	http.Redirect(c.w, c.req, url, code)
}

func (c *ctxImpl) JSON(code int, v any) error {
	c.mu.Lock()
	c.statusCode = code
	c.headerWritten = true
	c.mu.Unlock()
	
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	
	encoder := json.NewEncoder(c.w)
	return encoder.Encode(v)
}

func (c *ctxImpl) Text(code int, msg string) error {
	c.mu.Lock()
	c.statusCode = code
	c.headerWritten = true
	c.mu.Unlock()
	
	c.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.w.WriteHeader(code)
	
	_, err := c.w.Write([]byte(msg))
	return err
}

// === Session Methods ===

func (c *ctxImpl) Session() Session {
	return c.session
}

func (c *ctxImpl) Done() <-chan struct{} {
	return c.done
}

func (c *ctxImpl) Logger() *slog.Logger {
	return c.logger
}

// === Session Implementation ===

func newSession(r *http.Request) *sessionImpl {
	// TODO: Implement proper cookie-based session with encryption
	// For now, return an empty session
	return &sessionImpl{
		data: make(map[string]string),
	}
}

func (s *sessionImpl) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAuth
}

func (s *sessionImpl) UserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userID
}

func (s *sessionImpl) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *sessionImpl) Set(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
	s.modified = true
}

func (s *sessionImpl) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	s.modified = true
}

// SetAuthenticated sets the authentication status and user ID
func (s *sessionImpl) SetAuthenticated(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isAuth = true
	s.userID = userID
	s.modified = true
}

// Close should be called after the request to persist session changes
func (c *ctxImpl) Close() {
	close(c.done)
	// TODO: Write session cookie if modified
}