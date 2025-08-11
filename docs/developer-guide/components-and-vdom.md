# Components and Virtual DOM

## VNode model (mental model)
- Element nodes: `Tag` + `Props` + `Kids`
- Text nodes: `Text`
- Optional `Key` for list diffing; stable keys reduce DOM churn
- Optional Ref: callback receives DOM element after mount (WASM only)

## Render Cycle
1) A component function returns `*vdom.VNode`
2) Runtime (server or client) diffs against previous tree
3) Applier commits output (HTML string on server; DOM patches on client)
4) Event handlers fire and update state → mark affected components dirty → re-render

## Appliers
- Server: `pkg/renderer/html/applier.go` → generates string HTML
- Client (WASM or server-driven minimal runtime): applies patch opcodes to real DOM

## Diff/Patch Overview
The patch stream includes operations like:
- ReplaceText (id, text)
- SetAttribute (id, key, value)
- RemoveAttribute (id, key)
- InsertNode (id, parent, before)
- RemoveNode (id)
- MoveNode (id, parent, before)
- UpdateEvents (id, bitmask)

Provide stable keys for list items so moves can be minimal.

## Events & Props
- Element events (`onclick`, `oninput`, etc.) are stored in `Props` and wired differently per mode
- Use builder helpers for common events: `.OnClick`, `.OnInput`, `.OnSubmit`, `.OnChange`
- Non-standard attributes: `.Attr(key, val)`

## Controlled Inputs
- Bind `.Value` and listen to `.OnInput(func(string))`
- Keep form state in signals and update on input to avoid drift

## Composition Patterns
- Leaf components return a single node
- Layouts accept `...*vdom.VNode` children
- Pass data via parameters instead of global state when possible

## Performance Tips
- Prefer CSS classes over inline styles to reduce patch sizes
- Keep component functions pure (no I/O in render)
- Memoize expensive derived values using `Computed` signals
- Use keys and avoid reordering children unnecessarily
