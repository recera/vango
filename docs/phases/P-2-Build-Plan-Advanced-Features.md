title: "Build Plan: P-2 Advanced State, Real-Time & Security"
slug: P-2-Build-Plan-Advanced-Features
version: 1.1
status: ratified
requires:
  - P-1-Build-Plan-DX-Expansion
---

# Engineering Build Plan: Phase 2 - Advanced State, Real-Time & Security

**Document Purpose:** This document provides the detailed engineering plan for Phase 2. With a solid developer experience and core runtime in place, this phase focuses on building advanced capabilities that enable complex, real-time applications. It also introduces foundational security measures and a robust end-to-end testing strategy.

**Phase Goal:** To implement the Live Protocol for real-time server-client communication, build a sophisticated state management library, and establish a baseline of security and testing. This corresponds to the **Q1 2025 State Store & DevTools** milestone.

---

## Epic 1: The Live Protocol

**What & Why:** The nervous system of a running Vango application. After the initial SSR and hydration, all subsequent UI updates are driven by the server sending diff patches over a persistent WebSocket connection.

### Task 1.1: Opcode Serialization & Deserialization

*   **File:** `pkg/live/protocol.go`
*   **Plan:** Implement the binary serialization logic (using unsigned varints) for converting `vdom.Patch` structs into the specified opcode byte stream, and vice-versa.
*   **Testing:** Write unit tests that serialize and deserialize `vdom.Patch` structs to ensure the binary format is correct.

### Task 1.2: Server-Side WebSocket Hub

*   **File:** `pkg/live/server.go`
*   **Plan:** Create a WebSocket handler and a connection hub. When a component's state changes, generate patches and write them to the appropriate client connections, respecting the back-pressure mechanism (`conn.BufferedAmount()`) defined in the blueprint.

### Task 1.3: Client-Side Reliability & Reconnection

*   **File:** `internal/assets/bootstrap.js`
*   **Plan:** Implement the full disconnect/reconnect logic as specified in `live-protocol.md`, including exponential backoff, the `HELLO resumable=true` handshake, and toggling the `.vango-offline` CSS class on the `<body>`.

### Task 1.4: Implement Soft Resync

*   **Files:** `pkg/live/client.go`, `pkg/live/server.go`
*   **Plan:** Implement the `REFETCH nodeID` mechanism. If the client receives a patch for an unknown node, it sends a `REFETCH` request. The server responds by re-rendering and sending the full subtree for that component.

---

## Epic 2: Advanced State Management

**What & Why:** Powerful primitives for managing complex data flows, asynchronous operations, and real-time collaboration.

### Task 2.1: Implement `Computed[T]`

*   **File:** `pkg/reactive/computed.go`
*   **Plan:** Implement the memoized value primitive, including dynamic dependency tracking based on `currentFiber` and, crucially, **cycle detection** to prevent infinite loops.

### Task 2.2: Implement `Resource` and `Suspense`

*   **Files:** `pkg/reactive/resource.go`, `pkg/scheduler/scheduler.go` (enhancement)
*   **Plan:**
    1.  Implement the `Resource` primitive for async operations.
    2.  Enhance the scheduler to pause and resume fibers that yield a `SuspensePromise`.
    3.  Implement the error handling path: `users.Value()` must return `(nil, err)` if the fetch fails, and the `vango.Suspense` component must correctly render its `Error` slot, as defined in `state-management.md`.

### Task 2.3: Implement `GlobalSignal[T]`

*   **Files:** `pkg/reactive/global.go`, `pkg/live/server.go`
*   **Plan:** Implement the global state primitive. A `Set()` on the server will trigger a message broadcast over the Live Protocol to all subscribed clients. For this phase, the conflict resolution strategy will be **last-write-wins**.

### Task 2.4: Implement State Persistence Layer

*   **File:** `pkg/reactive/persist.go`
*   **Plan:**
    1.  Define the `Storage` interface (`Load`/`Save`).
    2.  Create `LocalStorage` and `SessionStorage` implementations using `syscall/js`.
    3.  Implement the `persist.New()` function that loads initial state and subscribes to subsequent changes to save them.
*   **Testing:** Write WASM-based tests using a mock storage implementation to verify the save/load cycle.

---

## Epic 3: Foundational Security

**What & Why:** As Vango becomes more powerful, security becomes non-negotiable. This epic implements baseline security features from the `security.md` blueprint.

### Task 3.1: Implement CSRF Protection

*   **Files:** `pkg/server/middleware/csrf.go`, `pkg/live/server.go`
*   **Plan:** Implement the double-submit cookie strategy for forms and the token-based validation for WebSocket connections, using the `vango.Ctx` object to manage tokens.
*   **Testing:** Write integration tests that attempt form submissions and WebSocket connections with and without the correct CSRF token.

### Task 3.2: Implement CSP Middleware

*   **File:** `pkg/server/middleware/csp.go`
*   **Plan:** Create middleware to add the `Content-Security-Policy` header, including support for the per-request nonce workflow via `vango.GetCSPNonce(ctx)`.

---

## Epic 4: End-to-End (E2E) Testing Framework

**What & Why:** The highest level of confidence in the framework's correctness, validating full user flows in a real browser.

### Task 4.1: Implement WASM Test Harness

*   **File:** `internal/testharness/`
*   **Plan:** Build the proof-of-concept test harness as specified in `testing.md`. This is a critical, unblocking task for all WASM-based DOM testing. It will involve launching a headless browser and streaming test results back to the Go process.

### Task 4.2: Setup Playwright and CI Integration

*   **File:** `test/playwright/`, `.github/workflows/ci.yml`
*   **Plan:** Initialize the Playwright project and configure the CI pipeline to run the E2E test suite against the example applications on every commit.

---

**Phase 2 Success:** At the end of this phase, Vango is a feature-rich framework capable of building dynamic, real-time applications. The Live Protocol is operational, the state management library is powerful and expressive, and the project is protected by a solid baseline of security and E2E tests.