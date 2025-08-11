// This file adds WASI support on top of the standard wasm_exec.js

// First, load the original wasm_exec.js content
// (In production, we'd concatenate these files)

// Add WASI polyfill
(function() {
    'use strict';
    
    // Store original Go constructor if it exists
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
})();