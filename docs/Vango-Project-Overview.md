---
title: "Vango: A Comprehensive Project Overview"
slug: vango-project-overview
version: 1.0
status: ratified
---

# Vango: A Comprehensive Project Overview

**Document Purpose:** This document provides a single, authoritative, and detailed overview of the Vango framework. It is intended to serve as the primary context for engineering teams and AI agents, synthesizing all architectural blueprints, API contracts, and build plans into one coherent guide.

---

## 1. Core Vision & Mission

Vango is a Go-native, hybrid-rendered UI framework designed to challenge the status quo of JavaScript-first front-end stacks. It aims to provide a first-class developer experience, exceptional performance, and a robust, type-safe environment for building modern web applications.

**Guiding Principles:**
1.  **Foundation First:** A stable, performant core runtime is the prerequisite for all other features.
2.  **Specification-Driven:** Development is guided by detailed blueprints and API contracts to ensure quality and interoperability.
3.  **Incremental DX:** Developer experience (tooling, syntax, hot-reloading) is a core feature, not an afterthought.
4.  **Layered Complexity:** Advanced features (plugins, observability) are built on top of a proven, stable foundation.

---

## 2. Core Architecture

Vango's architecture is designed to eliminate the common pain points of traditional web development, such as the client-server divide and state management complexity.

### 2.1. The Single VDOM (Virtual DOM)

The bedrock of the entire framework is a single, unified Virtual DOM model.
*   **Immutable `VNode`:** A Go struct (`pkg/vdom/types.go`) that serves as an abstract representation of the UI. It is immutable, ensuring predictable rendering and diffing.
*   **Unified Diff Algorithm:** A single, highly-optimized diff algorithm (`pkg/vdom/diff.go`) compares two `VNode` trees and produces a series of `Patch` objects.
*   **Two Render Targets, One Source of Truth:** The `Patch` objects are the universal language for UI updates. Two "appliers" consume them:
    *   `htmlApplier`: For Server-Side Rendering (SSR).
    *   `domApplier`: For client-side DOM manipulation in WASM.
This model completely eliminates parity issues between server and client rendering logic. (See: `ADR-0001-single-vdom.md`)

### 2.2. The Rendering Pipeline

Vango operates in three distinct rendering modes:

1.  **SSR (Server-Side Rendering):** The `htmlApplier` walks a `VNode` tree and streams HTML to an `io.Writer`. For nodes with event listeners, it injects a `data-hid` attribute, which is the key to hydration.
2.  **Hydration:** When the page loads, a minimal JavaScript bootstrap (`internal/assets/bootstrap.js`) loads the application WASM. This script builds a sparse "previous" `VNode` tree from the `data-hid` attributes in the server-rendered HTML. It then runs the root component to get the "next" tree and uses the `domApplier` to apply the resulting patches. This process attaches event listeners without re-rendering the DOM.
3.  **Live Updates (Post-Hydration):** All subsequent UI updates are driven by the server over a persistent WebSocket connection using the **Live Protocol**. The server re-runs components, diffs the `VNode` trees, and sends the resulting binary patches to the client, which applies them to the live DOM.

### 2.3. The Cooperative Scheduler

To manage client-side execution within TinyGo's memory constraints, Vango uses a cooperative, non-preemptive scheduler.
*   **Fibers:** Each component instance is a lightweight `fiber` struct, not a full OS goroutine.
*   **Single Goroutine:** A central scheduler loop (`pkg/scheduler/scheduler.go`) runs on a single real goroutine.
*   **Dirty Queue:** When a component's state changes, its fiber is marked "dirty" and added to a queue. The scheduler processes this queue, re-renders the component, generates patches, and applies them. This avoids the high memory overhead of thousands of goroutines.

### 2.4. State Management & Reactivity

