package graphviewer

// Node represents a graph node
type Node struct {
	ID    string
	Label string
	X     float64
	Y     float64
	Size  float64
	Color string
}

// Edge represents a graph edge between two nodes by ID
type Edge struct {
	Source string
	Target string
	Weight float64
	Color  string
}

// Data holds the graph data
type Data struct {
	Nodes []Node
	Edges []Edge
}

// Options configures the viewer behavior and style
type Options struct {
	// Physics
	Repulsion       float64 // default 2000
	SpringLength    float64 // default 80
	SpringStiffness float64 // default 0.05
	Damping         float64 // default 0.85
	Gravity         float64 // default 0.0 (no gravity)

	// Rendering
	BackgroundColor string // default "#0b0e14"
	NodeColor       string // default "#6ea8fe"
	EdgeColor       string // default "#39424e"
	LabelColor      string // default "#eaeef3"

	// Viewport
	MinScale float64 // default 0.2
	MaxScale float64 // default 5.0

    // Interaction callbacks (optional)
    OnSelectNode func(id string)
    OnDblClickNode func(id string)
    OnHoverNode func(id string)
    OnViewportChange func(offsetX, offsetY, scale float64)
}

func (o *Options) withDefaults() Options {
	d := Options{
		Repulsion:       2000,
		SpringLength:    80,
		SpringStiffness: 0.05,
		Damping:         0.85,
		Gravity:         0.01, // Small centering force by default
		BackgroundColor: "#0b0e14",
		NodeColor:       "#6ea8fe",
		EdgeColor:       "#39424e",
		LabelColor:      "#eaeef3",
		MinScale:        0.2,
		MaxScale:        5.0,
	}
	if o == nil {
		return d
	}
	if o.Repulsion != 0 {
		d.Repulsion = o.Repulsion
	}
	if o.SpringLength != 0 {
		d.SpringLength = o.SpringLength
	}
	if o.SpringStiffness != 0 {
		d.SpringStiffness = o.SpringStiffness
	}
	if o.Damping != 0 {
		d.Damping = o.Damping
	}
	// Allow gravity to be explicitly set to 0 if needed
	if o.Gravity >= 0 {
		d.Gravity = o.Gravity
	}
	if o.BackgroundColor != "" {
		d.BackgroundColor = o.BackgroundColor
	}
	if o.NodeColor != "" {
		d.NodeColor = o.NodeColor
	}
	if o.EdgeColor != "" {
		d.EdgeColor = o.EdgeColor
	}
	if o.LabelColor != "" {
		d.LabelColor = o.LabelColor
	}
	if o.MinScale != 0 {
		d.MinScale = o.MinScale
	}
	if o.MaxScale != 0 {
		d.MaxScale = o.MaxScale
	}
	return d
}
