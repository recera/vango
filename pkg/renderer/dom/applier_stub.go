//go:build !js || !wasm
// +build !js !wasm

package dom

import (
	"fmt"

	"github.com/recera/vango/pkg/vango/vdom"
)

// DOMApplier applies VNode patches to the browser DOM (stub for non-WASM builds)
type DOMApplier struct{}

// NewDOMApplier creates a new DOM applier (stub)
func NewDOMApplier() *DOMApplier {
	return &DOMApplier{}
}

// Apply applies patches to transform the DOM (stub)
func (a *DOMApplier) Apply(patches []vdom.Patch) error {
	return fmt.Errorf("DOM applier is only available in WASM builds")
}

// HydrateFromDOM builds the node map from existing DOM elements (stub)
func (a *DOMApplier) HydrateFromDOM() error {
	return fmt.Errorf("DOM hydration is only available in WASM builds")
}