# Vango Troubleshooting Guide

## Common Issues and Solutions

### Build Issues

#### Problem: `tinygo: command not found`

**Solution:**
Install TinyGo from https://tinygo.org/getting-started/install/

```bash
# macOS
brew install tinygo

# Linux
wget https://github.com/tinygo-org/tinygo/releases/download/v0.31.0/tinygo_0.31.0_amd64.deb
sudo dpkg -i tinygo_0.31.0_amd64.deb

# Verify installation
tinygo version
```

#### Problem: `wasm_exec.js not found`

**Solution:**
Copy the wasm_exec.js file from your TinyGo installation:

```bash
cp $(tinygo env TINYGOROOT)/targets/wasm_exec.js public/static/
```

#### Problem: Build fails with `out of memory`

**Solution:**
Increase memory limit for TinyGo:

```bash
# Set higher stack size
export GOGC=off
tinygo build -scheduler=none -gc=leaking -o app.wasm .
```

### Runtime Issues

#### Problem: Counter doesn't update (server-driven mode)

**Symptoms:**
- Click events are sent
- Server logs show "Incremented to X"
- UI doesn't update

**Common Causes and Solutions:**

1. **Frame Type Mismatch**
   ```javascript
   // Wrong: checking for wrong frame type
   if (frameType === 0x02) { // This is FrameControl!
   
   // Correct: patches are 0x00
   if (frameType === 0x00) { // FramePatches
   ```

2. **Session ID Mismatch**
   - Check server logs for session IDs
   - Ensure WebSocket connects with same session
   - Look for "Session not found" errors

3. **Component Not Connected to Scheduler**
   ```go
   // Component must be connected when WebSocket connects
   [SchedulerBridge] Connecting existing component...
   ```

4. **Handler Registration Issues**
   ```go
   // Node IDs must match between server and client
   component.RegisterHandler(1, decrementHandler)
   // Client sends: nodeId: 1
   ```

#### Problem: WebSocket connection fails

**Symptoms:**
- Console shows "WebSocket connection failed"
- No real-time updates

**Solutions:**

1. **Check WebSocket endpoint:**
   ```javascript
   // Ensure correct protocol and path
   const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
   const ws = new WebSocket(protocol + '//' + window.location.host + '/vango/live/' + sessionID);
   ```

2. **Verify server handling:**
   ```go
   // Server must handle WebSocket upgrade
   mux.HandleFunc("/vango/live/", liveServer.HandleWebSocket)
   ```

3. **Check for proxy issues:**
   - Nginx/Apache must be configured for WebSocket
   - Add upgrade headers:
   ```nginx
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection "upgrade";
   ```

#### Problem: "Initializing server component..." never goes away

**Cause:** Component instance not found in context

**Solution:**
```go
// Ensure component is set in context before rendering
vctx.Set("component", component)
vnode := renderServerCounter(vctx)
```

#### Problem: Multiple increments per click

**Symptoms:**
- Counter jumps by 2 or more
- Multiple event handlers registered

**Solutions:**

1. **Check for duplicate initialization:**
   ```javascript
   // Add guard to prevent double init
   if (window.vangoInitialized) return;
   window.vangoInitialized = true;
   ```

2. **Verify single event listener:**
   ```javascript
   // Remove old listener before adding
   button.removeEventListener('click', handler);
   button.addEventListener('click', handler);
   ```

### Development Issues

#### Problem: Hot reload not working

**Solutions:**

1. **Check file watcher:**
   ```bash
   # Increase file descriptor limit
   ulimit -n 2048
   ```

2. **Clear build cache:**
   ```bash
   rm -rf ~/.cache/vango
   go clean -cache
   ```

3. **Restart dev server:**
   ```bash
   killall vango
   vango dev
   ```

#### Problem: Styles not applying

**Common Causes:**

1. **Scoped styles not registered:**
   ```go
   // Styles must be registered during build
   var styles = styling.New(`...`)
   // Use styles.Class("name") not just "name"
   ```

2. **Tailwind not compiling:**
   ```bash
   # Check for tailwind.config.js
   # Ensure npx is available
   which npx
   ```

3. **CSS not injected:**
   ```go
   // Check StyleRegistry is collecting styles
   styling.GetAllCSS() // Should return collected CSS
   ```

### Performance Issues

#### Problem: Slow initial page load

**Solutions:**

1. **Optimize WASM size:**
   ```bash
   # Use optimization flags
   tinygo build -opt=z -no-debug
   
   # Strip debug info
   wasm-opt -Oz input.wasm -o output.wasm
   ```

2. **Enable compression:**
   ```go
   // Serve compressed WASM
   w.Header().Set("Content-Encoding", "gzip")
   ```

3. **Implement code splitting:**
   ```go
   // Load components on demand
   if userNeedsFeature {
       loadComponent("heavy-feature")
   }
   ```

#### Problem: Memory leaks in WASM

**Symptoms:**
- Browser tab crashes
- "Out of memory" errors

**Solutions:**

