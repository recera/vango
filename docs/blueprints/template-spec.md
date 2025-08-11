---
title: VEX Template Macro Specification
slug: template-spec
version: 0.1
status: draft
requires:
  - syntax-gel
---

# `//vango:template` Macro – Full Code-gen Contract

> **Audience**: Engineers implementing the parser + generator.  
> **Scope**: Exact AST output for every directive.

## 1. File Header Requirements
* The magic comment **must** be at top-level (**not** inside a function):
  ```go
  //vango:template
  package routes
  ```
* The rest of the file is treated as raw template string **until EOF**.

## 2. Grammar (EBNF)
```
Template   = ( Text | Mustache )* .
Mustache   = "{{" ( If | ElseIf | Else | EndIf | For | RawExpr | Ident ) "}}" .
If         = "#if" WS Expr WS "}}" Template ( ElseIf | Else )* "{{/if}}" .
ElseIf     = "#elseif" WS Expr .
Else       = "#else" .
For        = "#for" WS Ident "in" WS Expr WS "}}" Template "{{/for}}" .
RawExpr    = "=" Expr .
Ident      = identifier .
Text       = any-char-except-"{{" .
```
`Expr` reuses Go’s expression grammar (parsed with `go/parser`).

## 2.5 Props Declaration
Developers declare input props via **single-line** directive placed before template body:
```go
//vango:props { Name string; Age int }
```
The schema inside braces follows Go struct literal fields. Generator parses and emits:
```go
type PageProps struct {
    Name string
    Age  int
}
func Page(ctx vango.Ctx, props PageProps) vango.VNode { ... }
```
Within template, the fields are accessible via `{{.Name}}`. If directive omitted, `struct{}` is used.

## 3. Code Generation Output
Each template file compiles to **one** exported Go function named `Page` with signature:
```go
func Page(ctx vango.Ctx, props struct{}) vango.VNode
```
The generator constructs a VNode tree by translating tokens to calls on the functional layer (VEX Layer 0). It appends the following imports automatically:
```go
import (
    "vango"
    "vango/gel/functional"
)
```

### 3.1 Mapping Table
| Template Construct | Generated Go | Notes |
|--------------------|--------------|-------|
| `<div>` | `functional.Element("div", nil` | attributes filled later |
| `<div class="x">` | `functional.Element("div", &vango.Props{Class:"x"}` | attributes struct literal deduped |
| `{{= user.Name }}` | `functional.Text(user.Name)` | any Go expr allowed |
| `{{#if cond}}A{{/if}}` | `if cond { children = append(children, functional.Text("A")) }` | `else` appends alternative children |
| `{{#for item in list}}` | `for _, item := range list { ... }` | `item` is a new identifier with correct type from `list` element |
| `@click="inc()"` | adds `vango.OnClick(inc)` to props | events mapped via helper table |
| `<MyCard />` | resolves import path via `goimports`; call `MyCard()` |

### 3.2 Event Mapping
| Template Event | Prop Function |
|----------------|--------------|
| `@click` | `vango.OnClick(fn)` |
| `@input` | `vango.OnInput(fn)` |
| `@submit` | `vango.OnSubmit(fn)` |
(_Add more in `event-map.json` used by generator_)

### 3.3 Slot / Children Handling
Self-closing custom component with `<MyCard>{{>slot}}</MyCard>` becomes:
```go
MyCard(MyCardProps{Child: childVNode})
```

## 4. Type Resolution & Imports
* Identifiers first resolve to local file scope, then to imported packages.  
* Component symbols detected by camel-case and presence of `func(...) vango.VNode` in the same package or imports.

## 5. Error Reporting
Generator emits **file-positioned** errors using `go/token.FileSet` so IDEs can underline problems.

## 6. Idempotence
Running code-gen twice **must not** change output if template unchanged – verified in CI via `git diff --exit-code`.

## 7. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
