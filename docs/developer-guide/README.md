# Vango Developer Guide Suite

This directory contains deep-dive documentation intended for developers and AI code editors. Each guide links directly to code and architecture to make implementation decisions clear and predictable.

Guides
- rendering-modes.md — Universal, Server-Driven, Client-Only; selection guide, lifecycles, examples
- vex-syntax.md — Layer 1 Builder (primary), Layer 0 Functional, Layer 2 VEX templates; full syntax and examples
- components-and-vdom.md — VNode model, render cycle, diff/patch ops, events, performance
- state-management.md — signals, computed, batching, patterns
- routing-and-codegen.md — file-based routing, typed params, layouts/middleware, generated artifacts
- styling.md — Tailwind strategies, global/scoped CSS, dev/prod
- dev-server.md — flags, endpoints, hot reload, diagnostics
- live-protocol.md — frames, patches, client/server responsibilities, security
- client-bootstrap-and-csr.md — bootstrap flow, navigation, prefetch, HMR
- configuration.md — vango.json fields with a full example
- testing.md — unit, wasm DOM, server-driven, E2E tips
- cookbook.md — task recipes (layouts, forms, counters, API routes, dark mode)

Start here
- New to Vango? Read `rendering-modes.md` → `vex-syntax.md` → `routing-and-codegen.md` → `state-management.md`
- Building a real app? Skim `styling.md`, `dev-server.md`, and keep `cookbook.md` open
