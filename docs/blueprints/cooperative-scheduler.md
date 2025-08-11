---
title: Cooperative Scheduler
slug: cooperative-scheduler
version: 0.1
phase: P-0
status: draft
requires:
  - ADR-0001-single-vdom
---

# Cooperative Scheduler Blueprint

> **Purpose**: Provide an efficient, TinyGo-friendly execution model that simulates "goroutine per component" without incurring the memory cost of real OS-level goroutines.

## 1. Motivation
TinyGo goroutines allocate ~2 kB stack each. Large apps with thousands of components would blow past the 16 MB memory ceiling imposed by some browsers and edge runtimes. We therefore implement a *logical* fiber per component and run them on a single real goroutine.

## 2. Terminology
* **Fiber** – lightweight struct holding component state and a message channel.
* **Scheduler** – central event-loop that drives dirty fibers.

## 3. Fiber Data Structure
```go
type fiber struct {
    id     uint32
    parent *fiber
    vnode  vdom.VNode   // last rendered tree
    scope  *reactive.Scope
    ch     chan struct{}  // wake-up signal
    dirty  bool
}
```

## 4. Scheduler Algorithm
```go
func startScheduler(root *fiber) {
    q := make([]*fiber, 0, 1024)
    for {
        if len(q) == 0 {
            f := <-globalWake
            q = append(q, f)
        }
        f := q[0]
        q = q[1:]
        if f.dirty {
            next := f.render()        // calls component Render()
            patches := diff(f.vnode, next)
            applyPatches(patches)
            f.vnode = next
            f.dirty = false
        }
    }
}
```

*Dirty Flag*: set by reactive signals or external events.  
*Global Wake*: buffered channel where any fiber can enqueue itself.

## 5. Interaction with Reactivity
Signals call `markDirty(fiber)` which sets `dirty = true` and sends fiber to `globalWake` once per animation frame (debounced).

## 6. Error Handling
If `Render()` panics, `OnError(err)` is invoked. Returning `true` continues schedule; otherwise the fiber is unmounted.

## 7. Performance Targets
| Metric | Budget |
|--------|--------|
| Scheduler overhead (10k fibers idle) | < 1 ms/frame |
| Memory per fiber | ≤ 320 bytes |

## 8. Extensibility Hooks
* **Priorities** – future feature may add priority lanes (e.g., input vs background).
* **Suspense** – a fiber can yield a `SuspensePromise` to defer rendering until data arrives.

## 9. Open Questions
* Should we batch `applyPatches` per frame for better DOM coherence?
* How to integrate Web Worker offloading for long tasks?

## 10. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|First draft|
