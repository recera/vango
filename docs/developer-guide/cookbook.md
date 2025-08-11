# Cookbook

## Enable Tailwind
1) Set `styling.tailwind.enabled = true` in `vango.json`
2) Create `styles/input.css` with Tailwind directives
3) Link `/styles.css` in `public/index.html`
4) Run `vango dev`

## JSON API Route
```go
// app/routes/api/health.go
package api

import "github.com/recera/vango/pkg/server"

type Health struct{ OK bool }

func Page(ctx server.Ctx) (Health, error) { return Health{OK:true}, nil }
```

## Dynamic Route with Typed Param
```go
// app/routes/user/[id:int].go
package user

import (
  "github.com/recera/vango/pkg/server"
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  return builder.Div().Text("User "+ctx.Param("id")).Build(), nil
}

// usage (dev)
builder.A().Href(router.UserId(42)).Text("View user").Build()
```

## Layout Wrapper
```go
// app/routes/_layout.go
package routes

func Layout(children ...*vdom.VNode) *vdom.VNode {
  return builder.Div().Class("container mx-auto p-6").Children(children...).Build()
}

// child route calls Layout(...)
```

## Directory Middleware
```go
// app/routes/admin/_middleware.go
package admin

// Register middleware during router generation or in your production builder routing code
// e.g., wrap handlers with auth checks (convention-based)
```

## Client-Side Navigation Hook
```js
// In bootstrap context
window.__vango_navigate = function(path, component, params){
  // Dispatch into your WASM renderer by component name
}
```

## Server-Driven Counter (minimal)
```go
//go:build vango_server && !wasm
package routes

func Page(ctx server.Ctx) (*vdom.VNode, error) {
  count := 0
  return builder.Html().Children(
    builder.Head().Build(),
    builder.Body().Children(
      builder.Div().ID("counter-display").Text(fmt.Sprint(count)).Build(),
      builder.Button().Text("+1").OnClick(func(){ count++ }).Build(),
    ).Build(),
  ).Build(), nil
}
```

## Controlled Form with Signals
```go
name := reactive.CreateState("")
email := reactive.CreateState("")

func Form() *vdom.VNode {
  return builder.Form().OnSubmit(func(){ /* save */ }).Children(
    builder.Input().Type("text").Value(name.Get()).OnInput(func(v string){ name.Set(v) }).Build(),
    builder.Input().Type("email").Value(email.Get()).OnInput(func(v string){ email.Set(v) }).Build(),
    builder.Button().Text("Save").Build(),
  ).Build()
}
```

## Fetch Data on SSR (Universal)
```go
func Page(ctx server.Ctx) (*vdom.VNode, error) {
  users := fetchUsers() // server-side
  items := make([]*vdom.VNode, 0, len(users))
  for _, u := range users {
    items = append(items, builder.Li().Text(u.Name).Build())
  }
  return builder.Ul().Children(items...).Build(), nil
}
```

## Link Helpers
```go
// dev: use generated helpers
a := builder.A().Href(router.BlogSlug("intro")).Text("Read").Build()
```

## Dark Mode Class Strategy
- In Tailwind, configure `darkMode: 'class'`
- Toggle `document.documentElement.classList.toggle('dark')` via a button
