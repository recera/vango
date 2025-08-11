# Vango Examples

This directory contains examples demonstrating the three render modes in Vango.

## Counter Example (Pure Client-Side)

The `counter` example demonstrates pure client-side rendering (CSR) with WASM.

```bash
cd counter
./build.sh
# Open public/index.html in browser
```

**Mode**: `ModeClient`
- All state management happens in WASM
- No server required after initial load
- Best for SPAs and interactive applications

## Counter-SSR Example (Multiple Modes)

The `counter-ssr` example demonstrates all three rendering modes:

```bash
cd counter-ssr
go build -o server main.go
./server
```

Then visit:

### 1. Static SSR: http://localhost:8080/
**Mode**: `ModeSSRStatic`
- Server renders HTML
- No client-side JavaScript
- No interactivity
- Best for static content, SEO

### 2. SSR with Hydration: http://localhost:8080/with-hydration
**Mode**: `ModeSSRStatic` → `ModeClient`
- Server renders initial HTML
- Client hydrates and takes over
- Full interactivity via WASM
- Best for universal apps needing SEO + interactivity

### 3. Server-Driven: http://localhost:8080/server-driven
**Mode**: `ModeServerDriven`
- Server renders initial HTML
- Client sends events to server via WebSocket
- Server manages state and sends patches
- Minimal client-side code (3KB vs 800KB WASM)
- Best for low-powered devices, regulatory compliance

## Architecture

### Render Mode Detection

Components check `ctx.Mode` to determine behavior:

```go
switch ctx.Mode {
case vango.ModeClient:
    // Client-side state management
    count := reactive.NewState(0, ctx.Scheduler)
    
case vango.ModeServerDriven:
    // Events forwarded to server
    onClick := func() {
        vango.EmitEvent(ctx, "increment", nil)
    }
    
case vango.ModeSSRStatic:
    // No interactivity
}
```

### Live Protocol

Server-driven mode uses binary WebSocket protocol:

**Opcodes**:
- `0x01`: ReplaceText
- `0x02`: SetAttribute
- `0x03`: RemoveNode
- `0x04`: InsertNode
- `0x05`: UpdateEvents

**Event Flow**:
1. Client detects user interaction
2. Sends event to server via WebSocket
3. Server updates state
4. Server diffs VNodes
5. Server sends binary patches
6. Client applies patches to DOM

## Development

### Building WASM

```bash
cd counter
./build.sh
```

Uses TinyGo for small binary size (< 800KB).

### Running SSR Server

```bash
cd counter-ssr
go run main.go
```

Serves all three render modes on port 8080.

## Testing Render Modes

1. **Static SSR**: View source to see complete HTML, buttons don't work
2. **Hydration**: View source shows HTML, buttons work after JS loads
3. **Server-Driven**: Buttons work via WebSocket, check Network tab for WS messages

Each mode shows a "Render Mode" indicator for debugging.