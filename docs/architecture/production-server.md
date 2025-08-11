---
title: Vango Production Server Architecture
slug: production-server
status: stable
---

### Purpose

Provide a production-ready entrypoint that statically registers routes, serves the Live protocol, and exposes assets and the client route table. The production build also ensures the bootstrap script runs in production mode and WASM is optimized.

### Code Generation

During `vango build`, we now run the Production Builder:

```123:186:cmd/vango/build.go
// Generate production routing code
if pb, err := routes.NewProductionBuilder("app/routes"); err == nil {
    _ = pb.Build()
    _ = pb.GenerateProductionServer()
}
```

This creates:

- `internal/generated/routes/router_gen.go` – registers API/universal/server-driven routes into `pkg/server.Router`
- `internal/generated/routes/server_components_gen.go` – wrappers for server-driven components
- `main_gen.go` – production `main()` with mux wiring
- `router/table.json` – client route table (matching dev JSON shape)

### Generated Router Registration

```85:139:cmd/vango/internal/routes/production_builder.go
func RegisterRoutes(router *server.Router, liveServer *live.Server, sessionMgr *SessionManager) {
    // API routes → AddAPIRoute (auto JSON serialization)
    // Universal/SSR → AddRoute (VNode rendering)
    // Server-driven → AddRoute with InjectServerDrivenClient
}
```

Server-driven injection:

```104:131:cmd/vango/internal/routes/production_builder.go
vnode, err := routesX.Handler(ctx)
vnode = server.InjectServerDrivenClient(vnode, sessionID)
```

### Generated main()

```427:499:cmd/vango/internal/routes/production_builder.go
func main() {
  liveServer := live.NewServer()
  sessionMgr := routes.NewSessionManager(liveServer)
  router := server.NewRouter()
  routes.RegisterRoutes(router, liveServer, sessionMgr)

  mux := http.NewServeMux()
  mux.HandleFunc("/vango/live/", liveServer.HandleWebSocket)
  mux.HandleFunc("/router/table.json", serveTableJSON)
  mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("dist/assets"))))
  mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
  mux.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("dist"))))
  mux.Handle("/", router)
  _ = http.ListenAndServe(addr, mux)
}
```

`/router/table.json` handler uses the generated file if present, falling back to an empty payload.

### Route Table Format (Prod)

Matches dev:

```371:425:cmd/vango/internal/routes/production_builder.go
type paramDef struct { Name, Type string }
type routeEntry struct { Path, Component string; Params []paramDef }
payload := struct { Routes []routeEntry; Generated bool }{ ... }
```

### WASM and Bootstrap Layout

`vango build` emits assets to `dist/assets/`:

- `dist/assets/app.wasm` – built with TinyGo; optimized when `--optimize`
- `dist/assets/wasm_exec.js` – TinyGo runtime
- `dist/assets/bootstrap.js` – production-mutated bootstrap (NODE_ENV='production')

```90:141:cmd/vango/build.go
// Copy wasm_exec.js and bootstrap.js → dist/assets/
```

### Integration Points

- Live protocol WebSocket: `/vango/live/`
- Client route table: `/router/table.json`
- Assets: `/assets/`, `public/`, `dist/`

### Extending Production Routing

- Add new route files under `app/routes/` with bracket-style params `[slug]`, typed params `[id:int]`, catch-all `[...rest]`
- Keep API routes under `app/routes/api/` and return `(any, error)`
- For server-driven pages, include server-tagged handlers (e.g. `//go:build vango_server && !wasm`)


