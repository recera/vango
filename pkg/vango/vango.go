package vango

import (
	"fmt"
	"net/http"

	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/scheduler"
	"github.com/recera/vango/pkg/vango/vdom"
)

// App represents a Vango application
type App struct {
	Title     string
	scheduler *scheduler.Scheduler
	router    Router
}

// Component represents a UI component
type Component interface {
	Render(ctx *Context) *vdom.VNode
}

// RenderMode represents the rendering mode for a component
type RenderMode uint8

const (
	// ModeSSRStatic - Server static pass, no interactivity
	ModeSSRStatic RenderMode = iota
	// ModeClient - WASM owns state, full client-side interactivity
	ModeClient
	// ModeServerDriven - Server authoritative, patches streamed via WebSocket
	ModeServerDriven
)

// Context provides component context
type Context struct {
	Props     map[string]interface{}
	Params    map[string]string
	Query     map[string]string
	Fiber     *scheduler.Fiber
	Scheduler *scheduler.Scheduler
	Mode      RenderMode // Rendering mode for this component
	SessionID string     // Session ID for server-driven mode
	Data      map[string]interface{} // Additional context data
}

// Event represents a DOM event
type Event struct {
	Type   string
	Target struct {
		Value   string
		Checked bool
		ID      string
	}
	Key     string
	KeyCode int
}

// Router handles routing
type Router interface {
	Match(path string) (Component, map[string]string)
	HandleFunc(pattern string, handler http.HandlerFunc)
}

// FC creates a functional component
func FC(render func(ctx *Context) *vdom.VNode) Component {
	return &funcComponent{render: render}
}

type funcComponent struct {
	render func(ctx *Context) *vdom.VNode
}

func (c *funcComponent) Render(ctx *Context) *vdom.VNode {
	return c.render(ctx)
}

// New creates a new Vango application
func New() *App {
	sched := scheduler.NewScheduler()

	return &App{
		Title:     "Vango App",
		scheduler: sched,
	}
}

// Run starts the application
func (app *App) Run() error {
	// In a real implementation, this would:
	// 1. Set up the HTTP server
	// 2. Handle routing
	// 3. Start the scheduler
	// 4. Manage the application lifecycle

	app.scheduler.Start()

	// For now, just return nil
	return nil
}

// State creates a new reactive state
func State[T any](initial T) *reactive.State[T] {
	// Get scheduler from context (in real implementation)
	return reactive.CreateState(initial)
}

// Computed creates a new computed value
func Computed[T any](compute func() T) *reactive.Computed[T] {
	// Get scheduler from context (in real implementation)
	return reactive.CreateComputed(compute)
}

// Batch runs multiple state updates in a batch
func Batch(fn func()) {
	// Get scheduler from context (in real implementation)
	// For now, just run the function
	fn()
}

// Element shortcuts for common HTML elements
var (
	Div = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("div", props, children...)
	}
	Span = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("span", props, children...)
	}
	P = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("p", props, children...)
	}
	H1 = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("h1", props, children...)
	}
	H2 = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("h2", props, children...)
	}
	H3 = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("h3", props, children...)
	}
	Button = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("button", props, children...)
	}
	Input = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("input", props, children...)
	}
	Form = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("form", props, children...)
	}
	Label = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("label", props, children...)
	}
	Ul = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("ul", props, children...)
	}
	Li = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("li", props, children...)
	}
	Nav = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("nav", props, children...)
	}
	Header = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("header", props, children...)
	}
	Footer = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("footer", props, children...)
	}
	Main = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("main", props, children...)
	}
	Article = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("article", props, children...)
	}
	Section = func(props vdom.Props, children ...*vdom.VNode) *vdom.VNode {
		return vdom.NewElement("section", props, children...)
	}
	Text     = vdom.NewText
	Fragment = vdom.NewFragment
)

// Props is a convenience alias
type Props = vdom.Props

// Logging helpers
func Logf(format string, args ...interface{}) {
	fmt.Printf("[Vango] "+format+"\n", args...)
}

func Errorf(format string, args ...interface{}) {
	fmt.Printf("[Vango Error] "+format+"\n", args...)
}

// IsServerRendered returns true if the component is server-rendered (SSR or server-driven)
func (c *Context) IsServerRendered() bool {
	return c.Mode == ModeSSRStatic || c.Mode == ModeServerDriven
}

// IsClientRendered returns true if the component is client-rendered
func (c *Context) IsClientRendered() bool {
	return c.Mode == ModeClient
}

// IsServerDriven returns true if the component is server-driven (live updates via WebSocket)
func (c *Context) IsServerDriven() bool {
	return c.Mode == ModeServerDriven
}

// EmitEvent sends an event to the server (for server-driven components)
// This is a placeholder that will be implemented with the live protocol
func EmitEvent(ctx *Context, eventType string, data interface{}) {
	if ctx.Mode != ModeServerDriven {
		return
	}
	// TODO: Implement event emission via WebSocket
	Logf("EmitEvent: %s (server-driven mode)", eventType)
}

// NewContext creates a new context with the given mode
func NewContext(mode RenderMode) *Context {
	return &Context{
		Mode:   mode,
		Props:  make(map[string]interface{}),
		Params: make(map[string]string),
		Query:  make(map[string]string),
		Data:   make(map[string]interface{}),
	}
}

// WithScheduler sets the scheduler for this context
func (c *Context) WithScheduler(s *scheduler.Scheduler) *Context {
	c.Scheduler = s
	return c
}

// WithSessionID sets the session ID for server-driven mode
func (c *Context) WithSessionID(id string) *Context {
	c.SessionID = id
	return c
}

// Get retrieves a value from context data
func (c *Context) Get(key string) (interface{}, bool) {
	if c.Data == nil {
		return nil, false
	}
	val, ok := c.Data[key]
	return val, ok
}

// Set stores a value in context data
func (c *Context) Set(key string, value interface{}) {
	if c.Data == nil {
		c.Data = make(map[string]interface{})
	}
	c.Data[key] = value
}

// IsStatic returns true if this is static SSR
func (c *Context) IsStatic() bool {
	return c.Mode == ModeSSRStatic
}

// IsClient returns true if this is a client-side component
func (c *Context) IsClient() bool {
	return c.Mode == ModeClient
}
