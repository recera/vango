# Development Server Deep Dive

`vango dev` runs a watch/compile server with hot reload and live updates.

## CLI Flags
- `--port, -p` port (default 5173 or config)
- `--host, -H` host (default `localhost` or config)
- `--cwd` working directory
- `--no-tailwind` disable Tailwind runner

## Responsibilities
- Load `vango.json` and initialize Tailwind runner if enabled
- Initialize Live server for server-driven components
- Watch files: `.go`, `.vex`, `.css`, `.html`, `.js`
- Compile VEX → Go
- Scan routes, generate router artifacts (params, paths, table)
- Build WASM via TinyGo (`app/client/main.go` or fallback `app/main.go`)
- Serve app routes and static endpoints

## Endpoints
- `/vango/live/` — WebSocket for server-driven
- `/app.wasm`, `/wasm_exec.js`, `/vango/bootstrap.js`
- `/styles.css`, `/styles/**`
- `/router/table.json`

## Composite Routing in Dev
- API + Universal/SSR: compiled to a subprocess; requests proxied to it
- Server-Driven: served by dynamic live handler with injection
- Fallback: static file server (for CSR apps without server handlers)

## Hot Reload
- Go or VEX change → rebuild WASM, regenerate routes → notify clients
- CSS change → reload styles only

## Diagnostics
- Check logs for route scan counts and WASM size
- Visit `/router/table.json` to debug CSR matching
- Watch console logs for live WS HELLO/PING/PONG events
