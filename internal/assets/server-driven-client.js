// Minimal client runtime for server-driven components (~3KB gzipped)
// This handles WebSocket connection and patch application

(function() {
    'use strict';
    
    console.log('🔮 Vango Server-Driven Client v0.1');
    
    // Configuration
    const config = {
        reconnectDelay: 1000,
        maxReconnectDelay: 30000,
        heartbeatInterval: 30000
    };
    
    // State
    let ws = null;
    let sessionID = null;
    let reconnectTimer = null;
    let reconnectDelay = config.reconnectDelay;
    let heartbeatTimer = null;
    let nodeMap = new Map(); // Maps data-hid to DOM elements
    
    // Initialize session ID
    function initSession() {
        // Check if we have a session ID in meta tag
        const meta = document.querySelector('meta[name="vango-session"]');
        if (meta) {
            sessionID = meta.content;
        } else {
            // Generate a new session ID
            sessionID = 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        }
        console.log('📍 Session ID:', sessionID);
    }
    
    // Build node map from data-hid attributes
    function buildNodeMap() {
        nodeMap.clear();
        const elements = document.querySelectorAll('[data-hid]');
        elements.forEach(el => {
            const hid = el.dataset.hid;
            nodeMap.set(hid, el);
            // Also store numeric ID if present
            const numId = parseInt(hid);
            if (!isNaN(numId)) {
                nodeMap.set(numId, el);
            }
        });
        console.log('🗺️ Built node map with', nodeMap.size, 'entries');
    }
    
    // Connect to WebSocket
    function connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${window.location.host}/vango/live/${sessionID}`;
        
        console.log('🔌 Connecting to', url);
        ws = new WebSocket(url);
        ws.binaryType = 'arraybuffer';
        
        ws.onopen = handleOpen;
        ws.onmessage = handleMessage;
        ws.onclose = handleClose;
        ws.onerror = handleError;
    }
    
    // Handle WebSocket open
    function handleOpen() {
        console.log('✅ Connected to server');
        reconnectDelay = config.reconnectDelay; // Reset delay
        
        // Update connection status
        updateConnectionStatus(true);
        
        // Send hello message
        sendControl('HELLO', { resumable: false });
        
        // Start heartbeat
        startHeartbeat();
        
        // Clear reconnect timer
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
    }
    
    // Handle incoming messages
    function handleMessage(event) {
        if (typeof event.data === 'string') {
            // JSON message (legacy/debug)
            handleJSONMessage(JSON.parse(event.data));
        } else {
            // Binary message (patches)
            handleBinaryMessage(new Uint8Array(event.data));
        }
    }
    
    // Handle JSON messages
    function handleJSONMessage(data) {
        console.log('📥 JSON message:', data);
        
        if (data.type === 'update' && data.value !== undefined) {
            // Legacy counter update
            const counter = document.getElementById('counter-display') || 
                           document.getElementById('server-counter');
            if (counter) {
                counter.textContent = data.value;
                animateElement(counter);
            }
        }
    }
    
    // Handle binary messages (patches)
    function handleBinaryMessage(data) {
        if (data.length === 0) return;
        
        const frameType = data[0];
        console.log('📥 Binary frame type:', frameType);
        
        switch (frameType) {
            case 0x00: // Patches
                applyPatches(data);
                break;
            case 0x02: // Control
                handleControl(data);
                break;
            default:
                console.warn('Unknown frame type:', frameType);
        }
    }
    
    // Apply patches to DOM
    function applyPatches(data) {
        const view = new DataView(data.buffer);
        let offset = 1; // Skip frame type
        
        // Read patch count
        const patchCount = view.getUint8(offset++);
        console.log('🔧 Applying', patchCount, 'patches');
        
        for (let i = 0; i < patchCount; i++) {
            const opcode = view.getUint8(offset++);
            
            switch (opcode) {
                case 0x01: // ReplaceText
                    const nodeId = readVarInt(view, offset);
                    offset += nodeId.bytes;
                    const textLen = view.getUint8(offset++);
                    const text = new TextDecoder().decode(
                        data.slice(offset, offset + textLen)
                    );
                    offset += textLen;
                    
                    const node = nodeMap.get(nodeId.value) || 
                                document.querySelector(`[data-hid="${nodeId.value}"]`);
                    if (node) {
                        node.textContent = text;
                        animateElement(node);
                        console.log('📝 Updated text for node', nodeId.value, ':', text);
                    }
                    break;
                    
                case 0x02: // SetAttribute
                    // TODO: Implement attribute updates
                    console.log('SetAttribute patch (not implemented)');
                    break;
                    
                // Add other patch types as needed
                default:
                    console.warn('Unknown patch opcode:', opcode);
            }
        }
    }
    
    // Read variable-length integer (LEB128)
    function readVarInt(view, offset) {
        let value = 0;
        let shift = 0;
        let bytes = 0;
        let byte;
        
        do {
            byte = view.getUint8(offset + bytes);
            value |= (byte & 0x7F) << shift;
            shift += 7;
            bytes++;
        } while ((byte & 0x80) !== 0);
        
        return { value, bytes };
    }
    
    // Handle control messages
    function handleControl(data) {
        // Parse control message
        const decoder = new TextDecoder();
        const message = decoder.decode(data.slice(1));
        console.log('🎮 Control message:', message);
    }
    
    // Handle WebSocket close
    function handleClose() {
        console.log('❌ Disconnected from server');
        ws = null;
        
        // Update connection status
        updateConnectionStatus(false);
        
        // Stop heartbeat
        stopHeartbeat();
        
        // Schedule reconnection with exponential backoff
        reconnectTimer = setTimeout(() => {
            console.log('🔄 Attempting to reconnect...');
            connect();
        }, reconnectDelay);
        
        // Increase delay for next attempt (exponential backoff)
        reconnectDelay = Math.min(reconnectDelay * 2, config.maxReconnectDelay);
    }
    
    // Handle WebSocket error
    function handleError(error) {
        console.error('🔴 WebSocket error:', error);
    }
    
    // Send control message
    function sendControl(type, data) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        
        const message = JSON.stringify({ type, ...data });
        ws.send(message);
        console.log('📤 Sent control:', type);
    }
    
    // Send event to server
    function sendEvent(nodeId, eventType, data = {}) {
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            console.warn('Cannot send event: WebSocket not connected');
            return;
        }
        
        // Build event frame
        const encoder = new TextEncoder();
        const eventData = encoder.encode(JSON.stringify(data));
        
        const buffer = new ArrayBuffer(1 + 1 + 4 + eventData.length);
        const view = new DataView(buffer);
        
        view.setUint8(0, 0x01); // FrameEvent
        view.setUint8(1, getEventTypeCode(eventType));
        view.setUint32(2, nodeId, true); // Little-endian
        
        // Copy event data
        const uint8View = new Uint8Array(buffer);
        uint8View.set(eventData, 6);
        
        ws.send(buffer);
        console.log('📤 Sent event:', eventType, 'for node', nodeId);
    }
    
    // Get event type code
    function getEventTypeCode(eventType) {
        const codes = {
            'click': 0x01,
            'increment': 0x02,
            'decrement': 0x03,
            'reset': 0x04,
            'input': 0x05,
            'submit': 0x06
        };
        return codes[eventType] || 0x00;
    }
    
    // Update connection status UI
    function updateConnectionStatus(connected) {
        const status = document.getElementById('connection-status');
        if (status) {
            status.className = 'connection-status ' + (connected ? 'connected' : 'disconnected');
            status.textContent = connected ? 'Connected' : 'Disconnected';
        }
        
        // Add class to body for CSS hooks
        document.body.classList.toggle('vango-offline', !connected);
    }
    
    // Animate element (visual feedback)
    function animateElement(element) {
        element.style.transform = 'scale(1.1)';
        element.style.transition = 'transform 0.2s ease';
        setTimeout(() => {
            element.style.transform = 'scale(1)';
        }, 200);
    }
    
    // Start heartbeat
    function startHeartbeat() {
        stopHeartbeat();
        heartbeatTimer = setInterval(() => {
            sendControl('PING', {});
        }, config.heartbeatInterval);
    }
    
    // Stop heartbeat
    function stopHeartbeat() {
        if (heartbeatTimer) {
            clearInterval(heartbeatTimer);
            heartbeatTimer = null;
        }
    }
    
    // Handle click events on server-driven elements
    function handleClick(event) {
        const target = event.target.closest('[data-server-event]');
        if (!target) return;
        
        event.preventDefault();
        
        const eventType = target.dataset.serverEvent;
        const hid = target.dataset.hid;
        const nodeId = parseInt(hid) || 0;
        
        console.log('🖱️ Click on server element:', eventType, 'node:', nodeId);
        sendEvent(nodeId, eventType);
    }
    
    // Initialize
    function init() {
        console.log('🚀 Initializing server-driven client');
        
        // Initialize session
        initSession();
        
        // Build node map
        buildNodeMap();
        
        // Set up event delegation
        document.addEventListener('click', handleClick);
        
        // Connect to server
        connect();
        
        // Rebuild node map on DOM changes (for dynamic content)
        const observer = new MutationObserver(() => {
            buildNodeMap();
        });
        observer.observe(document.body, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['data-hid']
        });
    }
    
    // Wait for DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    
    // Export for debugging
    window.__VangoLive = {
        ws: () => ws,
        sessionID: () => sessionID,
        nodeMap: () => nodeMap,
        sendEvent: sendEvent,
        reconnect: connect
    };
})();