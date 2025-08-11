//go:build js && wasm
// +build js,wasm

package live

import "encoding/binary"

// EncodeEvent encodes an event to binary format (WASM version)
func EncodeEvent(evt Event) []byte {
	var buf []byte
	
	// Frame type
	buf = append(buf, byte(FrameEvent))
	
	// Event type
	buf = append(buf, byte(evt.Type))
	
	// Node ID
	buf = appendUvarint(buf, uint64(evt.NodeID))
	
	// TODO: Encode additional event data
	
	return buf
}

// Helper function to append uvarint to byte slice
func appendUvarint(buf []byte, v uint64) []byte {
	tmp := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(tmp, v)
	return append(buf, tmp[:n]...)
}