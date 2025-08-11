---
title: Routing and Code Generation
slug: routing-and-codegen
status: stable
---

### Overview

Vango uses file-based routing with bracket-style parameters to generate:

- A radix-tree matcher (for SSR/universal, used in examples and demos)
- Typed parameter structs and path helpers
- A portable client-side route table (`router/table.json`)

Two scanner/generator paths exist: a dev-time generator (under `cmd/vango/internal/router`) and a production builder (under `cmd/vango/internal/routes`). Both converge on the same route table shape.

### File-Based Conventions

- Static route: `app/routes/about.go` → `/about`
- Index: `app/routes/index.go` → `/`
- Param: `app/routes/blog/[slug].go` → `/blog/[slug]`
- Typed param: `[id:int]`, `[id:int64]`, `[id:uuid]`
- Catch-all: `[...rest]`
- API routes: `app/routes/api/*.go` → `/api/...`

### Scanner (Dev)

```45:66:cmd/vango/internal/router/scanner.go
func NewScanner(routesDir string) *Scanner
func (s *Scanner) Scan() ([]RouteInfo, error)
```

Dev `RouteInfo` captures `URLPath`, `Params`, and metadata such as `HasLayout`, `HasMiddleware`, `IsAPI`.

### Generator Outputs (Dev)

```55:91:cmd/vango/internal/router/codegen.go
Generate() → generateRadixTree, generateParams, generatePaths, generateRouterTable
```

- `router/params.go` – typed params + parse helpers

```243:276:cmd/vango/internal/router/codegen.go
// type XParams { Field types... }
// func ParseXParams(map[string]string) (XParams, error)
```

- `router/paths.go` – type-safe path builders

```348:453:cmd/vango/internal/router/codegen.go
// func Blog(slug string) string { return "/blog/[slug]" -> replacements }
```

- `router/table.json` – CSR source of truth

```455:511:cmd/vango/internal/router/codegen.go
type routeEntry { Path, Component, Params[], Middleware[] }
```

### Scanner (Production)

Production’s `routes.Scanner` is used by the `ProductionBuilder` and adheres to bracket-style params. It returns `RouteFile` with `URLPattern`, `IsAPI`, `HasServer`, `HasClient`, `Params[]`.

```56:111:cmd/vango/internal/routes/scanner.go
func (s *Scanner) ScanRoutes() ([]RouteFile, error)
```

### Production Builder

Generates a function to register routes, a small production `main`, and the route table JSON.

```46:64:cmd/vango/internal/routes/production_builder.go
func (b *ProductionBuilder) Build() error
func (b *ProductionBuilder) GenerateProductionServer() error
```

### Runtime Router

`pkg/server.Router` is a radix-like tree that expects bracket-style params at registration time. It validates typed params at match time and wraps API results into JSON responses.

```60:92:pkg/server/router.go
func (r *Router) AddRoute(path string, handler HandlerFunc, middleware ...Middleware)
func (r *Router) AddAPIRoute(path string, handler APIHandlerFunc, middleware ...Middleware)
```

Typed validation:

```367:394:pkg/server/router.go
validateParam(value, paramType string) bool // int, int64, uuid, string
```

### Client Route Table

Both dev and prod emit the same shape to `router/table.json`:

```json
{
  "routes": [
    { "path": "/blog/[slug]", "component": "Page", "params": [{"name":"slug","type":"string"}] }
  ]
}
```

The bootstrap fetches this JSON to enable CSR transitions.

### Best Practices

- Use bracket-style parameters everywhere (`[name[:type]]`, `[...rest]`)
- Keep API under `/api` with `AddAPIRoute` signatures
- For prod SSR/universal, prefer generated router registration; for server-driven, ensure client injection


