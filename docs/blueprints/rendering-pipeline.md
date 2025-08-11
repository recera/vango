---
title: Rendering Pipeline
slug: rendering-pipeline
version: 1.1
phase: P-0
status: ratified
requires:
  - ADR-0001-single-vdom
---

# Rendering Pipeline Blueprint

> **Purpose**: Define how Vango converts component functions into visual output across three modes: SSR, Hydration, and Client-Only WASM.

## 1. Overview Diagram
```mermaid
graph LR
  C[Component Render()] --> V(VNode Tree)
  subgraph Server
    V --> D(Diff Engine) --> H(HTML Stream)
    D -->|Serialize Patch| P[Patch Stream]
  end
  H -->|HTTP| Browser
  subgraph Browser
    Browser -->|Hydrate| V2(VNode Tree*)
    P -->|WS| V2
    V2 --> DOM
  end
```

## 2. VNode Contract
```go
type VNode struct {
    Kind  VKind          // Element | Text | Fragment | Portal
    Tag   string         // e.g. "div"
    Props Props          // stable map[string]any
    Kids  []VNode
    Key   string
    Flags uint8          // bitmask, see VNodeFlags
}
```
*Immutable*: once created, never mutate; diff relies on referential equality.

## 3. Diff Algorithm
Algorithm: keyed-depth-first with O(n) complexity.
* Stable keys speed up list reordering.
* Text nodes diff via byte-compare.
* Outputs opcode stream (see Live Protocol spec).

## 4. Render Modes
### 4.1 SSR Stream
* `htmlApplier.Apply(nil, root)` produces incremental HTML chunks.
* Flushes after closing `</head>` or explicit `goliath.Flush()`.

### 4.2 Hydration
* Server injects `data-hid` IDs on eventful nodes.
* Browser boot retrieves root DOM node, builds sparse previous-tree, then `domApplier.Apply(prev, next)`; no DOM creation expected.

### 4.3 Client-Only WASM
* Component flagged `//vango:client` bypasses SSR; build system injects `//go:build vango_client` into the file.
* Server outputs a placeholder `<div data-vgci="123"></div>` (vgci = Vango Client Instance).
* After TinyGo-compiled WASM boots, `domApplier` reconciles the subtree and attaches any event listeners.

### 4.4 Server-Only Live Patch Stream
* Component flagged `//vango:server` *or* route option `vango.ServerOnly()`.
* Server streams full HTML, keeps the component running, and on each reactive change diff-produces patches that are pushed over the Live WebSocket (`/vango/live/:sessID`).
* Browser executes only `bootstrap.js` (≈ 3 kB) containing:
  1. WebSocket client with exponential back-off.
  2. Tiny `applyPatch()` loop calling `domApplier` compiled to JS via `syscall/js` shims.
* Ideal for read-mostly pages, low-end devices, or code-secrecy contexts.

### 4.5 Mode Comparison
| Mode | HTML First Byte | Payload Size | Interactivity Source | Best For |
|------|-----------------|--------------|----------------------|----------|
| Universal (default) | Streaming SSR | WASM ≤ 800 kB | In-browser scheduler | Apps, forms, dashboards |
| Server-only | Streaming SSR | 3 kB JS | Server diff → patches (RTT) | Content, admin lists, regulated apps |
| Client-only | Minimal HTML | WASM ≤ 800 kB | In-browser scheduler | Heavy canvas/WebGL, embeds |

### 4.6 Build-Tag Injection
During `vango build` the pragma scanner rewrites source files:
```go
//vango:server → //go:build vango_server
//vango:client → //go:build vango_client
```
The CLI then performs two compilations:
1. `go build -tags vango_server` → server binary.
2. `tinygo build -tags vango_client -o app.wasm` → WASM bundle.

Developers can flip a component between modes by editing (or deleting) the pragma—no other code changes required.

## 5. Performance Targets
| Metric | Budget |
|--------|--------|
| SSR first byte | < 50 ms |
| WASM binary (gzip) | ≤ 800 kB |
| Hydration time (1k nodes) | < 30 ms |
| Patch latency (server→client) | < 50 ms P95 |

## 6. Extensibility Hooks
* **Partial Hydration** – future flag `data-vango-static` skips nodes.
* **Edge Adapter** – replace `http.ResponseWriter` with Edge `Context` without touching diff logic.

## 7. Open Questions
* Should static sub-trees be pre-rendered into string constants in Go code?  
* Do we expose a public `Compare(prev, next)` for userland snapshot testing?

## 8. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
