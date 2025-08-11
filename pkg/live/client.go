// +build js,wasm

package live

import (
	"syscall/js"
	"log"
)

// Client handles WebSocket communication from the browser
type Client struct {
	ws       js.Value
	url      string
	onPatch  func([]byte)
	onReady  func()
	onError  func(error)
}

// NewClient creates a new live protocol client
func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

// Connect establishes WebSocket connection
func (c *Client) Connect() error {
	// Create WebSocket
	c.ws = js.Global().Get("WebSocket").New(c.url)
	
	// Set binary type
	c.ws.Set("binaryType", "arraybuffer")
	
	// Set up event handlers
	c.ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("[Live Client] Connected")
		if c.onReady != nil {
			c.onReady()
		}
		return nil
	}))
	
	c.ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		data := event.Get("data")
		
		// Convert ArrayBuffer to byte slice
		buffer := js.Global().Get("Uint8Array").New(data)
		length := buffer.Get("length").Int()
		bytes := make([]byte, length)
		js.CopyBytesToGo(bytes, buffer)
		
		// Handle patch data
		if c.onPatch != nil {
			c.onPatch(bytes)
		}
		
		return nil
	}))
	
	c.ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("[Live Client] WebSocket error")
		if c.onError != nil {
			// TODO: Extract error details
			c.onError(nil)
		}
		return nil
	}))
	
	c.ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("[Live Client] Disconnected")
		// TODO: Implement reconnection logic
		return nil
	}))
	
	return nil
}

// SendEvent sends an event to the server
func (c *Client) SendEvent(evt Event) error {
	if c.ws.IsNull() || c.ws.IsUndefined() {
		return nil
	}
	
	// Encode event
	data := EncodeEvent(evt)
	
	// Convert to Uint8Array
	arrayBuffer := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arrayBuffer, data)
	
	// Send via WebSocket
	c.ws.Call("send", arrayBuffer)
	
	return nil
}

// Close closes the WebSocket connection
func (c *Client) Close() {
	if !c.ws.IsNull() && !c.ws.IsUndefined() {
		c.ws.Call("close")
	}
}

// OnPatch sets the patch handler
func (c *Client) OnPatch(handler func([]byte)) {
	c.onPatch = handler
}

// OnReady sets the ready handler
func (c *Client) OnReady(handler func()) {
	c.onReady = handler
}

// OnError sets the error handler
func (c *Client) OnError(handler func(error)) {
	c.onError = handler
}