package live

// MessageType represents the type of live protocol message
type MessageType uint8

const (
	// Frame types
	FramePatches MessageType = 0x00
	FrameEvent   MessageType = 0x01
	FrameControl MessageType = 0x02
)

// EventType represents client-side event types
type EventType uint8

const (
	EventClick     EventType = 0x01
	EventIncrement EventType = 0x02
	EventDecrement EventType = 0x03
	EventReset     EventType = 0x04
	EventInput     EventType = 0x05
	EventSubmit    EventType = 0x06
)

// Event represents a client-side event
type Event struct {
	Type   EventType
	NodeID uint32
	Data   map[string]interface{}
}