//go:build !wasm
// +build !wasm

package live

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/recera/vango/pkg/vango/vdom"
)

// Server handles WebSocket connections for live updates
type Server struct {
	upgrader websocket.Upgrader
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Session represents a live connection session
type Session struct {
	ID         string
	conn       *websocket.Conn
	state      map[string]interface{}
	lastSeq    uint64
	sendChan   chan []byte
	sendTextChan chan []byte  // For JSON/text messages
	closeChan  chan struct{}
	mu         sync.RWMutex
}

// NewServer creates a new live protocol server
func NewServer() *Server {
	return &Server{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		sessions: make(map[string]*Session),
	}
}

// HandleWebSocket handles WebSocket upgrade and session management
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path
	sessionID := r.URL.Path[len("/vango/live/"):]
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Live Server] Failed to upgrade connection: %v", err)
		return
	}

	// Create or get session
	session := s.getOrCreateSession(sessionID, conn)
	
	// Handle the session
	go session.handleConnection()
}

// getOrCreateSession gets an existing session or creates a new one
func (s *Server) getOrCreateSession(sessionID string, conn *websocket.Conn) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		// Update connection for existing session
		session.mu.Lock()
		if session.conn != nil {
			session.conn.Close()
		}
		session.conn = conn
		// Reset the close channel if needed
		select {
		case <-session.closeChan:
			// Channel was closed, create a new one
			session.closeChan = make(chan struct{})
		default:
			// Channel is still open
		}
		session.mu.Unlock()
		return session
	}

	// Create new session
	session := &Session{
		ID:        sessionID,
		conn:      conn,
		state:     make(map[string]interface{}),
		sendChan:  make(chan []byte, 256),
		sendTextChan: make(chan []byte, 256),
		closeChan: make(chan struct{}),
	}
	s.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID
func (s *Server) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[sessionID]
	return session, exists
}

// RemoveSession removes a session
func (s *Server) RemoveSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// handleConnection manages the WebSocket connection for a session
func (s *Session) handleConnection() {
	// Ensure cleanup happens only once
	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			s.conn.Close()
			// Signal writer to stop
			select {
			case <-s.closeChan:
				// Already closed
			default:
				close(s.closeChan)
			}
		})
	}
	defer cleanup()

	// Start writer goroutine
	writerReady := make(chan struct{})
	go func() {
		close(writerReady)
		s.writer()
	}()
	
	// Wait for writer to be ready
	<-writerReady
	
	// Send hello message after writer is running
	s.sendHello()
	log.Printf("[Live Session %s] Sent server HELLO", s.ID)
	
	// Create scheduler session if we have a bridge
	if bridge := GetBridge(); bridge != nil {
		log.Printf("[Live Session %s] Creating scheduler session", s.ID)
		bridge.CreateSessionScheduler(s.ID)
	}

	// Set up ping/pong to detect disconnects - but with longer initial timeout
	s.conn.SetReadDeadline(time.Now().Add(300 * time.Second)) // 5 minutes initially
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	// Read messages
	for {
		messageType, data, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Live Session %s] Unexpected close: %v", s.ID, err)
			} else {
				log.Printf("[Live Session %s] Read error: %v", s.ID, err)
			}
			break
		}

		log.Printf("[Live Session %s] Received message type %d, size %d bytes", s.ID, messageType, len(data))
		
		if messageType == websocket.BinaryMessage {
			s.handleBinaryMessage(data)
		} else if messageType == websocket.TextMessage {
			s.handleTextMessage(data)
		}
	}
}

// writer handles writing messages to the WebSocket
func (s *Session) writer() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-s.sendChan:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			log.Printf("[Live Session %s] Writing binary message to WebSocket: %d bytes", s.ID, len(message))
			if err := s.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Printf("[Live Session %s] Failed to write message: %v", s.ID, err)
				return
			}
			log.Printf("[Live Session %s] Binary message sent successfully", s.ID)
			
		case message, ok := <-s.sendTextChan:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				return
			}

			log.Printf("[Live Session %s] Writing text message to WebSocket: %d bytes", s.ID, len(message))
			if err := s.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Live Session %s] Failed to write text message: %v", s.ID, err)
				return
			}
			log.Printf("[Live Session %s] Text message sent successfully", s.ID)

		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-s.closeChan:
			return
		}
	}
}

