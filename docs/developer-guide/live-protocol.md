# Live Protocol (Server-Driven) Deep Dive

Server-driven pages stream DOM patches over a binary WebSocket protocol.

## Endpoint and Session
- WS endpoint: `/vango/live/<sessionId>`
- Session ID is stored in a `<meta name="vango-session" content="...">` and generated if missing

## Frames (`pkg/live/types.go`)
- `FramePatches (0x00)`: `[0x00][patchCount varint][patch*]`
- `FrameEvent   (0x01)`: `[0x01][eventType u8][nodeId varint][data?]`
- `FrameControl (0x02)`: `[0x02][len+"HELLO"][resumable varint][lastSeq varint]` and other control strings (e.g., `PING`, `PONG`)

## Patch Opcodes (client applier)
- ReplaceText
- SetAttribute
- RemoveAttribute
- InsertNode (parent/before ids)
- RemoveNode
- UpdateEvents (bitmask)
- MoveNode

DOM elements are addressed by numeric IDs embedded as `data-hid="h<id>"`.

## Client Runtime (`internal/assets/server-driven-client.js`)
- Connects with exponential backoff
- Applies patches by decoding varints and strings
- Delegates DOM events (`click`, `input`, `submit`) to WS FrameEvent
- Maintains a small nodeId→Element cache map for performance

## Server (`pkg/live/server.go`)
- Manages sessions; writer goroutine handles pings and outbound frames
- `SendPatches([]vdom.Patch)` serializes and enqueues patches
- Event handling can bridge to a scheduler via `live.NewSchedulerBridge`

## Reconnect Behavior
- Client attempts reconnect with backoff on close
- Control frames can be extended to support resumable sessions (last sequence id)

## Security Considerations
- Implement proper origin checks and auth on WS upgrades before production
- Sanitize any custom event payloads; only server emits patches

## Debugging
- Enable verbose console logs in the minimal client
- Inspect received frame types and patch counts
- Watch server logs for session connect/disconnect and errors
