package components

import (
	"github.com/recera/vango/pkg/vex/builder"
	"github.com/recera/vango/pkg/vango/vdom"
)

// CardProps defines the properties for the Card component
type CardProps struct {
	Title       string
	Subtitle    string
	Content     *vdom.VNode
	Footer      *vdom.VNode
	Image       string
	ImageAlt    string
	Hoverable   bool
	Clickable   bool
	OnClick     func()
	Class       string
	ID          string
	Bordered    bool
	Shadow      string // "none", "sm", "md", "lg", "xl"
}

// Card creates a reusable card component
func Card(props CardProps) *vdom.VNode {
	// Build class names
	classes := []string{"card"}
	
	if props.Hoverable {
		classes = append(classes, "card-hoverable")
	}
	
	if props.Clickable {
		classes = append(classes, "card-clickable")
	}
	
	if props.Bordered {
		classes = append(classes, "card-bordered")
	}
	
	// Add shadow class
	if props.Shadow != "" && props.Shadow != "none" {
		classes = append(classes, "card-shadow-"+props.Shadow)
	} else if props.Shadow == "" {
		// Default shadow
		classes = append(classes, "card-shadow-md")
	}
	
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	// Build card
	card := builder.Div().Class(joinClasses(classes...))
	
	if props.ID != "" {
		card.ID(props.ID)
	}
	
	if props.OnClick != nil && props.Clickable {
		card.OnClick(props.OnClick)
	}
	
	// Build children
	var children []*vdom.VNode
	
	// Add image if provided
	if props.Image != "" {
		imgAlt := props.ImageAlt
		if imgAlt == "" {
			imgAlt = props.Title
		}
		
		children = append(children, 
			builder.Div().
				Class("card-image").
				Children(
					builder.Img().
						Src(props.Image).
						Alt(imgAlt).
						Build(),
				).Build(),
		)
	}
	
	// Add card body
	var bodyChildren []*vdom.VNode
	
	// Add header if title or subtitle exists
	if props.Title != "" || props.Subtitle != "" {
		var headerChildren []*vdom.VNode
		
		if props.Title != "" {
			headerChildren = append(headerChildren,
				builder.H3().
					Class("card-title").
					Text(props.Title).
					Build(),
			)
		}
		
		if props.Subtitle != "" {
			headerChildren = append(headerChildren,
				builder.P().
					Class("card-subtitle").
					Text(props.Subtitle).
					Build(),
			)
		}
		
		bodyChildren = append(bodyChildren,
			builder.Div().
				Class("card-header").
				Children(headerChildren...).
				Build(),
		)
	}
	
	// Add content
	if props.Content != nil {
		bodyChildren = append(bodyChildren,
			builder.Div().
				Class("card-content").
				Children(props.Content).
				Build(),
		)
	}
	
	children = append(children,
		builder.Div().
			Class("card-body").
			Children(bodyChildren...).
			Build(),
	)
	
	// Add footer if provided
	if props.Footer != nil {
		children = append(children,
			builder.Div().
				Class("card-footer").
				Children(props.Footer).
				Build(),
		)
	}
	
	return card.Children(children...).Build()
}

// CardGrid creates a responsive grid of cards
type CardGridProps struct {
	Cards   []*vdom.VNode // Pre-built card VNodes
	Columns int          // Number of columns (1-4)
	Gap     string       // "sm", "md", "lg"
	Class   string
}

func CardGrid(props CardGridProps) *vdom.VNode {
	// Default values
	if props.Columns == 0 {
		props.Columns = 3
	}
	if props.Gap == "" {
		props.Gap = "md"
	}
	
	classes := []string{
		"card-grid",
		"card-grid-" + string(rune(props.Columns+'0')),
		"card-grid-gap-" + props.Gap,
	}
	
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	return builder.Div().
		Class(joinClasses(classes...)).
		Children(props.Cards...).
		Build()
}