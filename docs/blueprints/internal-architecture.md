---
title: Internal Package Architecture
slug: internal-architecture
version: 0.1
status: draft
---

# Internal Dependency Rules

> **Goal**: Prevent circular imports and provide clear ownership for subsystems.

## 1. Package Layers
```
cmd/            ← CLI entrypoints (imports below)
pkg/server/     ← HTTP handling, router, ctx (imports vdom, reactive)
pkg/router/     ← generated matcher (imports nothing)
pkg/vdom/       ← VNode structs, diff algorithm (no external deps)
pkg/reactive/   ← State[T], scheduler (imports vdom for invalidation)
pkg/dom/        ← WASM DOM applier (imports vdom)
pkg/live/       ← WebSocket encoder/decoder (imports vdom)
pkg/vex/...     ← Syntax helpers (builder, template runtime)
internal/...    ← code-gen tools, schema, templates
```

### Allowed Import Matrix
| From \ To | server | router | vdom | reactive | dom | live | vex |
|-----------|--------|--------|------|----------|-----|------|-----|
| cmd       | ✅     | ✅     | ✅   | ✅       | ✅  | ✅   | ✅  |
| server    | —      | ✅     | ✅   | ✅       |     | ✅   |     |
| router    | —      | —      |      |          |     |      |     |
| vdom      | —      | —      | —    |          |     |      |     |
| reactive  | —      | —      | ✅   | —        |     |      |     |
| dom       | —      | —      | ✅   | ✅       | —   |      |     |
| live      | —      | —      | ✅   |          |     | —    |     |
| vex       | —      | —      | ✅   | ✅       |     |      | —   |

Rule of thumb: higher rows may import lower columns, never the reverse.

## 2. Public API Surface
Only `pkg/vango` aggregates stable shorthand re-exports for DX. Internal packages remain individually importable for advanced use and avoid cycles.

## 3. Versioning Policy
Breaking moves across layers require a major version bump pre-1.0, enforced via `gomodguard` linter.

## 4. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
