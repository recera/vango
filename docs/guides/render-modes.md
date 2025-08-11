---
title: Choosing Render Modes & Live Protocol Deep-Dive
description: Practical guide to SSR, Hydration, Server-Only streaming, and Client-Only WASM in Vango.
slug: render-modes
version: 1.0
status: draft
requires:
  - blueprints/rendering-pipeline
  - blueprints/live-protocol
---

# Guide: Choosing the Right Render Strategy

> **Audience** – Application developers deciding which Vango render mode to use and framework contributors who need an implementation reference.

---

## 1. Big-Picture Recap

```
Component() ─▶ VNode ─▶ Diff ─▶ Patch ─▶ { htmlApplier | domApplier }
```
* **VNode** – immutable Go struct, shared across server & client.
* **htmlApplier** – streams HTML on the server.
* **domApplier** – applies binary patches either **in WASM** or **in a tiny JS runtime**.
* **Live Protocol** – WebSocket transport for patches when server stays authoritative.

---

## 2. The Three Modes

| Mode (pragma) | First Paint | After First Paint | Typical Use-Cases |
|---------------|------------|-------------------|-------------------|
| Universal _(default)_ | Streaming SSR | Hydrate into WASM; local scheduler handles further UI | SPAs, dashboards, forms |
| Server-Only `//vango:server` | Streaming SSR | Server stays authoritative; patches streamed over Live WS | Marketing pages, low-power devices, compliance contexts |
| Client-Only `//vango:client` | Placeholder HTML | Full run in WASM only | Heavy canvas/WebGL, 3rd-party embeds |

### 2.1 Code Snippets

```go
//vango:server
package pages
// 👉 Render only on server, never compiled to WASM.
```

```go
//vango:client
package widget
// 👉 Skips SSR. Client mounts onto placeholder.
```

Universal mode… needs **no pragma**.

---

## 3. Live Protocol in Practice

### 3.1 Frame Anatomy
```
[frameLen uvarint] [opcode 0x01] [nodeID varint] [textLen varint] [text bytes] …
```
See `blueprints/live-protocol.md` for the full opcode table.

### 3.2 Minimal Browser Stub (≈ 3 kB)
```js
import { applyPatch } from "@vango/runtime-dom";
const ws = new WebSocket(`/vango/live/${session}`);
ws.binaryType = "arraybuffer";
ws.onmessage = ev => {
  const buf = new Uint8Array(ev.data);
  applyPatch(buf);
};
```

### 3.3 Server Integration
```go
handler := live.NewHandler()            // pkg/live
router.Handle("/vango/live/{sess}", handler)

ticker := time.NewTicker(16 * time.Millisecond)
for range ticker.C {
    dirty := scheduler.Flush()          // returns []*Patch
    handler.Broadcast(dirty)
}
```

---

## 4. Build Pipeline Mechanics

1. **Pragma Scan** – CLI finds `//vango:(server|client)` and injects build tags.
2. **Dual Compile**
   * `go build -tags vango_server`  → server binary.
   * `tinygo build -tags vango_client -o app.wasm` → WASM.
3. **Asset Manifest** – `bootstrap.js`, `app.wasm`, and hashed CSS emitted to `dist/`.
4. **HTML Injection** – SSR inserts the correct `<script src="/bootstrap.<hash>.js">` automatically.

---

## 5. Choosing a Mode – Decision Tree

```
Start      ─── Is UI interactive after paint? ── yes ── Is WASM allowed? ── yes ─▶ Universal
    │                               │                               │
    │                               │                               └───▶ Server-Only
    │                               └── no ──▶ Server-Only
    └── no ─▶ Server-Only
```

## 6. FAQ
1. **Can I mix modes on the same route?** Yes – each component file can pick its own pragma.
2. **What about SEO?** Universal & Server-Only are crawlable; Client-Only isn’t.
3. **How big is the stub JS?** ≈ 3 kB gzipped, includes reconnect and patch loop.
4. **How do I debug patches?** Enable Dev build → `window.__VangoLiveTap` logs opcodes.

---

## 7. Implementation Check-List (Framework Contributors)
- [ ] `pkg/live/codec.go` – varint encoder/decoder with zero alloc.
- [ ] `pkg/live/client.js` – ES module shipped in bootstrap.
- [ ] `pkg/renderer/dom` – ensure `Apply()` can run in pure JS via `syscall/js` shims.
- [ ] `cmd/vango/internal/pragma.go` – robust pragma scanner.

---

## 8. Deep-Dive — Answering “server vs client authority”

> **Hey devs 👋 –** below is a verbatim, expanded answer to the question raised in the working session (2025-08-05) about how a component discovers whether it should run logic locally or remain server-driven.

### 8.1 What currently happens
1. `vango.Context` exposes `Scheduler *scheduler.Scheduler`.
2. Components assume:
   * `Scheduler == nil` → static SSR (no interactivity).
   * `Scheduler != nil` → interactive & owns state locally.
3. That works for **pure SSR** and **pure client** routes, but *not* for **server-driven hydration** because both ends have a scheduler.

### 8.2 Why we need an explicit render-mode signal
* Compliance & secrets – server must stay authoritative.
* Thin devices – prefer 3 kB stub JS instead of 800 kB WASM.
* Multi-user consistency – single source of truth.

### 8.3 Proposed refinement (`RenderMode` enum)
```go
// pkg/vango/context.go

 type RenderMode uint8
 const (
     ModeSSRStatic RenderMode = iota // 0 – server static pass
     ModeClient                      // 1 – WASM owns state
     ModeServerDriven               // 2 – server authoritative
 )
 type Context struct {
     Scheduler *scheduler.Scheduler // nil in ModeSSRStatic
     Mode      RenderMode
 }
```
Population rules are automated by the pragma scanner:
| Pragma | Server HTML pass | Browser bootstrap | Context.Mode |
|--------|------------------|-------------------|--------------|
| (none) | `ModeSSRStatic`  | WASM → `ModeClient` | Universal |
| `//vango:client` | placeholder | WASM → `ModeClient` | Client-only |
| `//vango:server` | `ModeServerDriven` | stub JS (no WASM) → `ModeServerDriven` | Server-only |

### 8.4 Component pattern
```go
switch ctx.Mode {
case vango.ModeClient:
    count := reactive.NewState(0, ctx.Scheduler)
    onClick := func() { count.Set(count.Get()+1) }
case vango.ModeServerDriven:
    onClick := func() { vango.EmitEvent(ctx, "increment") }
}
```
Server listens, updates state, diffs, streams patches.

### 8.5 Implementation tasks (added to Phase 0 backlog)
* Define `RenderMode` enum & update `Context`  — `pkg/vango/context.go`.
* Enhance pragma scanner & bootstrap JSON — `cmd/vango/internal/pragma.go`, `internal/assets/bootstrap.js`.
* Add `vango.EmitEvent` helper & server dispatcher — `pkg/live/events.go`.
* Update examples (`counter-ssr`) to demonstrate `ModeServerDriven`.

Feel free to ping @Cascade (AI) if anything here is unclear or if edge-cases pop up during implementation!

---

## 9. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|1.0|Initial draft.

