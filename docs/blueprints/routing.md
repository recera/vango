---
title: Routing
slug: routing
version: 0.2
phase: P-0
status: draft
requires:
  - rendering-pipeline
---

# Routing Blueprint

> **Goal**: Provide zero-config file-based routing with type-safe params, SSR & WASM parity, and fast radix-tree matching.

## 1. File-System Conventions
```
app/
 └─ routes/
     ├─ index.go          → "/"
     ├─ about.go          → "/about"
     ├─ blog/
     │   ├─ [slug].go     → "/blog/:slug"
     │   └─ _layout.go    → wraps all blog pages
     ├─ api/
     │   └─ users.go      → "/api/users" (JSON)
     └─ _middleware.go    → runs before every route
```

### 1.1 Special Files
| Name | Role |
|------|------|
| `_layout.go` | Wraps sibling pages, provides shared CSS/head |
| `_middleware.go` | Exports `Before`/`After` hooks |

#### Example `_layout.go`
```go
// routes/blog/_layout.go
package blog

func Layout(child vango.VNode) vango.VNode {
    return vango.Div(vango.Class("layout"),
        Header(),                 // shared header component
        vango.Main(nil, child),   // render the page being wrapped
        Footer(),                 // shared footer component
    )
}
```
`Layout` must be an exported identifier that accepts exactly **one** `vango.VNode` parameter. The code-gen wraps every page VNode in the same directory with the result of `Layout(child)`.


## 2. Code-Gen Output
* `pkg/internal/router/tree_gen.go` – radix tree matcher.  
* `router/paths.go` – helpers like `router.Blog(slug) string`.  
* `Params` struct per dynamic route with typed fields.

## 3. API Examples
```go
// routes/blog/[slug].go
type Params struct { Slug string `param:"slug"` }
func Page(p Params) vango.VNode {
    return BlogPost(slug: p.Slug)
}
```
Client link helper:
```go
vango.A(vango.Href(router.Blog("hello-vango")), vango.Text("Read"))
```

## 4. Imperative Additions
```go
router.Add("/legacy", LegacyHandler, vango.NoBuild())
```
Flags: `NoBuild()`, `NoAuth()`, `ClientOnly()`.

### 4.1 API Route Example
Files placed under `routes/api/` are treated as JSON (or raw bytes) endpoints.
```go
// routes/api/users.go
package api

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Page acts like an http.HandlerFunc; returning a value serialises to JSON
func Page() ([]User, error) {
    users := fetchUsersFromDB()
    return users, nil  // -> HTTP 200 with JSON body
}
```
If the `Page` function returns `error` only, Vango writes a `500` status. Use `ctx.JSON(status, data)` for manual control.


## 5. Client-Side Navigation
* On hydration, radix tree is exported as a typed JS object (`const routes = {...}`) for instant `pushState` transitions.  
* Prefetch on viewport `IntersectionObserver`.

## 6. Middleware Lifecycle
```go
type Middleware interface {
    Before(ctx *vango.Ctx) error // return vango.Stop to abort chain
    After(ctx *vango.Ctx) error  // always called if Before succeeded
}
```
Executed server-side on each request; during client-side navigation the same `Before` hook is replayed in WASM to guarantee parity.

### 6.1 Example Auth Middleware
```go
// routes/_middleware.go
package routes

type AuthMW struct{}

func (AuthMW) Before(ctx *vango.Ctx) error {
    if !ctx.Session().IsAuthenticated() {
        ctx.Redirect("/login", 302)
        return vango.Stop // sentinel: stop further handlers & page render
    }
    return nil
}
func (AuthMW) After(ctx *vango.Ctx) error {
    ctx.Logger().Info("Request handled", "path", ctx.Path(), "status", ctx.StatusCode())
    return nil
}

func Middleware() vango.Middleware { return AuthMW{} }
```


## 7. Performance Targets
| Metric | Budget |
|--------|--------|
| Route match | O(log n) <5 µs |
| Client nav TTI | <100 ms |

## 8. Open Questions
* Should `_layout` support nested slots like React’s `Outlet`?  
* Do we allow optional catch-all `[...slug].go`?  
* Need strategy for incremental static regen.

## 8.1 Not-Found & Error Routes
If no matcher fires the router searches for fallback pages:
| File | Purpose |
|------|---------|
| `routes/_404.go` | Rendered for unmatched paths (status 404) |
| `routes/_500.go` | Rendered on panic / internal error |

Example 404 page:
```go
// routes/_404.go
package routes

func Page() vango.VNode {
    return vango.H1(nil, vango.Text("Oops – page not found (404)"))
}
```


## 9. Acceptance & Validation
| Check | Command | Expected |
|-------|---------|----------|
| Radix Benchmark | `go test ./pkg/router -run=^$ -bench=.` | `ns/op` < goal in §7 |
| WASM Prefetch | `make e2e ROUTE=prefetch` | Links prefetch before hover |
| Middleware Parity | `go test ./pkg/router -run TestMiddlewareParity` | Client & server results equal |
| 404 Fallback | `curl -I /does-not-exist` | `HTTP/1.1 404` |

### 9.1 Golden Fixtures
Golden route trees stored in `testdata/radix/*.json`. Use `UPDATE_GOLDEN=1 go test` to regenerate after intentional changes.

## 10. Cross-References
* Codegen spec: `@docs/blueprints/codegen-routing-spec.md`
* Ctx contract: `@docs/blueprints/api-contracts.md`
* Live protocol nav event: `@docs/blueprints/live-protocol.md`

## 11. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
|2025-08-06|0.2|Add acceptance table, cross refs, bump version|
