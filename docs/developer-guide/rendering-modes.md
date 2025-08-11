# Rendering Modes Deep Dive

Vango routes can be authored in three modes. Mix modes per route and choose per-page.

- Universal (SSR + Hydration): default if no pragma/build tag is present
- Server-Driven (Live over WebSocket): server authoritative state, minimal client
- Client-Only (CSR via WASM): browser-only logic, no server round-trips

## Mode selection (how to opt in)
- Universal: export `Page(ctx server.Ctx) (*vdom.VNode, error)` (or similar) with no extra tags
- Server-Driven: `//vango:server` or `//go:build vango_server && !wasm`
- Client-Only: `//vango:client` or `//go:build vango_client`

## Universal — Example
```go
// app/routes/about.go
package routes

import (
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  return builder.Div().Class("prose mx-auto p-8").Children(
    builder.H1().Text("About").Build(),
    builder.P().Text("Rendered on the server; hydrated by WASM.").Build(),
  ).Build(), nil
}
```
Lifecycle
- Server renders HTML (great SEO and TTFB)
- Client loads WASM and hydrates event handlers
- Subsequent updates happen on the client

## Server-Driven — Example
```go
//go:build vango_server && !wasm
package routes

import (
  "fmt"
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  count := 0
  incr := func(){ count++ }
  return builder.Html().Children(
    builder.Head().Build(),
    builder.Body().Children(
      builder.H1().Text("Server Counter").Build(),
      builder.Div().ID("counter-display").Text(fmt.Sprint(count)).Build(),
      builder.Button().Text("+1").OnClick(incr).Build(),
    ).Build(),
  ).Build(), nil
}
```
Lifecycle
- HTML includes minimal client via injection
- Client connects to `/vango/live/<session>`
- User events are forwarded to the server; server re-renders and sends binary patches

## Client-Only — Example
```go
//go:build vango_client
package components

import (
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func InteractiveChart() *vdom.VNode {
  // Browser-only logic; no server round-trips
  return builder.Div().Class("chart").Text("Chart here").Build()
}
```
Lifecycle
- Server serves a shell; client builds everything in WASM
- No WS dependency; CSR navigation recommended

## Selection Guide
- Universal: marketing pages, content-heavy routes, SEO-critical pages
- Server-Driven: admin dashboards, collaborative tools, forms with heavy server validation
- Client-Only: offline/low-latency interactivity, canvas/WebGL, large client-only widgets

## Pitfalls and Tips
- Server-Driven requires WS reachability; handle reconnects gracefully (client does by default)
- Client-Only needs CSR routes to avoid full reloads; implement `window.__vango_navigate`
- Universal hydration: ensure your SSR DOM matches what the client expects (same VDOM tree)
- Mixed app: it’s normal to have Universal pages that embed Client-Only components and some Server-Driven routes

## Migration Strategy
- Start with Universal for simplicity and SEO
- Promote particular views to Server-Driven when you need real-time server authority
- Carve out Client-Only components for highly interactive custom UI or offline scenarios
