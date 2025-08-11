// Vango Bootstrap Script
// This script loads and initializes the WASM module for client-side rendering

(function() {
    'use strict';

    // Configuration
    const WASM_PATH = '/app.wasm';
    const RECONNECT_DELAYS = [1000, 2000, 5000, 10000, 30000]; // Exponential backoff
    
    // Global state
    let wasmInstance = null;
    let wsConnection = null;
    let reconnectAttempt = 0;
    let isOffline = false;
    let routerTable = null;
    let currentPath = window.location.pathname;

    // Load wasm_exec.js support file
    function loadWasmExec() {
        return new Promise((resolve, reject) => {
            if (window.Go) {
                resolve();
                return;
            }

            const script = document.createElement('script');
            script.src = '/wasm_exec.js';
            script.onload = resolve;
            script.onerror = reject;
            document.head.appendChild(script);
        });
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
        const sessionId = getSessionId();
        const wsUrl = `${protocol}//${window.location.host}/vango/live/${sessionId}`;
        
        console.log('[Vango] Connecting to WebSocket:', wsUrl);
        wsConnection = new WebSocket(wsUrl);
        wsConnection.binaryType = 'arraybuffer'; // Important for binary protocol
        
        wsConnection.onopen = () => {
            console.log('[Vango] WebSocket connected to session:', sessionId);
            reconnectAttempt = 0;
            setOnlineState(true);
            
            // Send binary HELLO message for live protocol
            const hello = encodeHelloMessage(true, getLastSequence());
            console.log('[Vango] Sending HELLO message, bytes:', hello.length);
            wsConnection.send(hello);
        };
        
        wsConnection.onmessage = (event) => {
            console.log('[Vango] Message received from server');
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
        console.log('[Vango] Received message, type:', typeof data, 'size:', data.byteLength || data.length);
        try {
            // Check if it's binary data (patch stream)
            if (data instanceof ArrayBuffer || data instanceof Blob) {
                console.log('[Vango] Processing binary message');
                handleBinaryPatch(data);
            } else {
                // JSON message
                console.log('[Vango] Processing text message:', data);
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
        
        // Check frame type
        const frameType = view.getUint8(offset++);
        
        switch (frameType) {
            case 0x00: // FramePatches
                offset = applyPatchFrame(view, offset);
                break;
            case 0x01: // FrameEvent (shouldn't receive from server)
                console.warn('[Vango] Received unexpected event frame from server');
                break;
            case 0x02: // FrameControl
                offset = handleControlFrame(view, offset);
                break;
            default:
                console.error('[Vango] Unknown frame type:', frameType);
        }
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
            case 'ACK':
                // Dev server handshake acknowledgement; no action required
                break;
            case 'update':
                // Legacy JSON update for counter demo
                console.log('[Vango] Received update:', message.value);
                const counterElement = document.getElementById('counter-display');
                console.log('[Vango] Counter element found:', !!counterElement, counterElement);
                if (counterElement) {
                    const oldValue = counterElement.textContent;
                    counterElement.textContent = String(message.value);
                    console.log('[Vango] Updated counter display from', oldValue, 'to', message.value);
                    // Force a visual update
                    counterElement.style.transform = 'scale(1.1)';
                    setTimeout(() => {
                        counterElement.style.transform = 'scale(1)';
                    }, 100);
                } else {
                    console.warn('[Vango] Counter display element not found');
                    console.log('[Vango] Available elements with IDs:', 
                        Array.from(document.querySelectorAll('[id]')).map(el => el.id));
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
        
        // Call WASM hydration function
        if (window.__vango_hydrate) {
            window.__vango_hydrate(hydrationTree);
        }
    }

    // Load router table
    async function loadRouterTable() {
        try {
            const response = await fetch('/router/table.json');
            if (!response.ok) {
                throw new Error(`Failed to load router table: ${response.status}`);
            }
            routerTable = await response.json();
            console.log('[Vango] Router table loaded:', routerTable.routes.length, 'routes');
        } catch (error) {
            console.warn('[Vango] Could not load router table:', error);
            // Continue without client-side navigation
        }
    }

    // Match a path against the router table
    function matchRoute(path) {
        if (!routerTable || !routerTable.routes) {
            return null;
        }

        // Normalize path
        path = path.replace(/\/$/, '') || '/';

        // Try exact match first
        for (const route of routerTable.routes) {
            if (route.path === path) {
                return route;
            }
        }

        // Try pattern matching for dynamic routes
        for (const route of routerTable.routes) {
            const pattern = routeToRegex(route);
            const match = path.match(pattern.regex);
            if (match) {
                // Extract params
                const params = {};
                pattern.params.forEach((param, index) => {
                    params[param.name] = match[index + 1];
                });
                return { ...route, params };
            }
        }

        return null;
    }

    // Convert route pattern to regex
    function routeToRegex(route) {
        let pattern = route.path;
        const params = [];

        // Extract parameter definitions
        pattern = pattern.replace(/\[([^\]]+)\]/g, (match, param) => {
            // Handle catch-all params
            if (param.startsWith('...')) {
                const name = param.substring(3);
                params.push({ name, type: 'catchall' });
                return '(.*)';
            }

            // Handle typed params
            const [name, type = 'string'] = param.split(':');
            params.push({ name, type });

            // Return appropriate regex based on type
            switch (type) {
                case 'int':
                case 'int64':
                    return '(\\d+)';
                case 'uuid':
                    return '([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})';
                default:
                    return '([^/]+)';
            }
        });

        // Escape special regex characters
        pattern = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

        return {
            regex: new RegExp('^' + pattern + '$'),
            params
        };
    }

    // Navigate to a new path
    async function navigate(path, options = {}) {
        // Check if path is external
        if (path.startsWith('http://') || path.startsWith('https://')) {
            window.location.href = path;
            return;
        }

        // Match route
        const route = matchRoute(path);
        if (!route) {
            // No client-side route found, do a full page navigation
            window.location.href = path;
            return;
        }

        // Update browser history
        if (!options.replace) {
            window.history.pushState({ path }, '', path);
        } else {
            window.history.replaceState({ path }, '', path);
        }

        // Update current path
        currentPath = path;

        // Notify WASM about navigation
        if (window.__vango_navigate) {
            window.__vango_navigate(path, route.component, route.params || {});
        } else {
            // Fallback to full page reload if WASM navigation not available
            window.location.href = path;
        }
    }

    // Intercept link clicks
    function interceptLinks() {
        document.addEventListener('click', (event) => {
            // Check if click is on a link
            let link = event.target;
            while (link && link.tagName !== 'A') {
                link = link.parentElement;
            }

            if (!link) return;

            // Check if we should handle this link
            const href = link.getAttribute('href');
            if (!href) return;

            // Skip external links
            if (href.startsWith('http://') || href.startsWith('https://')) {
                return;
            }

            // Skip links with target attribute
            if (link.getAttribute('target')) {
                return;
            }

            // Skip if meta keys are pressed
            if (event.metaKey || event.ctrlKey || event.shiftKey) {
                return;
            }

            // Skip download links
            if (link.hasAttribute('download')) {
                return;
            }

            // Skip fragment-only links
            if (href.startsWith('#')) {
                return;
            }

            // Prevent default and navigate
            event.preventDefault();
            navigate(href);
        });
    }

    // Handle browser back/forward buttons
    function handlePopState() {
        window.addEventListener('popstate', (event) => {
            const path = window.location.pathname;
            if (path !== currentPath) {
                navigate(path, { replace: true });
            }
        });
    }

    // Prefetch links on hover
    function setupPrefetch() {
        let prefetchTimer = null;
        
        document.addEventListener('mouseover', (event) => {
            let link = event.target;
            while (link && link.tagName !== 'A') {
                link = link.parentElement;
            }

            if (!link) return;

            const href = link.getAttribute('href');
            if (!href || href.startsWith('#') || href.startsWith('http')) {
                return;
            }

            // Clear any existing timer
            if (prefetchTimer) {
                clearTimeout(prefetchTimer);
            }

            // Prefetch after a short delay
            prefetchTimer = setTimeout(() => {
                if (window.__vango_prefetch) {
                    window.__vango_prefetch(href);
                }
            }, 100);
        });

        document.addEventListener('mouseout', () => {
            if (prefetchTimer) {
                clearTimeout(prefetchTimer);
                prefetchTimer = null;
            }
        });
    }

    // DevTools hook
    if ('production' !== 'production') {
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
        Promise.all([
            loadWasm(),
            loadRouterTable()
        ])
            .then(() => {
                console.log('[Vango] WASM loaded successfully');
                hydrate();
                initWebSocket();
                interceptLinks();
                handlePopState();
                setupPrefetch();
                setupServerEventDelegation();
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
    
    // Set up event delegation for server-driven components
    function setupServerEventDelegation() {
        // Delegate click events
        document.addEventListener('click', (event) => {
            let target = event.target;
            
            // Walk up the DOM tree to find element with data-hid
            while (target && target !== document.body) {
                if (target.hasAttribute('data-hid')) {
                    const hid = target.getAttribute('data-hid');
                    const eventType = target.getAttribute('data-server-event') || 'click';
                    
                    // Extract node ID from hydration ID (format: "h123")
                    const nodeId = parseInt(hid.substring(1), 10);
                    
                    if (!isNaN(nodeId)) {
                        event.preventDefault();
                        event.stopPropagation();
                        window.__vango_sendEvent(nodeId, eventType);
                        return;
                    }
                }
                target = target.parentElement;
            }
        });
        
        // Delegate input events
        document.addEventListener('input', (event) => {
            const target = event.target;
            if (target.hasAttribute('data-hid')) {
                const hid = target.getAttribute('data-hid');
                const nodeId = parseInt(hid.substring(1), 10);
                
                if (!isNaN(nodeId)) {
                    // TODO: Send input value with event
                    window.__vango_sendEvent(nodeId, 'input');
                }
            }
        });
        
        // Delegate form submit events
        document.addEventListener('submit', (event) => {
            let target = event.target;
            if (target.hasAttribute('data-hid')) {
                const hid = target.getAttribute('data-hid');
                const nodeId = parseInt(hid.substring(1), 10);
                
                if (!isNaN(nodeId)) {
                    event.preventDefault();
                    // TODO: Send form data with event
                    window.__vango_sendEvent(nodeId, 'submit');
                }
            }
        });
        
        console.log('[Vango] Server event delegation set up');
    }

    // Start initialization
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    
    // ========== Binary Protocol Helpers ==========
    
    // Decode unsigned varint from DataView
    function decodeUvarint(view, offset) {
        let value = 0;
        let shift = 0;
        while (offset < view.byteLength) {
            const byte = view.getUint8(offset++);
            value |= (byte & 0x7F) << shift;
            if ((byte & 0x80) === 0) {
                return [value, offset];
            }
            shift += 7;
        }
        throw new Error('Invalid varint');
    }
    
    // Encode unsigned varint to array
    function encodeUvarint(value) {
        const bytes = [];
        while (value >= 0x80) {
            bytes.push((value & 0x7F) | 0x80);
            value >>= 7;
        }
        bytes.push(value & 0x7F);
        return bytes;
    }
    
    // Decode string from DataView
    function decodeString(view, offset) {
        const [length, newOffset] = decodeUvarint(view, offset);
        const bytes = new Uint8Array(view.buffer, view.byteOffset + newOffset, length);
        const string = new TextDecoder().decode(bytes);
        return [string, newOffset + length];
    }
    
    // Encode string to bytes
    function encodeString(str) {
        const encoder = new TextEncoder();
        const strBytes = encoder.encode(str);
        const lenBytes = encodeUvarint(strBytes.length);
        return [...lenBytes, ...strBytes];
    }
    
    // Encode HELLO message
    function encodeHelloMessage(resumable, lastSeq) {
        const helloStr = encodeString('HELLO');
        const resumableBytes = encodeUvarint(resumable ? 1 : 0);
        const lastSeqBytes = encodeUvarint(lastSeq);
        
        console.log('[Vango] HELLO encoding:', {
            helloStr: helloStr,
            resumableBytes: resumableBytes,
            lastSeqBytes: lastSeqBytes
        });
        
        const bytes = [
            0x02, // FrameControl
            ...helloStr,
            ...resumableBytes,
            ...lastSeqBytes
        ];
        return new Uint8Array(bytes);
    }
    
    // Encode event message
    function encodeEventMessage(eventType, nodeId, data = {}) {
        const bytes = [
            0x01, // FrameEvent
            eventType,
            ...encodeUvarint(nodeId)
            // TODO: Encode additional event data
        ];
        return new Uint8Array(bytes);
    }
    
    // Apply patch frame
    function applyPatchFrame(view, offset) {
        const [patchCount, newOffset] = decodeUvarint(view, offset);
        offset = newOffset;
        
        console.log(`[Vango] Applying ${patchCount} patches`);
        
        for (let i = 0; i < patchCount; i++) {
            offset = applySinglePatch(view, offset);
        }
        
        return offset;
    }
    
    // Apply a single patch
    function applySinglePatch(view, offset) {
        const opcode = view.getUint8(offset++);
        
        switch (opcode) {
            case 0: // OpReplaceText
                return applyReplaceText(view, offset);
            case 1: // OpSetAttribute
                return applySetAttribute(view, offset);
            case 2: // OpRemoveAttribute
                return applyRemoveAttribute(view, offset);
            case 3: // OpRemoveNode
                return applyRemoveNode(view, offset);
            case 4: // OpInsertNode
                return applyInsertNode(view, offset);
            case 5: // OpUpdateEvents
                return applyUpdateEvents(view, offset);
            case 6: // OpMoveNode
                return applyMoveNode(view, offset);
            default:
                console.error('[Vango] Unknown patch opcode:', opcode);
                throw new Error(`Unknown patch opcode: ${opcode}`);
        }
    }
    
    // Node ID to DOM element mapping
    const nodeMap = new Map();
    
    // Apply ReplaceText patch
    function applyReplaceText(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const [text, off2] = decodeString(view, off1);
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        if (element) {
            element.textContent = text;
            nodeMap.set(nodeId, element);
            console.log(`[Vango] Replaced text for node ${nodeId}: "${text}"`);
        } else {
            console.warn(`[Vango] Node ${nodeId} not found for text replacement`);
        }
        
        return off2;
    }
    
    // Apply SetAttribute patch
    function applySetAttribute(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const [key, off2] = decodeString(view, off1);
        const [value, off3] = decodeString(view, off2);
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        if (element) {
            // Special handling for certain attributes
            if (key === 'class') {
                element.className = value;
            } else if (key === 'checked' || key === 'selected' || key === 'disabled') {
                element[key] = value === 'true';
            } else if (key === 'value' && (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA')) {
                element.value = value;
            } else {
                element.setAttribute(key, value);
            }
            nodeMap.set(nodeId, element);
            console.log(`[Vango] Set attribute ${key}="${value}" on node ${nodeId}`);
        } else {
            console.warn(`[Vango] Node ${nodeId} not found for attribute setting`);
        }
        
        return off3;
    }
    
    // Apply RemoveAttribute patch
    function applyRemoveAttribute(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const [key, off2] = decodeString(view, off1);
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        if (element) {
            element.removeAttribute(key);
            console.log(`[Vango] Removed attribute ${key} from node ${nodeId}`);
        }
        
        return off2;
    }
    
    // Apply RemoveNode patch
    function applyRemoveNode(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        if (element && element.parentNode) {
            element.parentNode.removeChild(element);
            nodeMap.delete(nodeId);
            console.log(`[Vango] Removed node ${nodeId}`);
        }
        
        return off1;
    }
    
    // Apply InsertNode patch (simplified - full VNode tree serialization needed)
    function applyInsertNode(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const [parentId, off2] = decodeUvarint(view, off1);
        const [beforeId, off3] = decodeUvarint(view, off2);
        
        // TODO: Deserialize VNode tree and create DOM elements
        console.log(`[Vango] InsertNode: nodeId=${nodeId}, parentId=${parentId}, beforeId=${beforeId}`);
        
        return off3;
    }
    
    // Apply UpdateEvents patch
    function applyUpdateEvents(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const eventBits = view.getUint32(off1, true); // little endian
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        if (element) {
            element.setAttribute('data-events', eventBits.toString());
            // TODO: Actually attach/detach event listeners based on bits
            console.log(`[Vango] Updated events for node ${nodeId}: ${eventBits}`);
        }
        
        return off1 + 4;
    }
    
    // Apply MoveNode patch
    function applyMoveNode(view, offset) {
        const [nodeId, off1] = decodeUvarint(view, offset);
        const [parentId, off2] = decodeUvarint(view, off1);
        const [beforeId, off3] = decodeUvarint(view, off2);
        
        const element = nodeMap.get(nodeId) || document.querySelector(`[data-hid="h${nodeId}"]`);
        const parent = nodeMap.get(parentId) || document.querySelector(`[data-hid="h${parentId}"]`) || document.body;
        
        if (element && parent) {
            if (beforeId > 0) {
                const before = nodeMap.get(beforeId) || document.querySelector(`[data-hid="h${beforeId}"]`);
                if (before) {
                    parent.insertBefore(element, before);
                } else {
                    parent.appendChild(element);
                }
            } else {
                parent.appendChild(element);
            }
            console.log(`[Vango] Moved node ${nodeId} to parent ${parentId}`);
        }
        
        return off3;
    }
    
    // Handle control frame
    function handleControlFrame(view, offset) {
        const [msgType, off1] = decodeString(view, offset);
        
        switch (msgType) {
            case 'HELLO':
                const [lastSeq, off2] = decodeUvarint(view, off1);
                console.log(`[Vango] Server HELLO: lastSeq=${lastSeq}`);
                sessionStorage.setItem('vango-last-seq', lastSeq.toString());
                return off2;
                
            case 'PONG':
                console.log('[Vango] Received PONG');
                return off1;
                
            default:
                console.log(`[Vango] Unknown control message: ${msgType}`);
                return off1;
        }
    }
    
    // Send event to server
    window.__vango_sendEvent = function(nodeId, eventType) {
        if (!wsConnection || wsConnection.readyState !== WebSocket.OPEN) {
            console.warn('[Vango] WebSocket not connected');
            return;
        }
        
        const eventTypeMap = {
            'click': 0x01,
            'increment': 0x02,
            'decrement': 0x03,
            'reset': 0x04,
            'input': 0x05,
            'submit': 0x06
        };
        
        const typeCode = eventTypeMap[eventType] || 0x01;
        const message = encodeEventMessage(typeCode, nodeId);
        wsConnection.send(message);
        console.log(`[Vango] Sent ${eventType} event for node ${nodeId}`);
    };
})();