// sendHello sends the initial hello message
func (s *Session) sendHello() {
	// Create hello message
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	
	// Write control frame type
	encoder.WriteBytes([]byte{byte(FrameControl)})
	encoder.WriteString("HELLO")
	encoder.WriteUvarint(s.lastSeq)
	
	helloBytes := buf.Bytes()
	log.Printf("[Live Session %s] Sending HELLO message: %d bytes, hex: %x", s.ID, len(helloBytes), helloBytes)
	s.sendChan <- helloBytes
}

// handleBinaryMessage processes binary protocol messages
func (s *Session) handleBinaryMessage(data []byte) {
	if len(data) == 0 {
		return
	}

	frameType := MessageType(data[0])
	
	switch frameType {
	case FrameEvent:
		// Decode event
		event, err := DecodeEvent(data)
		if err != nil {
			log.Printf("[Live Session %s] Failed to decode event: %v", s.ID, err)
			return
		}
		s.handleEvent(event)
		
	case FrameControl:
		// Handle control messages
		decoder := NewDecoder(bytes.NewReader(data[1:]))
		msgType, err := decoder.ReadString()
		if err != nil {
			log.Printf("[Live Session %s] Failed to decode control message type: %v", s.ID, err)
			return
		}
		
		switch msgType {
		case "HELLO":
			resumable, err1 := decoder.ReadUvarint()
			lastSeq, err2 := decoder.ReadUvarint()
			if err1 != nil || err2 != nil {
				log.Printf("[Live Session %s] Failed to decode HELLO params: %v, %v", s.ID, err1, err2)
				return
			}
			log.Printf("[Live Session %s] Client hello: resumable=%v, lastSeq=%d", s.ID, resumable > 0, lastSeq)
			
		case "PING":
			// Send pong
			s.sendControl("PONG")
		}
	}
}

// handleTextMessage processes text protocol messages (for debugging)
func (s *Session) handleTextMessage(data []byte) {
	log.Printf("[Live Session %s] Text message: %s", s.ID, string(data))
}

// handleEvent processes client events
func (s *Session) handleEvent(event *Event) {
	log.Printf("[Live Session %s] Event: type=%v, nodeID=%d", s.ID, event.Type, event.NodeID)
	
	// Check if we have a scheduler bridge
	if bridge := GetBridge(); bridge != nil {
		// Convert event type to string
		var eventTypeStr string
		switch event.Type {
		case EventClick:
			eventTypeStr = "click"
		case EventIncrement:
			eventTypeStr = "increment"
		case EventDecrement:
			eventTypeStr = "decrement"
		case EventReset:
			eventTypeStr = "reset"
		default:
			eventTypeStr = "unknown"
		}
		
		// Route event to component via bridge
		if err := bridge.HandleComponentEvent(s.ID, event.NodeID, eventTypeStr); err != nil {
			log.Printf("[Live Session %s] Failed to handle component event: %v", s.ID, err)
			// Fall back to legacy handling
			s.handleEventLegacy(event)
		}
		// Component will handle state update and scheduler will send patches
		return
	}
	
	// Fall back to legacy event handling
	s.handleEventLegacy(event)
}

