# Vango Component Library

The Vango component library provides a comprehensive set of pre-built, customizable UI components for rapid application development.

## Installation

The component library is included with Vango:

```go
import "github.com/recera/vango/pkg/components"
```

## Components Overview

### Buttons

#### Basic Button

```go
components.Button(components.ButtonProps{
    Text:    "Click me",
    Variant: components.ButtonPrimary,
    OnClick: handleClick,
})
```

#### Button Variants

- `ButtonPrimary` - Primary action button
- `ButtonSecondary` - Secondary action button
- `ButtonDanger` - Destructive action button
- `ButtonSuccess` - Success/confirmation button
- `ButtonGhost` - Minimal, transparent button

#### Button Sizes

```go
components.Button(components.ButtonProps{
    Text: "Large Button",
    Size: components.ButtonLarge, // Small, Medium, Large
})
```

#### Loading State

```go
components.Button(components.ButtonProps{
    Text:    "Submit",
    Loading: isLoading,
    Disabled: isLoading,
})
```

#### Icon Button

```go
components.IconButton(components.ButtonProps{
    Icon: myIconVNode,
    Variant: components.ButtonGhost,
})
```

#### Button Group

```go
components.ButtonGroup(components.ButtonGroupProps{
    Direction: "horizontal",
    Buttons: []components.ButtonProps{
        {Text: "Save", Variant: components.ButtonPrimary},
        {Text: "Cancel", Variant: components.ButtonSecondary},
    },
})
```

### Cards

#### Basic Card

```go
components.Card(components.CardProps{
    Title:    "Card Title",
    Subtitle: "Card subtitle",
    Content:  contentVNode,
})
```

#### Card with Image

```go
components.Card(components.CardProps{
    Title:    "Product Card",
    Image:    "/images/product.jpg",
    ImageAlt: "Product image",
    Content:  descriptionVNode,
    Footer:   priceVNode,
})
```

#### Interactive Card

```go
components.Card(components.CardProps{
    Title:     "Clickable Card",
    Hoverable: true,
    Clickable: true,
    OnClick:   handleCardClick,
})
```

#### Card Grid

```go
components.CardGrid(components.CardGridProps{
    Columns: 3,
    Gap:     "md",
    Cards: []components.CardProps{
        {Title: "Card 1", Content: content1},
        {Title: "Card 2", Content: content2},
        {Title: "Card 3", Content: content3},
    },
})
```

### Modals

#### Basic Modal

```go
components.Modal(components.ModalProps{
    Title:   "Modal Title",
    Content: modalContent,
    IsOpen:  isModalOpen,
    OnClose: closeModal,
})
```

#### Modal Sizes

```go
components.Modal(components.ModalProps{
    Size: "lg", // "sm", "md", "lg", "xl", "full"
    // ...
})
```

#### Alert Dialog

```go
components.Alert(components.AlertProps{
    Title:   "Success!",
    Message: "Your changes have been saved.",
    Type:    "success", // "info", "success", "warning", "error"
    OnOk:    handleOk,
})
```

#### Confirmation Dialog

```go
components.Confirm(components.ConfirmProps{
    Title:       "Delete Item",
    Message:     "Are you sure you want to delete this item?",
    Dangerous:   true,
    OnConfirm:   handleDelete,
    OnCancel:    handleCancel,
    ConfirmText: "Delete",
    CancelText:  "Keep",
})
```

### Form Components

#### Text Input

```go
components.Input(components.InputProps{
    Type:        "email",
    Name:        "email",
    Label:       "Email Address",
    Placeholder: "user@example.com",
    Required:    true,
    Value:       email,
    OnInput:     handleEmailChange,
    ErrorText:   emailError,
})
```

#### Input Types

- `text` - Standard text input
- `email` - Email input with validation
- `password` - Password input
- `number` - Numeric input
- `tel` - Telephone input
- `url` - URL input
- `search` - Search input

#### Textarea

```go
components.Textarea(components.TextareaProps{
    Name:        "description",
    Label:       "Description",
    Rows:        5,
    MaxLength:   500,
    Value:       description,
    OnInput:     handleDescriptionChange,
    HelperText:  "Maximum 500 characters",
})
```