Vango provides a powerful and expressive reactive state management system.
*   **`State[T]` / `Signal[T]`:** The basic reactive primitive. Calling `.Get()` within a component automatically subscribes that component's fiber to the signal. Calling `.Set()` marks all subscribed fibers as dirty, triggering a re-render.
*   **`Computed[T]`:** A memoized value derived from other signals. It only recalculates when its dependencies change.
*   **`Resource` & `Suspense`:** A primitive for handling asynchronous operations. A component can attempt to read from a `Resource`; if the data is not yet available, the scheduler pauses the fiber and can render a `Fallback` UI. If the operation fails, it can render an `Error` UI.
*   **`GlobalSignal[T]`:** A cross-session reactive value synced via the Live Protocol, enabling real-time collaboration features.
*   **Persistence:** A simple API (`persist.New()`) allows signals to be persisted to `LocalStorage`, `SessionStorage`, or other storage backends.

---

## 3. Developer Experience (DX)

Vango is designed to be productive and ergonomic for developers.

### 3.1. The `vango` CLI

A single command-line tool manages the entire development lifecycle.
*   `vango dev`: Starts a development server with file watching (`fsnotify`), incremental TinyGo compilation, and hot-reloading via WebSockets for both WASM and CSS.
*   `vango build`: Creates an optimized, production-ready build.
*   `vango create`: Scaffolds a new project from official examples.
*   `vango.json`: An optional configuration file for overriding default behaviors.

### 3.2. File-Based Routing

URL structure is defined by the file system layout in `app/routes/`.
*   `index.go` -> `/`
*   `blog/[slug].go` -> `/blog/:slug`
*   `_layout.go`: Wraps sibling and child routes with a shared layout component.
*   `_middleware.go`: Applies server-side middleware to a directory of routes.
*   `api/users.go`: Creates a JSON API endpoint at `/api/users`.
A code generator creates a fast radix-tree matcher, type-safe `Params` structs, and link-builder functions.

### 3.3. Component Syntax (VEX)

Vango offers a progressive syntax stack, allowing developers to choose their preferred level of abstraction. All layers compile to the same core `VNode` API.
*   **Layer 0 (Functional Go):** `vango.Div(nil, vango.Text("hello"))`
*   **Layer 1 (Fluent Builder):** `Div().Class("card").Children(Text("hello"))`
*   **Layer 2 (Template Macro):** A special `//vango:template` comment turns a Go file into an HTML-like template with Go expressions (`{{.Name}}`), event binding (`@click="inc()"`), and conditionals (`{{#if ...}}`). Props are declared with a `//vango:props` directive.

### 3.4. Styling

*   **Global CSS:** Standard `<link>`-based stylesheets are supported.
*   **Tailwind CSS:** If a `tailwind.config.js` is detected, `vango dev` automatically runs the Tailwind CLI in watch mode.
*   **Scoped Styles:** A `vango.Style()` function allows you to define CSS directly in your Go code. At build time, the CSS is extracted, class names are hashed to prevent conflicts, and the call site is rewritten with a helper that provides the hashed names.

---

## 4. Production Readiness & Ecosystem

Vango is built with production deployment in mind.

### 4.1. Observability

First-class support for debugging and monitoring.
*   **Structured Logging:** Uses Go's standard `slog` library with contextual fields (e.g., `request_id`, `fiber_id`).
*   **Metrics:** Exposes a `/metrics` endpoint for Prometheus scraping.
*   **Tracing:** Integrates with OpenTelemetry for distributed tracing, configurable via environment variables.

### 4.2. Security

Secure-by-default principles are built-in.
*   **CSRF Protection:** Double-submit cookie strategy for forms and token validation for WebSockets.
*   **Content Security Policy (CSP):** Middleware for generating and applying strict CSP headers.
*   **Dependency Integrity:** An `addons.lock` file with SHA-256 checksums prevents supply-chain attacks in the plugin system.

### 4.3. Plugin System (Addons)

The core is kept lean, with advanced features available through a secure addon system.
*   **`Addon` Interface:** Third-party Go modules can implement hooks that tap into the CLI and build process.
*   **`vango addon install`:** A CLI command to securely install addons, verifying their checksums.

### 4.4. Testing

A multi-layered testing strategy ensures reliability.
1.  **Unit Tests:** Standard `go test` for core logic (diffing, scheduling).
2.  **WASM DOM Tests:** A special test harness runs TinyGo-compiled tests in a headless browser to validate DOM interactions.
3.  **Integration/E2E Tests:** Playwright is used to drive real browser tests against example applications.
