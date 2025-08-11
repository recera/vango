# Vango Developer Guide (Alpha)

This is the authoritative, end‑to‑end guide for building Vango applications. It reflects the current source code and architecture. Cross‑linking docs: development server, production server, routing/codegen, client bootstrap/CSR, and server‑driven live protocol.

Contents
- What is Vango, prerequisites
- Project layout and conventions
- Rendering modes and build pragmas
- Components and Virtual DOM APIs (functional, builder, templates)
- File‑based routing and generated artifacts
- Request context, middleware, API routes
- Development server lifecycle (watch, compile, serve)
- Client bootstrap, CSR navigation, hydration
- Server‑driven components and Live protocol
- Styling and Tailwind integration
- Production build and server
- Cookbook: common tasks (pages, params, API, live components)
- Troubleshooting and references

Deep-dive suite
- See `docs/developer-guide/`:
  - `rendering-modes.md`
  - `vex-syntax.md`
  - `components-and-vdom.md`
  - `state-management.md`
  - `routing-and-codegen.md`
  - `styling.md`
  - `dev-server.md`
  - `live-protocol.md`
  - `client-bootstrap-and-csr.md`
  - `configuration.md`
  - `testing.md`
  - `cookbook.md`

## What is Vango
- Go‑native UI framework using a single Virtual DOM across server and WASM client
- Three modes per route: Universal (SSR + hydrate), Server‑Driven (live over WS), Client‑Only (pure WASM)
- File‑based routing with typed params and generated helpers
- Minimal client runtime, efficient binary patch protocol for server‑driven mode

## Prerequisites
- Go 1.22+
- TinyGo for WASM builds
- Node.js optional (Tailwind)

## Project Layout
- `app/routes/**` defines routes
- `app/main.go` or `app/client/main.go` is the WASM entry; dev/prod will build one of these
- `public/` static assets served in dev and shipped in prod
- `router/` generated: `params.go`, `paths.go`, `table.json`
- `vango.json` config (dev server, styling, etc.)

## Rendering Modes and Pragmas
- Universal/SSR (default): exported handler returning `(*vdom.VNode, error)` with optional `server.Ctx`
- Server‑Driven: mark file as server code using either `//vango:server` pragma or `//go:build vango_server && !wasm`
- Client‑Only: mark with `//vango:client` or build tag `vango_client`
- The pragma scanner can auto‑inject build tags from `//vango:*` comments during `vango dev`

## Components and Virtual DOM
- VNode type under `pkg/vango/vdom` with element/text kinds, `Props` (attributes and events), `Kids`, and optional key/ref
- Appliers: `pkg/renderer/html` (SSR to string) and `pkg/renderer/dom` (WASM DOM patches)

APIs to create VNodes
- Functional: `github.com/recera/vango/pkg/vex/functional` provides tag functions and prop/event helpers
- Builder (fluent): `github.com/recera/vango/pkg/vex/builder` provides chainable methods per tag and attributes/events
- Templates (VEX): `//vango:template` and optional `//vango:props { ... }` compiled by `cmd/vango/internal/template`

Event wiring by mode
- Universal/CSR: event handlers hydrate via WASM
- Server‑Driven: minimal client delegates events; nodes use `data-hid`, optional `data-events`

## File‑Based Routing and Codegen
Dev route scan → codegen
- `cmd/vango/internal/router/scanner.go` discovers pages and API from `app/routes/**` using bracket params: `[name]`, `[id:int]`, `[...rest]`
- `cmd/vango/internal/router/codegen.go` emits:
  - `router/params.go` (typed param structs + `ParseXParams`)
  - `router/paths.go` (type‑safe path builders)
  - `router/table.json` (client route table for CSR)

Prod route scan → builder
- `cmd/vango/internal/routes/scanner.go` returns `RouteFile` with `HasServer`, `HasClient`, `IsAPI`, `Params`, `ImportPath`
- `cmd/vango/internal/routes/production_builder.go` generates:
  - `internal/generated/routes/router_gen.go` – registers routes into `pkg/server.Router`
  - `internal/generated/routes/server_components_gen.go` – wrappers for server components (if any)
  - `router/table.json` – CSR route table
  - `main_gen.go` – production entry wiring WS/static/router

Runtime router
- `pkg/server/router.go` radix‑like matcher with bracket params and typed validation
- Page: `AddRoute(path, HandlerFunc)`; API: `AddAPIRoute(path, APIHandlerFunc)` (auto JSON)
- Middleware: global and node‑level via Before/After hooks; `server.Stop()` to abort
- Custom 404/500 via `SetNotFound`, `SetErrorPage`

