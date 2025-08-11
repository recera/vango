---
title: Syntax – Vango Element Language (VEX)
slug: syntax-VEX
version: 0.2
phase: P-0
status: draft
requires:
  - ADR-0001-single-vdom
---

# VEX: Vango Element Language

> **Goal**: Provide a progressive syntax stack—from pure Go to template macros—that compiles to the same `VNode` API, giving developers choice without sacrificing type-safety.

## 1. Layers at a Glance
| Layer | Opt-in? | Example | Typical Use |
|-------|---------|---------|-------------|
| 0. Functional Go | Always | `vango.Div(nil, vango.Text("hi"))` | Library authors, code-gen tools |
| 1. Fluent Builder | `import . "vango/VEX/builder"` | `Div().Class("box").Children(Text("hi"))` | App code wanting brevity |
| 2. Template Macro | `//vango:template` | `` `<div>{{msg}}</div>` `` | Designers, HTML heavy pages |

All three compile to:
```go
return vango.Element("div", Props{}, vango.Text(msg))
```

## 2. Layer 0 – Functional API
### 2.1 Function Signatures
```go
func Div(p *Props, kids ...VNode) VNode
func Button(p *Props, kids ...VNode) VNode
```
*Props* is a nullable pointer to keep call-sites terse (`nil` = no attributes).

### 2.2 Event Helpers
```go
func OnClick(fn func()) Prop
func Href(url string) Prop
```
Under the hood they set fields on `Props` (generated code ensures stable key order).

## 3. Layer 1 – Fluent Builder
```go
import . "vango/VEX/builder"

func Card(msg string) VNode {
    return Div().Class("card").Children(
        Text(msg),
        Button().OnClick(func(){ alert("hi") }).Text("click"),
    )
}
```
### 3.1 Implementation Sketch
The builder types are `struct{ v vango.VNode }` with chainable methods returning a copy. Chain methods are code-generated from a YAML spec of HTML attributes.

## 4. Layer 2 – Template Macro
```go
//go:generate vango template
//vango:client        // optional directive
var tpl = `
<section class="hero">
  <h1>{{title}}</h1>
  <p>{{subtitle}}</p>
  <button @click="count++">Like {{count}}</button>
</section>`
```
### 4.1 Parsing Flow
1. CLI runs `vango template` → invokes PEG parser.  
2. Produces Go AST nodes under `*_tpl.go`.  
3. Auto-registers `UseWASM()` if `@click`/`@input` detected.

### 4.2 Supported Directives
| Syntax | Expands to |
|--------|------------|
| `{{expr}}` | `vango.Text(expr)` |
| `@click="code"` | `OnClick(func(){ code })` |
| `{{#if cond}} … {{/if}}` | inline ternary VNodes |

## 5. Editor & LSP Integration
VS Code plugin hooks into template parser for diagnostics, autocompletion of attributes, and “go to definition” for imported Go identifiers.

## 6. Open Questions
* Include JSX-style spread `{{...props}}`?  
* Should builder layer allow generic components `Comp[T]()`?  
* Do we want granular opt-out of escaping (`{{{rawHTML}}}`)?

## 7. YAML Builder Spec (Layer 1)
Generator reads `internal/spec/html.yml`:
```yaml
- tag: div
  attributes: [class, id, style]
- tag: button
  attributes:
    - type: enum ["button","submit","reset"]
    - disabled: bool
```
Produces in `pkg/vex/builder/elements_gen.go`:
```go
func Div() *Builder { return &Builder{tag:"div"} }
func (b *Builder) Class(v string) *Builder { b.addAttr("class", v); return b }
```

## 8. Acceptance & Validation
| Check | Command | Expected |
|-------|---------|----------|
| Template Parse | `go test ./cmd/vango/internal/template -run TestPEG` | PASS |
| Builder Gen | `go test ./cmd/vango/internal/gen -run TestBuilder` | Generated code matches golden |
| LSP Hover | `npm test --workspace tools/vscode-ext` | Hover shows prop types |

## 9. Cross-References
* Template spec: `@docs/blueprints/template-spec.md`
* HTML elements phase: `@docs/phases/P-1-html-elements.md`
* Build pipeline step 2: `@docs/blueprints/build-system.md#4-code-generation-steps`

## 10. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
|2025-08-06|0.2|Add builder spec example, acceptance table, cross refs|