1. **Use object pooling:**
   ```go
   var vnodePool = sync.Pool{
       New: func() interface{} {
           return &VNode{}
       },
   }
   ```

2. **Clear references:**
   ```go
   // Clear handlers when component unmounts
   component.handlers = nil
   component.LastVNode = nil
   ```

3. **Limit component instances:**
   ```go
   // Reuse components where possible
   if existing := registry.Get(id); existing != nil {
       return existing
   }
   ```

### Debugging Techniques

#### Enable Debug Logging

```go
// Set debug mode
os.Setenv("VANGO_DEBUG", "true")

// Add debug logs
if os.Getenv("VANGO_DEBUG") == "true" {
    log.Printf("[DEBUG] %s", message)
}
```

#### Browser DevTools

1. **Network Tab:**
   - Check WebSocket frames
   - Verify binary message format
   - Look for failed requests

2. **Console:**
   - Add strategic console.log
   - Check for JavaScript errors
   - Monitor WebSocket state

3. **Performance Tab:**
   - Profile rendering performance
   - Identify bottlenecks
   - Check memory usage

#### Server Logging

```go
// Add detailed logging
log.Printf("[Component %s] State: %+v", id, state)
log.Printf("[Scheduler] Dirty queue: %d", len(dirtyQueue))
log.Printf("[WebSocket] Sending %d patches", len(patches))
```

#### Binary Protocol Debugging

```javascript
// Decode and log binary messages
ws.onmessage = (event) => {
    const view = new DataView(event.data);
    const frameType = view.getUint8(0);
    console.log('Frame type:', frameType.toString(16));
    
    // Hex dump for debugging
    const hex = Array.from(new Uint8Array(event.data))
        .map(b => b.toString(16).padStart(2, '0'))
        .join(' ');
    console.log('Hex:', hex);
};
```

### Error Messages Explained

#### "Session not found"

**Meaning:** The WebSocket session doesn't exist in the scheduler bridge

**Fix:** Ensure session is created when WebSocket connects

#### "No handler for node X"

**Meaning:** Event received for unregistered node ID

**Fix:** Register handlers with correct node IDs

#### "Failed to decode event: event data too short"

**Meaning:** Binary event format is incorrect

**Fix:** Ensure event has [FrameType][EventType][NodeID] format

#### "Send buffer full"

**Meaning:** WebSocket send channel is blocked

**Fix:** Client may be disconnected or slow to process

### Platform-Specific Issues

#### macOS

- **File watching limits:** Increase with `ulimit -n`
- **Port already in use:** Find process with `lsof -i :8080`

#### Linux

- **TinyGo installation:** May need manual PATH setup
- **WebSocket connection refused:** Check firewall rules

#### Windows

- **Path separators:** Use filepath.Join() not hardcoded /
- **Build scripts:** Use .bat versions or WSL

### Getting Help

#### Before Asking for Help

1. Check this troubleshooting guide
2. Search existing GitHub issues
3. Try with a minimal reproduction
4. Collect relevant logs

#### Information to Provide

```markdown
### Environment
- OS: [e.g., macOS 14.0]
- Go version: [go version]
- TinyGo version: [tinygo version]
- Browser: [e.g., Chrome 120]

### Problem
[Clear description of the issue]

### Steps to Reproduce
1. [First step]
2. [Second step]
3. [See error]

### Expected Behavior
[What should happen]

### Actual Behavior
[What actually happens]

### Logs
```
[Relevant server logs]
```

### Code
```go
// Minimal reproduction code
```
```

#### Where to Get Help

- GitHub Issues: https://github.com/vango-ui/vango/issues
- Discord: https://discord.gg/vango
- Stack Overflow: Tag with `vango`
- Documentation: https://vango.dev/docs

### Common Gotchas

1. **Frame type constants are not what you think:**
   - FramePatches = 0x00 (not 0x02!)
   - FrameEvent = 0x01
   - FrameControl = 0x02

2. **Session IDs must match exactly:**
   - Generated on server during render
   - Must be same in WebSocket path

3. **Component instances are per-session:**
   - Not shared between users
   - Created on first render
   - Connected when WebSocket opens

4. **Binary protocol uses varint encoding:**
   - Not fixed-width integers
   - Must decode properly

5. **Scheduler must be running:**
   - Started when session created
   - Processes dirty queue
   - Generates patches

6. **Events need proper node IDs:**
   - Assigned during render
   - Must match between server and client
   - Used to route to handlers

### Performance Tips

1. **Minimize VNode creation:**
   ```go
   // Cache static nodes
   var staticHeader = builder.Header()./*...*/.Build()
   ```

2. **Use keys for lists:**
   ```go
   for _, item := range items {
       node := builder.Li().
           Key(item.ID). // Important!
           Text(item.Name).
           Build()
   }
   ```

3. **Batch state updates:**
   ```go
   scheduler.Batch(func() {
       state1.Set(value1)
       state2.Set(value2)
       // Renders once, not twice
   })
   ```

4. **Profile before optimizing:**
   ```bash
   go test -bench=. -cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

Remember: Most issues are simple configuration problems. Check the basics first!
