# Testing

## Unit Tests
- Use `go test ./...` for pure Go logic
- Component functions that return `*vdom.VNode` can be validated structurally (tags, classes, child counts)

## WASM DOM Tests
- Build and run tests targeting WebAssembly to exercise client-side behavior
- With standard toolchain: `GOOS=js GOARCH=wasm go test ./...`
- Interact with the DOM via `syscall/js` and test hydration or event handlers

## Server-Driven Tests
- Stand up a test server exposing `/vango/live/<session>` and simulate client messages
- Verify that events result in expected patch frames (binary) or use legacy JSON updates for simplified assertions

## E2E (Playwright/Cypress)
- Run `vango dev` in CI
- Navigate to pages, interact with buttons/inputs, and assert DOM changes
- For server-driven, ensure WS connection is established and reconnection is handled

## Tips
- Keep side effects out of render functions to simplify assertions
- Use data attributes (`data-testid`) for stable selectors in tests
