---
title: Client Bootstrap and Client-Side Routing (CSR)
slug: client-bootstrap
status: stable
---

### Overview

The embedded bootstrap (`internal/assets/bootstrap.js`) initializes the WASM runtime, establishes the Live WebSocket, loads the client route table, and implements client-side navigation by intercepting link clicks.

### Responsibilities

- Load `wasm_exec.js` and instantiate `app.wasm`
- Connect to `/vango/live/<session>` for server-driven updates
- Fetch `/router/table.json` and build client matchers
- Intercept `<a>` clicks for CSR
- Provide prefetch hooks and HMR in dev

### WASM Loading

```19:50:internal/assets/bootstrap.js
async function loadWasm() {
  await loadWasmExec()
  const go = new Go()
  const wasmBuffer = await (await fetch('/app.wasm')).arrayBuffer()
  const result = await WebAssembly.instantiate(wasmBuffer, go.importObject)
  go.run(result.instance)
}
```

### Live WebSocket

```88:123:internal/assets/bootstrap.js
const wsUrl = `${protocol}//${window.location.host}/vango/live/${sessionId}`
wsConnection = new WebSocket(wsUrl)
wsConnection.binaryType = 'arraybuffer'
```

Binary patch frames and control frames are handled to update the DOM and maintain resumable sessions.

### Router Table

```302:315:internal/assets/bootstrap.js
async function loadRouterTable() {
  const response = await fetch('/router/table.json')
  routerTable = await response.json()
}
```

Matching builds a regex per route entry respecting bracket syntax:

```350:387:internal/assets/bootstrap.js
// routeToRegex converts /blog/[slug] and [id:int]/[...rest] to regex with typed segments
```

### Navigation

```389:422:internal/assets/bootstrap.js
async function navigate(path, options = {}) {
  const route = matchRoute(path)
  if (!route) { window.location.href = path; return }
  if (!options.replace) history.pushState({ path }, '', path)
  if (window.__vango_navigate) {
    window.__vango_navigate(path, route.component, route.params || {})
  } else {
    // Fallback to full reload if WASM app doesn't expose a handler
    window.location.href = path
  }
}
```

Interception is global; apps can cooperate by defining `window.__vango_navigate` (and optionally `__vango_prefetch`) to take control of rendering on CSR transitions.

### Dev Hooks

In dev, the bootstrap also connects to the dev server’s HMR channel and reloads WASM or stylesheets on change events.

### Recommendations for App Authors

- Expose `window.__vango_navigate(path, component, params)` in the WASM client to handle CSR without reloads
- Use the generated `router/paths.go` for type-safe links
- For dynamic routes, prefer file-based `[slug]` structure so the router table includes correct patterns


