# VEX Templates Guide

VEX (Vango Expression) templates provide an HTML-like syntax for building Vango components while maintaining full type safety and Go integration.

## Overview

VEX templates compile to pure Go code at build time, ensuring zero runtime overhead and full type safety.

## Basic Syntax

### Creating a VEX Template

Create a `.vex` file:

```vex
//vango:template
package components

<div class="hello-world">
  <h1>Hello, World!</h1>
  <p>Welcome to Vango with VEX templates!</p>
</div>
```

### Compiling Templates

```bash
# Compile a single template
vango gen template components/hello.vex

# Compile all templates in a directory
vango gen template --dir components

# Watch mode for development
vango gen template --watch --dir components
```

## Props Declaration

### Basic Props

```vex
//vango:template
package components

//vango:props { Name string; Age int }

<div class="user-card">
  <h2>{{.Name}}</h2>
  <p>Age: {{.Age}}</p>
</div>
```

### Complex Props

```vex
//vango:props { 
  User struct {
    ID   int
    Name string
    Email string
  }
  Posts []Post
  OnEdit func(id int)
}

<div class="user-profile">
  <h1>{{.User.Name}}</h1>
  <p>{{.User.Email}}</p>
  <!-- ... -->
</div>
```

## Expressions

### Text Interpolation

```vex
<p>Hello, {{.Name}}!</p>
<span>You have {{.Count}} messages</span>
```

### Go Expressions

```vex
<p>Total: {{.Price * .Quantity}}</p>
<p>Status: {{if .IsActive}}Active{{else}}Inactive{{/if}}</p>
```

### Function Calls

```vex
<p>Formatted date: {{.FormatDate(.CreatedAt)}}</p>
<p>Uppercase name: {{strings.ToUpper(.Name)}}</p>
```

## Control Flow

### If Statements

```vex
{{#if .ShowHeader}}
  <header>
    <h1>{{.Title}}</h1>
  </header>
{{/if}}
```

### If-Else

```vex
{{#if .IsLoggedIn}}
  <p>Welcome back, {{.Username}}!</p>
{{#else}}
  <p>Please log in to continue.</p>
{{/if}}
```

### Else-If Chains

```vex
{{#if .Status == "pending"}}
  <span class="badge-pending">Pending</span>
{{#elseif .Status == "approved"}}
  <span class="badge-success">Approved</span>
{{#elseif .Status == "rejected"}}
  <span class="badge-danger">Rejected</span>
{{#else}}
  <span class="badge-default">Unknown</span>
{{/if}}
```

### For Loops

```vex
<ul>
  {{#for item in .Items}}
    <li>{{item.Name}} - ${{item.Price}}</li>
  {{/for}}
</ul>
```

### For Loop with Index

```vex
<ol>
  {{#for i, post in .Posts}}
    <li>
      <h3>{{i + 1}}. {{post.Title}}</h3>
      <p>{{post.Content}}</p>
    </li>
  {{/for}}
</ol>
```

## Event Handlers

### Click Events

```vex
<button @click="handleClick()">Click me</button>
<button @click="increment()">Count: {{.Count}}</button>
```

### Input Events

```vex
<input 
  type="text" 
  value="{{.SearchTerm}}"
  @input="updateSearch(event.target.value)"
/>
```

### Form Events

```vex
<form @submit="handleSubmit(event)">
  <input name="email" type="email" required />
  <button type="submit">Subscribe</button>
</form>
```

### Other Events

```vex
<div @mouseover="showTooltip()" @mouseout="hideTooltip()">
  Hover over me
</div>

<input @focus="onFocus()" @blur="onBlur()" />

<select @change="updateSelection(event.target.value)">
  <option value="1">Option 1</option>
  <option value="2">Option 2</option>
</select>
```

## Components

### Using Components

```vex
<div class="app">
  <Header title="My App" />
  
  <Card>
    <h2>Card Title</h2>
    <p>Card content goes here</p>
  </Card>
  
  <Footer year="2025" />
</div>
```

### Component Props

```vex
<UserCard 
  name="{{.User.Name}}"
  email="{{.User.Email}}"
  avatar="{{.User.Avatar}}"
  @onEdit="handleEdit({{.User.ID}})"
/>
```

### Component Slots

```vex
<Modal title="Confirmation" @onClose="closeModal()">
  <p>Are you sure you want to proceed?</p>
  <div class="actions">
    <button @click="confirm()">Yes</button>
    <button @click="cancel()">No</button>
  </div>
</Modal>
```

## Attributes

### Static Attributes

```vex
<div class="container" id="main-content">
  <img src="/logo.png" alt="Logo" width="200" height="50" />
</div>
```

### Dynamic Attributes

```vex
<div class="{{.ContainerClass}}" id="{{.ElementID}}">
  <a href="{{.Link}}" target="{{.Target}}">{{.LinkText}}</a>
</div>
```

### Conditional Attributes

```vex
<button 
  class="btn {{if .IsPrimary}}btn-primary{{else}}btn-secondary{{/if}}"
  disabled="{{.IsDisabled}}"
>
  Submit
</button>
```

### Data Attributes

```vex
<div 
  data-user-id="{{.UserID}}"
  data-role="{{.Role}}"
  data-active="{{.IsActive}}"
>
  User info
</div>
```

## Advanced Features

### Computed Values

