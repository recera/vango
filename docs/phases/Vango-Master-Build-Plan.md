title: "Vango Master Build Plan & Order of Operations"
slug: vango-master-build-plan
version: 1.1
status: ratified
requires:
  - P-0-Build-Plan-Core-Runtime
  - P-1-Build-Plan-DX-Expansion
  - P-2-Build-Plan-Advanced-Features
  - P-3-Build-Plan-Production-Readiness
---

# Vango Master Build Plan & Order of Operations

**Document Purpose:** This document serves as the high-level engineering directory, synthesizing the detailed phase plans into a single, coherent build strategy. It outlines the general order of operations for building the Vango framework from its initial commit to a stable 1.0 General Availability release.

**Guiding Principles:**

1.  **Foundation First:** The core runtime (VDOM, scheduler) must be built and proven before any developer-facing features (syntax, routing) are added. The stability of the entire framework depends on the correctness of this core.
2.  **Specification-Driven:** Each component must be built according to its corresponding detailed blueprint and API contract to ensure interoperability.
3.  **Incremental DX:** The developer experience (CLI, hot reload) is not an afterthought. It is built in parallel with the core, ensuring the framework is usable and testable by the development team at every stage.
4.  **Layered Complexity:** Advanced, cross-cutting concerns like the full plugin system, observability, and advanced security are layered on top of a stable core, not integrated prematurely.

---

## The Critical Path: A Sequential Overview

This is the general, sequential order for building Vango. Each step is a prerequisite for the next.

### **Phase 0: The Core Runtime & MVP**

**Goal:** Prove the fundamental architectural hypothesis. Can we build a hybrid SSR/WASM application with a single VDOM, a cooperative scheduler, and a hot-reload loop?

1.  **VDOM Primitives (`pkg/vdom`):**
    *   **What:** Define the immutable `VNode` struct and implement the core `diff` algorithm. The output `Patch` types must map directly to the opcodes defined in the `live-protocol.md` spec.
    *   **Why:** This is the universal language of the framework. Nothing can be rendered or updated without it.

2.  **SSR Renderer (`pkg/renderer/html`):**
    *   **What:** Implement the `htmlApplier` to render a `VNode` tree to an HTML string, including the injection of `data-hid` attributes for evented nodes.
    *   **Why:** This provides the first tangible output and the server-rendered content needed for hydration.

3.  **Cooperative Scheduler & Basic State (`pkg/scheduler`, `pkg/reactive`):**
    *   **What:** Implement the `fiber` and `Scheduler` loop. Implement the basic `State[T]` primitive that can mark fibers as dirty via dynamic dependency tracking.
    *   **Why:** This is the heart of the client-side execution model, designed to work within TinyGo's memory constraints.

4.  **WASM Renderer & Hydration (`pkg/renderer/dom`, `internal/assets/bootstrap.js`):**
    *   **What:** Implement the `domApplier` using `syscall/js` and the initial hydration logic within the client-side `bootstrap.js`.
    *   **Why:** This brings the application to life on the client, connecting the static server-rendered HTML with the live, interactive WASM module.

5.  **CLI & Dev Server (`cmd/vango`):**
    *   **What:** Build the `vango dev` command with file watching, WebSocket-based hot reloading, artifact caching, and support for `vango.json` configuration files.
    *   **Why:** This provides the essential, fast feedback loop required for all subsequent development.

**Outcome of Phase 0:** A working `counter` example that proves the core architecture is viable.

---

### **Phase 1: Developer Experience & Feature Expansion**

**Goal:** Make the framework productive and capable of building real applications.

1.  **File-Based Routing (`pkg/server`, `cmd/vango/internal/router`):**
    *   **What:** Implement the file-system scanner and code generator as specified in `codegen-routing-spec.md`. Implement the runtime router middleware, which is responsible for creating and managing the `vango.Ctx` object defined in `api-contracts.md`.
    *   **Why:** This is the foundation for any multi-page application.

2.  **Component Syntax (VEX) (`pkg/vex`, `cmd/vango/internal/template`):**
    *   **What:** Implement the Fluent Builder API and the HTML-like Template Macro system, adhering to the detailed `template-spec.md` for code generation, including the `//vango:props` directive.
    *   **Why:** To move beyond verbose, functional `VNode` creation and provide an ergonomic API.

3.  **Styling Strategy (`pkg/styling`, `cmd/vango/internal/styling`):**
    *   **What:** Implement the complete styling solution, including the two-part (runtime and build-time) system for `vango.Style` scoped styles.
    *   **Why:** A rich standard library and a flexible styling system are table stakes for a modern UI framework.

**Outcome of Phase 1:** The framework is now ergonomic and capable of building well-structured, styled, multi-page applications.

---

### **Phase 2: Advanced State, Real-Time & Security**

**Goal:** Enable complex, dynamic, and secure applications.

1.  **The Live Protocol (`pkg/live`, `internal/assets/bootstrap.js`):**
    *   **What:** Implement the binary patch protocol over WebSockets, including the full client-side reconnection and resync logic (`HELLO resumable=true`, `REFETCH`).
    *   **Why:** This is the nervous system for all real-time features and server-driven UI updates after the initial load.

2.  **Advanced State Management (`pkg/reactive`):**
    *   **What:** Implement `Computed[T]`, `GlobalSignal[T]`, and `Resource` with `Suspense`, including the specified error handling paths for async operations.
    *   **Why:** To give developers the powerful tools needed to manage complex data flows and asynchronous operations.

3.  **Foundational Security & E2E Testing (`pkg/server/middleware`, `test/`):**
    *   **What:** Implement CSRF and CSP protection. Set up the Playwright E2E testing suite and the critical WASM Test Harness PoC.
    *   **Why:** To establish a baseline of security and to create the highest level of testing confidence for complex user flows.

**Outcome of Phase 2:** Vango is now a feature-rich framework capable of building secure, real-time applications.

---

### **Phase 3: Production Readiness & Ecosystem**

**Goal:** Make the framework a stable, trustworthy, and extensible platform ready for public adoption.

1.  **Full Observability Suite (`pkg/observability`):**
    *   **What:** Integrate structured logging (`slog`), Prometheus metrics, and OTLP tracing throughout the framework, accessible via the `vango.Ctx` where appropriate.
    *   **Why:** To make production applications transparent, monitorable, and debuggable.

2.  **Plugin System (`pkg/vango/addon.go`, `cmd/vango/addon.go`):**
    *   **What:** Implement the `Addon` interface, the hook-based loading system, and the `vango addon` CLI for managing secure, checksummed plugins.
    *   **Why:** To ensure the framework is extensible and to foster a healthy third-party ecosystem.

3.  **Production Build & CI Hardening:**
    *   **What:** Implement PWA generation and add performance and bundle-size gates to CI.
    *   **Why:** To provide modern deployment options and to prevent regressions in key performance metrics.

4.  **Documentation & GA (`docs/`, `website/`):**
    *   **What:** Ratify all design documents, write comprehensive user guides, build the public documentation site, and freeze the API for a v1.0 release.
    *   **Why:** A project without documentation does not exist. This is the final step to prepare for widespread community adoption.

**Outcome of Phase 3:** Vango is a world-class framework, ready for a stable v1.0 release.

---

This master plan ensures a logical progression, where each phase builds upon the tested and validated output of the previous one. By following this order of operations, we can build the Vango framework efficiently and with a high degree of quality and stability.