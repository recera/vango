package components

import (
	"fmt"
	"github.com/recera/vango/pkg/vex/builder"
	"github.com/recera/vango/pkg/vango/vdom"
)

// ButtonVariant defines the visual style of the button
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
	ButtonSuccess   ButtonVariant = "success"
	ButtonWarning   ButtonVariant = "warning"
	ButtonGhost     ButtonVariant = "ghost"
)

// ButtonSize defines the size of the button
type ButtonSize string

const (
	ButtonSmall  ButtonSize = "small"
	ButtonMedium ButtonSize = "medium"
	ButtonLarge  ButtonSize = "large"
)

// ButtonProps defines the properties for the Button component
type ButtonProps struct {
	Text     string
	Variant  ButtonVariant
	Size     ButtonSize
	Disabled bool
	Loading  bool
	Icon     *vdom.VNode
	OnClick  func()
	Class    string
	ID       string
}

// Button creates a reusable button component
func Button(props ButtonProps) *vdom.VNode {
	// Default values
	if props.Variant == "" {
		props.Variant = ButtonPrimary
	}
	if props.Size == "" {
		props.Size = ButtonMedium
	}
	
	// Build class names
	classes := []string{"btn"}
	classes = append(classes, fmt.Sprintf("btn-%s", props.Variant))
	classes = append(classes, fmt.Sprintf("btn-%s", props.Size))
	
	if props.Disabled || props.Loading {
		classes = append(classes, "btn-disabled")
	}
	
	if props.Loading {
		classes = append(classes, "btn-loading")
	}
	
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	// Build button
	btn := builder.Button().
		Class(joinClasses(classes...)).
		Disabled(props.Disabled || props.Loading)
	
	if props.ID != "" {
		btn.ID(props.ID)
	}
	
	if props.OnClick != nil && !props.Disabled && !props.Loading {
		btn.OnClick(props.OnClick)
	}
	
	// Build children
	var children []*vdom.VNode
	
	// Add loading spinner if loading
	if props.Loading {
		children = append(children, LoadingSpinner(SpinnerProps{
			Size:  "small",
			Color: "currentColor",
		}))
	}
	
	// Add icon if provided
	if props.Icon != nil && !props.Loading {
		children = append(children, props.Icon)
	}
	
	// Add text
	if props.Text != "" {
		children = append(children, builder.Span().Text(props.Text).Build())
	}
	
	return btn.Children(children...).Build()
}

// IconButton creates a button with only an icon
func IconButton(props ButtonProps) *vdom.VNode {
	props.Class = joinClasses(props.Class, "btn-icon")
	return Button(props)
}

// ButtonGroup creates a group of buttons
type ButtonGroupProps struct {
	Buttons   []ButtonProps
	Direction string // "horizontal" or "vertical"
	Class     string
}

func ButtonGroup(props ButtonGroupProps) *vdom.VNode {
	if props.Direction == "" {
		props.Direction = "horizontal"
	}
	
	classes := []string{"btn-group", fmt.Sprintf("btn-group-%s", props.Direction)}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	
	var buttons []*vdom.VNode
	for _, btnProps := range props.Buttons {
		buttons = append(buttons, Button(btnProps))
	}
	
	return builder.Div().
		Class(joinClasses(classes...)).
		Children(buttons...).
		Build()
}

// Helper function to join classes
func joinClasses(classes ...string) string {
	result := ""
	for i, class := range classes {
		if class != "" {
			if i > 0 {
				result += " "
			}
			result += class
		}
	}
	return result
}