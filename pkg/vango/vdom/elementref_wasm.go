//go:build js && wasm
// +build js,wasm

package vdom

import "syscall/js"

// ElementRef is a reference to a DOM element in WASM builds
type ElementRef = js.Value
