title: Q1 2025 State Store & DevTools
slug: Q1-2025-state-store
version: 1.1
status: ratified
---

# Phase Q1 2025: State Store, Computed, DevTools

Timeline: **Jan–Mar 2025**

**Corresponds to:** [P-2 Advanced State, Real-Time & Security](P-2-Build-Plan-Advanced-Features.md)

## 1. Goals
- **Advanced State:** Implement `Computed[T]`, `GlobalSignal[T]`, and the `Resource` primitive with `Suspense` for asynchronous operations.
- **DevTools:** Ship an MVP browser extension that can inspect the Live Protocol patch stream and visualize the reactive dependency graph.
- **Real-Time Backend:** Upgrade the Live Protocol to handle broadcast deltas for `GlobalSignal` and implement the client-side reconnection logic.
- **Persistence:** Implement the state persistence layer for saving signals to `LocalStorage`.

## 2. Milestones & Deliverables
| Week | Deliverable | Corresponding Epic/Task |
|------|-------------|-------------------------|
| 1-2  | `Computed[T]` prototype with cycle detection & benchmarks. | P-2, Epic 2, Task 2.1 |
| 3-4  | `Resource` & `Suspense` implementation with error handling. | P-2, Epic 2, Task 2.2 |
| 5-6  | `GlobalSignal[T]` with WebSocket fan-out; Live Protocol reliability. | P-2, Epic 1 & Epic 2, Task 2.3 |
| 7-8  | DevTools MVP browser extension showing Live patch log. | P-3, Epic 1, Task 1.4 (overlaps) |
| 9-10 | State persistence layer (`persist.New`) implemented. | P-2, Epic 2, Task 2.4 |
| 11-12| Docs for all new state features & a `todo-mvc` example. | P-3, Epic 4, Task 4.3 (overlaps) |

## 3. Risks
*   **Graph cycles:** Add robust cycle detection in `Computed` and provide clear error messages.
*   **DevTools security:** Ensure the DevTools WebSocket endpoint is only exposed in dev mode (`vango dev`) and not in production builds.
*   **Real-time race conditions:** The initial `GlobalSignal` will use a last-write-wins strategy; document this limitation clearly.

## 4. Owners
- Lead: @dave