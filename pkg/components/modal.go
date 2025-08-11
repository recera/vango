package components

import (
	"github.com/recera/vango/pkg/vex/builder"
	"github.com/recera/vango/pkg/vango/vdom"
)

// ModalProps defines the properties for the Modal component
type ModalProps struct {
	Title        string
	Content      *vdom.VNode
	Footer       *vdom.VNode
	IsOpen       bool
	OnClose      func()
	Size         string // "sm", "md", "lg", "xl", "full"
	CloseOnEsc   bool
	CloseOnClick bool // Close when clicking outside
	ShowClose    bool
	Class        string
	ID           string
}

// Modal creates a reusable modal/dialog component
func Modal(props ModalProps) *vdom.VNode {
	if !props.IsOpen {
		return nil
	}
	
	// Default values
	if props.Size == "" {
		props.Size = "md"
	}
	if !props.ShowClose {
		props.ShowClose = true
	}
	
	// Build modal classes
	modalClasses := []string{"modal", "modal-" + props.Size}
	if props.Class != "" {
		modalClasses = append(modalClasses, props.Class)
	}
	
	// Build overlay
	overlay := builder.Div().
		Class("modal-overlay")
	
	if props.CloseOnClick && props.OnClose != nil {
		overlay.OnClick(props.OnClose)
	}
	
	// Build modal content
	var modalChildren []*vdom.VNode
	
	// Add header with title and close button
	if props.Title != "" || props.ShowClose {
		var headerChildren []*vdom.VNode
		
		if props.Title != "" {
			headerChildren = append(headerChildren,
				builder.H2().
					Class("modal-title").
					Text(props.Title).
					Build(),
			)
		}
		
		if props.ShowClose && props.OnClose != nil {
			headerChildren = append(headerChildren,
				builder.Button().
					Class("modal-close").
					OnClick(props.OnClose).
					Children(
						builder.Span().Text("×").Build(),
					).Build(),
			)
		}
		
		modalChildren = append(modalChildren,
			builder.Div().
				Class("modal-header").
				Children(headerChildren...).
				Build(),
		)
	}
	
	// Add content
	if props.Content != nil {
		modalChildren = append(modalChildren,
			builder.Div().
				Class("modal-body").
				Children(props.Content).
				Build(),
		)
	}
	
	// Add footer
	if props.Footer != nil {
		modalChildren = append(modalChildren,
			builder.Div().
				Class("modal-footer").
				Children(props.Footer).
				Build(),
		)
	}
	
	// Build modal dialog
	dialog := builder.Div().
		Class(joinClasses(modalClasses...)).
		Children(modalChildren...)
	
	if props.ID != "" {
		dialog.ID(props.ID)
	}
	
	// Add a click handler to prevent bubbling to overlay
	// This prevents the modal from closing when clicking inside
	dialog.Attr("data-modal-dialog", "true")
	
	// Create portal container
	return builder.Div().
		Class("modal-container").
		Children(
			overlay.Build(),
			dialog.Build(),
		).Build()
}

// AlertProps defines properties for alert dialogs
type AlertProps struct {
	Title   string
	Message string
	Type    string // "info", "success", "warning", "error"
	OnOk    func()
	OkText  string
}

// Alert creates a simple alert dialog
func Alert(props AlertProps) *vdom.VNode {
	if props.Type == "" {
		props.Type = "info"
	}
	if props.OkText == "" {
		props.OkText = "OK"
	}
	
	// Build alert icon based on type
	var icon *vdom.VNode
	switch props.Type {
	case "success":
		icon = builder.Span().Class("alert-icon alert-icon-success").Text("✓").Build()
	case "warning":
		icon = builder.Span().Class("alert-icon alert-icon-warning").Text("⚠").Build()
	case "error":
		icon = builder.Span().Class("alert-icon alert-icon-error").Text("✕").Build()
	default:
		icon = builder.Span().Class("alert-icon alert-icon-info").Text("ℹ").Build()
	}
	
	// Build content
	content := builder.Div().
		Class("alert-content").
		Children(
			icon,
			builder.P().Text(props.Message).Build(),
		).Build()
	
	// Build footer with OK button
	footer := builder.Div().
		Class("alert-actions").
		Children(
			Button(ButtonProps{
				Text:    props.OkText,
				Variant: ButtonPrimary,
				OnClick: props.OnOk,
			}),
		).Build()
	
	return Modal(ModalProps{
		Title:        props.Title,
		Content:      content,
		Footer:       footer,
		IsOpen:       true,
		OnClose:      props.OnOk,
		Size:         "sm",
		ShowClose:    false,
		CloseOnClick: false,
		Class:        "alert-modal alert-" + props.Type,
	})
}

// ConfirmProps defines properties for confirmation dialogs
type ConfirmProps struct {
	Title      string
	Message    string
	OnConfirm  func()
	OnCancel   func()
	ConfirmText string
	CancelText  string
	Dangerous   bool // Shows confirm button in danger color
}

// Confirm creates a confirmation dialog
func Confirm(props ConfirmProps) *vdom.VNode {
	if props.ConfirmText == "" {
		props.ConfirmText = "Confirm"
	}
	if props.CancelText == "" {
		props.CancelText = "Cancel"
	}
	
	// Build content
	content := builder.Div().
		Class("confirm-content").
		Children(
			builder.P().Text(props.Message).Build(),
		).Build()
	
	// Build footer with action buttons
	confirmVariant := ButtonPrimary
	if props.Dangerous {
		confirmVariant = ButtonDanger
	}
	
	footer := builder.Div().
		Class("confirm-actions").
		Children(
			Button(ButtonProps{
				Text:    props.CancelText,
				Variant: ButtonSecondary,
				OnClick: props.OnCancel,
			}),
			Button(ButtonProps{
				Text:    props.ConfirmText,
				Variant: confirmVariant,
				OnClick: props.OnConfirm,
			}),
		).Build()
	
	return Modal(ModalProps{
		Title:        props.Title,
		Content:      content,
		Footer:       footer,
		IsOpen:       true,
		OnClose:      props.OnCancel,
		Size:         "sm",
		CloseOnClick: true,
		Class:        "confirm-modal",
	})
}