---
title: Build and Distribution Pipeline
slug: build-system-architecture
status: stable
---

### Build Command

`vango build` produces a `dist/` directory with an optimized WASM bundle and a server binary (when server files exist). Key steps:

```37:186:cmd/vango/build.go
// 1) Clean output dir and create dist/assets
// 2) Tailwind build if config present
// 3) TinyGo build of client (prefers app/client/main.go)
// 4) Copy wasm_exec.js and bootstrap.js (NODE_ENV=production)
// 5) Copy public/
// 6) Production routing codegen (router + main)
// 7) go build -tags vango_server main_gen.go (if server files present)
```

### Output Layout (Typical)

```
dist/
  assets/
    app.wasm
    wasm_exec.js
    bootstrap.js
  index.html
  public/
```

If server sources (tagged for `vango_server`) are present, a `dist/server` binary is also created.

### Production Routing Codegen

`NewProductionBuilder("app/routes").Build()` emits `internal/generated/routes/*` and `router/table.json`. `GenerateProductionServer()` writes `main_gen.go` which is compiled into the `dist/server` binary.

```417:507:cmd/vango/internal/routes/production_builder.go
// main_gen.go template with mux and endpoints
```

### WASM Optimizations

- `-opt z` and `-gc leaking` when `--optimize`
- Removes debug symbols (`-no-debug`)
- Gzip size reporting for visibility

```106:121:cmd/vango/build.go
// log sizes and gzipped size of app.wasm
```

### Serving in Production

The generated `main` serves:

- Live WebSocket: `/vango/live/`
- Client route table: `/router/table.json`
- Assets: `/assets/`, static `public/` and `dist/`
- Application routes: `/` via `pkg/server.Router`

### Extending the Build

- Tailwind: Provide `tailwind.config.js`; files under `styles/`
- Templates: `vango gen template` compiles `.vex` to Go
- Routing: Place `.go` files under `app/routes/` and adhere to bracket parameter conventions


