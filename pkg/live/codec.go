//go:build !wasm
// +build !wasm

package live

import (
	"encoding/binary"
	"errors"
	"io"
)

// Encoder handles encoding of live protocol messages
type Encoder struct {
	w io.Writer
}

// NewEncoder creates a new encoder
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// WriteUvarint writes an unsigned varint
func (e *Encoder) WriteUvarint(v uint64) error {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, v)
	_, err := e.w.Write(buf[:n])
	return err
}

// WriteString writes a length-prefixed string
func (e *Encoder) WriteString(s string) error {
	if err := e.WriteUvarint(uint64(len(s))); err != nil {
		return err
	}
	_, err := e.w.Write([]byte(s))
	return err
}

// WriteBytes writes raw bytes
func (e *Encoder) WriteBytes(b []byte) error {
	_, err := e.w.Write(b)
	return err
}

// Decoder handles decoding of live protocol messages
type Decoder struct {
	r   io.Reader
	buf []byte
}

// NewDecoder creates a new decoder
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:   r,
		buf: make([]byte, 1024),
	}
}

// ReadUvarint reads an unsigned varint
func (d *Decoder) ReadUvarint() (uint64, error) {
	return binary.ReadUvarint(d)
}

// ReadByte implements io.ByteReader
func (d *Decoder) ReadByte() (byte, error) {
	var b [1]byte
	_, err := d.r.Read(b[:])
	return b[0], err
}

// ReadString reads a length-prefixed string
func (d *Decoder) ReadString() (string, error) {
	length, err := d.ReadUvarint()
	if err != nil {
		return "", err
	}
	
	if length > uint64(len(d.buf)) {
		d.buf = make([]byte, length)
	}
	
	n, err := io.ReadFull(d.r, d.buf[:length])
	if err != nil {
		return "", err
	}
	
	return string(d.buf[:n]), nil
}

// ReadBytes reads n bytes
func (d *Decoder) ReadBytes(n int) ([]byte, error) {
	if n > len(d.buf) {
		d.buf = make([]byte, n)
	}
	
	_, err := io.ReadFull(d.r, d.buf[:n])
	if err != nil {
		return nil, err
	}
	
	result := make([]byte, n)
	copy(result, d.buf[:n])
	return result, nil
}

// EncodeEvent encodes an event to binary format
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

// DecodeEvent decodes an event from binary format
func DecodeEvent(data []byte) (*Event, error) {
	if len(data) < 3 {
		return nil, errors.New("event data too short")
	}
	
	// Check frame type
	if data[0] != byte(FrameEvent) {
		return nil, errors.New("not an event frame")
	}
	
	evt := &Event{
		Type: EventType(data[1]),
	}
	
	// Decode node ID
	nodeID, n := binary.Uvarint(data[2:])
	if n <= 0 {
		return nil, errors.New("failed to decode node ID")
	}
	evt.NodeID = uint32(nodeID)
	
	// TODO: Decode additional event data
	
	return evt, nil
}

// Helper function to append uvarint to byte slice
func appendUvarint(buf []byte, v uint64) []byte {
	tmp := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(tmp, v)
	return append(buf, tmp[:n]...)
}