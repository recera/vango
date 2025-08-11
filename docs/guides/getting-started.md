# Getting Started with Vango

Welcome to Vango, the Go Frontend Framework! This guide will help you get up and running with your first Vango application.

## Prerequisites

Before you begin, make sure you have the following installed:

- **Go 1.22** or higher
- **TinyGo** (for WebAssembly compilation)
- **Node.js** (optional, for Tailwind CSS)

## Installation

Install the Vango CLI tool:

```bash
go install github.com/recera/vango/cmd/vango@latest
```

Verify the installation:

```bash
vango version
```

## Creating Your First Project

### Interactive Mode (Recommended)

The easiest way to create a new Vango project is using interactive mode:

```bash
vango create my-app
```

This will guide you through:
- Selecting a project template
- Choosing features (Tailwind CSS, routing, etc.)
- Configuring development settings

### Quick Start

For a quick start with defaults:

```bash
vango create my-app --template basic
cd my-app
vango dev
```

Your app will be running at `http://localhost:5173`!

## Project Structure

A typical Vango project has the following structure:

```
my-app/
├── app/
│   ├── routes/          # Page components (file-based routing)
│   ├── components/      # Reusable components
│   ├── layouts/         # Layout wrappers
│   └── styles/          # CSS files
├── public/              # Static assets
├── vango.json           # Configuration file
├── go.mod               # Go dependencies
└── README.md
```

## Core Concepts

### 1. Components

Vango components are Go functions that return virtual DOM nodes:

```go
func HelloWorld() *vdom.VNode {
    return builder.Div().
        Class("greeting").
        Children(
            builder.H1().Text("Hello, World!").Build(),
        ).Build()
}
```

### 2. VEX Templates

For a more familiar syntax, use VEX templates:

```vex
//vango:template
package components

//vango:props { Name string }

<div class="greeting">
  <h1>Hello, {{.Name}}!</h1>
</div>
```

Compile VEX templates:

```bash
vango gen template components/*.vex
```

### 3. Reactive State

Vango provides reactive state management:

```go
import "github.com/recera/vango/pkg/reactive"

func Counter() vango.Component {
    count := reactive.CreateSignal(0)
    
    increment := func() {
        count.Update(func(v int) int { return v + 1 })
    }
    
    return vango.FC(func(ctx *server.Ctx) *vdom.VNode {
        return builder.Div().Children(
            builder.P().Text(fmt.Sprintf("Count: %d", count.Get())).Build(),
            builder.Button().OnClick(increment).Text("Increment").Build(),
        ).Build()
    })
}
```

### 4. File-Based Routing

Routes are automatically generated from your file structure:

- `app/routes/index.go` → `/`
- `app/routes/about.go` → `/about`
- `app/routes/blog/[slug].go` → `/blog/:slug`
- `app/routes/api/users.go` → `/api/users` (JSON endpoint)

### 5. Styling Options

Vango supports multiple styling approaches:

#### CSS Files
```css
/* app/styles/main.css */
.my-class {
    color: blue;
}
```

#### Tailwind CSS
```bash
vango create my-app --features tailwind
```

#### Scoped Styles
```go
styles := vango.Style(`
    .card { 
        padding: 1rem;
        border-radius: 8px;
    }
`)

func Card() *vdom.VNode {
    return builder.Div().Class(styles.Class("card")).Build()
}
```

## Development Workflow

### Hot Reloading

The development server automatically reloads when you make changes:

```bash
vango dev
```

Features:
- ⚡ Instant WASM recompilation
- 🔄 CSS hot module replacement
- 🔌 WebSocket live updates
- 📝 Template auto-compilation

### Building for Production

Create an optimized production build:

```bash
vango build
```

This will:
- Optimize WASM bundle size
- Minify CSS
- Generate static assets
- Create deployment-ready files in `dist/`

## Using the Component Library

Vango comes with a built-in component library:

```go
import "github.com/recera/vango/pkg/components"

func MyPage() *vdom.VNode {
    return components.Card(components.CardProps{
        Title: "Welcome",
        Content: builder.P().Text("Hello from Vango!").Build(),
        Footer: components.Button(components.ButtonProps{
            Text: "Get Started",
            Variant: components.ButtonPrimary,
        }),
    })
}
```

## Next Steps

- [Component Documentation](./components.md)
- [Routing Guide](./routing.md)
- [State Management](./state-management.md)
- [Deployment Guide](./deployment.md)

## Examples

Check out the example projects:

- [Basic Example](../../examples/basic)
- [Counter Example](../../examples/counter)
- [Todo App](../../examples/todo)
- [Blog with Routing](../../examples/blog)
- [Full Stack App](../../examples/fullstack)

## Getting Help

- 📖 [Documentation](https://vango.dev/docs)
- 💬 [Discord Community](https://discord.gg/vango)
- 🐛 [Report Issues](https://github.com/recera/vango/issues)
- ⭐ [Star on GitHub](https://github.com/recera/vango)