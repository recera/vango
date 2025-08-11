// Vango Bootstrap Script
// This script loads and initializes the WASM module for client-side rendering

(function() {
    'use strict';
    
    // Define process.env for browser compatibility
    if (typeof process === 'undefined') {
        window.process = { env: { NODE_ENV: 'development' } };
    }

    // Configuration
    const WASM_PATH = '/app.wasm';
    const RECONNECT_DELAYS = [1000, 2000, 5000, 10000, 30000]; // Exponential backoff
    
    // Global state
    let wasmInstance = null;
    let wsConnection = null;
    let reconnectAttempt = 0;
    let isOffline = false;

    // Load wasm_exec.js support file
    function loadWasmExec() {
        return new Promise((resolve, reject) => {
            if (window.Go) {
                resolve();
                return;
            }

            const script = document.createElement('script');
            script.src = '/wasm_exec.js';
            script.onload = () => {
                // After loading wasm_exec.js, add WASI support
                addWASISupport();
                resolve();
            };
            script.onerror = reject;
            document.head.appendChild(script);
        });
    }
    
    // Add WASI polyfill support
    function addWASISupport() {
        // Store original Go constructor
        const OriginalGo = window.Go;
        
        // WASI polyfill functions
        const wasiPolyfill = {
            fd_write: function(fd, iovs_ptr, iovs_len, nwritten_ptr) {
                // Basic stdout/stderr support
                if (fd === 1 || fd === 2) {
                    const memory = this._inst.exports.memory;
                    const view = new DataView(memory.buffer);
                    let written = 0;
                    
                    for (let i = 0; i < iovs_len; i++) {
                        const ptr = iovs_ptr + i * 8;
                        const buf = view.getUint32(ptr, true);
                        const len = view.getUint32(ptr + 4, true);
                        
                        const bytes = new Uint8Array(memory.buffer, buf, len);
                        const text = new TextDecoder().decode(bytes);
                        
                        if (fd === 1) {
                            console.log(text);
                        } else {
                            console.error(text);
                        }
                        
                        written += len;
                    }
                    
                    view.setUint32(nwritten_ptr, written, true);
                    return 0;
                }
                return 8; // EBADF
            },
            
            fd_close: function(fd) {
                return 0;
            },
            
            fd_seek: function(fd, offset_low, offset_high, whence, newoffset_ptr) {
                return 0;
            },
            
            environ_get: function(environ, environ_buf) {
                return 0;
            },
            
            environ_sizes_get: function(count_ptr, size_ptr) {
                const memory = this._inst.exports.memory;
                const view = new DataView(memory.buffer);
                view.setUint32(count_ptr, 0, true);
                view.setUint32(size_ptr, 0, true);
                return 0;
            },
            
            clock_time_get: function(id, precision_low, precision_high, time_ptr) {
                const memory = this._inst.exports.memory;
                const view = new DataView(memory.buffer);
                const now = Date.now() * 1000000; // Convert to nanoseconds
                view.setBigUint64(time_ptr, BigInt(now), true);
                return 0;
            },
            
            proc_exit: function(code) {
                if (this.exited) return;
                this.exited = true;
                this.exitCode = code;
                throw 'wasm exited with code ' + code;
            },
            
            random_get: function(buf, buf_len) {
                const memory = this._inst.exports.memory;
                const bytes = new Uint8Array(memory.buffer, buf, buf_len);
                crypto.getRandomValues(bytes);
                return 0;
            }
        };
        
        // Extended Go class with WASI support
        class GoWithWASI extends OriginalGo {
            constructor() {
                super();
                
                // Add WASI imports to the import object
                this.importObject.wasi_snapshot_preview1 = {};
                
                // Bind WASI functions to this instance
                for (const [name, fn] of Object.entries(wasiPolyfill)) {
                    this.importObject.wasi_snapshot_preview1[name] = fn.bind(this);
                }
            }
        }
        
        // Replace global Go with our extended version
        window.Go = GoWithWASI;
    }

    // Load and instantiate WASM module
    async function loadWasm() {
        await loadWasmExec();

        const go = new Go();
        const response = await fetch(WASM_PATH);
        const wasmBuffer = await response.arrayBuffer();
        
        const result = await WebAssembly.instantiate(wasmBuffer, go.importObject);
        wasmInstance = result.instance;
        
        // Run the Go program
        go.run(wasmInstance);
        
        return wasmInstance;
    }

    // Build sparse VNode tree from hydration IDs
    function buildHydrationTree() {
        const nodes = document.querySelectorAll('[data-hid]');
        const hydrationMap = {};
        
        nodes.forEach(node => {
            const hid = node.getAttribute('data-hid');
            hydrationMap[hid] = {
                element: node,
                tagName: node.tagName.toLowerCase(),
                attributes: getAttributes(node),
                events: getEventMask(node)
            };
        });
        
        return hydrationMap;
    }

    // Extract attributes from a DOM element
    function getAttributes(element) {
        const attrs = {};
        for (let i = 0; i < element.attributes.length; i++) {
            const attr = element.attributes[i];
            if (attr.name !== 'data-hid') {
                attrs[attr.name] = attr.value;
            }
        }
        return attrs;
    }

    // Get event mask from data attribute
    function getEventMask(element) {
        const eventsAttr = element.getAttribute('data-events');
        return eventsAttr ? parseInt(eventsAttr, 10) : 0;
    }

    // Initialize WebSocket connection for live updates
    function initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/vango/live/${getSessionId()}`;
        
        wsConnection = new WebSocket(wsUrl);
        
        wsConnection.onopen = () => {
            console.log('[Vango] WebSocket connected');
            reconnectAttempt = 0;
            setOnlineState(true);
            
            // Send hello message
            wsConnection.send(JSON.stringify({
                type: 'HELLO',
                resumable: true,
                lastSeq: getLastSequence()
            }));
        };
        
        wsConnection.onmessage = (event) => {
            handleLiveUpdate(event.data);
        };
        
        wsConnection.onclose = () => {
            console.log('[Vango] WebSocket disconnected');
            setOnlineState(false);
            scheduleReconnect();
        };
        
        wsConnection.onerror = (error) => {
            console.error('[Vango] WebSocket error:', error);
        };
    }

    // Handle live updates from server
    function handleLiveUpdate(data) {
        try {
            // Check if it's binary data (patch stream)
            if (data instanceof ArrayBuffer || data instanceof Blob) {
                handleBinaryPatch(data);
            } else {
                // JSON message
                const message = JSON.parse(data);
                handleControlMessage(message);
            }
        } catch (error) {
            console.error('[Vango] Error handling live update:', error);
        }
    }

    // Handle binary patch data
    async function handleBinaryPatch(data) {
        // Convert to ArrayBuffer if needed
        const buffer = data instanceof Blob ? await data.arrayBuffer() : data;
        const view = new DataView(buffer);
        let offset = 0;
        
        // Read frame type
        const frameType = view.getUint8(offset);
        offset++;
        
        if (frameType === 0x00) { // Patches frame
            // Parse and apply patches
            while (offset < buffer.byteLength) {
                const opcode = view.getUint8(offset);
                offset++;
                
                switch (opcode) {
                    case 0x01: // ReplaceText
                        const nodeId = readUvarint(view, offset);
                        offset = nodeId.offset;
                        const text = readString(view, offset);
                        offset = text.offset;
                        
                        console.log(`[Live] ReplaceText node=${nodeId.value} text="${text.value}"`);
                        
                        // Apply patch via WASM if available
                        if (window.__vango_applyPatches) {
                            window.__vango_applyPatches([{
                                op: 'ReplaceText',
                                nodeID: nodeId.value,
                                value: text.value
                            }]);
                        }
                        break;
                        
                    case 0x04: // InsertNode
                        const parentId = readUvarint(view, offset);
                        offset = parentId.offset;
                        const beforeId = readUvarint(view, offset);
                        offset = beforeId.offset;
                        // TODO: Read serialized VNode tree
                        console.log(`[Live] InsertNode parent=${parentId.value} before=${beforeId.value}`);
                        break;
                        
                    case 0x05: // UpdateEvents
                        const eventNodeId = readUvarint(view, offset);
                        offset = eventNodeId.offset;
                        const eventBits = readUvarint(view, offset);
                        offset = eventBits.offset;
                        console.log(`[Live] UpdateEvents node=${eventNodeId.value} bits=${eventBits.value}`);
                        break;
                        
                    default:
                        console.warn(`[Live] Unknown opcode: ${opcode}`);
                        return;
                }
            }
        }
    }
    
    // Read unsigned varint from DataView
    function readUvarint(view, offset) {
        let value = 0;
        let shift = 0;
        let byte;
        
        do {
            byte = view.getUint8(offset);
            offset++;
            value |= (byte & 0x7F) << shift;
            shift += 7;
        } while (byte & 0x80);
        
        return { value, offset };
    }
    
    // Read length-prefixed string from DataView
    function readString(view, offset) {
        const length = readUvarint(view, offset);
        offset = length.offset;
        
        const bytes = new Uint8Array(view.buffer, offset, length.value);
        const value = new TextDecoder().decode(bytes);
        offset += length.value;
        
        return { value, offset };
    }

    // Handle control messages
    function handleControlMessage(message) {
        switch (message.type) {
            case 'FULL-RESYNC':
                window.location.reload();
                break;
            case 'RELOAD':
                if (message.target === 'wasm') {
                    reloadWasm();
                } else if (message.target === 'css') {
                    reloadStyles();
                }
                break;
            default:
                console.warn('[Vango] Unknown message type:', message.type);
        }
    }

    // Set online/offline state
    function setOnlineState(online) {
        isOffline = !online;
        document.body.classList.toggle('vango-offline', isOffline);
        
        // Fire custom event
        window.dispatchEvent(new CustomEvent(online ? '__vangoLive_online' : '__vangoLive_offline'));
    }

    // Schedule WebSocket reconnection
    function scheduleReconnect() {
        const delay = RECONNECT_DELAYS[Math.min(reconnectAttempt, RECONNECT_DELAYS.length - 1)];
        reconnectAttempt++;
        
        console.log(`[Vango] Reconnecting in ${delay}ms (attempt ${reconnectAttempt})`);
        
        setTimeout(() => {
            if (!wsConnection || wsConnection.readyState === WebSocket.CLOSED) {
                initWebSocket();
            }
        }, delay);
    }

    // Get or create session ID
    function getSessionId() {
        let sessionId = sessionStorage.getItem('vango-session-id');
        if (!sessionId) {
            sessionId = generateSessionId();
            sessionStorage.setItem('vango-session-id', sessionId);
        }
        return sessionId;
    }

    // Generate a random session ID
    function generateSessionId() {
        return Array.from(crypto.getRandomValues(new Uint8Array(16)))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }

    // Get last sequence number for resumable connections
    function getLastSequence() {
        return parseInt(sessionStorage.getItem('vango-last-seq') || '0', 10);
    }

    // Hot reload WASM module
    async function reloadWasm() {
        console.log('[Vango] Hot reloading WASM...');
        
        try {
            // Store current state
            const state = window.__vango_getState ? window.__vango_getState() : null;
            
            // Reload WASM
            await loadWasm();
            
            // Restore state
            if (state && window.__vango_setState) {
                window.__vango_setState(state);
            }
            
            // Re-hydrate
            hydrate();
        } catch (error) {
            console.error('[Vango] Failed to hot reload WASM:', error);
            window.location.reload();
        }
    }

    // Hot reload stylesheets
    function reloadStyles() {
        console.log('[Vango] Hot reloading styles...');
        
        const links = document.querySelectorAll('link[rel="stylesheet"]');
        links.forEach(link => {
            const href = link.getAttribute('href');
            const url = new URL(href, window.location.href);
            url.searchParams.set('_', Date.now());
            link.setAttribute('href', url.toString());
        });
    }

    // Hydrate the application
    function hydrate() {
        const hydrationTree = buildHydrationTree();
        
        // Call WASM hydration function with config
        if (window.__vango_hydrate) {
            window.__vango_hydrate({
                serverDriven: true,
                hydrationTree: hydrationTree
            });
        }
        
        // Store WebSocket reference for WASM
        window.__vango_ws = wsConnection;
    }

    // DevTools hook
    if (typeof process !== 'undefined' && process.env && process.env.NODE_ENV !== 'production') {
        window.__VangoLiveTap = {
            getConnection: () => wsConnection,
            getHydrationTree: buildHydrationTree,
            isOffline: () => isOffline,
            forceReconnect: () => {
                if (wsConnection) {
                    wsConnection.close();
                }
                reconnectAttempt = 0;
                initWebSocket();
            }
        };
    }

    // Initialize everything when DOM is ready
    function init() {
        loadWasm()
            .then(() => {
                console.log('[Vango] WASM loaded successfully');
                hydrate();
                initWebSocket();
            })
            .catch(error => {
                console.error('[Vango] Failed to initialize:', error);
                document.body.innerHTML = `
                    <div style="padding: 20px; font-family: monospace; color: red;">
                        <h1>Vango Initialization Error</h1>
                        <pre>${error.stack || error.message}</pre>
                    </div>
                `;
            });
    }

    // Start initialization
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();