```vex
//vango:props { 
  Items []Item
  TaxRate float64
}

//vango:computed {
  Subtotal() float64 {
    total := 0.0
    for _, item := range .Items {
      total += item.Price * float64(item.Quantity)
    }
    return total
  }
  
  Tax() float64 {
    return .Subtotal() * .TaxRate
  }
  
  Total() float64 {
    return .Subtotal() + .Tax()
  }
}

<div class="invoice">
  <p>Subtotal: ${{.Subtotal()}}</p>
  <p>Tax: ${{.Tax()}}</p>
  <p>Total: ${{.Total()}}</p>
</div>
```

### Methods

```vex
//vango:methods {
  FormatCurrency(amount float64) string {
    return fmt.Sprintf("$%.2f", amount)
  }
  
  IsExpensive(price float64) bool {
    return price > 100
  }
}

<div class="product">
  <p>Price: {{.FormatCurrency(.Price)}}</p>
  {{#if .IsExpensive(.Price)}}
    <span class="badge-premium">Premium Product</span>
  {{/if}}
</div>
```

## Best Practices

### 1. Keep Templates Simple

Move complex logic to Go code:

```go
// component.go
func (c *Component) GetStatusClass() string {
    switch c.Status {
    case "active":
        return "status-active"
    case "pending":
        return "status-pending"
    default:
        return "status-default"
    }
}
```

```vex
<span class="{{.GetStatusClass()}}">{{.Status}}</span>
```

### 2. Use Semantic HTML

```vex
<article class="blog-post">
  <header>
    <h1>{{.Title}}</h1>
    <time datetime="{{.PublishedAt}}">{{.FormatDate(.PublishedAt)}}</time>
  </header>
  
  <main>
    {{.Content}}
  </main>
  
  <footer>
    <p>By {{.Author}}</p>
  </footer>
</article>
```

### 3. Accessibility

```vex
<button 
  @click="toggleMenu()"
  aria-label="Toggle navigation menu"
  aria-expanded="{{.IsMenuOpen}}"
>
  <span class="hamburger-icon"></span>
</button>

<nav role="navigation" aria-label="Main navigation">
  <!-- Navigation items -->
</nav>
```

### 4. Component Organization

```
components/
├── button/
│   ├── button.vex
│   ├── button.go
│   └── button_test.go
├── card/
│   ├── card.vex
│   ├── card.go
│   └── card_test.go
└── modal/
    ├── modal.vex
    ├── modal.go
    └── modal_test.go
```

## Error Handling

### Template Compilation Errors

VEX provides clear error messages:

```
error: components/card.vex:12:5: undefined prop "Username"
  12 |     <h2>{{.Username}}</h2>
                  ^^^^^^^^^
hint: did you mean "UserName"?
```

### Runtime Type Safety

Generated code maintains full Go type safety:

```go
// Generated from card.vex
func Page(ctx server.Ctx, props PageProps) (*vdom.VNode, error) {
    // Type-safe prop access
    title := props.Title // string
    count := props.Count // int
    // ...
}
```

## Integration with Go

### Importing Go Packages

```vex
//vango:template
package components

//vango:import (
  "strings"
  "fmt"
  "time"
)

<p>{{strings.ToUpper(.Name)}}</p>
<p>{{fmt.Sprintf("Count: %d", .Count)}}</p>
<p>{{time.Now().Format("2006-01-02")}}</p>
```

### Using Go Types

```vex
//vango:props {
  CreatedAt time.Time
  Tags      []string
  Metadata  map[string]interface{}
}

<p>Created: {{.CreatedAt.Format("Jan 2, 2006")}}</p>
<div class="tags">
  {{#for tag in .Tags}}
    <span class="tag">{{tag}}</span>
  {{/for}}
</div>
```

## Performance

### Build-Time Compilation

VEX templates are compiled to Go code at build time:
- Zero runtime parsing overhead
- Full compiler optimizations
- Dead code elimination

### Generated Code Example

Input VEX:
```vex
<div class="card">
  <h2>{{.Title}}</h2>
</div>
```

Generated Go:
```go
func Page(ctx server.Ctx, props PageProps) (*vdom.VNode, error) {
    return builder.Div().Class("card").Children(
        builder.H2().Text(props.Title).Build(),
    ).Build(), nil
}
```

## Debugging

### Source Maps

VEX generates source maps for debugging:

```bash
vango gen template --sourcemaps components/*.vex
```

### Development Mode

Enable template debugging:

```bash
vango dev --debug-templates
```

## Migration Guide

### From HTML

```html
<!-- Before: HTML -->
<div class="user">
  <h2>John Doe</h2>
  <p>john@example.com</p>
</div>
```

```vex
<!-- After: VEX -->
//vango:props { Name string; Email string }

<div class="user">
  <h2>{{.Name}}</h2>
  <p>{{.Email}}</p>
</div>
```

### From React/JSX

```jsx
// Before: React
function User({ name, email, onClick }) {
  return (
    <div className="user" onClick={onClick}>
      <h2>{name}</h2>
      <p>{email}</p>
    </div>
  );
}
```

```vex
// After: VEX
//vango:props { Name string; Email string; OnClick func() }

<div class="user" @click="OnClick()">
  <h2>{{.Name}}</h2>
  <p>{{.Email}}</p>
</div>
```

## Resources

- [VEX Template Specification](../blueprints/template-spec.md)
- [Component Examples](../../examples/components)
- [VEX Playground](https://vango.dev/playground)
- [Video Tutorial](https://vango.dev/tutorials/vex)