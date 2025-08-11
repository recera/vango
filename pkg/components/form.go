package components

import (
	"github.com/recera/vango/pkg/vex/builder"
	"github.com/recera/vango/pkg/vango/vdom"
)

// InputProps defines properties for input components
type InputProps struct {
	Type        string // "text", "email", "password", "number", "tel", "url", "search"
	Name        string
	Value       string
	Placeholder string
	Label       string
	HelperText  string
	ErrorText   string
	Required    bool
	Disabled    bool
	ReadOnly    bool
	AutoFocus   bool
	MaxLength   int
	Pattern     string
	OnInput     func(value string)
	OnChange    func(value string)
	OnFocus     func()
	OnBlur      func()
	Class       string
	ID          string
}

// Input creates a form input component with label and validation
func Input(props InputProps) *vdom.VNode {
	// Default type
	if props.Type == "" {
		props.Type = "text"
	}
	
	// Generate ID if not provided
	inputID := props.ID
	if inputID == "" && props.Name != "" {
		inputID = "input-" + props.Name
	}
	
	// Build container
	containerClasses := []string{"form-field"}
	if props.ErrorText != "" {
		containerClasses = append(containerClasses, "form-field-error")
	}
	if props.Disabled {
		containerClasses = append(containerClasses, "form-field-disabled")
	}
	
	var children []*vdom.VNode
	
	// Add label if provided
	if props.Label != "" {
		labelBuilder := builder.Label().
			Class("form-label").
			Text(props.Label)
		
		if inputID != "" {
			labelBuilder.For(inputID)
		}
		
		if props.Required {
			labelBuilder.Children(
				builder.Span().Text(props.Label).Build(),
				builder.Span().Class("form-required").Text(" *").Build(),
			)
		}
		
		children = append(children, labelBuilder.Build())
	}
	
	// Build input
	inputClasses := []string{"form-input"}
	if props.Class != "" {
		inputClasses = append(inputClasses, props.Class)
	}
	
	input := builder.Input().
		Type(props.Type).
		Class(joinClasses(inputClasses...)).
		Value(props.Value)
	
	if inputID != "" {
		input.ID(inputID)
	}
	
	if props.Name != "" {
		input.Name(props.Name)
	}
	
	if props.Placeholder != "" {
		input.Placeholder(props.Placeholder)
	}
	
	if props.Required {
		input.Required(true)
	}
	
	if props.Disabled {
		input.Disabled(true)
	}
	
	if props.ReadOnly {
		input.ReadOnly(true)
	}
	
	if props.AutoFocus {
		input.AutoFocus(true)
	}
	
	if props.MaxLength > 0 {
		input.MaxLength(props.MaxLength)
	}
	
	if props.Pattern != "" {
		input.Pattern(props.Pattern)
	}
	
	// Add event handlers
	if props.OnInput != nil {
		input.OnInput(props.OnInput)
	}
	
	if props.OnChange != nil {
		input.OnChange(props.OnChange)
	}
	
	if props.OnFocus != nil {
		input.OnFocus(props.OnFocus)
	}
	
	if props.OnBlur != nil {
		input.OnBlur(props.OnBlur)
	}
	
	children = append(children, input.Build())
	
	// Add helper or error text
	if props.ErrorText != "" {
		children = append(children,
			builder.Span().
				Class("form-error").
				Text(props.ErrorText).
				Build(),
		)
	} else if props.HelperText != "" {
		children = append(children,
			builder.Span().
				Class("form-helper").
				Text(props.HelperText).
				Build(),
		)
	}
	
	return builder.Div().
		Class(joinClasses(containerClasses...)).
		Children(children...).
		Build()
}

// TextareaProps defines properties for textarea components
type TextareaProps struct {
	Name        string
	Value       string
	Placeholder string
	Label       string
	HelperText  string
	ErrorText   string
	Rows        int
	Cols        int
	Required    bool
	Disabled    bool
	ReadOnly    bool
	AutoFocus   bool
	MaxLength   int
	OnInput     func(value string)
	OnChange    func(value string)
	Class       string
	ID          string
}

