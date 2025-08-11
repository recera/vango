---
title: Vango Development Server Architecture
slug: dev-server
status: stable
---

### Purpose

The development server focuses on fast iteration: file watching, TinyGo incremental builds, live WebSocket updates, and dynamic route compilation. It multiplexes API, SSR/universal, and server-driven routes behind a single handler with hot reload.

### Responsibilities

- Load configuration (`vango.json`) and Tailwind settings
- Initialize Live protocol server for server-driven components
- Watch `.go`, `.vex`, `.css`, `.html`, `.js` and trigger rebuilds
- Scan file-based routes, generate routing code + client route table
- Serve WASM, bootstrap, styles, and router table endpoints
- Dispatch requests across API/SSR and server-driven routes

### Bootstrap and Startup Flow

```16:37:cmd/vango/dev.go
type devServer struct {
    port int
    host string
    watcher *fsnotify.Watcher
    liveServer *live.Server
    routeHandler http.Handler
    routeCompiler *routes.Compiler
    apiPatterns, serverDrivenPatterns, ssrPagePatterns []string
}
```

```84:118:cmd/vango/dev.go
// Load config, init build cache & live server
cfg, _ := config.Load(".")
liveServer := live.NewServer()
live.InitBridge(liveServer)
```

```134:153:cmd/vango/dev.go
// Watch for changes and optionally start Tailwind runner
watcher, _ := fsnotify.NewWatcher()
server.watcher = watcher
if !server.disableTailwind { _ = server.startTailwind() }
```

### Route Scanning and Codegen (Dev)

On startup and whenever `app/routes/**` changes, the dev server scans routes and generates code artifacts (radix tree, typed params, path helpers) and the client route table.

```155:173:cmd/vango/dev.go
routeManifest, err := router.ScanRoutes("app/routes")
if err == nil {
  _ = router.GenerateRouteTree(routeManifest)
  _ = router.GenerateClientRouteTable(routeManifest)
}
```

The generator produces:

- `router/params.go` – typed param structs
- `router/paths.go` – path helper functions
- `router/table.json` – JSON used by the client bootstrap for CSR

See generator entrypoints:

```68:89:cmd/vango/internal/router/integration.go
func GenerateRouteTree(manifest *RouteManifest) error
func GenerateClientRouteTable(manifest *RouteManifest) error
```

### Runtime Routing Topology (Dev)

Dev compiles routes into a subprocess for API + SSR/universal pages and wires a Live handler for server-driven components. A composite handler chooses the correct backend per request.

```636:704:cmd/vango/dev.go
// Prefer compiler (API + SSR/universal)
if comp, err := routes.NewCompiler("app/routes"); err == nil {
  if h, err := comp.CompileAll(); err == nil {
     server.routeCompiler = comp
     compiledHandler = h
  }
}

// Live handler for server-driven routes (or static fallback)
var liveOrStatic http.Handler = staticHandler
if s.liveServer != nil {
  if lh, err := routes.NewLiveHandler("app/routes", s.liveServer, staticHandler); err == nil {
    liveOrStatic = lh
  }
}

// Composite selector
s.routeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  path := r.URL.Path
  switch {
  case s.isAPIRoute(path), s.isSSRPageRoute(path):
    compiledHandler.ServeHTTP(w, r)
  case s.isServerDrivenRoute(path):
    liveOrStatic.ServeHTTP(w, r)
  default:
    staticHandler.ServeHTTP(w, r)
  }
})
```

Route classification is derived from the route scan and supports bracket-style typed parameters:

```862:925:cmd/vango/dev.go
// matchPathRoute supports [name[:type]] and catch-all [...name]
```

### Dev Compiler (API + SSR/Universal)

The compiler generates a tiny entry that registers discovered routes into `pkg/server.Router` then serves from a subprocess. The dev server reverse-proxies requests to that subprocess.

```84:118:cmd/vango/internal/routes/compiler.go
// Generated CreateRouter registers AddAPIRoute / AddRoute based on signatures
```

```360:449:cmd/vango/internal/routes/compiler.go
// compileAsSubprocess → build a small server, run it, and proxy via httputil.ReverseProxy
```

### Live Handler (Server-Driven)

Server-driven routes are loaded dynamically; responses inject a minimal client and connect to the Live WebSocket to receive binary patch frames.

```47:116:cmd/vango/internal/routes/live_handler.go
func (h *LiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { /* match, mw chain, render, inject */ }
```

```283:340:cmd/vango/internal/routes/loader.go
// renderServerComponent builds <html>, adds session meta, injects client
html = server.InjectServerDrivenClient(html, sessionID)
```

### File Watching and Hot Reload

```331:373:cmd/vango/dev.go
// fsnotify debounce loop → handleFileChanges(events)
```

```408:491:cmd/vango/dev.go
// VEX changed → compile templates; routes changed → regenerate + recompile;
// Go or CSS changed → rebuild WASM or live-reload styles; notify clients over WS
```

### Static and Special Endpoints

```199:229:cmd/vango/dev.go
mux := http.NewServeMux()
mux.HandleFunc("/vango/live/", server.handleWebSocket)
mux.HandleFunc("/app.wasm", server.serveWASM)
mux.HandleFunc("/wasm_exec.js", server.serveWasmExec)
mux.HandleFunc("/vango/bootstrap.js", server.serveBootstrap)
mux.HandleFunc("/styles.css", server.serveRootStyles)
mux.HandleFunc("/styles/", server.serveStyles)
mux.HandleFunc("/router/table.json", server.serveRouterTable)
```

### Tips for Contributors

- Favor bracket-style params `[slug]`, `[id:int]`, `[...rest]` so matching and codegen remain consistent
- Keep API handlers return `(any, error)` to enable autowrap JSON responses in `AddAPIRoute`
- If a route should be fully server-driven, add a server handler under appropriate build tags (see `//go:build vango_server && !wasm`)
- Extending classifier logic? Update `refreshRouteClassifiers()` and `matchPathRoute()` in tandem


