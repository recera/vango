//go:build js && wasm
// +build js,wasm

package graphviewer

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// Viewer renders an interactive canvas-based graph viewer (WASM)
func Viewer(data Data, opts *Options) *vdom.VNode {
	o := opts.withDefaults()

	// Reactive state
	scale := reactive.CreateState(1.0)
	offsetX := reactive.CreateState(0.0)
	offsetY := reactive.CreateState(0.0)
	dragging := reactive.CreateState(false)
	dragNode := reactive.CreateState(-1)
	hoverIdx := reactive.CreateState(-1)
	selectedIdx := reactive.CreateState(-1)
	lastX := reactive.CreateState(0.0)
	lastY := reactive.CreateState(0.0)
	mouseDownX := reactive.CreateState(0.0)
	mouseDownY := reactive.CreateState(0.0)
	didMove := reactive.CreateState(false)

	var canvasRef js.Value

	screenToWorld := func(x, y float64) (wx, wy float64) {
		return (x - offsetX.Get()) / scale.Get(), (y - offsetY.Get()) / scale.Get()
	}

	onWheel := func(deltaY float64) {
		c := canvasRef
		if c.IsUndefined() || c.IsNull() {
			return
		}
		mx, my := lastX.Get(), lastY.Get()
		factor := 1.0 - math.Max(-0.5, math.Min(0.5, deltaY/500.0))
		newScale := scale.Get() * factor
		if newScale < o.MinScale {
			newScale = o.MinScale
		}
		if newScale > o.MaxScale {
			newScale = o.MaxScale
		}
		wx, wy := screenToWorld(mx, my)
		scale.Set(newScale)
		offsetX.Set(mx - wx*newScale)
		offsetY.Set(my - wy*newScale)
		requestDraw(c)
	}

	onMouseDown := func(x, y float64) {
		c := canvasRef
		if c.IsUndefined() || c.IsNull() {
			return
		}
		dragging.Set(true)
		lastX.Set(x)
		lastY.Set(y)
		mouseDownX.Set(x)
		mouseDownY.Set(y)
		didMove.Set(false)
		wx, wy := screenToWorld(x, y)
		picked := -1
		for i, n := range data.Nodes {
			r := n.Size
			if r <= 0 {
				r = 8
			}
			dx := wx - n.X
			dy := wy - n.Y
			if dx*dx+dy*dy <= r*r {
				picked = i
				break
			}
		}
		dragNode.Set(picked)
	}

	onMouseMove := func(x, y float64) {
		c := canvasRef
		if c.IsUndefined() || c.IsNull() {
			return
		}
		dx := x - lastX.Get()
		dy := y - lastY.Get()
		lastX.Set(x)
		lastY.Set(y)
		// Update hover
		// inline pick for hover (avoid extra symbol)
		wxh, wyh := screenToWorld(x, y)
		idx := -1
		for i := range data.Nodes {
			r := data.Nodes[i].Size
			if r <= 0 {
				r = 8
			}
			dxh := wxh - data.Nodes[i].X
			dyh := wyh - data.Nodes[i].Y
			if dxh*dxh+dyh*dyh <= r*r {
				idx = i
				break
			}
		}
		if idx != hoverIdx.Get() {
			hoverIdx.Set(idx)
			if idx >= 0 && o.OnHoverNode != nil {
				o.OnHoverNode(data.Nodes[idx].ID)
			}
		}
		if !dragging.Get() {
			return
		}
		if idx := dragNode.Get(); idx >= 0 && idx < len(data.Nodes) {
			wx, wy := screenToWorld(x, y)
			data.Nodes[idx].X = wx
			data.Nodes[idx].Y = wy
		} else {
			offsetX.Set(offsetX.Get() + dx)
			offsetY.Set(offsetY.Get() + dy)
		}
		// detect movement for click vs drag
		if !didMove.Get() {
			dsx := x - mouseDownX.Get()
			dsy := y - mouseDownY.Get()
			if dsx*dsx+dsy*dsy > 9 { // 3px threshold
				didMove.Set(true)
			}
		}
		requestDraw(c)
	}

	onMouseUp := func() {
		dragging.Set(false)
		if idx := dragNode.Get(); idx >= 0 && !didMove.Get() {
			selectedIdx.Set(idx)
			if o.OnSelectNode != nil {
				o.OnSelectNode(data.Nodes[idx].ID)
			}
		}
		dragNode.Set(-1)
	}

	var vx = make([]float64, len(data.Nodes))
	var vy = make([]float64, len(data.Nodes))
	layoutTick := func(dt float64) {
		// Repulsion forces between all nodes
		for i := range data.Nodes {
			for j := i + 1; j < len(data.Nodes); j++ {
				dx := data.Nodes[j].X - data.Nodes[i].X
				dy := data.Nodes[j].Y - data.Nodes[i].Y
				dist2 := dx*dx + dy*dy + 0.01
				force := o.Repulsion / dist2
				invDist := 1.0 / math.Sqrt(dist2)
				fx := force * dx * invDist
				fy := force * dy * invDist
				vx[i] -= fx
				vy[i] -= fy
				vx[j] += fx
				vy[j] += fy
			}
		}

		// Spring forces for edges
		for _, e := range data.Edges {
			si := indexOfNodeWASM(data.Nodes, e.Source)
			ti := indexOfNodeWASM(data.Nodes, e.Target)
			if si < 0 || ti < 0 {
				continue
			}
			dx := data.Nodes[ti].X - data.Nodes[si].X
			dy := data.Nodes[ti].Y - data.Nodes[si].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist == 0 {
				continue
			}
			diff := dist - o.SpringLength
			k := o.SpringStiffness
			fx := k * diff * dx / dist
			fy := k * diff * dy / dist
			vx[si] += fx
			vy[si] += fy
			vx[ti] -= fx
			vy[ti] -= fy
		}

		// Apply gravity/centering force to prevent drift
		// This pulls nodes toward the center (0, 0)
		if o.Gravity > 0 {
			for i := range data.Nodes {
				// Calculate center of mass
				cx := data.Nodes[i].X
				cy := data.Nodes[i].Y

				// Apply weak centering force toward origin
				vx[i] -= cx * o.Gravity
				vy[i] -= cy * o.Gravity
			}
		}

		// Update positions
		for i := range data.Nodes {
			if dragNode.Get() == i {
				vx[i] = 0
				vy[i] = 0
				continue
			}
			vx[i] *= o.Damping
			vy[i] *= o.Damping
			data.Nodes[i].X += vx[i] * dt
			data.Nodes[i].Y += vy[i] * dt
		}
	}

	draw := func(c js.Value) {
		if c.IsUndefined() || c.IsNull() {
			return
		}
		rect := c.Call("getBoundingClientRect")
		widthCss := rect.Get("width").Float()
		heightCss := rect.Get("height").Float()
		dprVal := js.Global().Get("window").Get("devicePixelRatio")
		pixelRatio := 1.0
		if dprVal.Truthy() {
			pixelRatio = dprVal.Float()
		}
		// Set backing store size in device pixels
		c.Set("width", int(widthCss*pixelRatio))
		c.Set("height", int(heightCss*pixelRatio))
		ctx := c.Call("getContext", "2d")
		if !ctx.Truthy() {
			return
		}
		ctx.Call("save")
		// Normalize to CSS pixel coordinate system
		ctx.Call("scale", pixelRatio, pixelRatio)
		// Clear background in CSS pixels
		ctx.Set("fillStyle", o.BackgroundColor)
		ctx.Call("fillRect", 0, 0, widthCss, heightCss)
		// Apply viewport transform (CSS px), then world scale
		ctx.Call("translate", offsetX.Get(), offsetY.Get())
		ctx.Call("scale", scale.Get(), scale.Get())
		ctx.Set("strokeStyle", o.EdgeColor)
		ctx.Set("lineWidth", 1.0/scale.Get())
		for _, e := range data.Edges {
			si := indexOfNodeWASM(data.Nodes, e.Source)
			ti := indexOfNodeWASM(data.Nodes, e.Target)
			if si < 0 || ti < 0 {
				continue
			}
			ctx.Call("beginPath")
			ctx.Call("moveTo", data.Nodes[si].X, data.Nodes[si].Y)
			ctx.Call("lineTo", data.Nodes[ti].X, data.Nodes[ti].Y)
			ctx.Call("stroke")
		}
		for _, n := range data.Nodes {
			r := n.Size
			if r <= 0 {
				r = 8
			}
			ctx.Set("fillStyle", nonEmptyWASM(n.Color, o.NodeColor))
			ctx.Call("beginPath")
			ctx.Call("arc", n.X, n.Y, r, 0, math.Pi*2)
			ctx.Call("fill")
			// Highlight selected
			if selectedIdx.Get() >= 0 && data.Nodes[selectedIdx.Get()].ID == n.ID {
				ctx.Set("strokeStyle", "#ffcf33")
				ctx.Set("lineWidth", 2.0/scale.Get())
				ctx.Call("beginPath")
				ctx.Call("arc", n.X, n.Y, r+3/scale.Get(), 0, math.Pi*2)
				ctx.Call("stroke")
			}
			// Highlight hover
			if hoverIdx.Get() >= 0 && data.Nodes[hoverIdx.Get()].ID == n.ID {
				ctx.Set("strokeStyle", "#9ad0ff")
				ctx.Set("lineWidth", 1.5/scale.Get())
				ctx.Call("beginPath")
				ctx.Call("arc", n.X, n.Y, r+2/scale.Get(), 0, math.Pi*2)
				ctx.Call("stroke")
			}
			if n.Label != "" {
				ctx.Set("fillStyle", o.LabelColor)
				ctx.Set("font", fmt.Sprintf("%fpx sans-serif", 12.0/scale.Get()))
				ctx.Call("fillText", n.Label, n.X+r+4/scale.Get(), n.Y)
			}
		}
		ctx.Call("restore")
	}

	var raf js.Func
	raf = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !canvasRef.Truthy() {
			return nil
		}
		layoutTick(0.016)
		draw(canvasRef)
		if o.OnViewportChange != nil {
			o.OnViewportChange(offsetX.Get(), offsetY.Get(), scale.Get())
		}
		js.Global().Get("window").Call("requestAnimationFrame", raf)
		return nil
	})
	started := false
	onRef := func(el js.Value) {
		canvasRef = el
		if !started {
			started = true
			// Defer initial centering to after first layout tick; then fit bounds
			js.Global().Get("window").Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// Compute bounds of graph
				if len(data.Nodes) > 0 {
					minx, miny := data.Nodes[0].X, data.Nodes[0].Y
					maxx, maxy := minx, miny
					for _, n := range data.Nodes {
						if n.X < minx {
							minx = n.X
						}
						if n.Y < miny {
							miny = n.Y
						}
						if n.X > maxx {
							maxx = n.X
						}
						if n.Y > maxy {
							maxy = n.Y
						}
					}
					rect := el.Call("getBoundingClientRect")
					w := rect.Get("width").Float()
					h := rect.Get("height").Float()
					gw := maxx - minx
					if gw <= 0 {
						gw = 1
					}
					gh := maxy - miny
					if gh <= 0 {
						gh = 1
					}
					padding := 40.0
					sx := (w - 2*padding) / gw
					sy := (h - 2*padding) / gh
					s := sx
					if sy < s {
						s = sy
					}
					if s <= 0 {
						s = 1
					}
					scale.Set(s)
					offsetX.Set(w*0.5 - (minx+gw*0.5)*s)
					offsetY.Set(h*0.5 - (miny+gh*0.5)*s)
				}
				js.Global().Get("window").Call("requestAnimationFrame", raf)
				return nil
			}))
		}
		requestDraw(el)
	}

	// Node picking helper
	pickNode := func(x, y float64) int {
		wx, wy := screenToWorld(x, y)
		for i := range data.Nodes {
			r := data.Nodes[i].Size
			if r <= 0 {
				r = 8
			}
			dx := wx - data.Nodes[i].X
			dy := wy - data.Nodes[i].Y
			if dx*dx+dy*dy <= r*r {
				return i
			}
		}
		return -1
	}
	// (removed pickNodeAt helper to keep symbol usage minimal)
	onDblClick := func(x, y float64) {
		idx := pickNode(x, y)
		if idx >= 0 && o.OnDblClickNode != nil {
			o.OnDblClickNode(data.Nodes[idx].ID)
		}
	}

	return builder.Canvas().
		Style("width:100%;height:100%;display:block;touch-action:none").
		Ref(onRef).
		OnWheel(onWheel).
		OnMouseDown(onMouseDown).
		OnMouseMove(func(x, y float64) { lastX.Set(x); lastY.Set(y); onMouseMove(x, y) }).
		OnMouseUp(func() { onMouseUp() }).
		OnDblClick(func(x, y float64) { onDblClick(x, y) }).
		Build()
}

func requestDraw(c js.Value) {
	if !c.Truthy() {
		return
	}
	js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	}))
}

func indexOfNodeWASM(nodes []Node, id string) int {
	for i := range nodes {
		if nodes[i].ID == id {
			return i
		}
	}
	return -1
}

func nonEmptyWASM(s, def string) string {
	if s != "" {
		return s
	}
	return def
}
