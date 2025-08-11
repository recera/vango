//go:build js && wasm
// +build js,wasm

package graphviewer

import (
    "syscall/js"
)

// Controller provides imperative controls via ref
type Controller struct {
    canvas js.Value
    getState func() (offx, offy, scale float64)
    setState func(offx, offy, scale float64)
    getData  func() *Data
}

func (c *Controller) valid() bool { return c.canvas.Truthy() }

// FitGraph resets viewport to fit all nodes
func (c *Controller) FitGraph(padding float64) {
    if !c.valid() { return }
    d := c.getData(); if d == nil || len(d.Nodes) == 0 { return }
    minx, miny := d.Nodes[0].X, d.Nodes[0].Y
    maxx, maxy := minx, miny
    for _, n := range d.Nodes {
        if n.X < minx { minx = n.X }
        if n.Y < miny { miny = n.Y }
        if n.X > maxx { maxx = n.X }
        if n.Y > maxy { maxy = n.Y }
    }
    rect := c.canvas.Call("getBoundingClientRect")
    w := rect.Get("width").Float(); h := rect.Get("height").Float()
    gw := maxx - minx; gh := maxy - miny
    if gw <= 0 { gw = 1 }; if gh <= 0 { gh = 1 }
    sx := (w - 2*padding) / gw
    sy := (h - 2*padding) / gh
    s := sx; if sy < s { s = sy }
    if s <= 0 { s = 1 }
    offx := padding - minx*s
    offy := padding - miny*s
    c.setState(offx, offy, s)
}

// Reset resets zoom/pan to defaults
func (c *Controller) Reset() { c.setState(0, 0, 1) }

// FocusNode centers viewport on a node ID
func (c *Controller) FocusNode(id string, scale float64) {
    if !c.valid() { return }
    d := c.getData(); if d == nil { return }
    for _, n := range d.Nodes {
        if n.ID == id {
            rect := c.canvas.Call("getBoundingClientRect")
            w := rect.Get("width").Float(); h := rect.Get("height").Float()
            offx := w*0.5 - n.X*scale
            offy := h*0.5 - n.Y*scale
            c.setState(offx, offy, scale)
            return
        }
    }
}

// ExportPNG returns a data URL of the current canvas
func (c *Controller) ExportPNG() string {
    if !c.valid() { return "" }
    return c.canvas.Call("toDataURL", "image/png").String()
}