// Textarea creates a form textarea component
func Textarea(props TextareaProps) *vdom.VNode {
	// Default rows
	if props.Rows == 0 {
		props.Rows = 4
	}
	
	// Generate ID if not provided
	textareaID := props.ID
	if textareaID == "" && props.Name != "" {
		textareaID = "textarea-" + props.Name
	}
	
	// Build container
	containerClasses := []string{"form-field"}
	if props.ErrorText != "" {
		containerClasses = append(containerClasses, "form-field-error")
	}
	if props.Disabled {
		containerClasses = append(containerClasses, "form-field-disabled")
	}
	
	var children []*vdom.VNode
	
	// Add label if provided
	if props.Label != "" {
		labelBuilder := builder.Label().
			Class("form-label").
			Text(props.Label)
		
		if textareaID != "" {
			labelBuilder.For(textareaID)
		}
		
		if props.Required {
			labelBuilder.Children(
				builder.Span().Text(props.Label).Build(),
				builder.Span().Class("form-required").Text(" *").Build(),
			)
		}
		
		children = append(children, labelBuilder.Build())
	}
	
	// Build textarea
	textareaClasses := []string{"form-textarea"}
	if props.Class != "" {
		textareaClasses = append(textareaClasses, props.Class)
	}
	
	textarea := builder.Textarea().
		Class(joinClasses(textareaClasses...)).
		Rows(props.Rows).
		Text(props.Value)
	
	if textareaID != "" {
		textarea.ID(textareaID)
	}
	
	if props.Name != "" {
		textarea.Name(props.Name)
	}
	
	if props.Placeholder != "" {
		textarea.Placeholder(props.Placeholder)
	}
	
	if props.Cols > 0 {
		textarea.Cols(props.Cols)
	}
	
	if props.Required {
		textarea.Required(true)
	}
	
	if props.Disabled {
		textarea.Disabled(true)
	}
	
	if props.ReadOnly {
		textarea.ReadOnly(true)
	}
	
	if props.AutoFocus {
		textarea.AutoFocus(true)
	}
	
	if props.MaxLength > 0 {
		textarea.MaxLength(props.MaxLength)
	}
	
	// Add event handlers
	if props.OnInput != nil {
		textarea.OnInput(props.OnInput)
	}
	
	if props.OnChange != nil {
		textarea.OnChange(props.OnChange)
	}
	
	children = append(children, textarea.Build())
	
	// Add helper or error text
	if props.ErrorText != "" {
		children = append(children,
			builder.Span().
				Class("form-error").
				Text(props.ErrorText).
				Build(),
		)
	} else if props.HelperText != "" {
		children = append(children,
			builder.Span().
				Class("form-helper").
				Text(props.HelperText).
				Build(),
		)
	}
	
	return builder.Div().
		Class(joinClasses(containerClasses...)).
		Children(children...).
		Build()
}

// SelectOption represents an option in a select dropdown
type SelectOption struct {
	Value    string
	Label    string
	Disabled bool
	Selected bool
}

// SelectProps defines properties for select components
type SelectProps struct {
	Name        string
	Options     []SelectOption
	Value       string
	Label       string
	Placeholder string
	HelperText  string
	ErrorText   string
	Required    bool
	Disabled    bool
	Multiple    bool
	OnChange    func(value string)
	Class       string
	ID          string
}

