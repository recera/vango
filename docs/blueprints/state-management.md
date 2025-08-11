---
title: State Management
slug: state-management
version: 0.1
phase: Q1-2025
status: draft
requires:
  - cooperative-scheduler
---

# State Management Blueprint

> **Goal**: Provide a reactive data model that scales from local component state to multi-session global signals and structured stores.

## 1. Reactive Primitives
| API | Generic? | Description |
|-----|----------|-------------|
| `State[T]` | ✅ | Local mutable value, triggers component re-render on `Set`. |
| `Computed[T]` | ✅ | Memoised function recalculated when dependencies change. |
| `Signal[T]` | ✅ | Alias for `State[T]`; kept for readability. |
| `GlobalSignal[T]` | ✅ | Cross-session reactive value synced via Live WS. |

### 1.1 Example
```go
count := vango.State(0)
dbl := vango.Computed(func() int { return count.Get()*2 })
```

## 2. Dependency Tracking
* Implemented via dynamically scoped `currentFiber` pointer.  
* During `Get()`, the signal records dependency onto active fiber.  
* On `Set()`, mark dependent fibers dirty and enqueue.

## 3. Structured Store Add-on
```go
type TodoStore struct {
    Items vango.Signal[[]Todo]
}
func (t *TodoStore) Add(text string) { ... }
var todos = store.New(&TodoStore{})
```
* Generates typed hooks: `useTodos()` returns `*TodoStore` with reactive fields.
* Actions mutate signals but batch patches until function returns.

## 4. Resource & Suspense
```go
users := vango.Resource(fetchUsers)
list, err := users.Value() // blocks fiber until resolved
```
* If unresolved, fiber yields `SuspensePromise` → scheduler pauses it.  
* When fetch completes, fiber is re-enqueued.
* **Error path**: if fetch returns `(nil, err)` the promise resolves with `ResourceError{Err: err}`; scheduler continues fiber where `users.Value()` returns `(nil, err)` so component can branch.  
  ```go
  list, err := users.Value()
  if err != nil {
      return ErrorView(err)
  }
  ```
* `vango.Suspense()` component accepts `Fallback` and `Error` slots:
  ```go
  vango.Suspense(vango.Fallback(Spinner()), vango.Error(func(e error) vango.VNode {
      return vango.Text("failed: "+e.Error())
  }), UsersList())
  ```

## 5. DevTools Support
* Graph visualisation: nodes = signals, edges = deps.  
* Time-travel: record `Set` deltas; scheduler can rewind.

## 6. Persistence Layer
`persist.New(signal, persist.LocalStorage("todos"))` enables saving to LS, URL, or cookies.

## 7. Performance Targets
| Metric | Budget |
|--------|--------|
| Signal `Get()` | <20 ns |
| Commit batch 1k signals | <1 ms |
| GlobalSignal WS latency | <80 ms |

## 8. Open Questions
* Conflict resolution strategy for concurrent GlobalSignal `Set` from two clients? Last-write-wins vs CRDT.
* Should `Computed` be lazy (on-demand) or eager (on any dep change)?

## 9. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
