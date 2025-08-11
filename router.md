Dev server: route discovery, codegen, dispatch
•️ Startup (scan + codegen): On vango dev, the server:
Loads config, starts the live WebSocket server, watches the FS.
Scans app/routes via the routing scanner and generates router code (radix tree + typed params + path helpers + router table JSON).

dev.go
  // Scan and generate routes
  routeManifest, err := router.ScanRoutes("app/routes")
  ...
  // Generate route tree and helpers
  if err := router.GenerateRouteTree(routeManifest); err != nil { ... }
  // Generate client route table
  if err := router.GenerateClientRouteTable(routeManifest); err != nil { ... }

•️ Serving dev assets:
WASM: /app.wasm, TinyGo wasm_exec.js.
Bootstrap: /vango/bootstrap.js.
Router table: /router/table.json (served from generated file).

dev.go
  // Serve router table for client-side routing
  mux.HandleFunc("/router/table.json", server.serveRouterTable)
  ...
  if server.routeHandler != nil {
      mux.Handle("/", server.routeHandler)
  } else {
      mux.HandleFunc("/", server.serveStatic)
  }

  •️ Route handlers (composite):
Preferred path uses a compiler to build an http.Handler for API + SSR/universal routes (subprocess or plugin).
Server-driven routes go through a “live” handler that injects a minimal client and links the scheduler bridge.
Client-only routes fall back to static (so the WASM app takes over CSR).

dev.go
  s.routeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      path := r.URL.Path
      if s.isAPIRoute(path) {
          compiledHandler.ServeHTTP(w, r); return
      }
      if s.isServerDrivenRoute(path) {
          liveOrStaticHandler.ServeHTTP(w, r); return
      }
      if s.isSSRPageRoute(path) {
          compiledHandler.ServeHTTP(w, r); return
      }
      staticHandler.ServeHTTP(w, r)
  })

  •️ Classification (used only to choose handler):
  dev.go
    // refreshRouteClassifiers: collects API, server-driven, SSR patterns
  ...
  func (s *devServer) isAPIRoute(path string) bool { ... matchPathRoute ... }
  func (s *devServer) isServerDrivenRoute(path string) bool { ... }
  func (s *devServer) isSSRPageRoute(path string) bool { ... }

  •️ Dev “compiler” behavior:
Generates a tiny main that registers discovered API + SSR/universal routes on a server.Router, runs an HTTP server in a subprocess, and the dev server reverse-proxies to it. Server-driven routes are explicitly skipped here (they’re handled by the live handler).

  // CreateRouter builds a server.Router and registers API & SSR/Universal routes
  router.AddAPIRoute("{{.URLPattern}}", func(ctx server.Ctx) (any, error) { ... })
  ...
  router.AddRoute("{{.URLPattern}}", func(ctx server.Ctx) (*vdom.VNode, error) { ... })

  ...
    // compileAsSubprocess → returns an http.Handler that reverse-proxies to route server