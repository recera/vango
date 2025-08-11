---
title: Build & Tooling
slug: build-system
version: 0.2
phase: P-0
status: draft
requires:
  - rendering-pipeline
---

# Build System Blueprint

> **Goal**: Offer a single `vango` CLI that covers dev server, production build, and project scaffolding without Node.js (except optional Tailwind binary).

## 1. Commands
| Command | Flags | Description |
|---------|-------|-------------|
| `vango dev` | `--proxy`, `--open`, `--port` | Incremental compile (TinyGo `-gc=leaking`), WASM hot reload, CSS HMR, proxy backend. |
| `vango build` | `--release`, `--pwa`, `--split` | TinyGo `-opt=z`, tree-shake assets, optional code-splitting chunks. |
| `vango create my-app` | `--template=counter` | Generate starter folder. |
| `vango docs build` | | Build static docs via mdBook. |

## 2. Dev Server Stack
* File watcher via `fsnotify`.  
* On `.go` change → `TinyGo build -o app.wasm` (incremental cache) then WebSocket push `reload:wasm`.  
* On `.css` change → compute hash, push `reload:style` with new href.

## 3. Production Build Pipeline
```
(vango build)
     ↓
TinyGo compile → strip DWARF → wasm-opt -Oz
     ↓
Asset graph (HTML, CSS, images) → content hash
     ↓
PWA manifest & service worker (if --pwa)
```

## 4. Code Generation Steps
1. CLI scans `app/routes/**.go` → emit router tree.  
2. Collect `//vango:template` macros → run template parser.  
3. Run `tailwindcss -m` if config present.  
4. Rewrite `vango.Style()` calls and extract CSS.

## 4.1 Configuration Overrides – `vango.json`
Users place an optional file at project root to tweak defaults:
```jsonc
{
  "tailwindConfig": "./styles/tailwind.config.js", // non-default path
  "routesDir": "./app/pages",                      // change folder
  "wasmTarget": "wasm",                            // or wasm-mvp, wasm32-wasi
  "pwa": {
    "enabled": true,
    "manifest": "./pwa/manifest.json"
  }
}
```
Schema (JSON Schema draft-07) lives in `internal/schema/vango.schema.json`; CLI validates and prints useful errors.

## 4.2 Client-Side Bootstrap JS
Minimal JS (~3 kB gzipped) is embedded in the CLI binary under `internal/assets/bootstrap.js`.
Responsibilities:
1. Load `wasm_exec.js` (vendored TinyGo runtime) and instantiate the compiled `app.wasm`.
2. Establish Live WebSocket and handle reconnect logic (see Live Protocol §5).
3. Listen for dev-server HMR messages (`reload:wasm`, `reload:style`).

During `vango build` the bootstrap is content-hashed and written to `dist/bootstrap.<hash>.js`; SSR inserts `<script defer src="/bootstrap.<hash>.js"></script>` automatically. Users **should not** import it manually.



## 5. Caching & Rebuild Times
* Artifact cache keyed by `go.mod` + file hashes; stored under `$HOME/.cache/vango/`.

## 6. CI Integration
`make ci` executes:
```bash
go vet ./...
go test ./...
vango build --release --pwa
scripts/check-size.sh dist/app.wasm 800k
```

## 7. Open Questions
* Use `wasm-opt` (binaryen) for further size reduction?  
* Provide `vango eject` to expose raw TinyGo flags?

## 8. Legend
* `↓` denotes a step in the build pipeline.
* `→` denotes a transformation or processing step.

## 9. Acceptance & Validation
| Check | Command | Expected |
|-------|---------|----------|
| Lint  | `staticcheck ./cmd/vango/...` | 0 issues |
| Unit  | `go test ./internal/... -run TestBuildPipeline` | PASS |
| WASM Size | `scripts/check-size.sh dist/app.wasm 800k` | ≤ target |
| CLI Help | `vango --help` | Lists commands in §1 |
| Smoke | `vango dev --port=0` | Serves and HMR works (watcher log) |

### 9.1 CI Matrix
```yaml
name: vango-build
on: [push]
jobs:
  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: make ci
  build-mac:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - run: make ci
```

## 10. Example `vango.json`
```jsonc
{
  "tailwindConfig": "./styles/tailwind.config.js",
  "routesDir": "./app/pages",
  "wasmTarget": "wasm32-wasi",
  "pwa": {
    "enabled": true,
    "manifest": "./pwa/manifest.json"
  }
}
```
Use `vango validate` to lint this file against `internal/schema/vango.schema.json`.

## 11. Cross-References
* Codegen steps: `@docs/blueprints/codegen-routing-spec.md`
* Styling extractor: `@docs/blueprints/styling.md`
* Live reload protocol: `@docs/blueprints/live-protocol.md`

## 12. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
|2025-08-06|0.2|Add acceptance matrix, example config, cross-refs|
