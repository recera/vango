# Client Bootstrap and CSR Deep Dive

The bootstrap (`internal/assets/bootstrap.js`) initializes WASM, Live WS, CSR routing, and HMR in dev.

## Init Flow
1) Load `wasm_exec.js`
2) Fetch and instantiate `/app.wasm`
3) Load `router/table.json`
4) Hydrate existing SSR DOM using `data-hid`
5) Connect to `/vango/live/<session>`
6) Intercept links and install popstate handler

## CSR Navigation
- `navigate(path, {replace?: bool})` attempts a client-side route; falls back to full reload on miss
- Apps can define `window.__vango_navigate = (path, component, params) => { … }`

### Example: Minimal CSR handler in WASM
```js
// In your WASM-exposed bindings
window.__vango_navigate = function(path, component, params){
  // component is a symbol/name from the route table; use it to dispatch or render
  // e.g., call into a Go-exported function to render a component by name
}
```

## Prefetch
- `window.__vango_prefetch(path)` may be invoked on link hover
- Implement prefetch to warm caches or request data prior to navigate

## Route Matching
- `routeToRegex` converts bracket params to typed regex, extracts params
- Dynamic and catch-all segments are supported

## Dev HMR
- Receives `RELOAD` for `wasm` or `css`; reloads accordingly

## Progressive Enhancement
- If CSR matching fails, users still get full-page navigation to server routes
- External links and special cases (`target`, modified clicks, hash links) are ignored by the interceptor
