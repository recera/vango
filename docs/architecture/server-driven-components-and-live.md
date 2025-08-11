---
title: Server-Driven Components and Live Protocol
slug: server-driven
status: stable
---

### Overview

Server-driven pages/components render DOM server-side and keep the server authoritative for state. The client is minimal; events are delegated to the server over a binary WebSocket protocol; the server responds with patch frames to mutate the DOM.

### Key Pieces

- Live server: `pkg/live.Server` lifecycle and session management
- Scheduler bridge: `live.NewSchedulerBridge` for component instances
- Session manager (prod): associates components to sessions
- Minimal client injection: adds a small JS runtime to handle events/patches

### Dev Handler Pipeline

```47:116:cmd/vango/internal/routes/live_handler.go
func (h *LiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  // 1) Match route using loader.router
  // 2) Run middleware Before chain
  // 3) Execute handler → VNode
  // 4) If server-driven → inject minimal client + create component instance
  // 5) Render HTML and write response
}
```

VNode injection:

```336:340:cmd/vango/internal/routes/loader.go
html = server.InjectServerDrivenClient(html, sessionID)
```

### Production Injection

```104:131:cmd/vango/internal/routes/production_builder.go
// Registered server-driven routes inject minimal client using InjectServerDrivenClient
```

### Client-Side Minimal Runtime

The injected script (see `pkg/server/server_driven_helper.go`) sets up event delegation for clicks/inputs/submits and forwards binary event frames to the server.

```9:60:pkg/server/server_driven_helper.go
// InjectServerDrivenClient(doc *vdom.VNode, sessionID string) *vdom.VNode
```

### WebSocket Endpoint

All server-driven sessions connect to `/vango/live/<sessionId>`. The client starts with a HELLO control frame, and receives binary patch frames (`FramePatches`) that mutate nodes marked by hydration IDs (`data-hid`).

```88:123:internal/assets/bootstrap.js
// initWebSocket() – choose ws/wss, send HELLO, handle control + patch frames
```

### Building Server-Driven Pages

- Mark the page as server-only with build tags or pragmas (e.g., `//go:build vango_server && !wasm`)
- Render elements with hydration IDs (e.g., `data-hid: "h42"`) for precise patching
- Use scheduler bridge to manage component instances per session

### Example (Demo)

```1:40:app/routes/server_counter.go
//go:build vango_server && !wasm
func ServerCounterPage(ctx server.Ctx) (*vdom.VNode, error) { /* ... */ }
```

### Notes

- The client bootstrap and minimal injected client are distinct. The injected client is only added for server-driven pages.
- In dev, the live handler returns a minimal shell HTML to bootstrap and connect; in production, the registered handler injects client into the actual VNode returned by the route handler.


