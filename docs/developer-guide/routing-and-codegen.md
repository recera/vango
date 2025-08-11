# Routing and Codegen Deep Dive

## File → Route Conventions
```
app/routes/
  index.go          → /
  about.go          → /about
  blog/
    index.go        → /blog
    [slug].go       → /blog/[slug]
  user/
    [id:int].go     → /user/[id:int]
  [...catch].go     → /[...catch]
  api/
    users.go        → /api/users
  _layout.go        → directory layout wrapper (optional)
  _middleware.go    → directory middleware (optional)
  _404.go           → custom 404 (optional)
  _500.go           → custom 500 (optional)
```
- Bracket params can be typed; catch-all consumes the rest

## Typed Parameters
- Supported: `int`, `int64`, `uuid`, default `string`
- In dev, `router/params.go` contains parse helpers for deduped param sets
- In handlers, use `ctx.Param("name")` and convert as needed (prod and dev compatible)

## Path Helpers (dev)
- `router/paths.go` exposes functions based on paths, e.g. `BlogSlug(slug string) string`
- Use in components to avoid stringly-typed hrefs

## Client Route Table
- `router/table.json` is generated and fetched by bootstrap for CSR matching
- Entries include `path`, `component`, optional typed `params`

## Dev Codegen
- `cmd/vango/internal/router/scanner.go` scans files → manifest
- `cmd/vango/internal/router/codegen.go` writes:
  - `router/params.go`
  - `router/paths.go`
  - `router/table.json`

## Production Builder
- `cmd/vango/internal/routes/scanner.go` discovers `RouteFile{URLPattern, IsAPI, HasServer, HasClient, Params}`
- `production_builder.go` writes:
  - `internal/generated/routes/router_gen.go` (registers routes into `pkg/server.Router`)
  - `internal/generated/routes/server_components_gen.go` (if server-driven routes exist)
  - `router/table.json`
  - `main_gen.go` (prod entrypoint)

## Runtime Router (`pkg/server/router.go`)
- `AddRoute` for pages, `AddAPIRoute` for API (auto-JSON)
- Radix-like matcher supports bracket params and catch-all
- Typed validation at match time
- Global + node middleware with `Before/After` hooks; return `server.Stop()` to abort chain
- `SetNotFound` and `SetErrorPage` for error pages

## Layouts and Middleware
- `_layout.go` in a directory can wrap children routes (convention; implement by calling layout inside child handlers)
- `_middleware.go` can register middleware for a subtree (convention; implement by adding middleware in registration)

## Example: Dynamic Page
```go
// app/routes/blog/[slug].go
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
