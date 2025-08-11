---
title: Plugin / Add-on System
slug: plugin-system
version: 0.1
phase: Q2-2025
status: draft
---

# Plugin / Add-on System Blueprint

> **Goal**: Allow third-party extensions (state store, RPC generator, design system) without bloating the core runtime.

## 1. Key Concepts
* **Addon** – Go module implementing `vango.Addon` interface.
* **Hook** – callback into CLI or dev server life-cycle.
* **Manifest** – YAML describing addon metadata and SHA-256.

## 2. `Addon` Interface
```go
type Addon interface {
    Init(cfg Config) error           // CLI start
    DevServerHook(h *http.ServeMux)  // register endpoints
    BuildHook(out *build.Artifacts)  // mutate output folder
}
```
Addons are loaded by reflection via `init()` when present under `extras/` or vendored module.

## 3. CLI Commands
| Command | Purpose |
|---------|---------|
| `vango addon install github.com/foo/bar@v1.2.3` | Fetches module, verifies checksum, updates `addons.lock`. |
| `vango addon list` | Shows installed addons. |

## 4. Sandboxing & Security
* Addons run in separate Go modules; no access to internal packages.  
* Checksum in `addons.lock` prevents supply-chain attacks.

## 5. Built-in Addons
1. `tailwind` – style builder.  
2. `otel` – telemetry exporter.  
3. `rpcgen` – schema-fused RPC generator.

## 6. Versioning
* Semantic versioning; breaking API requires major bump.  
* CLI warns if addon major version differs from Vango’s.

## 7. Open Questions
* Should addons be allowed to ship WASM side-car?  
* Marketplace UI for discovery?

## 8. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
