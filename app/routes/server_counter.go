//go:build vango_server && !wasm
// +build vango_server,!wasm

package routes

import (
	"fmt"
	"sync"
	"time"

	"github.com/recera/vango/pkg/live"
	"github.com/recera/vango/pkg/server"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// Global state store for demo purposes
// In production, use proper session state management
var (
	counters = make(map[string]int)
	mu       sync.RWMutex
)

// ServerCounterPage demonstrates a fully server-driven counter component
func ServerCounterPage(ctx server.Ctx) (*vdom.VNode, error) {
	// Get session ID from context
	sessionID := ctx.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		// Generate a session ID for this demo
		sessionID = fmt.Sprintf("demo_%d", GenerateID())
	}

	// Get or initialize counter value
	mu.RLock()
	count := counters[sessionID]
	mu.RUnlock()

	// Create the page structure with hydration IDs for live updates
	return functional.Div(
		vdom.Props{
			"class": "min-h-screen flex items-center justify-center bg-gray-100",
		},
		functional.Div(
			vdom.Props{
				"class": "bg-white rounded-lg shadow-lg p-8 max-w-md w-full",
			},
			// Title
			functional.H1(
				vdom.Props{
					"class": "text-3xl font-bold text-center mb-2",
				},
				functional.Text("Server-Driven Counter"),
			),

			// Mode indicator
			functional.Div(
				vdom.Props{
					"class": "text-center mb-6",
				},
				functional.Span(
					vdom.Props{
						"class": "inline-block px-3 py-1 bg-red-500 text-white rounded-full text-sm font-semibold",
					},
					functional.Text("🔴 Server Mode"),
				),
			),

			// Counter display
			functional.Div(
				vdom.Props{
					"class": "text-center mb-8",
				},
				functional.Div(
					vdom.Props{
						"id":       "counter-display",
						"data-hid": "h1", // Hydration ID for live updates
						"class":    "text-6xl font-bold text-blue-600 transition-transform",
					},
					functional.Text(fmt.Sprintf("%d", count)),
				),
			),

			// Button container
			functional.Div(
				vdom.Props{
					"class": "flex gap-4 justify-center mb-6",
				},
				// Decrement button
				functional.Button(
					vdom.Props{
						"data-hid":          "h2",
						"data-server-event": "decrement",
						"class":             "px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-semibold",
					},
					functional.Text("− Decrement"),
				),

				// Reset button
				functional.Button(
					vdom.Props{
						"data-hid":          "h3",
						"data-server-event": "reset",
						"class":             "px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors font-semibold",
					},
					functional.Text("↺ Reset"),
				),

				// Increment button
				functional.Button(
					vdom.Props{
						"data-hid":          "h4",
						"data-server-event": "increment",
						"class":             "px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors font-semibold",
					},
					functional.Text("+ Increment"),
				),
			),

			// Info box
			functional.Div(
				vdom.Props{
					"class": "bg-blue-50 border-l-4 border-blue-500 p-4 rounded",
				},
				functional.P(
					vdom.Props{
						"class": "text-sm text-blue-800",
					},
					functional.Strong(vdom.Props{}, functional.Text("Server-Driven Mode: ")),
					functional.Text("All state is managed on the server. Click events are sent via WebSocket and patches are applied to update the UI."),
				),
			),

			// Connection status (will be updated via patches)
			functional.Div(
				vdom.Props{
					"id":       "connection-status",
					"data-hid": "h5",
					"class":    "text-center mt-4 text-sm text-gray-600",
				},
				functional.Text("⚫ Connecting..."),
			),
		),
	), nil
}

// RegisterServerHandlers sets up the event handlers for the server-driven counter
func RegisterServerHandlers() {
	// This would be called during server initialization
	// to register handlers for specific node IDs and event types

	bridge := live.GetBridge()
	if bridge == nil {
		return
	}

	// The actual event handling will be done through the scheduler bridge
	// which will update state and generate patches
}

// GenerateID generates a unique ID (simplified for demo)
func GenerateID() uint32 {
	// In production, use a proper ID generator
	return uint32(time.Now().UnixNano() & 0xFFFFFFFF)
}
