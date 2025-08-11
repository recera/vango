# VEX, Builder, and Functional Syntax (Complete Guide)

Vango offers three complementary authoring layers for UI:
- Layer 1 (Primary): Fluent Builder API in Go
- Layer 0: Functional API in Go (low-level, ideal for codegen)
- Layer 2: VEX Templates (HTML-like syntax compiled to Go)

Use Layer 1 as your day-to-day authoring style. Use Layer 2 when you want HTML ergonomics and control-flow macros; use Layer 0 for generated or very terse code.

---

## Layer 1 — Fluent Builder API (Primary)
Package: `github.com/recera/vango/pkg/vex/builder`

Builder creates a `*vdom.VNode` via chainable methods. You pick an element constructor, set attributes/events, add children, then `.Build()` the node.

### Basics
```go
import (
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/builder"
)

func Hello(name string) *vdom.VNode {
  return builder.Div().
    Class("p-6 rounded bg-white shadow").
    Children(
      builder.H1().Class("text-xl font-bold").Text("Hello, "+name).Build(),
      builder.P().Text("Welcome to Vango.").Build(),
    ).
    Build()
}
```

- Constructors: `Div()`, `H1()`, `A()`, `Button()`, `Input()`, … (see `builder_gen.go` for the full set)
- Attributes: common helpers like `.Class`, `.ID`, `.Style`, `.Href`, `.Target`, `.Disabled`, `.Name`, `.Value`, etc. (see `attributes.go`)
- Events: `.OnClick(func())`, `.OnInput(func(string))`, `.OnSubmit(func())`, `.OnChange(func(string))`, and more
- Children: `.Children(child1, child2, …)` or `.Text(str)`

You can chain in any order before `.Build()`:
```go
btn := builder.Button().
  Class("px-3 py-2 bg-blue-600 text-white rounded").
  OnClick(onClick).
  Title("Click me").
  Build()
```

### Composition and Components
Builder returns `*vdom.VNode`, so you can nest and reuse easily:
```go
func Card(title, body string) *vdom.VNode {
  return builder.Div().
    Class("rounded border p-4").
    Children(
      builder.H2().Class("font-bold mb-2").Text(title).Build(),
      builder.P().Text(body).Build(),
    ).Build()
}

func Page() *vdom.VNode {
  return builder.Div().
    Class("space-y-4").
    Children(
      Card("Welcome", "This is Vango."),
      Card("Next Steps", "Check the developer guide."),
    ).Build()
}
```

### Lists and Keys
Use stable keys when rendering collections to avoid DOM churn.
```go
func TodoList(items []string) *vdom.VNode {
  children := make([]*vdom.VNode, 0, len(items))
  for i, text := range items {
    children = append(children,
      builder.Li().
        // Use IDs or stable identifiers when available
        Attr("data-key", fmt.Sprintf("todo-%d", i)).
        Text(text).
        Build(),
    )
  }
  return builder.Ul().Children(children...).Build()
}
```

### Forms and Controlled Inputs
- `.Value(v string)` on inputs updates the `value` attribute.
- For controlled inputs, handle `OnInput(func(string))` and keep state in a signal.
```go
func NameForm(value string, setValue func(string)) *vdom.VNode {
  return builder.Form().
    OnSubmit(func(){ /* handle submit */ }).
    Children(
      builder.Input().Type("text").
        Class("border rounded p-2").
        Value(value).
        OnInput(func(v string){ setValue(v) }).
        Build(),
      builder.Button().Class("ml-2").Text("Save").Build(),
    ).Build()
}
```

### Navigation Links
Use generated helpers from `router/paths.go` in dev for type-safe hrefs.
```go
link := builder.A().Href(router.BlogSlug("my-post")).Text("Open").Build()
```

### Events Across Modes
- Universal/CSR: WASM hydrates and runs event handlers client-side
- Server‑Driven: handlers are wired by the minimal client; `OnClick` maps to an event forwarded over WS

---

## Layer 0 — Functional API
Package: `github.com/recera/vango/pkg/vex/functional`

A terse, functional constructor interface. Great for generated code and places where you want compact expressions.

```go
import (
  vdom "github.com/recera/vango/pkg/vango/vdom"
  "github.com/recera/vango/pkg/vex/functional"
)

func Banner(msg string) *vdom.VNode {
  return functional.Div(
    vdom.Props{"class": "p-4 bg-yellow-100 border-l-4 border-yellow-500"},
    functional.H1(nil, functional.Text(msg)),
  )
}
```

- Children are positional variadic args
- Props are `vdom.Props` map; use helpers like `functional.OnClick(fn)` or `functional.MergeProps(...)`

---

## Layer 2 — VEX Templates
Files annotated with `//vango:template` compile to Go. They support HTML-like syntax plus control flow macros.

### Minimal Template
```html
//vango:template
package routes

<div class="prose mx-auto p-8">
  <h1>Hello</h1>
  <p>Welcome to Vango.</p>
</div>
```
Compiles into a `Page(ctx server.Ctx) (*vdom.VNode, error)` that returns the above structure using the Builder API.

### Props
Declare a `PageProps` struct via pragma, then call `Page(ctx, PageProps{...})`.
```html
//vango:template
package routes
//vango:props { Title string; Items []string }

<div>
  <h1>{{.Title}}</h1>
  {{#if len(.Items) > 0}}
    <ul>
      {{#for item in .Items}}
        <li>{{item}}</li>
      {{/for}}
    </ul>
  {{else}}
    <p>No items</p>
  {{/if}}
</div>
```
- `{{.Field}}` becomes `props.Field`
- In conditions/loops, `.Field` also maps to `props.Field`

### Control Flow
- If/ElseIf/Else:
```html
{{#if cond}}
  <div>True</div>
{{#elseif other}}
  <div>Other</div>
{{else}}
  <div>False</div>
{{/if}}
```
- For loops:
```html
{{#for user in .Users}}
  <p>{{user.Name}}</p>
{{/for}}
```

### Attributes and Events
- Normal HTML attributes map to builder methods when known (`class`, `id`, `href`, `src`, `type`), else to `.Attr("key", "val")`
- Events use `@event="goCode"` and compile into appropriate builder `.OnClick(...)` etc.
```html
<button class="btn" @click="doSomething()">Click</button>
```

### Components vs Elements
- Lowercase tags are HTML; Capitalized tags are treated as Go component calls with `Props` and children
```html
<MyWidget title="Hi">
  <span>Child</span>
</MyWidget>
```
(Your project must define `func MyWidget(...) *vdom.VNode` or similar; VEX will emit a call.)

### Self-Closing and Nesting
```html
<img src="/logo.png" alt="logo" />
<input type="text" />
```

### Generated Function Shape
- Single simple node → `return node, nil`
- Complex/conditional → builds arrays and appends children, then returns a wrapper element

### Mode Pragmas in Templates
You can combine with build tags in adjacent Go files, but templates themselves focus on structure. Route mode (universal/server/client) is typically driven by file build tags/pragma in the surrounding Go route file, not in the template body.

---

## Choosing a Layer
- Use Builder (Layer 1) for most app code (strong typing, readable Go, great IDE support)
- Use VEX (Layer 2) when authoring large static structures with conditionals/loops feels more natural in HTML
- Use Functional (Layer 0) for compact or generated code paths

## Best Practices
- Keep components small and reuse via composition
- Prefer `Class` utilities (e.g., Tailwind) to minimize patch sizes
- Always provide stable keys for long lists
- Separate state from render: pass signals/values down, avoid side effects in render functions