#### Select Dropdown

```go
components.Select(components.SelectProps{
    Name:        "country",
    Label:       "Country",
    Placeholder: "Select a country",
    Options: []components.SelectOption{
        {Value: "us", Label: "United States"},
        {Value: "uk", Label: "United Kingdom"},
        {Value: "ca", Label: "Canada"},
    },
    Value:    selectedCountry,
    OnChange: handleCountryChange,
})
```

#### Checkbox

```go
components.Checkbox(components.CheckboxProps{
    Name:     "terms",
    Label:    "I agree to the terms and conditions",
    Checked:  agreedToTerms,
    OnChange: handleTermsChange,
})
```

### Loading Components

#### Spinner

```go
components.LoadingSpinner(components.SpinnerProps{
    Size:  "large",
    Color: "#3b82f6",
    Text:  "Loading...",
})
```

#### Skeleton Loader

```go
components.Skeleton(components.SkeletonProps{
    Type: "text",  // "text", "title", "avatar", "image", "button"
    Lines: 3,
})
```

#### Loading Overlay

```go
components.LoadingOverlay(components.LoadingOverlayProps{
    IsVisible: isLoading,
    Text:      "Processing...",
    Blur:      true,
})
```

## Styling Components

### Using CSS Classes

All components accept a `Class` prop for additional styling:

```go
components.Button(components.ButtonProps{
    Text:  "Custom Button",
    Class: "my-custom-class",
})
```

### Tailwind CSS Support

Components work seamlessly with Tailwind CSS:

```go
components.Card(components.CardProps{
    Class: "shadow-xl hover:shadow-2xl transition-shadow",
    // ...
})
```

### Component Styles

Default component styles are included in:

```css
@import "github.com/recera/vango/pkg/components/styles.css";
```

## Creating Custom Components

### Component Pattern

```go
package mycomponents

import (
    "github.com/recera/vango/pkg/vex/builder"
    "github.com/recera/vango/pkg/vango/vdom"
)

type MyComponentProps struct {
    Title   string
    Content *vdom.VNode
    OnClick func()
}

func MyComponent(props MyComponentProps) *vdom.VNode {
    return builder.Div().
        Class("my-component").
        Children(
            builder.H2().Text(props.Title).Build(),
            props.Content,
        ).
        OnClick(props.OnClick).
        Build()
}
```

### Composition Example

```go
func ProfileCard(user User) *vdom.VNode {
    return components.Card(components.CardProps{
        Title:    user.Name,
        Subtitle: user.Role,
        Image:    user.Avatar,
        Content: builder.Div().Children(
            builder.P().Text(user.Bio).Build(),
            components.ButtonGroup(components.ButtonGroupProps{
                Buttons: []components.ButtonProps{
                    {Text: "Message", Variant: components.ButtonPrimary},
                    {Text: "Follow", Variant: components.ButtonSecondary},
                },
            }),
        ).Build(),
    })
}
```

## Best Practices

### 1. Props Validation

Always provide sensible defaults:

```go
func MyComponent(props MyComponentProps) *vdom.VNode {
    // Provide defaults
    if props.Size == "" {
        props.Size = "medium"
    }
    // ...
}
```

### 2. Event Handling

Check for nil handlers:

```go
if props.OnClick != nil {
    element.OnClick(props.OnClick)
}
```

### 3. Accessibility

Include ARIA attributes:

```go
button.
    Attr("aria-label", "Close dialog").
    Attr("role", "button")
```

### 4. Performance

Use memoization for expensive computations:

```go
import "github.com/recera/vango/pkg/reactive"

computedValue := reactive.CreateComputed(func() string {
    // Expensive computation
    return result
})
```

## Component Playground

Try components in the interactive playground:

```bash
vango dev --playground
```

Visit `http://localhost:5173/playground` to experiment with components.

## Contributing

To contribute new components:

1. Create component in `pkg/components/`
2. Add tests in `pkg/components/component_test.go`
3. Update documentation
4. Submit a pull request

## Resources

- [Component API Reference](../api/components.md)
- [Styling Guide](./styling.md)
- [Examples](../../examples/components)
- [Component Tests](../../pkg/components)