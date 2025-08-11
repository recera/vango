package server

import (
	"fmt"
	
	"github.com/recera/vango/pkg/vango/vdom"
)

// InjectServerDrivenClient adds the minimal client script for server-driven components
func InjectServerDrivenClient(doc *vdom.VNode, sessionID string) *vdom.VNode {
	if doc == nil || doc.Kind != vdom.KindElement || doc.Tag != "html" {
		return doc
	}
	
	// Create meta tag for session ID
	sessionMeta := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "meta",
		Props: vdom.Props{
			"name": "vango-session",
			"content": sessionID,
		},
	}
	
	// Use the embedded minimal client script
	// In production, this would be loaded from internal/assets/server-driven-client.js
	scriptContent := []byte(getMinimalClientScript())
	
	// Create script element
	clientScript := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "script",
		Props: vdom.Props{
			"type": "text/javascript",
		},
		Kids: []vdom.VNode{
			{
				Kind: vdom.KindText,
				Text: string(scriptContent),
			},
		},
	}
	
	// Find head and body in the document
	for i := range doc.Kids {
		child := &doc.Kids[i]
		
		if child.Tag == "head" {
			// Add session meta to head
			child.Kids = append(child.Kids, *sessionMeta)
		}
		
		if child.Tag == "body" {
			// Add script to end of body
			child.Kids = append(child.Kids, *clientScript)
		}
	}
	
	return doc
}

// getMinimalClientScript returns a minimal fallback client script
func getMinimalClientScript() string {
	return fmt.Sprintf(`
// Minimal Vango server-driven client (fallback)
(function() {
    console.log('🔮 Vango Server-Driven Client (minimal)');
    
    const sessionID = document.querySelector('meta[name="vango-session"]')?.content || 
                     'session_' + Date.now();
    
    let ws = null;
    
    // Helper to read varint from DataView
    function readVarint(view, offset) {
        let value = 0;
        let shift = 0;
        while (offset < view.byteLength) {
            const byte = view.getUint8(offset++);
            value |= (byte & 0x7F) << shift;
            if ((byte & 0x80) === 0) {
                return { value, offset };
            }
            shift += 7;
        }
        return { value: 0, offset };
    }
    
    function connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(protocol + '//' + window.location.host + '/vango/live/' + sessionID);
        ws.binaryType = 'arraybuffer';
        
        ws.onopen = () => {
            console.log('✅ Connected to server');
            updateStatus(true);
        };
        
        ws.onmessage = (event) => {
            if (event.data instanceof ArrayBuffer) {
                // Handle binary patches from server
                const view = new DataView(event.data);
                const frameType = view.getUint8(0);
                
                console.log('📦 Received binary message, frame type:', frameType);
                
                if (frameType === 0x00) { // FramePatches - THIS WAS THE BUG!
                    console.log('🔧 Received patch frame');
                    
                    // Parse binary patch format
                    let offset = 1; // Skip frame type
                    
                    // Read patch count (varint)
                    const patchCount = readVarint(view, offset);
                    offset = patchCount.offset;
                    
                    console.log('📦 Patch count:', patchCount.value);
                    
                    for (let i = 0; i < patchCount.value; i++) {
                        // Read opcode
                        const opcode = view.getUint8(offset++);
                        console.log('🔨 Patch opcode:', opcode);
                        
                        if (opcode === 0x01) { // OpReplaceText
                            // Read node ID (varint)
                            const nodeId = readVarint(view, offset);
                            offset = nodeId.offset;
                            
                            // Read string value
                            const strLen = readVarint(view, offset);
                            offset = strLen.offset;
                            
                            const decoder = new TextDecoder();
                            const value = decoder.decode(new DataView(event.data, offset, strLen.value));
                            offset += strLen.value;
                            
                            console.log('✏️ ReplaceText: nodeId=' + nodeId.value + ', value="' + value + '"');
                            
                            // Update the counter display
                            const counter = document.getElementById('counter-display');
                            if (counter) {
                                counter.textContent = value;
                                counter.style.transform = 'scale(1.1)';
                                setTimeout(() => {
                                    counter.style.transform = 'scale(1)';
                                }, 200);
                                console.log('✅ Updated counter to:', value);
                            }
                        }
                    }
                } else if (frameType === 0x02) { // FrameControl
                    console.log('🎉 Control message (HELLO etc)');
                }
            } else if (typeof event.data === 'string') {
                // Legacy JSON handling
                const data = JSON.parse(event.data);
                if (data.type === 'update' && data.value !== undefined) {
                    const counter = document.getElementById('counter-display');
                    if (counter) {
                        counter.textContent = data.value;
                        counter.style.transform = 'scale(1.1)';
                        setTimeout(() => {
                            counter.style.transform = 'scale(1)';
                        }, 200);
                    }
                }
            }
        };
        
        ws.onclose = () => {
            console.log('❌ Disconnected');
            updateStatus(false);
            setTimeout(connect, 2000);
        };
    }
    
    function updateStatus(connected) {
        const status = document.getElementById('connection-status');
        if (status) {
            status.className = 'connection-status ' + (connected ? 'connected' : 'disconnected');
            status.textContent = connected ? 'Connected' : 'Disconnected';
        }
    }
    
    // Handle clicks on server elements
    document.addEventListener('click', (e) => {
        const target = e.target.closest('[data-server-event]');
        if (!target) return;
        
        e.preventDefault();
        const eventType = target.dataset.serverEvent;
        
        if (ws && ws.readyState === WebSocket.OPEN) {
            // Send properly formatted event:
            // [FrameEvent=0x01, EventType, NodeID as varint]
            const nodeId = parseInt(target.dataset.hid?.substring(1) || '0', 10) || 0;
            
            // Encode varint for node ID (simple version for small numbers)
            const encodeVarint = (n) => {
                const bytes = [];
                while (n >= 0x80) {
                    bytes.push((n & 0x7F) | 0x80);
                    n >>= 7;
                }
                bytes.push(n & 0x7F);
                return bytes;
            };
            
            const nodeIdBytes = encodeVarint(nodeId);
            const event = new Uint8Array([0x01, getEventCode(eventType), ...nodeIdBytes]);
            ws.send(event);
            console.log('📤 Sent event:', eventType, 'nodeId:', nodeId);
        }
    });
    
    function getEventCode(type) {
        const codes = {
            'increment': 0x02,
            'decrement': 0x03,
            'reset': 0x04
        };
        return codes[type] || 0x01;
    }
    
    // Initialize
    connect();
    
    // Add transition styles
    const style = document.createElement('style');
    style.textContent = %s;
    document.head.appendChild(style);
})();
`, "`#counter-display { transition: transform 0.2s ease; }`")
}