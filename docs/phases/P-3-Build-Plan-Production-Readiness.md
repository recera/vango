title: "Build Plan: P-3 Production Readiness & Ecosystem"
slug: P-3-Build-Plan-Production-Readiness
version: 1.1
status: ratified
requires:
  - P-2-Build-Plan-Advanced-Features
---

# Engineering Build Plan: Phase 3 - Production Readiness & Ecosystem

**Document Purpose:** This document outlines the final major engineering phase before a General Availability (GA) release. With a feature-complete and stable core, this phase focuses on the cross-cutting concerns required to run Vango applications reliably and securely in production.

**Phase Goal:** To transition Vango from a powerful framework into a stable, trustworthy, and extensible platform. This involves implementing a full observability suite, a secure plugin architecture, hardening the production build pipeline, and publishing a complete documentation site. This corresponds to the **Beta 0.9** and **v1.0 GA** milestones.

---

## Epic 1: Full Observability Suite

**What & Why:** Production applications are black boxes without good observability. This epic implements the `observability.md` blueprint to empower operators and developers to debug issues, monitor performance, and understand system behavior.

### Task 1.1: Integrate Structured Logging (`slog`)

*   **Files:** Across the entire `pkg/` directory.
*   **Plan:**
    1.  Refactor the entire codebase to use Go 1.22's `slog` for all internal logging.
    2.  Add contextual attributes at key points (e.g., `request_id` in server middleware, `fiber_id` in the scheduler).
    3.  The logger instance will be passed via the `vango.Ctx` object, accessible via `ctx.Logger()`.

### Task 1.2: Implement Prometheus Metrics

*   **Files:** `pkg/observability/metrics.go`, `pkg/server/middleware/metrics.go`
*   **Plan:**
    1.  Add the Prometheus Go client library as a dependency.
    2.  Create a central registry for all framework metrics defined in `observability.md`.
    3.  Create an HTTP middleware that exposes the `/metrics` endpoint for scraping.
    4.  Instrument the code at key locations (e.g., request latency, active fibers, patch bytes sent).

### Task 1.3: Implement OTLP Tracing

*   **File:** `pkg/observability/tracing.go`
*   **Plan:**
    1.  Add the OpenTelemetry Go SDK as a dependency.
    2.  Create a tracer provider configurable via `OTEL_EXPORTER_OTLP_ENDPOINT`.
    3.  Instrument the code by creating and propagating spans around critical operations (`ssr.render`, `scheduler.commit`, `wasm.hydrate`, etc.).
*   **Testing:** Implement the `TestTraceExport` unit test which uses an in-memory exporter to validate span creation and attributes.

### Task 1.4: DevTools Extension MVP

*   **Files:** `tools/devtools-extension/`
*   **Plan:**
    1.  Create a browser extension sub-project.
    2.  The extension will connect to a special WebSocket endpoint on the dev server.
    3.  The framework (in debug mode) will push the reactive graph, live patch logs, and fiber timelines to the extension for visualization.
*   **Pitfalls:** This is a complex task. The initial MVP should focus on one feature, like the Live patch log, and build from there.

---

## Epic 2: Plugin System

**What & Why:** A plugin system (`Addon`s) is the key to extensibility and a vibrant ecosystem, allowing features to be added without bloating the core runtime.

### Task 2.1: Define the `Addon` Interface and Hook System

*   **Files:** `pkg/vango/addon.go`, `cmd/vango/main.go`
*   **Plan:**
    1.  Define the `Addon` interface in Go as specified in `plugin-system.md`.
    2.  Implement the reflection-based loading mechanism in the `vango` CLI that discovers installed addons, instantiates them, and calls their lifecycle hooks (`Init`, `DevServerHook`, `BuildHook`).

### Task 2.2: Implement `vango addon` CLI

*   **File:** `cmd/vango/addon.go`
*   **Plan:**
    1.  Implement the `vango addon install <module_path>` command, which will use `go get`, calculate a SHA-256 checksum, and update the `addons.lock` file.
    2.  Implement `vango addon list`.
    3.  The `vango` CLI must verify checksums from `addons.lock` on every run to prevent supply-chain attacks.

### Task 2.3: Build Internal Addons

*   **Plan:** To prove the plugin system works, refactor existing functionality and build the planned "built-in" addons:
    1.  **`tailwind` addon:** Convert the existing Tailwind integration into a proper addon.
    2.  **`otel` addon:** Package the OTLP exporter logic from Epic 1 as an addon.
    3.  **`rpcgen` addon:** Build a new addon that reads a schema and generates type-safe RPC stubs.

---

## Epic 3: Production Build & Security Finalization

**What & Why:** This epic focuses on the final touches that make an application ready for public deployment.

### Task 3.1: PWA & Service Worker Generation

*   **File:** `cmd/vango/build.go`
*   **Plan:**
    1.  Implement the `--pwa` flag for the `vango build` command.
    2.  When enabled, the build process will generate a `manifest.json` (from `vango.json` config) and a basic caching service worker, injecting the necessary links into the final HTML.

### Task 3.2: Finalize Security Posture

*   **Plan:**
    1.  **`auth` addon:** Build a reference implementation of an authentication addon that provides session cookie management (using the `vango.Ctx.Session()` interface) and route guards.
    2.  **SRI Hashes:** Add a build step to calculate Subresource Integrity (SRI) hashes for all external assets (`bootstrap.js`) and add the `integrity` attribute to the `<script>` tags.

### Task 3.3: CI Hardening (Perf & Size Gates)

*   **File:** `.github/workflows/ci.yml`, `scripts/check-size.sh`
*   **Plan:**
    1.  Implement the bundle size check script. The CI pipeline will run `vango build --release` and then execute this script, failing the build if `app.wasm` exceeds the budget (e.g., 800kB).
    2.  Add a performance benchmark step to CI that runs the `bench_test.go` suite and fails the build on significant performance regressions.

---

## Epic 4: Documentation & GA

**What & Why:** A project without documentation does not exist. This is the final step to prepare for a v1.0 release.

### Task 4.1: Build Docs Site Generator

*   **File:** `cmd/vango-docgen/main.go`
*   **Plan:** Create a CLI tool that reads all markdown files in `docs/`, parses their front-matter to build navigation, and renders a static HTML site.

### Task 4.2: Ratify All Blueprints & Write Guides

*   **Plan:**
    1.  Conduct a final review of every document in `docs/adr/` and `docs/blueprints/`, ensuring they accurately reflect the final implementation. Change their `status` to `ratified`.
    2.  Write new, user-friendly tutorials for every major feature (e.g., "Building your first multi-page app," "Advanced state management with GlobalSignal").

### Task 4.3: API Freeze for v1.0

*   **Plan:**
    1.  Conduct a thorough review of all public-facing APIs in `pkg/`.
    2.  Make any final breaking changes.
    3.  Tag the release as `v1.0.0`. From this point on, all changes must follow semantic versioning.

---

**Phase 3 Success:** With the completion of this phase, Vango is a world-class framework. It is not only powerful and feature-rich but also stable, secure, observable, and extensible. It is accompanied by a comprehensive test suite, a polished documentation site, and a stable v1.0 API.