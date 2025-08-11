---
title: Observability & Telemetry
slug: observability
version: 0.1
phase: Q2-2025
status: draft
requires:
  - cooperative-scheduler
---

# Observability Blueprint

> **Goal**: Provide first-class metrics, tracing, and logging across SSR, WASM, and Live channels.

## 1. Structured Logging
* Uses Go 1.22 `slog` by default; adapter for Zap.  
* Contextual fields: `req.id`, `component`, `fiber.id`, `opBytes`.

## 2. Metrics
| Metric | Label Set | Exporter |
|--------|-----------|----------|
| `vango_http_ttfb_ms` | route | Prometheus |
| `vango_live_patch_bytes` | opcode | Prometheus |
| `vango_fiber_active` | | Prometheus |
| `vango_wasm_alloc_bytes` | | DevTools panel |

### 2.1 Implementation
Prometheus registry under `/metrics`. WASM side collects counters and flushes via Live WS frame `METRICS` every 5 s.

## 3. Tracing
* OTLP spans:  
  – `ssr.render` (attributes: nodeCount)  
  – `scheduler.commit` (fiber.id)  
  – `live.patch` (bytes, ops)  
  – `wasm.hydrate`.
* Users enable with `export OTEL_EXPORTER_OTLP_ENDPOINT`.

## 4. DevTools Inspector
* Browser extension adds panel:  
  – reactive graph visualization  
  – fiber timeline flamechart  
  – Live patch log with opcode decode.

## 5. Testing & CI
`go test -run TestTraceExport` ensures spans are emitted with correct attrs.

## 6. Open Questions
* Should we ship a built-in Jaeger UI in dev server?  
* Privacy default—disable metrics in production unless env var set?

## 7. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