// handleEventLegacy is the old hardcoded event handler (for backward compatibility)
func (s *Session) handleEventLegacy(event *Event) {
	// Handle counter events with hardcoded logic
	var newValue int
	currentValue, exists := s.state["counter"]
	if exists {
		if val, ok := currentValue.(int); ok {
			newValue = val
		}
	}
	
	switch event.Type {
	case EventIncrement:
		newValue++
		s.state["counter"] = newValue
		log.Printf("[Live Session %s] Counter incremented to %d (legacy)", s.ID, newValue)
		
	case EventDecrement:
		newValue--
		s.state["counter"] = newValue
		log.Printf("[Live Session %s] Counter decremented to %d (legacy)", s.ID, newValue)
		
	case EventReset:
		newValue = 0
		s.state["counter"] = newValue
		log.Printf("[Live Session %s] Counter reset to %d (legacy)", s.ID, newValue)
		
	default:
		log.Printf("[Live Session %s] Unknown event type: %v", s.ID, event.Type)
		return
	}
	
	// Send update back to client
	// For now, send a simple JSON update (in production, this would be binary patches)
	update := map[string]interface{}{
		"type": "update",
		"value": newValue,
	}
	
	if data, err := json.Marshal(update); err == nil {
		select {
		case s.sendTextChan <- data:
			log.Printf("[Live Session %s] Sent update: %d (legacy)", s.ID, newValue)
		default:
			log.Printf("[Live Session %s] Send buffer full, dropping update", s.ID)
		}
	}
}

// sendControl sends a control message
func (s *Session) sendControl(msgType string) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	
	encoder.WriteBytes([]byte{byte(FrameControl)})
	encoder.WriteString(msgType)
	
	s.sendChan <- buf.Bytes()
}

// SendPatches sends a batch of patches to the client
func (s *Session) SendPatches(patches []vdom.Patch) error {
	if len(patches) == 0 {
		return nil
	}

	// Encode patches
	data, err := EncodePatches(patches)
	if err != nil {
		return fmt.Errorf("failed to encode patches: %w", err)
	}

	// Send via channel (non-blocking)
	select {
	case s.sendChan <- data:
		s.lastSeq++
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// EncodePatches encodes patches to binary format
func EncodePatches(patches []vdom.Patch) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	
	// Write frame type
	encoder.WriteBytes([]byte{byte(FramePatches)})
	
	// Write patch count
	encoder.WriteUvarint(uint64(len(patches)))
	
	// Write each patch
	for _, patch := range patches {
		// Write opcode
		encoder.WriteBytes([]byte{byte(patch.Op)})
		
		switch patch.Op {
		case vdom.OpReplaceText:
			encoder.WriteUvarint(uint64(patch.NodeID))
			encoder.WriteString(patch.Value)
			
		case vdom.OpSetAttribute:
			encoder.WriteUvarint(uint64(patch.NodeID))
			encoder.WriteString(patch.Key)
			encoder.WriteString(patch.Value)
			
		case vdom.OpRemoveAttribute:
			encoder.WriteUvarint(uint64(patch.NodeID))
			encoder.WriteString(patch.Key)
			
		case vdom.OpRemoveNode:
			encoder.WriteUvarint(uint64(patch.NodeID))
			
		case vdom.OpInsertNode:
			encoder.WriteUvarint(uint64(patch.NodeID))
			encoder.WriteUvarint(uint64(patch.ParentID))
			encoder.WriteUvarint(uint64(patch.BeforeID))
			// TODO: Serialize VNode tree
			
		case vdom.OpUpdateEvents:
			encoder.WriteUvarint(uint64(patch.NodeID))
			tmp := make([]byte, 4)
			binary.LittleEndian.PutUint32(tmp, patch.EventBits)
			encoder.WriteBytes(tmp)
			
		case vdom.OpMoveNode:
			encoder.WriteUvarint(uint64(patch.NodeID))
			encoder.WriteUvarint(uint64(patch.ParentID))
			encoder.WriteUvarint(uint64(patch.BeforeID))
		}
	}
	
	return buf.Bytes(), nil
}

// UpdateState updates the session state and generates patches
func (s *Session) UpdateState(key string, value interface{}) {
	s.mu.Lock()
	oldValue := s.state[key]
	s.state[key] = value
	s.mu.Unlock()
	
	// In a real implementation, this would:
	// 1. Re-render the affected components
	// 2. Diff the old and new VNode trees
	// 3. Generate patches
	// 4. Send patches to client
	
	log.Printf("[Live Session %s] State updated: %s = %v (was %v)", s.ID, key, value, oldValue)
}

// GetState retrieves a state value
func (s *Session) GetState(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.state[key]
	return value, exists
}