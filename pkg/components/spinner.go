package components

import (
	"fmt"
	"github.com/recera/vango/pkg/vex/builder"
	"github.com/recera/vango/pkg/vango/vdom"
)

// SpinnerProps defines the properties for the LoadingSpinner component
type SpinnerProps struct {
	Size  string // "small", "medium", "large"
	Color string // CSS color value
	Text  string // Optional loading text
	Class string
}

// LoadingSpinner creates a loading spinner component
func LoadingSpinner(props SpinnerProps) *vdom.VNode {
	// Default values
	if props.Size == "" {
		props.Size = "medium"
	}
	if props.Color == "" {
		props.Color = "#3b82f6"
	}
	
	// Build classes
	classes := []string{"spinner", "spinner-" + props.Size}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	// Create spinner SVG
	var width, height string
	switch props.Size {
	case "small":
		width, height = "16", "16"
	case "large":
		width, height = "48", "48"
	default:
		width, height = "24", "24"
	}
	
	spinner := builder.Svg().
		Attr("width", width).
		Attr("height", height).
		Attr("viewBox", "0 0 24 24").
		Attr("fill", "none").
		Class(joinClasses(classes...)).
		Children(
			builder.Circle().
				Attr("cx", "12").
				Attr("cy", "12").
				Attr("r", "10").
				Attr("stroke", props.Color).
				Attr("stroke-width", "2").
				Attr("stroke-opacity", "0.25").
				Build(),
			builder.Circle().
				Attr("cx", "12").
				Attr("cy", "12").
				Attr("r", "10").
				Attr("stroke", props.Color).
				Attr("stroke-width", "2").
				Attr("stroke-linecap", "round").
				Attr("stroke-dasharray", "32").
				Attr("stroke-dashoffset", "32").
				Class("spinner-track").
				Build(),
		).Build()
	
	// If text is provided, wrap spinner and text
	if props.Text != "" {
		return builder.Div().
			Class("spinner-container").
			Children(
				spinner,
				builder.Span().
					Class("spinner-text").
					Text(props.Text).
					Build(),
			).Build()
	}
	
	return spinner
}

// SkeletonProps defines properties for skeleton loading placeholders
type SkeletonProps struct {
	Type   string // "text", "title", "avatar", "image", "button"
	Width  string // CSS width value
	Height string // CSS height value
	Lines  int    // Number of lines for text skeleton
	Class  string
}

// Skeleton creates a skeleton loading placeholder
func Skeleton(props SkeletonProps) *vdom.VNode {
	// Default type
	if props.Type == "" {
		props.Type = "text"
	}
	
	classes := []string{"skeleton", "skeleton-" + props.Type}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	// Build skeleton based on type
	switch props.Type {
	case "title":
		return builder.Div().
			Class(joinClasses(classes...)).
			Style(fmt.Sprintf("width: %s; height: 2rem", getWidth(props.Width, "60%"))).
			Build()
			
	case "avatar":
		size := getWidth(props.Width, "40px")
		return builder.Div().
			Class(joinClasses(classes...)).
			Style(fmt.Sprintf("width: %s; height: %s", size, size)).
			Build()
			
	case "image":
		return builder.Div().
			Class(joinClasses(classes...)).
			Style(fmt.Sprintf("width: %s; height: %s", 
				getWidth(props.Width, "100%"),
				getHeight(props.Height, "200px"))).
			Build()
			
	case "button":
		return builder.Div().
			Class(joinClasses(classes...)).
			Style(fmt.Sprintf("width: %s; height: 2.5rem", getWidth(props.Width, "100px"))).
			Build()
			
	case "text":
		lines := props.Lines
		if lines == 0 {
			lines = 3
		}
		
		var skeletonLines []*vdom.VNode
		for i := 0; i < lines; i++ {
			// Last line is shorter
			width := "100%"
			if i == lines-1 {
				width = "75%"
			}
			
			skeletonLines = append(skeletonLines,
				builder.Div().
					Class("skeleton-line").
					Style(fmt.Sprintf("width: %s", width)).
					Build(),
			)
		}
		
		return builder.Div().
			Class(joinClasses(classes...)).
			Children(skeletonLines...).
			Build()
			
	default:
		return builder.Div().
			Class(joinClasses(classes...)).
			Style(fmt.Sprintf("width: %s; height: %s",
				getWidth(props.Width, "100%"),
				getHeight(props.Height, "1rem"))).
			Build()
	}
}

// LoadingOverlay creates a full-screen loading overlay
type LoadingOverlayProps struct {
	IsVisible bool
	Text      string
	Blur      bool // Blur background content
}

func LoadingOverlay(props LoadingOverlayProps) *vdom.VNode {
	if !props.IsVisible {
		return nil
	}
	
	classes := []string{"loading-overlay"}
	if props.Blur {
		classes = append(classes, "loading-overlay-blur")
	}
	
	return builder.Div().
		Class(joinClasses(classes...)).
		Children(
			builder.Div().
				Class("loading-overlay-content").
				Children(
					LoadingSpinner(SpinnerProps{
						Size: "large",
						Text: props.Text,
					}),
				).Build(),
		).Build()
}

// Helper functions
func getWidth(provided, defaultVal string) string {
	if provided != "" {
		return provided
	}
	return defaultVal
}

func getHeight(provided, defaultVal string) string {
	if provided != "" {
		return provided
	}
	return defaultVal
}