# Vango - The Go Frontend Framework - ALPHA RELEASE: EXPERIMENTAL

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![TinyGo Compatible](https://img.shields.io/badge/TinyGo-Compatible-00ADD8?style=for-the-badge)](https://tinygo.org)
[![WebAssembly](https://img.shields.io/badge/WebAssembly-Powered-654FF0?style=for-the-badge&logo=webassembly)](https://webassembly.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

**Build modern web applications entirely in Go. No JavaScript required.**

[Documentation](https://vango.dev/docs) • [Examples](examples/) • [API Reference](https://pkg.go.dev/github.com/recera/vango) • [Discord](https://discord.gg/vango)

</div>

---

## What is Vango?

Vango is a revolutionary web framework that brings the power, simplicity, and type safety of Go to frontend development. By compiling to WebAssembly, Vango enables you to build entire web applications—from backend to frontend—using only Go.

### Key Features

- **100% Go** - Write your entire stack in Go, no JavaScript required
- **Near-Native Performance** - WebAssembly compilation for blazing-fast execution
- **Multiple Syntax Options** - Choose from functional, builder, or template syntax
- **Type-Safe Throughout** - Catch errors at compile time, not runtime
- **Hot Reload** - See changes instantly during development
- **Three Rendering Modes** - Client-side, SSR with hydration, or server-driven
- **File-Based Routing** - Zero-config routing based on file structure
- **Component Library** - Pre-built, accessible UI components
- **Flexible Styling** - Tailwind CSS, scoped styles, or inline styles
- **Progressive Enhancement** - Works without JavaScript, enhances when available

## Quick Start

### Prerequisites

Before getting started, ensure you have:

- **Go 1.22+** ([install](https://go.dev/dl/))
- **TinyGo** for WebAssembly compilation ([install](https://tinygo.org/getting-started/))
- **Node.js** (optional, for Tailwind CSS) ([install](https://nodejs.org/))

### Installation

Install the Vango CLI:

```bash
go install github.com/recera/vango/cmd/vango@latest
```

Verify installation:

```bash
vango version
# Output: Vango v0.1.0-preview
```

### Create Your First App

#### Interactive Mode (Recommended)

The easiest way to start is with the interactive CLI:

```bash
vango create my-app
```

This will launch a beautiful TUI that guides you through:
- Project naming and setup
- Template selection (basic, counter, todo, blog, fullstack)
- Feature configuration
- Development preferences

#### Quick Start Mode

For a quick start with defaults:

```bash
# Create a new project
vango create my-app --template base --no-interactive

# Navigate to project
cd my-app

# Start development server
vango dev
```

Your app is now running at `http://localhost:5173`

### Development Server Flags

```bash
vango dev --help
  -p, --port int        Port to run the dev server on (default 5173)
  -H, --host string     Host to bind the dev server to (default "localhost")
      --cwd string      Working directory of the app (defaults to current)
      --no-tailwind     Disable Tailwind CSS watcher
```

## Project Structure

A typical Vango project looks like this:

```
my-app/
├── app/
│   ├── routes/              # Pages (file-based routing)
│   │   ├── index.go         # Home page (/)
│   │   ├── about.go         # About page (/about)
│   │   ├── blog/
│   │   │   ├── [slug].go    # Dynamic route (/blog/[slug])
│   │   │   └── _layout.go   # (optional) Layout wrapper for blog pages
│   │   ├── _404.go          # 404 error page (optional)
│   │   └── _500.go          # 500 error page (optional)
│   ├── components/          # Reusable components
│   ├── layouts/             # Layout templates (optional)
│   ├── client/              # (optional) WASM entrypoint: app/client/main.go
│   └── main.go              # WASM entrypoint if client/ is missing
├── public/                  # Static assets (served in dev)
│   ├── index.html
│   └── favicon.ico
├── styles/                  # CSS (Tailwind input, etc.)
├── internal/               # Internal packages
├── pkg/                    # Public packages
├── vango.json              # Configuration
├── go.mod                  # Go dependencies
├── tailwind.config.js      # Tailwind config (optional)
└── README.md
```

Notes:
- Dynamic and typed parameters use bracket syntax (`[slug]`, `[id:int]`, `[...rest]`).
- If `app/client/main.go` exists, builds use it. Otherwise `app/main.go` is used.

## Component Syntax Options

Vango offers three progressively enhanced syntax options:

### 1. Functional API (Layer 0)

Pure Go functions for maximum control:

```go
package components

import (
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/vex/functional"
)

func Button(text string, onClick func()) *vdom.VNode {
    return functional.Button(
        functional.MergeProps(
            functional.Class("btn btn-primary"),
            functional.OnClick(onClick),
        ),
        functional.Text(text),
    )
}
```

### 2. Fluent Builder API (Layer 1)

Chainable methods for ergonomic component creation:

```go
package components

import "github.com/recera/vango/pkg/vex/builder"

func Card(title, content string) *vdom.VNode {
    return builder.Div().
        Class("card shadow-lg").
        Children(
            builder.H2().
                Class("card-title").
                Text(title).
                Build(),
            builder.P().
                Class("card-content").
                Text(content).
                Build(),
        ).Build()
}
```

### 3. VEX Templates (Layer 2)

HTML-like templates that compile to Go:

```html
//vango:template
package components

//vango:props { Title string; Items []string }

<div class="container">
    <h1>{{.Title}}</h1>
    
    {{#if len(.Items) > 0}}
        <ul>
            {{#for item in .Items}}
                <li>{{item}}</li>
            {{/for}}
        </ul>
    {{else}}
        <p>No items to display</p>
    {{/if}}
    
    <button @click="addItem()">Add Item</button>
</div>
```

## State Management

Vango provides reactive state management inspired by modern frameworks:

### Signals (Basic Reactive State)

```go
import "github.com/recera/vango/pkg/reactive"

func Counter() *vdom.VNode {
    // Create reactive state
    count := reactive.NewSignal(0)
    
    increment := func() {
        count.Set(count.Get() + 1)
    }
    
    return builder.Div().Children(
        builder.H2().Text(fmt.Sprintf("Count: %d", count.Get())).Build(),
        builder.Button().
            OnClick(increment).
            Text("Increment").
            Build(),
    ).Build()
}
```

### Computed Values

Derive values from other reactive state:

```go
price := reactive.NewSignal(100.0)
quantity := reactive.NewSignal(2)

total := reactive.Computed(func() float64 {
    return price.Get() * float64(quantity.Get())
})
```

### Resources & Suspense

Handle async operations elegantly:

```go
userResource := reactive.NewResource(fetchUser)

func UserProfile() *vdom.VNode {
    return reactive.Suspense(
        userResource,
        func(user User) *vdom.VNode {
            // Render user data
            return UserCard(user)
        },
        LoadingSpinner(),  // Fallback while loading
        ErrorMessage(),    // Error fallback
    )
}
```

## File-Based Routing

Routes are automatically generated from your file structure:

```
app/routes/
├── index.go                 → /
├── about.go                 → /about
├── blog/
│   ├── index.go            → /blog
│   ├── [slug].go           → /blog/:slug
│   └── _layout.go          → Layout wrapper
├── api/
│   └── users.go            → /api/users (JSON endpoint)
├── admin/
│   ├── _middleware.go      → Auth middleware
│   └── dashboard.go        → /admin/dashboard
└── [...catchall].go        → Catch-all route
```

### Dynamic Routes

```go
// app/routes/blog/[slug].go
package blog

import (
    "github.com/recera/vango/pkg/server"
    "github.com/recera/vango/pkg/vango/vdom"
)

// Recommended universal/SSR signature
func Page(ctx server.Ctx) (*vdom.VNode, error) {
    slug := server.Param(ctx, "slug") // or ctx.Param("slug") helper
    post := fetchPost(slug)
    return BlogPost(post), nil
}
```

### API Routes

```go
// app/routes/api/users.go
package api

import "github.com/recera/vango/pkg/server"

// Returning (any, error) is serialized to JSON by the router
func Page(ctx server.Ctx) ([]User, error) {
    users := fetchUsers()
    return users, nil
}
```

### Server-Driven Routes

Server-driven pages keep state on the server and stream DOM patches to the client. Implement handlers under server build tags:

```go
//go:build vango_server && !wasm
// +build vango_server,!wasm

package routes

import (
  "github.com/recera/vango/pkg/server"
  "github.com/recera/vango/pkg/vango/vdom"
)

func CounterPage(ctx server.Ctx) (*vdom.VNode, error) {
  // Return an HTML tree with hydration IDs (data-hid) where needed
  return RenderCounter(ctx), nil
}
```

The dev and prod servers inject a minimal client to wire events and patches automatically.

## Rendering Modes

Vango supports three rendering modes for different use cases:

### 1. Client-Side Rendering (CSR)

Pure client-side app with WASM:

```go
// Default for interactive SPAs
func Page() *vdom.VNode {
    // All state management in WASM
    return App()
}
```

### 2. Server-Side Rendering (SSR) with Hydration

Initial render on server, hydrate on client:

```go
// Perfect for SEO + interactivity
func Page(ctx *vango.Ctx) (*vdom.VNode, error) {
    if ctx.IsServer() {
        // Server-side data fetching
        data := fetchInitialData()
        return ServerApp(data), nil
    }
    // Client takes over after hydration
    return ClientApp(), nil
}
```

### 3. Server-Driven Components

Minimal client, server manages state:

```go
// Ultra-light client (3KB vs 800KB WASM)
func Page(ctx *vango.Ctx) *vdom.VNode {
    // Events sent to server via WebSocket
    // Server sends DOM patches back
    return ServerDrivenApp(ctx)
}
```

## CLI Commands

### Development

```bash
vango dev                   # Start dev server with hot reload
vango dev -p 3000 -H 0.0.0.0
vango dev --no-tailwind     # Disable Tailwind watcher
```

### Code Generation

```bash
vango gen router            # Generate router code from app/routes/**
vango gen template          # Compile VEX templates → Go
vango gen builder           # Generate HTML element builders
```

### Production Build

```bash
# Create production build
vango build

# Options
vango build -o dist --optimize --sourcemap=false
```

What `vango build` does:

- Compiles WASM (TinyGo) to `dist/assets/app.wasm`
- Copies `wasm_exec.js` and production bootstrap to `dist/assets/`
- Copies static files from `public/`
- Runs production routing codegen (registers routes, generates `main_gen.go` and `router/table.json`)
- Builds a server binary at `dist/server` when server-tagged files exist

Run the production server (from project root so it can read `router/table.json`):

```bash
./dist/server -host 0.0.0.0 -port 8080
# Live WS:          /vango/live/
# Client route JSON: /router/table.json
# Assets:           /assets/ (WASM, bootstrap, wasm_exec)
# App routes:       served by the registered router
```

### Project Creation

```bash
# Interactive project creation
vango create my-app

# Use specific template
vango create my-app --template blog

# Non-interactive with defaults
vango create my-app --no-interactive
```

## Styling Options

### 1. Tailwind CSS (Recommended)

Vango automatically detects and configures Tailwind:

```go
func StyledCard() *vdom.VNode {
    return builder.Div().
        Class("bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow").
        Children(/* ... */).
        Build()
}
```

### 2. Scoped Component Styles

```go
styles := vango.Style(`
    .card {
        background: white;
        border-radius: 8px;
        padding: 1rem;
    }
    .card:hover {
        box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    }
`)

func Card() *vdom.VNode {
    return builder.Div().
        Class(styles.Class("card")).
        Children(/* ... */).
        Build()
}
```

### 3. CSS Modules

Place CSS files in `styles/` directory:

```css
/* styles/components.css */
.button {
    @apply px-4 py-2 rounded-lg font-semibold;
}
```

## Building for Production

### Optimization Process

```bash
vango build

# Output structure
dist/
  assets/
    app.wasm
    wasm_exec.js
    bootstrap.js
  index.html
  public/
  server (optional)
```

### Deployment Options

#### Docker Container

```dockerfile
# Multi-stage build
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN vango build --release

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
```

#### Running the Generated Server (SSR / server-driven)

```bash
# From the project root
./dist/server -port 8080
```

It serves:
- `/assets/` (WASM + bootstrap)
- `/router/table.json` (client routing)
- `/vango/live/` (WebSocket for server-driven updates)
- Application routes at `/` via the generated router

## Performance Targets

Targets (indicative):

| Metric | Target |
|--------|--------|
| WASM Bundle Size | <1MB gzipped |
| Route Match | <5µs avg |
| Live Patch Latency | <50ms |

## Configuration

### vango.json

Configure your project with `vango.json`:

```json
{
  "name": "my-app",
  "version": "1.0.0",
  "dev": {
    "port": 5173,
    "host": "localhost",
    "https": false,
    "proxy": {
      "/api": "http://localhost:8080"
    }
  },
  "build": {
    "output": "dist",
    "minify": true,
    "sourceMaps": false
  },
  "styling": {
    "tailwind": {
      "enabled": true,
      "config": "./tailwind.config.js",
      "watch": true
    }
  },
  "routes": {
    "dir": "./app/routes"
  }
}
```

## Testing

### Unit Testing

```go
// component_test.go
func TestButton(t *testing.T) {
    btn := Button("Click me", func(){})
    
    if btn.Tag != "button" {
        t.Errorf("Expected button tag, got %s", btn.Tag)
    }
}
```

### WASM DOM Testing

```go
// Run tests in WASM environment
func TestDOMInteraction(t *testing.T) {
    doc := js.Global().Get("document")
    elem := vnodeToDOM(MyComponent())
    
    // Test DOM manipulation
    doc.Call("body").Call("appendChild", elem)
    // Assert DOM state
}
```

### E2E Testing (Example)

```typescript
// e2e/app.spec.ts
test('counter increments', async ({ page }) => {
  await page.goto('http://localhost:5173')
  await page.click('button:text("Increment")')
  await expect(page.locator('#count')).toHaveText('1')
})
```

## Live Updates (Server-Driven)

Server-driven components use efficient binary protocol:

```go
// Automatic DOM patching via WebSocket
func LiveComponent(ctx *vango.Ctx) *vdom.VNode {
    ctx.OnEvent("click", func(e Event) {
        // Handle on server
        updateState()
        // Client receives patches automatically
    })
    
    return builder.Button().
        OnClick("server:click").
        Text("Server-Handled Click").
        Build()
}
```

> Advanced features such as PWA and code splitting are on the roadmap. See below.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/recera/vango.git
cd vango

# Install dependencies
go mod download

# Run tests
go test ./...

# Build CLI
go build -o vango cmd/vango/main.go
```

## Learning Resources

### Official Documentation
- [Getting Started Guide](https://vango.dev/docs/getting-started)
- [Component Guide](https://vango.dev/docs/components)
- [Routing Guide](https://vango.dev/docs/routing)
- [State Management](https://vango.dev/docs/state)
- [API Reference](https://pkg.go.dev/github.com/recera/vango)

### Example Projects
- [Basic Template](examples/basic/) - Simple starter
- [Counter App](examples/counter/) - State management demo
- [Todo App](examples/todo/) - CRUD operations
- [Blog Template](examples/blog/) - Full-featured blog with Tailwind
- [Showcase](examples/showcase/) - All features demonstration

### Video Tutorials
- [Building Your First Vango App](https://youtube.com/vango-intro)
- [Deep Dive: Reactive State](https://youtube.com/vango-state)
- [Deploying Vango Apps](https://youtube.com/vango-deploy)

## Community

- **Discord**: [Join our Discord](https://discord.gg/vango)
- **Twitter**: [@vango_dev](https://twitter.com/vango_dev)
- **GitHub Discussions**: [Ask questions](https://github.com/recera/vango/discussions)
- **Stack Overflow**: [#vango](https://stackoverflow.com/questions/tagged/vango)

## Roadmap

### Phase 1 (Current) - Developer Experience
- [x] File-based routing
- [x] VEX template syntax
- [x] Fluent builder API
- [x] Hot module reloading
- [x] Tailwind integration
- [x] Interactive CLI

### Phase 2 - Advanced Features
- [ ] Plugin system
- [ ] DevTools extension
- [ ] Form validation library
- [ ] Animation library
- [ ] Testing utilities
- [ ] i18n support

### Phase 3 - Production Ready
- [ ] Performance monitoring
- [ ] Error boundaries
- [ ] SEO utilities
- [ ] Analytics integration
- [ ] CI/CD templates
- [ ] Enterprise features

## Architecture Docs

For a deep dive into the servers and routing pipeline:

- Development server: `docs/architecture/dev-server.md`
- Production server: `docs/architecture/production-server.md`
- Routing & codegen: `docs/architecture/routing-and-codegen.md`
- Client bootstrap & CSR: `docs/architecture/client-bootstrap-and-csr.md`
- Server-driven & Live protocol: `docs/architecture/server-driven-components-and-live.md`
- Build & distribution: `docs/architecture/build-and-distribution.md`

## License

Vango is MIT licensed. See [LICENSE](LICENSE) for details.

## Acknowledgments

Vango stands on the shoulders of giants:

- The Go team for the incredible language and toolchain
- TinyGo team for making Go → WASM possible
- The WebAssembly community for pushing the web forward
- All our contributors and early adopters

---

<div align="center">

**Ready to revolutionize web development with Go?**

[Get Started](https://vango.dev/docs) • [Star on GitHub](https://github.com/recera/vango) • [Join Discord](https://discord.gg/vango)

Made with love by the Vango Team

</div>