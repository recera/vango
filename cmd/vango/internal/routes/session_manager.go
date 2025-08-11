package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// SessionManager manages sessions for server-driven components
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ManagedSession
	
	// Dependencies
	liveServer *live.Server
	bridge     *live.SchedulerBridge
	registry   *server.ComponentRegistry
	
	// Configuration
	cookieName   string
	cookiePath   string
	cookieDomain string
	cookieSecure bool
	maxAge       time.Duration
}

// ManagedSession represents a managed session with components
type ManagedSession struct {
	ID         string
	CreatedAt  time.Time
	LastAccess time.Time
	
	// Session data
	Data       map[string]interface{}
	
	// Authentication
	IsAuthenticated bool
	UserID          string
	
	// Components in this session
	Components map[string]*server.ComponentInstance
	
	// Live session reference
	LiveSession *live.Session
	
	mu sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager(liveServer *live.Server) *SessionManager {
	return &SessionManager{
		sessions:     make(map[string]*ManagedSession),
		liveServer:   liveServer,
		bridge:       live.NewSchedulerBridge(liveServer),
		registry:     server.NewComponentRegistry(),
		cookieName:   "vango-session",
		cookiePath:   "/",
		cookieDomain: "",
		cookieSecure: false,
		maxAge:       24 * time.Hour,
	}
}

// GetOrCreateSession gets or creates a session for a request
func (sm *SessionManager) GetOrCreateSession(w http.ResponseWriter, r *http.Request) (*ManagedSession, error) {
	// Try to get session ID from cookie
	sessionID := ""
	if cookie, err := r.Cookie(sm.cookieName); err == nil {
		sessionID = cookie.Value
	}
	
	// Try to get existing session
	if sessionID != "" {
		sm.mu.RLock()
		session, exists := sm.sessions[sessionID]
		sm.mu.RUnlock()
		
		if exists {
			// Update last access time
			session.mu.Lock()
			session.LastAccess = time.Now()
			session.mu.Unlock()
			return session, nil
		}
	}
	
	// Create new session
	sessionID = generateSessionID()
	session := &ManagedSession{
		ID:         sessionID,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		Data:       make(map[string]interface{}),
		Components: make(map[string]*server.ComponentInstance),
	}
	
	// Store session
	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()
	
	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sm.cookieName,
		Value:    sessionID,
		Path:     sm.cookiePath,
		Domain:   sm.cookieDomain,
		Secure:   sm.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.maxAge.Seconds()),
	})
	
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*ManagedSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, exists := sm.sessions[sessionID]
	return session, exists
}

// CreateServerComponent creates a server component for a session
func (sm *SessionManager) CreateServerComponent(
	session *ManagedSession,
	componentID string,
	handler server.HandlerFunc,
) (*server.ComponentInstance, error) {
	// Create render function that wraps the handler
	renderFunc := func(ctx *vango.Context) *vdom.VNode {
		// Create a context wrapper that provides session data
		serverCtx := &sessionCtx{
			session: session,
			ctx:     ctx,
		}
		
		// Call the handler
		vnode, err := handler(serverCtx)
		if err != nil {
			// Return error component
			return functional.Div(
				vdom.Props{"class": "error"},
				functional.Text(fmt.Sprintf("Error: %v", err)),
			)
		}
		
		return vnode
	}
	
	// Create component via bridge
	component, err := sm.bridge.CreateServerComponent(
		session.ID,
		componentID,
		renderFunc,
	)
	if err != nil {
		return nil, err
	}
	
	// Register component in session
	session.mu.Lock()
	session.Components[componentID] = component
	session.mu.Unlock()
	
	// Register in global registry
	sm.registry.Register(component)
	
	return component, nil
}

// CleanupSession removes a session and its components
func (sm *SessionManager) CleanupSession(sessionID string) {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		return
	}
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
	
	// Cleanup components
	session.mu.Lock()
	for id, component := range session.Components {
		sm.registry.Unregister(component.ID)
		delete(session.Components, id)
	}
	session.mu.Unlock()
}

// CleanupExpiredSessions removes expired sessions
func (sm *SessionManager) CleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastAccess) > sm.maxAge {
			delete(sm.sessions, id)
			
			// Cleanup components
			for _, component := range session.Components {
				sm.registry.Unregister(component.ID)
			}
		}
	}
}

// StartCleanupRoutine starts a background routine to cleanup expired sessions
func (sm *SessionManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			sm.CleanupExpiredSessions()
		}
	}()
}

// sessionCtx wraps a session as a server.Ctx
type sessionCtx struct {
	session *ManagedSession
	ctx     *vango.Context
	req     *http.Request
	w       http.ResponseWriter
	params  map[string]string
	status  int
}

func (s *sessionCtx) Request() *http.Request      { return s.req }
func (s *sessionCtx) Path() string                { return s.req.URL.Path }
func (s *sessionCtx) Method() string              { return s.req.Method }
func (s *sessionCtx) Query() url.Values           { return s.req.URL.Query() }
func (s *sessionCtx) Param(key string) string     { return s.params[key] }
func (s *sessionCtx) Status(code int)             { s.status = code }
func (s *sessionCtx) StatusCode() int             { return s.status }
func (s *sessionCtx) Header() http.Header         { return s.w.Header() }
func (s *sessionCtx) SetHeader(key, val string)   { s.w.Header().Set(key, val) }
func (s *sessionCtx) Redirect(url string, code int) {
	http.Redirect(s.w, s.req, url, code)
}
func (s *sessionCtx) JSON(code int, v any) error {
	s.w.Header().Set("Content-Type", "application/json")
	s.w.WriteHeader(code)
	return json.NewEncoder(s.w).Encode(v)
}
func (s *sessionCtx) Text(code int, msg string) error {
	s.w.Header().Set("Content-Type", "text/plain")
	s.w.WriteHeader(code)
	_, err := s.w.Write([]byte(msg))
	return err
}
func (s *sessionCtx) Session() server.Session {
	return &sessionImpl{
		data:   s.session.Data,
		isAuth: s.session.IsAuthenticated,
		userID: s.session.UserID,
	}
}
func (s *sessionCtx) Done() <-chan struct{} { 
	// Return a closed channel for now
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (s *sessionCtx) Logger() *slog.Logger {
	return slog.Default().With("session", s.session.ID)
}

// sessionImpl implements server.Session
type sessionImpl struct {
	data     map[string]interface{}
	isAuth   bool
	userID   string
	modified bool
	mu       sync.RWMutex
}

func (s *sessionImpl) IsAuthenticated() bool { return s.isAuth }
func (s *sessionImpl) UserID() string         { return s.userID }
func (s *sessionImpl) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
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

// generateSessionID generates a secure random session ID
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}