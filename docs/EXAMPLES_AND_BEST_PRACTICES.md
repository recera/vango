# Vango Examples and Best Practices

## Table of Contents

1. [Complete Example Applications](#complete-example-applications)
2. [Component Patterns](#component-patterns)
3. [State Management Best Practices](#state-management-best-practices)
4. [Performance Patterns](#performance-patterns)
5. [Error Handling Patterns](#error-handling-patterns)
6. [Testing Patterns](#testing-patterns)
7. [Security Best Practices](#security-best-practices)
8. [Code Organization](#code-organization)
9. [Common Pitfalls](#common-pitfalls)
10. [Production Checklist](#production-checklist)

## Complete Example Applications

### 1. Todo List Application (Universal Mode)

```go
// app/routes/index.go
package routes

import (
    "github.com/recera/vango/pkg/vex/builder"
    "github.com/recera/vango/pkg/reactive"
    "github.com/recera/vango/pkg/server"
    "github.com/recera/vango/pkg/styling"
    "github.com/recera/vango/pkg/vango/vdom"
)

// Define styles
var todoStyles = styling.New(`
    .todo-app {
        max-width: 600px;
        margin: 0 auto;
        padding: 2rem;
        font-family: system-ui, -apple-system, sans-serif;
    }
    
    .todo-input {
        width: 100%;
        padding: 0.75rem;
        font-size: 1rem;
        border: 2px solid #e2e8f0;
        border-radius: 0.5rem;
        margin-bottom: 1rem;
    }
    
    .todo-item {
        display: flex;
        align-items: center;
        padding: 1rem;
        background: white;
        border-radius: 0.5rem;
        margin-bottom: 0.5rem;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    }
    
    .todo-item.completed {
        opacity: 0.6;
    }
    
    .todo-item.completed .todo-text {
        text-decoration: line-through;
    }
    
    .todo-checkbox {
        margin-right: 1rem;
        width: 20px;
        height: 20px;
        cursor: pointer;
    }
    
    .todo-text {
        flex: 1;
        font-size: 1rem;
    }
    
    .todo-delete {
        padding: 0.5rem 1rem;
        background: #ef4444;
        color: white;
        border: none;
        border-radius: 0.25rem;
        cursor: pointer;
    }
    
    .todo-delete:hover {
        background: #dc2626;
    }
    
    .todo-stats {
        display: flex;
        justify-content: space-between;
        padding: 1rem;
        background: #f8fafc;
        border-radius: 0.5rem;
        margin-top: 1rem;
    }
    
    .filter-buttons {
        display: flex;
        gap: 0.5rem;
        margin-bottom: 1rem;
    }
    
    .filter-button {
        padding: 0.5rem 1rem;
        background: #e2e8f0;
        border: none;
        border-radius: 0.25rem;
        cursor: pointer;
    }
    
    .filter-button.active {
        background: #3b82f6;
        color: white;
    }
`)

type Todo struct {
    ID        string
    Text      string
    Completed bool
}

type Filter string

const (
    FilterAll       Filter = "all"
    FilterActive    Filter = "active"
    FilterCompleted Filter = "completed"
)

func TodoApp(ctx server.Ctx) (*vdom.VNode, error) {
    // State
    todos := reactive.CreateState([]Todo{})
    inputValue := reactive.CreateState("")
    filter := reactive.CreateState(FilterAll)
    
    // Persist todos to localStorage
    todos.Persist(reactive.LocalStorage[[]Todo]("todos"))
    
    // Computed values
    filteredTodos := reactive.CreateComputed(func() []Todo {
        all := todos.Get()
        f := filter.Get()
        
        switch f {
        case FilterActive:
            var active []Todo
            for _, todo := range all {
                if !todo.Completed {
                    active = append(active, todo)
                }
            }
            return active
        case FilterCompleted:
            var completed []Todo
            for _, todo := range all {
                if todo.Completed {
                    completed = append(completed, todo)
                }
            }
            return completed
        default:
            return all
        }
    })
    
    activeTodoCount := reactive.CreateComputed(func() int {
        count := 0
        for _, todo := range todos.Get() {
            if !todo.Completed {
                count++
            }
        }
        return count
    })
    
    // Event handlers
    addTodo := func() {
        text := inputValue.Get()
        if text == "" {
            return
        }
        
        newTodo := Todo{
            ID:        generateID(),
            Text:      text,
            Completed: false,
        }
        
        current := todos.Get()
        todos.Set(append(current, newTodo))
        inputValue.Set("")
    }
    
    toggleTodo := func(id string) {
        current := todos.Get()
        for i, todo := range current {
            if todo.ID == id {
                current[i].Completed = !current[i].Completed
                break
            }
        }
        todos.Set(current)
    }
    
    deleteTodo := func(id string) {
        current := todos.Get()
        var updated []Todo
        for _, todo := range current {
            if todo.ID != id {
                updated = append(updated, todo)
            }
        }
        todos.Set(updated)
    }
    
    clearCompleted := func() {
        current := todos.Get()
        var active []Todo
        for _, todo := range current {
            if !todo.Completed {
                active = append(active, todo)
            }
        }
        todos.Set(active)
    }
    
    // Build UI
    return builder.Div().
        Class(todoStyles.Class("todo-app")).
        Children(
            // Header
            builder.H1().Text("Todo List").Build(),
            
            // Input
            builder.Form().
                OnSubmit(func(e Event) {
                    e.PreventDefault()
                    addTodo()
                }).
                Children(
                    builder.Input().
                        Class(todoStyles.Class("todo-input")).
                        Type("text").
                        Placeholder("What needs to be done?").
                        Value(inputValue.Get()).
                        OnInput(func(val string) {
                            inputValue.Set(val)
                        }).
                        Build(),
                ).Build(),
            
            // Filter buttons
            builder.Div().
                Class(todoStyles.Class("filter-buttons")).
                Children(
                    filterButton("All", FilterAll, filter),
                    filterButton("Active", FilterActive, filter),
                    filterButton("Completed", FilterCompleted, filter),
                ).Build(),
            
            // Todo list
            builder.Div().
                Children(
                    renderTodos(filteredTodos.Get(), toggleTodo, deleteTodo)...,
                ).Build(),
            
            // Stats
            builder.Div().
                Class(todoStyles.Class("todo-stats")).
                Children(
                    builder.Span().
                        TextF("%d active items", activeTodoCount.Get()).
                        Build(),
                    builder.Button().
                        Text("Clear completed").
                        OnClick(clearCompleted).
                        Build(),
                ).Build(),
        ).Build(), nil
}

func renderTodos(todos []Todo, toggle, delete func(string)) []*vdom.VNode {
    var nodes []*vdom.VNode
    for _, todo := range todos {
        todoID := todo.ID // Capture for closure
        
        itemClass := todoStyles.Class("todo-item")
        if todo.Completed {
            itemClass += " " + todoStyles.Class("completed")
        }
        
        nodes = append(nodes,
            builder.Div().
                Key(todo.ID).
                Class(itemClass).
                Children(
                    builder.Input().
                        Type("checkbox").
                        Class(todoStyles.Class("todo-checkbox")).
                        Checked(todo.Completed).
                        OnChange(func() { toggle(todoID) }).
                        Build(),
                    builder.Span().
                        Class(todoStyles.Class("todo-text")).
                        Text(todo.Text).
                        Build(),
                    builder.Button().
                        Class(todoStyles.Class("todo-delete")).
                        Text("Delete").
                        OnClick(func() { delete(todoID) }).
                        Build(),
                ).Build(),
        )
    }
    return nodes
}

func filterButton(text string, value Filter, current *reactive.State[Filter]) *vdom.VNode {
    class := todoStyles.Class("filter-button")
    if current.Get() == value {
        class += " " + todoStyles.Class("active")
    }
    
    return builder.Button().
        Class(class).
        Text(text).
        OnClick(func() { current.Set(value) }).
        Build()
}

func generateID() string {
    return fmt.Sprintf("todo_%d", time.Now().UnixNano())
}
```

### 2. Real-Time Chat (Server-Driven Mode)

```go
//vango:server
// app/routes/chat.go
package routes

import (
    "fmt"
    "time"
    "github.com/recera/vango/pkg/vex/builder"
    "github.com/recera/vango/pkg/server"
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/reactive"
)

type Message struct {
    ID        string
    User      string
    Text      string
    Timestamp time.Time
}

// Global state shared across all sessions
var (
    messages = reactive.CreateGlobalSignal([]Message{})
    users    = reactive.CreateGlobalSignal(map[string]bool{})
)

func ChatRoom(ctx server.Ctx) (*vdom.VNode, error) {
    // Get username from session
    username := ctx.Session().Get("username")
    if username == "" {
        return LoginForm(ctx)
    }
    
    // Get component instance for server-driven updates
    component := ctx.Get("component").(*server.ComponentInstance)
    
    // Local state for input
    inputValue := ""
    
    // Add user to active users
    currentUsers := users.Get()
    currentUsers[username] = true
    users.Set(currentUsers)
    
    // Clean up on disconnect
    component.OnCleanup(func() {
        currentUsers := users.Get()
        delete(currentUsers, username)
        users.Set(currentUsers)
    })
    
    // Register message send handler
    component.RegisterHandler(1, func() {
        if inputValue == "" {
            return
        }
        
        newMessage := Message{
            ID:        generateID(),
            User:      username,
            Text:      inputValue,
            Timestamp: time.Now(),
        }
        
        current := messages.Get()
        messages.Set(append(current, newMessage))
        
        inputValue = ""
    })
    
    // Register input handler
    component.RegisterHandler(2, func(value string) {
        inputValue = value
    })
    
    // Build UI
    return builder.Div().
        Class("chat-room").
        Children(
            // Header
            builder.Div().
                Class("chat-header").
                Children(
                    builder.H2().Text("Chat Room").Build(),
                    builder.Div().
                        Class("users-online").
                        TextF("%d users online", len(users.Get())).
                        Build(),
                ).Build(),
            
            // Messages
            builder.Div().
                Class("messages").
                ID("message-list").
                Children(
                    renderMessages(messages.Get(), username)...,
                ).Build(),
            
            // Input
            builder.Form().
                Class("message-input").
                Attr("data-hid", "h1").
                OnSubmit(nil). // Handler registered above
                Children(
                    builder.Input().
                        Type("text").
                        Placeholder("Type a message...").
                        Value(inputValue).
                        Attr("data-hid", "h2").
                        OnInput(nil). // Handler registered above
                        Build(),
                    builder.Button().
                        Type("submit").
                        Text("Send").
                        Build(),
                ).Build(),
        ).Build(), nil
}

func renderMessages(msgs []Message, currentUser string) []*vdom.VNode {
    var nodes []*vdom.VNode
    for _, msg := range msgs {
        class := "message"
        if msg.User == currentUser {
            class += " own-message"
        }
        
        nodes = append(nodes,
            builder.Div().
                Key(msg.ID).
                Class(class).
                Children(
                    builder.Div().
                        Class("message-header").
                        Children(
                            builder.Span().
                                Class("message-user").
                                Text(msg.User).
                                Build(),
                            builder.Span().
                                Class("message-time").
                                Text(msg.Timestamp.Format("15:04")).
                                Build(),
                        ).Build(),
                    builder.Div().
                        Class("message-text").
                        Text(msg.Text).
                        Build(),
                ).Build(),
        )
    }
    return nodes
}
```

### 3. Interactive Dashboard (Client-Only Mode)

```go
//vango:client
// app/components/dashboard.go
package components

import (
    "math"
    "github.com/recera/vango/pkg/vex/builder"
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/reactive"
)

type DataPoint struct {
    Label string
    Value float64
}

func InteractiveDashboard() *vdom.VNode {
    // Local state (runs entirely in WASM)
    data := reactive.CreateState(generateData())
    selectedChart := reactive.CreateState("bar")
    animationFrame := reactive.CreateState(0)
    
    // Animation loop
    reactive.Effect(func() {
        frame := animationFrame.Get()
        requestAnimationFrame(func() {
            animationFrame.Set(frame + 1)
        })
    })
    
    // Chart renderer based on type
    renderChart := func() *vdom.VNode {
        chartType := selectedChart.Get()
        currentData := data.Get()
        
        switch chartType {
        case "bar":
            return BarChart(currentData, animationFrame.Get())
        case "line":
            return LineChart(currentData, animationFrame.Get())
        case "pie":
            return PieChart(currentData, animationFrame.Get())
        default:
            return builder.Div().Text("Unknown chart type").Build()
        }
    }
    
    return builder.Div().
        Class("dashboard").
        Children(
            // Controls
            builder.Div().
                Class("controls").
                Children(
                    builder.Select().
                        Value(selectedChart.Get()).
                        OnChange(func(val string) {
                            selectedChart.Set(val)
                        }).
                        Children(
                            builder.Option().Value("bar").Text("Bar Chart").Build(),
                            builder.Option().Value("line").Text("Line Chart").Build(),
                            builder.Option().Value("pie").Text("Pie Chart").Build(),
                        ).Build(),
                    builder.Button().
                        Text("Refresh Data").
                        OnClick(func() {
                            data.Set(generateData())
                        }).
                        Build(),
                ).Build(),
            
            // Chart
            builder.Div().
                Class("chart-container").
                Child(renderChart()).
                Build(),
            
            // Stats
            builder.Div().
                Class("stats").
                Children(
                    renderStats(data.Get())...,
                ).Build(),
        ).Build()
}

func BarChart(data []DataPoint, frame int) *vdom.VNode {
    maxValue := getMaxValue(data)
    
    var bars []*vdom.VNode
    for i, point := range data {
        height := (point.Value / maxValue) * 300
        // Animate height based on frame
        animatedHeight := height * math.Min(1.0, float64(frame-i*5)/30.0)
        
        bars = append(bars,
            builder.Div().
                Class("bar").
                Style(fmt.Sprintf(
                    "height: %dpx; left: %dpx; background: hsl(%d, 70%%, 50%%)",
                    int(animatedHeight),
                    i*60,
                    i*30,
                )).
                Children(
                    builder.Span().
                        Class("bar-label").
                        Text(point.Label).
                        Build(),
                    builder.Span().
                        Class("bar-value").
                        TextF("%.0f", point.Value).
                        Build(),
                ).Build(),
        )
    }
    
    return builder.Div().
        Class("bar-chart").
        Children(bars...).
        Build()
}
```

## Component Patterns

### 1. Compound Components

```go
// Create related components that work together
type AccordionContext struct {
    activeIndex *reactive.State[int]
}

func Accordion(items []AccordionItem) *vdom.VNode {
    ctx := &AccordionContext{
        activeIndex: reactive.CreateState(-1),
    }
    
    var children []*vdom.VNode
    for i, item := range items {
        children = append(children,
            AccordionItem(ctx, i, item),
        )
    }
    
    return builder.Div().
        Class("accordion").
        Children(children...).
        Build()
}

func AccordionItem(ctx *AccordionContext, index int, item AccordionItem) *vdom.VNode {
    isActive := ctx.activeIndex.Get() == index
    
    return builder.Div().
        Class("accordion-item").
        Children(
            builder.Button().
                Class("accordion-header").
                Text(item.Title).
                OnClick(func() {
                    if isActive {
                        ctx.activeIndex.Set(-1)
                    } else {
                        ctx.activeIndex.Set(index)
                    }
                }).
                Build(),
            builder.Div().
                Class("accordion-content").
                If(isActive, func(b *builder.ElementBuilder) {
                    b.Style("display: block")
                }).
                Unless(isActive, func(b *builder.ElementBuilder) {
                    b.Style("display: none")
                }).
                Children(item.Content...).
                Build(),
        ).Build()
}
```

### 2. Render Props Pattern

```go
// Component that delegates rendering to caller
func DataFetcher[T any](
    url string,
    render func(data T, loading bool, error error) *vdom.VNode,
) *vdom.VNode {
    resource := reactive.CreateResource(func() (T, error) {
        return fetchJSON[T](url)
    })
    
    data, err := resource.Read()
    loading := resource.IsLoading()
    
    return render(data, loading, err)
}

// Usage
func UserProfile(userID string) *vdom.VNode {
    return DataFetcher(
        fmt.Sprintf("/api/users/%s", userID),
        func(user User, loading bool, err error) *vdom.VNode {
            if loading {
                return Spinner()
            }
            if err != nil {
                return ErrorMessage(err)
            }
            return UserCard(user)
        },
    )
}
```

### 3. Higher-Order Components

```go
// Add functionality to existing components
func WithErrorBoundary(component func() *vdom.VNode) func() *vdom.VNode {
    return func() *vdom.VNode {
        defer func() {
            if r := recover(); r != nil {
                // Return error UI
                return builder.Div().
                    Class("error-boundary").
                    Children(
                        builder.H2().Text("Something went wrong").Build(),
                        builder.P().TextF("Error: %v", r).Build(),
                        builder.Button().
                            Text("Reload").
                            OnClick(func() {
                                window.Location.Reload()
                            }).
                            Build(),
                    ).Build()
            }
        }()
        
        return component()
    }
}

// Usage
SafeDashboard := WithErrorBoundary(Dashboard)
```

### 4. Provider Pattern

```go
// Context for sharing state down the tree
type ThemeContext struct {
    theme    *reactive.State[string]
    setTheme func(string)
}

var themeContext *ThemeContext

func ThemeProvider(children ...*vdom.VNode) *vdom.VNode {
    theme := reactive.CreateState("light")
    theme.Persist(reactive.LocalStorage[string]("theme"))
    
    themeContext = &ThemeContext{
        theme:    theme,
        setTheme: theme.Set,
    }
    
    return builder.Div().
        Class(fmt.Sprintf("theme-%s", theme.Get())).
        Children(children...).
        Build()
}

func useTheme() *ThemeContext {
    if themeContext == nil {
        panic("useTheme must be called within ThemeProvider")
    }
    return themeContext
}

// Usage in child component
func ThemedButton(text string) *vdom.VNode {
    theme := useTheme()
    
    return builder.Button().
        Class(fmt.Sprintf("btn--%s", theme.theme.Get())).
        Text(text).
        Build()
}
```

## State Management Best Practices

### 1. State Colocation

```go
// BAD: Global state for everything
var (
    globalUserName = reactive.CreateState("")
    globalUserAge  = reactive.CreateState(0)
    globalUserEmail = reactive.CreateState("")
)

// GOOD: Colocate related state
type UserState struct {
    Name  string
    Age   int
    Email string
}

func UserForm() *vdom.VNode {
    // State is local to where it's used
    user := reactive.CreateState(UserState{})
    
    // ...
}
```

### 2. Derived State

```go
// BAD: Duplicate state
firstName := reactive.CreateState("John")
lastName := reactive.CreateState("Doe")
fullName := reactive.CreateState("John Doe") // Duplicated!

// GOOD: Compute derived values
firstName := reactive.CreateState("John")
lastName := reactive.CreateState("Doe")
fullName := reactive.CreateComputed(func() string {
    return fmt.Sprintf("%s %s", firstName.Get(), lastName.Get())
})
```

### 3. State Normalization

```go
// BAD: Nested, denormalized state
type AppState struct {
    Posts []Post
    // Each post contains full user object
}

// GOOD: Normalized state
type AppState struct {
    Posts map[string]Post   // Posts by ID
    Users map[string]User   // Users by ID
    PostIDs []string        // Ordered post IDs
}

// Helper to get post with user
func getPostWithUser(state AppState, postID string) PostWithUser {
    post := state.Posts[postID]
    user := state.Users[post.UserID]
    return PostWithUser{Post: post, User: user}
}
```

### 4. Action Creators

```go
// Define actions as functions
type TodoActions struct {
    todos *reactive.State[[]Todo]
}

func (a *TodoActions) Add(text string) {
    current := a.todos.Get()
    a.todos.Set(append(current, Todo{
        ID:   generateID(),
        Text: text,
    }))
}

func (a *TodoActions) Remove(id string) {
    current := a.todos.Get()
    filtered := filter(current, func(t Todo) bool {
        return t.ID != id
    })
    a.todos.Set(filtered)
}

func (a *TodoActions) Toggle(id string) {
    current := a.todos.Get()
    for i, todo := range current {
        if todo.ID == id {
            current[i].Completed = !current[i].Completed
            break
        }
    }
    a.todos.Set(current)
}
```

## Performance Patterns

### 1. List Virtualization

```go
func VirtualList(items []Item, containerHeight int) *vdom.VNode {
    itemHeight := 50
    scrollTop := reactive.CreateState(0)
    
    // Calculate visible range
    startIndex := scrollTop.Get() / itemHeight
    endIndex := startIndex + (containerHeight / itemHeight) + 1
    
    // Only render visible items
    var visibleItems []*vdom.VNode
    for i := startIndex; i < endIndex && i < len(items); i++ {
        visibleItems = append(visibleItems,
            builder.Div().
                Key(items[i].ID).
                Style(fmt.Sprintf(
                    "position: absolute; top: %dpx; height: %dpx",
                    i*itemHeight,
                    itemHeight,
                )).
                Text(items[i].Name).
                Build(),
        )
    }
    
    return builder.Div().
        Style(fmt.Sprintf("height: %dpx; overflow-y: auto", containerHeight)).
        OnScroll(func(e ScrollEvent) {
            scrollTop.Set(e.ScrollTop)
        }).
        Children(
            builder.Div().
                Style(fmt.Sprintf(
                    "height: %dpx; position: relative",
                    len(items)*itemHeight,
                )).
                Children(visibleItems...).
                Build(),
        ).Build()
}
```

### 2. Debouncing

```go
func DebouncedSearch() *vdom.VNode {
    searchTerm := reactive.CreateState("")
    results := reactive.CreateState([]SearchResult{})
    
    // Debounced search function
    var debounceTimer *time.Timer
    search := func(term string) {
        if debounceTimer != nil {
            debounceTimer.Stop()
        }
        
        debounceTimer = time.AfterFunc(300*time.Millisecond, func() {
            if term == "" {
                results.Set([]SearchResult{})
                return
            }
            
            // Perform search
            searchResults := performSearch(term)
            results.Set(searchResults)
        })
    }
    
    return builder.Div().
        Children(
            builder.Input().
                Type("text").
                Placeholder("Search...").
                Value(searchTerm.Get()).
                OnInput(func(val string) {
                    searchTerm.Set(val)
                    search(val)
                }).
                Build(),
            builder.Div().
                Children(
                    renderResults(results.Get())...,
                ).Build(),
        ).Build()
}
```

### 3. Lazy Loading

```go
func LazyImage(src string, placeholder string) *vdom.VNode {
    loaded := reactive.CreateState(false)
    inView := reactive.CreateState(false)
    
    // Use IntersectionObserver for viewport detection
    reactive.Effect(func() {
        if !inView.Get() {
            return
        }
        
        // Load image when in viewport
        img := new(Image)
        img.OnLoad = func() {
            loaded.Set(true)
        }
        img.Src = src
    })
    
    currentSrc := placeholder
    if loaded.Get() {
        currentSrc = src
    }
    
    return builder.Img().
        Src(currentSrc).
        Class(If(loaded.Get(), "loaded", "loading")).
        Ref(func(el Element) {
            observeElement(el, func(isInView bool) {
                inView.Set(isInView)
            })
        }).
        Build()
}
```

## Error Handling Patterns

### 1. Error Boundaries

```go
type ErrorBoundary struct {
    fallback func(error) *vdom.VNode
    onError  func(error)
}

func (e *ErrorBoundary) Wrap(component func() *vdom.VNode) *vdom.VNode {
    defer func() {
        if r := recover(); r != nil {
            err := fmt.Errorf("component error: %v", r)
            
            if e.onError != nil {
                e.onError(err)
            }
            
            if e.fallback != nil {
                return e.fallback(err)
            }
            
            // Default error UI
            return builder.Div().
                Class("error-boundary").
                Text("An error occurred").
                Build()
        }
    }()
    
    return component()
}
```

### 2. Async Error Handling

```go
func AsyncComponent() *vdom.VNode {
    data := reactive.CreateResource(fetchData)
    
    // Handle different states
    if data.IsLoading() {
        return LoadingSpinner()
    }
    
    if err := data.Error(); err != nil {
        return ErrorDisplay(err)
    }
    
    value, _ := data.Read()
    return DataDisplay(value)
}
```

### 3. Form Validation

```go
type FormErrors map[string]string

func ValidatedForm() *vdom.VNode {
    email := reactive.CreateState("")
    password := reactive.CreateState("")
    errors := reactive.CreateState(FormErrors{})
    
    validate := func() bool {
        errs := FormErrors{}
        
        if !isValidEmail(email.Get()) {
            errs["email"] = "Invalid email address"
        }
        
        if len(password.Get()) < 8 {
            errs["password"] = "Password must be at least 8 characters"
        }
        
        errors.Set(errs)
        return len(errs) == 0
    }
    
    handleSubmit := func() {
        if !validate() {
            return
        }
        
        // Submit form
        submitForm(email.Get(), password.Get())
    }
    
    return builder.Form().
        OnSubmit(func(e Event) {
            e.PreventDefault()
            handleSubmit()
        }).
        Children(
            // Email field
            builder.Div().
                Class("form-field").
                Children(
                    builder.Input().
                        Type("email").
                        Value(email.Get()).
                        OnInput(func(val string) {
                            email.Set(val)
                            // Clear error on input
                            errs := errors.Get()
                            delete(errs, "email")
                            errors.Set(errs)
                        }).
                        Build(),
                    renderError(errors.Get()["email"]),
                ).Build(),
            
            // Password field
            builder.Div().
                Class("form-field").
                Children(
                    builder.Input().
                        Type("password").
                        Value(password.Get()).
                        OnInput(func(val string) {
                            password.Set(val)
                            errs := errors.Get()
                            delete(errs, "password")
                            errors.Set(errs)
                        }).
                        Build(),
                    renderError(errors.Get()["password"]),
                ).Build(),
            
            builder.Button().
                Type("submit").
                Text("Submit").
                Build(),
        ).Build()
}

func renderError(err string) *vdom.VNode {
    if err == "" {
        return builder.Fragment().Build()
    }
    
    return builder.Span().
        Class("error-message").
        Text(err).
        Build()
}
```

## Testing Patterns

### 1. Component Testing

```go
func TestCounter(t *testing.T) {
    // Render component
    vnode := Counter(CounterProps{Initial: 5})
    
    // Check initial render
    assert.NotNil(t, vnode)
    assert.Equal(t, "div", vnode.Tag)
    
    // Find counter display
    display := vnode.Find(func(n *vdom.VNode) bool {
        return n.Props["id"] == "count"
    })
    assert.Equal(t, "5", display.Text)
    
    // Simulate click
    button := vnode.Find(func(n *vdom.VNode) bool {
        _, ok := n.Props["onclick"]
        return ok && n.Text == "Increment"
    })
    
    clickHandler := button.Props["onclick"].(func())
    clickHandler()
    
    // Re-render and check
    vnode = Counter(CounterProps{Initial: 5})
    display = vnode.Find(func(n *vdom.VNode) bool {
        return n.Props["id"] == "count"
    })
    assert.Equal(t, "6", display.Text)
}
```

### 2. State Testing

```go
func TestReactiveState(t *testing.T) {
    // Create state
    count := reactive.CreateState(0)
    
    // Track changes
    var changes []int
    count.Subscribe(func(val int) {
        changes = append(changes, val)
    })
    
    // Make changes
    count.Set(1)
    count.Set(2)
    count.Set(3)
    
    // Verify
    assert.Equal(t, []int{1, 2, 3}, changes)
    assert.Equal(t, 3, count.Get())
}
```

### 3. Integration Testing

```go
func TestTodoAppIntegration(t *testing.T) {
    // Create test context
    ctx := server.NewTestContext()
    
    // Render app
    vnode, err := TodoApp(ctx)
    assert.NoError(t, err)
    
    // Add a todo
    input := findByPlaceholder(vnode, "What needs to be done?")
    form := findParentForm(input)
    
    // Simulate input
    inputHandler := input.Props["oninput"].(func(string))
    inputHandler("Test todo")
    
    // Simulate submit
    submitHandler := form.Props["onsubmit"].(func(Event))
    submitHandler(Event{})
    
    // Re-render
    vnode, _ = TodoApp(ctx)
    
    // Check todo was added
    todos := findByClass(vnode, "todo-item")
    assert.Len(t, todos, 1)
    assert.Contains(t, todos[0].Text, "Test todo")
}
```

## Security Best Practices

### 1. Input Sanitization

```go
func sanitizeHTML(input string) string {
    // Use a proper HTML sanitizer library
    p := bluemonday.UGCPolicy()
    return p.Sanitize(input)
}

func UserContent(html string) *vdom.VNode {
    // Never trust user input
    safe := sanitizeHTML(html)
    
    return builder.Div().
        Class("user-content").
        // Use safe HTML rendering (if implemented)
        InnerHTML(safe).
        Build()
}
```

### 2. CSRF Protection

```go
func ProtectedForm(ctx server.Ctx) *vdom.VNode {
    // Get CSRF token from session
    token := ctx.Session().Get("csrf_token")
    if token == "" {
        token = generateCSRFToken()
        ctx.Session().Set("csrf_token", token)
    }
    
    return builder.Form().
        Method("POST").
        Children(
            // Include token in form
            builder.Input().
                Type("hidden").
                Name("csrf_token").
                Value(token).
                Build(),
            // Form fields...
        ).Build()
}
```

### 3. Content Security Policy

```go
func SecureHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set security headers
        w.Header().Set("Content-Security-Policy", 
            "default-src 'self'; "+
            "script-src 'self' 'wasm-unsafe-eval'; "+
            "style-src 'self' 'unsafe-inline'; "+
            "img-src 'self' data: https:; "+
            "connect-src 'self' wss:")
        
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
```

## Code Organization

### 1. Project Structure

```
my-vango-app/
├── app/
│   ├── routes/           # Page components
│   │   ├── index.go      # Home page
│   │   ├── about.go      # About page
│   │   └── blog/         # Blog section
│   │       ├── index.go
│   │       └── [slug].go
│   ├── components/       # Reusable components
│   │   ├── header.go
│   │   ├── footer.go
│   │   └── button.go
│   ├── layouts/          # Layout components
│   │   └── default.go
│   └── styles/           # Style definitions
│       └── theme.go
├── pkg/                  # Business logic
│   ├── models/           # Data models
│   ├── services/         # Service layer
│   └── utils/            # Utilities
├── public/               # Static assets
│   ├── images/
│   └── fonts/
├── config/               # Configuration
│   └── config.go
├── go.mod
├── go.sum
└── vango.json            # Vango configuration
```

### 2. Component Organization

```go
// components/button/button.go
package button

import (
    "github.com/recera/vango/pkg/vex/builder"
    "github.com/recera/vango/pkg/styling"
    "github.com/recera/vango/pkg/vango/vdom"
)

// Styles
var buttonStyles = styling.New(`
    .button { /* styles */ }
    .button--primary { /* styles */ }
    .button--secondary { /* styles */ }
`)

// Types
type ButtonVariant string

const (
    Primary   ButtonVariant = "primary"
    Secondary ButtonVariant = "secondary"
)

type ButtonProps struct {
    Text     string
    Variant  ButtonVariant
    OnClick  func()
    Disabled bool
}

// Component
func Button(props ButtonProps) *vdom.VNode {
    class := buttonStyles.Class("button")
    if props.Variant != "" {
        class += " " + buttonStyles.Class("button--"+string(props.Variant))
    }
    
    return builder.Button().
        Class(class).
        Text(props.Text).
        OnClick(props.OnClick).
        Disabled(props.Disabled).
        Build()
}
```

## Common Pitfalls

### 1. Closure Issues

```go
// BAD: Captures loop variable
for i, item := range items {
    button.OnClick(func() {
        handleClick(i) // Always uses last value of i!
    })
}

// GOOD: Capture in local variable
for i, item := range items {
    index := i // Capture
    button.OnClick(func() {
        handleClick(index)
    })
}
```

### 2. Memory Leaks

```go
// BAD: Subscription without cleanup
func LeakyComponent() *vdom.VNode {
    state.Subscribe(func(val int) {
        // This keeps component alive!
    })
    // ...
}

// GOOD: Clean up subscriptions
func CleanComponent() *vdom.VNode {
    unsubscribe := state.Subscribe(func(val int) {
        // ...
    })
    
    // Register cleanup
    onCleanup(unsubscribe)
    // ...
}
```

### 3. Unnecessary Re-renders

```go
// BAD: Creates new object every render
func BadComponent() *vdom.VNode {
    return builder.Div().
        Style(map[string]string{ // New object!
            "color": "red",
        }).
        Build()
}

// GOOD: Stable references
var divStyle = map[string]string{
    "color": "red",
}

func GoodComponent() *vdom.VNode {
    return builder.Div().
        Style(divStyle). // Same object
        Build()
}
```

## Production Checklist

### Before Deployment

- [ ] **Performance**
  - [ ] Enable production build flags
  - [ ] Minimize WASM size
  - [ ] Enable compression (gzip/brotli)
  - [ ] Implement caching strategy
  - [ ] Optimize images

- [ ] **Security**
  - [ ] Set security headers
  - [ ] Enable HTTPS
  - [ ] Implement rate limiting
  - [ ] Validate all inputs
  - [ ] Set up CORS properly

- [ ] **Error Handling**
  - [ ] Add error boundaries
  - [ ] Set up error logging
  - [ ] Create 404/500 pages
  - [ ] Test error scenarios

- [ ] **Monitoring**
  - [ ] Set up logging
  - [ ] Add performance monitoring
  - [ ] Configure alerts
  - [ ] Track user analytics

- [ ] **Testing**
  - [ ] Run all tests
  - [ ] Perform load testing
  - [ ] Test on target browsers
  - [ ] Check accessibility

- [ ] **Documentation**
  - [ ] Update README
  - [ ] Document API endpoints
  - [ ] Create deployment guide
  - [ ] Document environment variables

### Build Configuration

```json
// vango.json
{
  "build": {
    "mode": "production",
    "minify": true,
    "sourceMaps": false,
    "optimization": "z",
    "compression": "gzip"
  },
  "server": {
    "port": 8080,
    "host": "0.0.0.0",
    "https": {
      "enabled": true,
      "cert": "/path/to/cert.pem",
      "key": "/path/to/key.pem"
    }
  },
  "security": {
    "csp": "strict",
    "cors": {
      "origins": ["https://example.com"]
    }
  }
}
```

### Deployment Script

```bash
#!/bin/bash
# deploy.sh

# Build for production
vango build --release

# Run tests
go test ./...

# Check bundle size
MAX_SIZE=800000
WASM_SIZE=$(stat -f%z dist/app.wasm)
if [ $WASM_SIZE -gt $MAX_SIZE ]; then
    echo "WASM too large: $WASM_SIZE bytes"
    exit 1
fi

# Deploy
rsync -avz dist/ server:/var/www/app/

# Restart service
ssh server "systemctl restart vango-app"

echo "Deployment complete!"
```

This completes the comprehensive examples and best practices guide for the Vango framework.