// Select creates a form select dropdown component
func Select(props SelectProps) *vdom.VNode {
	// Generate ID if not provided
	selectID := props.ID
	if selectID == "" && props.Name != "" {
		selectID = "select-" + props.Name
	}
	
	// Build container
	containerClasses := []string{"form-field"}
	if props.ErrorText != "" {
		containerClasses = append(containerClasses, "form-field-error")
	}
	if props.Disabled {
		containerClasses = append(containerClasses, "form-field-disabled")
	}
	
	var children []*vdom.VNode
	
	// Add label if provided
	if props.Label != "" {
		labelBuilder := builder.Label().
			Class("form-label").
			Text(props.Label)
		
		if selectID != "" {
			labelBuilder.For(selectID)
		}
		
		if props.Required {
			labelBuilder.Children(
				builder.Span().Text(props.Label).Build(),
				builder.Span().Class("form-required").Text(" *").Build(),
			)
		}
		
		children = append(children, labelBuilder.Build())
	}
	
	// Build select
	selectClasses := []string{"form-select"}
	if props.Class != "" {
		selectClasses = append(selectClasses, props.Class)
	}
	
	selectBuilder := builder.Select().
		Class(joinClasses(selectClasses...))
	
	if selectID != "" {
		selectBuilder.ID(selectID)
	}
	
	if props.Name != "" {
		selectBuilder.Name(props.Name)
	}
	
	if props.Required {
		selectBuilder.Required(true)
	}
	
	if props.Disabled {
		selectBuilder.Disabled(true)
	}
	
	if props.Multiple {
		selectBuilder.Multiple(true)
	}
	
	if props.OnChange != nil {
		selectBuilder.OnChange(props.OnChange)
	}
	
	// Build options
	var options []*vdom.VNode
	
	// Add placeholder option if provided
	if props.Placeholder != "" {
		options = append(options,
			builder.Option().
				Value("").
				Disabled(true).
				Selected(props.Value == "").
				Text(props.Placeholder).
				Build(),
		)
	}
	
	// Add regular options
	for _, opt := range props.Options {
		option := builder.Option().
			Value(opt.Value).
			Text(opt.Label)
		
		if opt.Disabled {
			option.Disabled(true)
		}
		
		if opt.Selected || opt.Value == props.Value {
			option.Selected(true)
		}
		
		options = append(options, option.Build())
	}
	
	selectBuilder.Children(options...)
	children = append(children, selectBuilder.Build())
	
	// Add helper or error text
	if props.ErrorText != "" {
		children = append(children,
			builder.Span().
				Class("form-error").
				Text(props.ErrorText).
				Build(),
		)
	} else if props.HelperText != "" {
		children = append(children,
			builder.Span().
				Class("form-helper").
				Text(props.HelperText).
				Build(),
		)
	}
	
	return builder.Div().
		Class(joinClasses(containerClasses...)).
		Children(children...).
		Build()
}

// CheckboxProps defines properties for checkbox components
type CheckboxProps struct {
	Name     string
	Label    string
	Checked  bool
	Disabled bool
	OnChange func(checked bool)
	Class    string
	ID       string
}

// Checkbox creates a form checkbox component
func Checkbox(props CheckboxProps) *vdom.VNode {
	// Generate ID if not provided
	checkboxID := props.ID
	if checkboxID == "" && props.Name != "" {
		checkboxID = "checkbox-" + props.Name
	}
	
	// Build checkbox input
	checkbox := builder.Input().
		Type("checkbox").
		Class("form-checkbox").
		Checked(props.Checked)
	
	if checkboxID != "" {
		checkbox.ID(checkboxID)
	}
	
	if props.Name != "" {
		checkbox.Name(props.Name)
	}
	
	if props.Disabled {
		checkbox.Disabled(true)
	}
	
	if props.OnChange != nil {
		// Create a wrapper that converts string to bool
		// The browser sends the checkbox value as a string
		checkbox.Attr("onchange", func() {
			// Toggle the checked state
			props.OnChange(!props.Checked)
		})
	}
	
	// Build label
	labelClasses := []string{"form-checkbox-label"}
	if props.Disabled {
		labelClasses = append(labelClasses, "form-checkbox-label-disabled")
	}
	if props.Class != "" {
		labelClasses = append(labelClasses, props.Class)
	}
	
	label := builder.Label().
		Class(joinClasses(labelClasses...))
	
	if checkboxID != "" {
		label.For(checkboxID)
	}
	
	return builder.Div().
		Class("form-checkbox-container").
		Children(
			checkbox.Build(),
			label.Text(props.Label).Build(),
		).Build()
}