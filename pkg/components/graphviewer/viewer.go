//go:build !js || !wasm

package graphviewer

import "github.com/recera/vango/pkg/vango/vdom"

// Viewer is stubbed out for non-WASM builds
func Viewer(_ Data, _ *Options) *vdom.VNode {
	return vdom.NewElement("div", vdom.Props{"style": "width:100%;height:100%"}, vdom.NewText("Graph viewer requires WASM"))
}
