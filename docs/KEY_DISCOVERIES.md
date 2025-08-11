# Key Discoveries and Gotchas - Vango Development

## Critical Bug Fixes That Took Hours to Find

### 1. The Frame Type Bug (CRITICAL)
**Symptom**: Server-driven counter events were sent but UI didn't update
**Root Cause**: Client was checking wrong frame type constant
```javascript
// WRONG - This was the bug that took hours to find!
if (frameType === 0x02) { // This is FrameControl, not patches!

// CORRECT
if (frameType === 0x00) { // FramePatches
```
**Location**: `pkg/server/server_driven_helper.go:107`
**Lesson**: Binary protocol constants must be triple-checked

### 2. Component Instance Not in Context
**Symptom**: "Initializing server component..." never went away
**Root Cause**: Component wasn't being set in the context
```go
// WRONG
vnode := renderServerCounter(vctx)

// CORRECT  
vctx.Set("component", component)  // Must set BEFORE rendering!
vnode := renderServerCounter(vctx)
```
**Lesson**: Context must be fully populated before render

### 3. Double Counter Increment
**Symptom**: Counter jumped by 2 each click
**Root Cause**: Both WASM and server-driven handlers were firing
```go
// Solution: Check render mode and only initialize appropriate handlers
if ctx.IsServerDriven() {
    // Server handlers only
} else {
    // Client handlers only  
}
```

## Architecture Insights

### 1. Rendering Modes Are Per-Component, Not Per-Project
**Wrong Thinking**: "User chooses rendering mode for whole app"
**Right Thinking**: "Each component can use different mode via pragma"
```go
// Same project can have all three:
func HomePage() { }           // Universal (default)
//vango:server
func Dashboard() { }           // Server-driven
//vango:client  
func Game() { }                // Client-only
```

### 2. The Missing Integration Layer
Phase 0 built the pieces but not the glue:
- ✅ VDOM exists
- ✅ Scheduler exists  
- ✅ Live Protocol exists
- ❌ But they weren't connected!

We had to create:
- `ComponentInstance` - Server-side state
- `SchedulerBridge` - Connect scheduler to WebSocket
- Event routing from client → component → scheduler → patches → client

### 3. Binary Protocol vs JSON
**Discovery**: Binary protocol is 5-10x smaller than JSON
```go
// JSON: {"type":"patch","nodeId":1,"value":"42"} = 43 bytes
// Binary: [0x00, 0x01, 0x01, 0x02, '4', '2'] = 6 bytes
```
**Trade-off**: Much harder to debug - need hex dumps

## Styling System Realities

### 1. Tailwind Without Integration is Useless
**Problem**: Config exists but doesn't run automatically
**Impact**: Developers get frustrated and leave
**Solution**: Must be zero-config - just work with `vango dev`

### 2. CSS-in-Go vs Tailwind is False Choice
**Reality**: Developers want both
- Tailwind for rapid prototyping
- CSS-in-Go for complex/dynamic styles
**Solution**: Support both, make Tailwind primary

### 3. Dark Mode Must Be Automatic
**Problem**: Manually adding `.dark` variants everywhere
**Solution**: Component library with dark mode built-in
```go
// Bad: Developer handles dark mode
Class("bg-white dark:bg-gray-800")

// Good: Component handles it internally
ui.Card() // Automatically has dark variant
```

## Developer Experience Lessons

### 1. First Run Must Be Perfect
**Reality**: You have 30 seconds to impress
**Requirements**:
- No manual steps
- No errors
- Working examples
- Clear next steps

### 2. Errors Are Teaching Moments
**Bad Error**: "Style not found"
**Good Error**: "Style 'card' not found. Did you mean to import 'cardStyles' from './styles.go'?"

### 3. Examples > Documentation
**Nobody reads docs first**
**Everyone copies examples**
**Examples must be**:
- Complete
- Working
- Well-commented
- Show best practices

## Technical Gotchas

### 1. TinyGo Limitations
- No `reflect` package
- Limited goroutine support
- No `init()` functions in WASM
- Memory constraints (use pooling)

### 2. WASM Specific Issues
```go
// This works on server, fails in WASM
go func() { 
    // TinyGo can't create goroutines in WASM
}

// Use scheduler instead
scheduler.Queue(func() {
    // This works
})
```

### 3. Builder Pattern Memory
```go
// Bad: Creates intermediate objects
builder.Div().Class("a").Class("b").Build()

// Good: Accumulate then build once
div := builder.Div()
for _, class := range classes {
    div.Class(class)
}
div.Build()
```

## File System Gotchas

### 1. Path Separators
```go
// Bad: Breaks on Windows
path := "app/routes/" + name + ".go"

// Good: Platform agnostic
path := filepath.Join("app", "routes", name+".go")
```

### 2. File Watching Limits
```bash
# macOS default is too low
ulimit -n 2048  # Increase file descriptor limit
```

### 3. Hot Reload Race Conditions
```go
// Must debounce file changes
var lastBuild time.Time
if time.Since(lastBuild) < 100*time.Millisecond {
    return // Skip, too soon
}
```

## WebSocket Protocol Gotchas

### 1. Session Timing
**Problem**: Component created before WebSocket connects
**Solution**: Create component on first render, connect when WS opens
```go
// On page render
component := CreateComponent()
registry.Store(component)

// On WebSocket connect  
component := registry.Get(sessionID)
bridge.Connect(component)
```

### 2. Message Ordering
**Problem**: Patches arrive out of order
**Solution**: Sequence numbers in protocol
```go
type Patch struct {
    Seq uint64  // Monotonic sequence
    Ops []Op
}
```

### 3. Reconnection Handling
**Problem**: Client disconnects, state is lost
**Solution**: Server maintains state, client can resume
```go
// Client sends last seen sequence on reconnect
{type: "HELLO", lastSeq: 42}
// Server replays missed patches
```

## Platform-Specific Issues

### macOS
- File watching requires increased ulimits
- Tailwind via Homebrew has different paths
- Port 5173 might be taken by Vite

### Linux
- Requires manual TinyGo installation
- Different path for wasm_exec.js
- Systemd service needs special config

### Windows
- Path separators break everything
- NPM/NPX commands need `.cmd` extension
- File watching is unreliable

## What We Learned About Frameworks

### 1. Vango Shouldn't Compete on React's Terms
**Don't**: Try to be a better React
**Do**: Be the best Go web framework
**Unique Value**: Server-driven mode for real-time apps

### 2. The JavaScript Ecosystem is Required
**Reality**: Can't avoid JS tooling completely
**Approach**: Integrate the best parts (Tailwind, bundlers)
**Hide Complexity**: Make it zero-config

### 3. Developer Experience > Features
**Better to have**:
- 10 features that work perfectly
**Than**:
- 100 features that sorta work

### 4. Opinionated Defaults Win
**Don't**: Ask developers to choose everything
**Do**: Make smart defaults they can override
**Example**: Tailwind by default, CSS-in-Go as escape hatch

## The Most Important Lesson

**The CLI setup is everything.** If `vango create my-app` doesn't produce a working, impressive app in under a minute with zero manual steps, developers will never give Vango a second chance. This is why the interactive CLI setup must be the #1 priority after fixing the foundations.