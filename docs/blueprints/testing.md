---
title: Testing Strategy
slug: testing
version: 0.1
phase: P-0
status: draft
requires:
  - rendering-pipeline
---

# Testing Strategy Blueprint

> **Goal**: Ensure reliability across SSR, WASM, and Live update layers with fast, maintainable tests.

## 1. Pyramid
1. **Unit Tests** (80 %) – `go test`, cover VDOM diff, signals, router matcher.  
2. **WASM DOM Tests** (10 %) – TinyGo runner executes in `jsdom`.  
3. **Integration / E2E** (10 %) – Playwright drives browser against `examples/`.

## 2. Unit Testing
```bash
go test ./pkg/... -run TestDiff
```
* Golden snapshots under `testdata/`.

## 3. WASM DOM Tests
* **Task 0 – Harness PoC**: Implement `internal/testharness` to launch headless Chrome with `go test -exec` hooking `wazero run`. Goal: run `t.Run("button click")` within 30 s on CI.  
* Command after harness:
  ```bash
  tinygo test -target wasm ./pkg/renderer/dom/... -exec "testharness"
  ```
* The harness:
  1. Builds the test WASM file.
  2. Serves simple HTML with `<script src="wasm_exec.js">`.
  3. Uses Chrome DevTools Protocol to stream `console.log` back to Go process mapping to `testing.T` output.


## 4. Integration Tests
```ts
// tests/counter.spec.ts
await page.goto("http://localhost:5173");
await page.click("button");
expect(await page.textContent("#count")).toBe("1");
```
CI uses Playwright’s Docker image.

## 5. Hot Reload Harness
`test/hotreload_test.go` watches a dummy file, triggers rebuild, asserts websocket `reload:wasm` frame.

## 6. Performance Benchmarks
* `bench_test.go` renders 1k VNodes 100 times.  
* Bundle-size gate script fails build if `dist/app.wasm` > 800 kB.

## 7. Code Coverage
Generate via `go test -coverprofile`, upload to Codecov.

## 8. Open Questions
* Should fuzzing be used on template parser?  
* Chaos tests for websocket disconnects?

## 9. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
