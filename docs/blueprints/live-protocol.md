---
title: Live Protocol
slug: live-protocol
version: 1.1
phase: P-0
status: ratified
requires:
  - ADR-0001-single-vdom
---

# Live Protocol Blueprint

> **Purpose**: Specify the binary patch format and transport rules used to propagate state changes from server-side components to hydrated clients.

## 1. Transport
* Default: WebSocket per browser session (`/vango/live/:sessID`).
* Future: WebTransport (`h3`) adapter behind same interface.

## 2. Opcode Stream
All integers are *unsigned varints* (LEB128) for size. String payloads are UTF-8 length-prefixed varints.

| Opcode | Payload | Meaning |
|--------|---------|---------|
| 0x01 | nodeID, text | Replace text node content |
| 0x02 | nodeID, attrKey, attrVal | Set/replace attribute |
| 0x03 | nodeID | Remove node |
| 0x04 | parentID, beforeID, serialized subtree | Insert new node |
| 0x05 | nodeID, eventBits | Update event subscription |

*`nodeID`* is `uint32` assigned during hydration (`data-hid`).

## 3. Framing
Each patch burst is: `[frameLen varint][ops...]`. Allows multiplexing over shared WS with future RPC.

## 4. Compression
Outgoing server stream passes through `sync/flate` w/ `DefaultCompression`. Browser uses `Compression Streams API` when available.

## 5. Reliability & Reconnection
* At-most-once semantics; idempotent opcodes safe to replay. Lost frames cause soft resync: client requests full component subtree (`REFETCH nodeID`).
* **Disconnect Handling**:
  1. Client listens to `close` event; sets `state = disconnected` and fires `window.__vangoLive_offline`.
  2. Exponential back-off reconnect: `1s, 2s, 5s, 10s, max 30s`.
  3. Upon reconnect, client sends `HELLO resumable=true lastSeq=<n>` allowing server to resume stream. If `lastSeq` low-watermark expired, server replies `FULL-RESYNC` and streams HTML diff for root.
  4. UI fallback: CSS class `vango-offline` toggled on `<body>` so apps can show toast – default stylesheet includes subtle banner.


## 6. Back-Pressure
Server monitors `conn.BufferedAmount()` (from Gorilla WS). If >32 kB, batch further diffs until below threshold.

## 7. DevTools Hook
Debug builds add opcode logging with human-readable names, available under `window.__VangoLiveTap`.

## 8. Security
* Frame is encrypted by TLS (wss).  
* nodeID mapping lives per session; no global IDs → mitigates XS-Leak.

## 9. Developer Usage
### 9.1 Server Side
```go
liveConn := live.Acquire(ctx)                  // pkg/live
liveConn.Subscribe(componentID)                // push patches automatically
```
*Call `live.MarkDirty(fiber)` after state change — scheduler handles batching.*

### 9.2 Client Side Stub
```js
import { applyPatch } from '/vango/runtime-dom.js'
const ws = new VangoLiveSocket('/vango/live/'+session)
ws.onPatch = applyPatch
```
`VangoLiveSocket` auto-reconnects and dispatches `vango-offline` CSS toggle.

### 9.3 Testing
Use `internal/testharness/live` to spin up a fake WS server and assert patch ordering.

## 10. Implementation Guide (Phase P-0 tasks)
| Task | File | Owner |
|------|------|-------|
| Binary encoder/decoder | `pkg/live/codec.go` | @alice |
| Go WS handler | `pkg/live/server.go` | @bob |
| TinyGo patch applier | `pkg/renderer/dom/apply.go` | @carol |
| JS bootstrap client | `internal/assets/bootstrap.js` | @dave |

CI job `make test-live` runs against headless Chrome (`wazero` polyfill) verifying a 10k-patch burst ≤ 50 ms.

## 11. Open Questions
* Should we CRC each frame for integrity? WebSocket already has mask; unsure.
* Binary choice: Protobuf vs custom varint – custom currently smaller.

## 12. Changelog
| Date | Version | Notes |
|------|---------|-------|
|2025-08-05|0.1|Initial draft|