## Context API
`pkg/server/context.go` `server.Ctx` provides:
- Request: `Request()`, `Path()`, `Method()`, `Query()`, `Param(key)`
- Response: `Status`, `Header`, `SetHeader`, `Redirect`, `JSON`, `Text`
- Session: cookie‑backed placeholder (`Get/Set/Delete`, `IsAuthenticated`, `UserID`)
- Logger and `Done()`

## Development Server
- Command: `vango dev` (`cmd/vango/dev.go`)
- Loads `vango.json`; initializes Live server; compiles VEX → Go; scans routes; generates router artifacts; builds WASM with TinyGo
- Serves:
  - `/app.wasm`, `/wasm_exec.js`, `/vango/bootstrap.js`
  - `/styles.css` and `/styles/**`
  - `/router/table.json`
  - `/vango/live/<session>` WebSocket
  - `/*` composite route handler (API/SSR via subprocess; server‑driven via live handler; fallback static)

## Client Bootstrap and CSR
- `internal/assets/bootstrap.js` loads WASM, fetches router table, hydrates DOM, intercepts links for CSR, connects Live WS
- CSR API: `navigate(path, {replace})` and hooks `window.__vango_navigate`, `window.__vango_prefetch`
- Hydration uses a sparse `data-hid` map for precise listener attachment

## Server‑Driven Components and Live Protocol
- WS endpoint: `/vango/live/<sessionId>`
- Frames (`pkg/live/types.go`): `FramePatches=0x00`, `FrameEvent=0x01`, `FrameControl=0x02`
- Server (`pkg/live/server.go`): manages sessions, encodes patches (`EncodePatches`) and handles events; optional scheduler bridge dispatches to component instances
- Client (`internal/assets/server-driven-client.js` and injected script from `pkg/server/server_driven_helper.go`): receives patches, applies to `data-hid` nodes, delegates `click/input/submit`
- Injection: `server.InjectServerDrivenClient(htmlVNode, sessionID)` adds meta + client script

## Styling and Tailwind
- `vango.json` → `styling.tailwind` with `enabled`, `strategy` (`auto|npm|standalone`), `config/input/output`, `watch`, `autoDownload`
- Dev server detects config, runs Tailwind runner, writes `/public/styles.css`; basic CSS under `/styles/**` and `/public/`

## Production Build and Server
- `vango build` emits:
  - `dist/assets/`: `app.wasm`, `wasm_exec.js`, `bootstrap.js`
  - Generated router and `main_gen.go`
- Production server serves WS, router table, assets, and app routes

## Cookbook
Create a page
```go
// app/routes/about.go
package routes

import (
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  return builder.Div().
    Class("prose mx-auto p-8").
    Children(
      builder.H1().Text("About").Build(),
      builder.P().Text("Built with Vango").Build(),
    ).Build(), nil
}
```

Dynamic route with typed param
```go
// app/routes/blog/[slug].go → /blog/[slug]
package blog

import (
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  slug := ctx.Param("slug")
  return builder.Div().Text("Post: "+slug).Build(), nil
}
```

Path helpers (dev)
```go
href := router.BlogSlug("hello-world") // "/blog/hello-world"
```

API route
```go
// app/routes/api/users.go → /api/users
package api

import "github.com/recera/vango/pkg/server"

type User struct{ ID int; Name string }

func Page(ctx server.Ctx) ([]User, error) {
  return []User{{ID:1, Name:"Ada"}}, nil
}
```

Server‑driven counter
```go
//go:build vango_server && !wasm
package routes

import (
  "fmt"
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func ServerCounterPage(ctx server.Ctx) (*vdom.VNode, error) {
  count := 0
  increment := func() { count++ }

  return builder.Html().Children(
    builder.Head().Build(),
    builder.Body().Children(
      builder.H1().Text("Server Counter").Build(),
      builder.Div().ID("counter-display").Text(fmt.Sprint(count)).Build(),
      builder.Button().Text("Increment").OnClick(increment).Build(),
    ).Build(),
  ).Build(), nil
}
```

Dev tips
- `vango dev` regenerates routes, compiles VEX, builds WASM; hot reload for styles and WASM
- Tailwind runner writes `/public/styles.css`
- Inspect `/router/table.json` for CSR issues

Troubleshooting
- WASM build fails → ensure TinyGo installed and WASM entry present
- Route not matched → verify file path → URL mapping and bracket params
- Live WS not connecting → check console logs and presence of `meta[name="vango-session"]`

Further Reading
- `docs/architecture/dev-server.md`
- `docs/architecture/production-server.md`
- `docs/architecture/routing-and-codegen.md`
- `docs/architecture/client-bootstrap-and-csr.md`
- `docs/architecture/server-driven-components-and-live.md`


