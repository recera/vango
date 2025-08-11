# State Management Deep Dive

Vango’s reactive system is signal-based and scheduler-aware, enabling fine-grained updates.

## Signals
```go
import "github.com/recera/vango/pkg/reactive"

count := reactive.CreateState(0)

// Read within render to subscribe the current fiber
_ = count.Get()

// Update triggers re-render of dependent fibers
count.Set(count.Get() + 1)
```
- Dependencies are tracked per fiber via `reactive.SetCurrentFiber` (wired by the runtime)
- `Subscribe`/`Unsubscribe` happens automatically when `Get()` is called during render

## Computed Values
```go
price := reactive.CreateState(100.0)
qty := reactive.CreateState(2)

total := reactive.CreateComputed(func() float64 {
  return price.Get() * float64(qty.Get())
})

_ = total.Get() // subscribes to total (which internally subscribes to price/qty)
```
- Computed invalidates when any source changes and recomputes lazily on next `Get()`

## Batching
Group multiple updates to avoid redundant renders.
```go
reactive.RunBatch(sched, func(){
  count.Set(count.Get()+1)
  qty.Set(qty.Get()+3)
})
```
- The batch collects dirty fibers and marks them once at commit
- Get `sched` from your runtime context

## Server-Driven State
- Server receives events over WS, updates state, re-renders, diffs VDOM, and sends patches via `live.Server.SendPatches`
- Keep server state per session in the live session store or within your handlers

## Patterns
- Keep renders pure: compute derived values via `Computed`
- Avoid circular dependencies between signals
- Consider colocating related signals in a small module per component

## Example: Controlled Form
```go
name := reactive.CreateState("")
email := reactive.CreateState("")

func Form() *vdom.VNode {
  return builder.Form().OnSubmit(func(){ /* save */ }).Children(
    builder.Input().Type("text").Value(name.Get()).OnInput(func(v string){ name.Set(v) }).Build(),
    builder.Input().Type("email").Value(email.Get()).OnInput(func(v string){ email.Set(v) }).Build(),
    builder.Button().Text("Save").Build(),
  ).Build()
}
```
