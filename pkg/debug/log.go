//go:build js && wasm
// +build js,wasm

package debug

import (
	"fmt"
	"syscall/js"
	
	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/scheduler"
)

// EnableLogging enables debug logging for scheduler and reactive packages
func EnableLogging() {
	logFn := func(args ...interface{}) {
		js.Global().Get("console").Call("log", args...)
	}
	
	scheduler.SetDebugLog(logFn)
	reactive.SetDebugLog(logFn)
}

// Log logs a message to the console
func Log(args ...interface{}) {
	js.Global().Get("console").Call("log", args...)
}

// Logf logs a formatted message to the console
func Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	js.Global().Get("console").Call("log", msg)
}