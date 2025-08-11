package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register("blog", &BlogTemplate{})
}

// BlogTemplate generates a fully-featured blog with markdown support
type BlogTemplate struct{}

func (t *BlogTemplate) Name() string {
	return "blog"
}

func (t *BlogTemplate) Description() string {
	return "Full-featured blog with markdown support and automatic post discovery"
}

func (t *BlogTemplate) Generate(config *ProjectConfig) error {
	// Create blog-specific directories
	blogDirs := []string{
		"app/routes/blog",
		"app/lib",
		"app/lib/markdown",
		"app/components",
		"content",
		"content/posts",
		"public/images",
		"public/images/blog",
	}
	
	for _, dir := range blogDirs {
		if err := os.MkdirAll(filepath.Join(config.Directory, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	// Create main.go for blog
	if err := t.createMainFile(config); err != nil {
		return err
	}
	
	// Create markdown loader
	if err := t.createMarkdownLoader(config); err != nil {
		return err
	}
	
	// Create blog types
	if err := t.createBlogTypes(config); err != nil {
		return err
	}
	
	// Create blog index page (home)
	if err := t.createBlogIndex(config); err != nil {
		return err
	}
	
	// Create blog post page
	if err := t.createBlogPostPage(config); err != nil {
		return err
	}
	
	// Create blog components
	if err := t.createBlogComponents(config); err != nil {
		return err
	}
	
	// Create comprehensive sample posts
	if err := t.createComprehensiveSamplePosts(config); err != nil {
		return err
	}
	
	// Add Tailwind config with typography plugin
	if err := t.createEnhancedTailwindConfig(config); err != nil {
		return err
	}
	
	// Create enhanced styles
	if err := t.createEnhancedStyles(config); err != nil {
		return err
	}
	
	return nil
}

func (t *BlogTemplate) createMainFile(config *ProjectConfig) error {
	content := fmt.Sprintf(`package main

import (
	"strings"
	"syscall/js"
	
	// Import our packages
	"%s/app/lib/markdown"
	routes "%s/app/routes"
	"github.com/recera/vango/pkg/vango/vdom"
)

func main() {
	js.Global().Get("console").Call("log", "🚀 Vango Blog starting...")
	
	// Load blog posts from markdown files
	if err := markdown.LoadPosts(); err != nil {
		js.Global().Get("console").Call("error", "Failed to load posts:", err.Error())
	}
	
	// Initialize app
	initApp()
	
	// Keep the WASM runtime alive
	select {}
}

func initApp() {
	document := js.Global().Get("document")
	
	// Wait for DOM ready
	if document.Get("readyState").String() != "loading" {
		onReady()
	} else {
		document.Call("addEventListener", "DOMContentLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			onReady()
			return nil
		}))
	}
}

func onReady() {
	console := js.Global().Get("console")
	console.Call("log", "DOM ready, initializing Blog...")
	
	// Initialize dark mode from localStorage or system preference
	initDarkMode()
	
	// Get current route
	path := js.Global().Get("window").Get("location").Get("pathname").String()
	
	// Get sorted posts
	posts := markdown.GetSortedPosts()
	
	// Simple routing
	var vnode *vdom.VNode
	if path == "/" || path == "/blog" || path == "" {
		vnode = routes.BlogIndex(posts)
	} else if strings.HasPrefix(path, "/blog/") {
		slug := strings.TrimPrefix(path, "/blog/")
		slug = strings.TrimSuffix(slug, "/")
		
		// Find the post
		var foundPost *markdown.BlogPost
		for _, post := range posts {
			if post.Slug == slug {
				foundPost = &post
				break
			}
		}
		
		if foundPost != nil {
			vnode = routes.BlogPostPage(*foundPost)
		} else {
			vnode = routes.NotFound()
		}
	} else if path == "/about" {
		vnode = routes.AboutPage()
	} else if path == "/archive" {
		vnode = routes.ArchivePage(posts)
	} else {
		vnode = routes.BlogIndex(posts)
	}
	
	renderVNode(vnode)
	
	// Set up dark mode toggle handler
	setupDarkModeToggle()
	
	// Set up client-side navigation
	setupClientNavigation()
}

func initDarkMode() {
	document := js.Global().Get("document")
	localStorage := js.Global().Get("localStorage")
	
	// Check localStorage first
	darkMode := localStorage.Call("getItem", "darkMode").String()
	
	if darkMode == "true" {
		document.Get("documentElement").Get("classList").Call("add", "dark")
	} else if darkMode == "false" {
		document.Get("documentElement").Get("classList").Call("remove", "dark")
	} else {
		// Check system preference
		if js.Global().Get("window").Call("matchMedia", "(prefers-color-scheme: dark)").Get("matches").Bool() {
			document.Get("documentElement").Get("classList").Call("add", "dark")
		}
	}
}

func setupDarkModeToggle() {
	js.Global().Set("toggleDarkMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		document := js.Global().Get("document")
		localStorage := js.Global().Get("localStorage")
		html := document.Get("documentElement")
		classList := html.Get("classList")
		
		if classList.Call("contains", "dark").Bool() {
			classList.Call("remove", "dark")
			localStorage.Call("setItem", "darkMode", "false")
		} else {
			classList.Call("add", "dark")
			localStorage.Call("setItem", "darkMode", "true")
		}
		
		return nil
	}))
}

func setupClientNavigation() {
	// Handle link clicks for client-side navigation
	js.Global().Set("navigateTo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			path := args[0].String()
			js.Global().Get("window").Get("history").Call("pushState", nil, "", path)
			onReady() // Re-render with new route
		}
		return nil
	}))
	
	// Handle browser back/forward buttons
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		onReady() // Re-render with new route
		return nil
	}))
}

func renderVNode(vnode *vdom.VNode) {
	console := js.Global().Get("console")
	document := js.Global().Get("document")
	
	console.Call("log", "Rendering Blog VNode...")
	
	// Get the app root
	appRoot := document.Call("getElementById", "app")
	if appRoot.IsNull() || appRoot.IsUndefined() {
		console.Call("error", "Could not find #app element")
		return
	}
	
	// Clear existing content
	appRoot.Set("innerHTML", "")
	
	// Render the actual VNode to DOM
	domNode := vnodeToDOM(vnode)
	if !domNode.IsNull() && !domNode.IsUndefined() {
		appRoot.Call("appendChild", domNode)
		console.Call("log", "✅ Blog rendered successfully!")
		
		// Scroll to top on navigation
		js.Global().Get("window").Call("scrollTo", 0, 0)
	}
}

func vnodeToDOM(vnode *vdom.VNode) js.Value {
	document := js.Global().Get("document")
	
	if vnode == nil {
		return js.Null()
	}
	
	switch vnode.Kind {
	case vdom.KindText:
		return document.Call("createTextNode", vnode.Text)
		
	case vdom.KindElement:
		elem := document.Call("createElement", vnode.Tag)
		
		// Set properties
		if vnode.Props != nil {
			for key, value := range vnode.Props {
				switch key {
				case "class", "className":
					if v, ok := value.(string); ok {
						elem.Set("className", v)
					}
				case "id":
					if v, ok := value.(string); ok {
						elem.Set("id", v)
					}
				case "href":
					if v, ok := value.(string); ok {
						elem.Set("href", v)
					}
				case "src":
					if v, ok := value.(string); ok {
						elem.Set("src", v)
					}
				case "alt":
					if v, ok := value.(string); ok {
						elem.Set("alt", v)
					}
				case "onclick":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "onclick", v)
					}
				case "style":
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", "style", v)
					}
				case "target":
					if v, ok := value.(string); ok {
						elem.Set("target", v)
					}
				case "rel":
					if v, ok := value.(string); ok {
						elem.Set("rel", v)
					}
				case "loading":
					if v, ok := value.(string); ok {
						elem.Set("loading", v)
					}
				default:
					if v, ok := value.(string); ok {
						elem.Call("setAttribute", key, v)
					}
				}
			}
		}
		
		// Render children
		for _, child := range vnode.Kids {
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				elem.Call("appendChild", childNode)
			}
		}
		
		return elem
		
	case vdom.KindFragment:
		fragment := document.Call("createDocumentFragment")
		for _, child := range vnode.Kids {
			childNode := vnodeToDOM(&child)
			if !childNode.IsNull() && !childNode.IsUndefined() {
				fragment.Call("appendChild", childNode)
			}
		}
		return fragment
		
	default:
		return js.Null()
	}
}`, config.Module, config.Module)
	
	return WriteFile(filepath.Join(config.Directory, "app/main.go"), content)
}

func (t *BlogTemplate) createMarkdownLoader(config *ProjectConfig) error {
	content := `package markdown

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BlogPost represents a single blog post with metadata
type BlogPost struct {
	Slug        string
	Title       string
	Date        time.Time
	Author      string
	AuthorImage string
	Tags        []string
	Excerpt     string
	Content     string // HTML content after markdown processing
	RawContent  string // Raw markdown content
	HeroImage   string
	ReadingTime int
	Published   bool
}

var posts []BlogPost

// LoadPosts loads all markdown posts from content/posts directory
func LoadPosts() error {
	// In a real implementation, this would scan the content/posts directory
	// For now, we'll create posts programmatically
	
	// Since we're in WASM, we can't read files directly
	// In production, you'd have a build step that generates this data
	posts = generateSamplePosts()
	return nil
}

// GetSortedPosts returns all posts sorted by date (newest first)
func GetSortedPosts() []BlogPost {
	sorted := make([]BlogPost, len(posts))
	copy(sorted, posts)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.After(sorted[j].Date)
	})
	
	// Filter out unpublished posts
	var published []BlogPost
	for _, post := range sorted {
		if post.Published {
			published = append(published, post)
		}
	}
	
	return published
}

// GetPostBySlug returns a post by its slug
func GetPostBySlug(slug string) (*BlogPost, error) {
	for _, post := range posts {
		if post.Slug == slug && post.Published {
			return &post, nil
		}
	}
	return nil, fmt.Errorf("post not found: %s", slug)
}

// GetPostsByTag returns all posts with a specific tag
func GetPostsByTag(tag string) []BlogPost {
	var tagged []BlogPost
	for _, post := range GetSortedPosts() {
		for _, t := range post.Tags {
			if strings.ToLower(t) == strings.ToLower(tag) {
				tagged = append(tagged, post)
				break
			}
		}
	}
	return tagged
}

// GetRelatedPosts returns posts related to the given post
func GetRelatedPosts(post BlogPost, limit int) []BlogPost {
	var related []BlogPost
	sorted := GetSortedPosts()
	
	// Find posts with matching tags
	for _, p := range sorted {
		if p.Slug == post.Slug {
			continue
		}
		
		// Check for matching tags
		for _, tag := range post.Tags {
			for _, t := range p.Tags {
				if tag == t {
					related = append(related, p)
					goto next
				}
			}
		}
		next:
		
		if len(related) >= limit {
			break
		}
	}
	
	return related
}

// CalculateReadingTime estimates reading time based on word count
func CalculateReadingTime(content string) int {
	words := strings.Fields(content)
	// Average reading speed: 200-250 words per minute
	minutes := len(words) / 225
	if minutes < 1 {
		return 1
	}
	return minutes
}

// generateSamplePosts creates sample blog posts
func generateSamplePosts() []BlogPost {
	return []BlogPost{
		{
			Slug:        "getting-started-with-vango",
			Title:       "Getting Started with Vango: Build Modern Web Apps in Go",
			Date:        time.Date(2024, 12, 15, 10, 0, 0, 0, time.UTC),
			Author:      "Sarah Chen",
			AuthorImage: "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=150&h=150&fit=crop",
			Tags:        []string{"tutorial", "vango", "golang", "getting-started"},
			Excerpt:     "Learn how to build modern, reactive web applications using Vango - the Go-native frontend framework that compiles to WebAssembly. No JavaScript required!",
			HeroImage:   "https://images.unsplash.com/photo-1461749280684-dccba630e2f6?w=1200&h=600&fit=crop",
			ReadingTime: 8,
			Published:   true,
			Content: ` + "`" + `
<div class="prose prose-lg dark:prose-invert max-w-none">
<p class="lead">Welcome to Vango! If you're a Go developer looking to build modern web applications without leaving the comfort of Go, you're in the right place. Vango is a revolutionary framework that brings the power of Go to frontend development through WebAssembly.</p>

<img src="https://images.unsplash.com/photo-1461749280684-dccba630e2f6?w=800&h=400&fit=crop" alt="Code on screen" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Why Vango?</h2>

<p>Traditional web development requires juggling multiple languages, build tools, and ecosystems. With Vango, you can:</p>

<ul>
<li><strong>Write Everything in Go</strong> - Use a single language for your entire application</li>
<li><strong>Type Safety</strong> - Catch errors at compile time, not runtime</li>
<li><strong>Native Performance</strong> - WebAssembly provides near-native speed</li>
<li><strong>No Build Complexity</strong> - Say goodbye to webpack, babel, and npm</li>
<li><strong>Familiar Patterns</strong> - Use Go idioms and patterns you already know</li>
</ul>

<h2>Setting Up Your First Vango Project</h2>

<p>Getting started with Vango is incredibly simple. First, make sure you have Go 1.22+ installed, then:</p>

<pre><code class="language-bash"># Install Vango CLI
go install github.com/recera/vango/cmd/vango@latest

# Create a new blog project
vango create my-blog --template blog

# Navigate to your project
cd my-blog

# Start the development server
vango dev</code></pre>

<p>That's it! Your blog is now running at <code>http://localhost:5173</code> with hot-reloading enabled.</p>

<h2>Understanding the Project Structure</h2>

<p>When you create a new Vango blog, you'll see this structure:</p>

<pre><code>my-blog/
├── app/
│   ├── main.go           # Application entry point
│   ├── routes/           # Page components
│   ├── components/       # Reusable UI components
│   └── lib/             # Utilities and helpers
├── content/
│   └── posts/           # Your markdown blog posts
├── public/              # Static assets
├── styles/              # CSS files
└── vango.json          # Configuration</code></pre>

<h2>Creating Your First Blog Post</h2>

<p>Creating a new blog post is as simple as adding a markdown file to the <code>content/posts/</code> directory:</p>

<pre><code class="language-markdown">---
title: "My First Post"
date: 2024-12-20
author: "Your Name"
tags: ["tutorial", "vango"]
excerpt: "This is my first blog post using Vango!"
hero_image: "https://example.com/image.jpg"
published: true
---

# Welcome to My Blog!

This is my first post using Vango's blog template...
</code></pre>

<h2>The Power of Go Components</h2>

<p>Unlike traditional frameworks, Vango components are just Go functions that return VNodes:</p>

<pre><code class="language-go">func BlogCard(post BlogPost) *vdom.VNode {
    return functional.Article(
        functional.Class("blog-card"),
        functional.H2(nil, functional.Text(post.Title)),
        functional.P(nil, functional.Text(post.Excerpt)),
        functional.A(
            functional.Href("/blog/" + post.Slug),
            functional.Text("Read more →"),
        ),
    )
}</code></pre>

<img src="https://images.unsplash.com/photo-1516116216624-53e697fedbea?w=800&h=400&fit=crop" alt="Programming setup" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Reactive State Management</h2>

<p>Vango includes a powerful reactive state system inspired by modern frontend frameworks:</p>

<pre><code class="language-go">// Create reactive state
count := reactive.Signal(0)

// Update state
count.Set(count.Get() + 1)

// Components automatically re-render when state changes
func Counter() *vdom.VNode {
    return functional.Div(nil,
        functional.Text(fmt.Sprintf("Count: %d", count.Get())),
        functional.Button(
            functional.OnClick(func() { count.Set(count.Get() + 1) }),
            functional.Text("Increment"),
        ),
    )
}</code></pre>

<h2>Styling Your Blog</h2>

<p>This blog template comes with Tailwind CSS pre-configured. You can use utility classes or write custom CSS:</p>

<pre><code class="language-go">functional.Div(
    functional.Class("bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6"),
    // Your content here
)</code></pre>

<h2>Dark Mode Support</h2>

<p>The blog template includes automatic dark mode support that respects system preferences and allows manual toggling. Try clicking the moon icon in the navigation bar!</p>

<h2>Next Steps</h2>

<p>Now that you have your blog running, here are some things to explore:</p>

<ol>
<li><strong>Customize the theme</strong> - Modify <code>tailwind.config.js</code> to match your brand</li>
<li><strong>Add new pages</strong> - Create new routes in <code>app/routes/</code></li>
<li><strong>Enhance SEO</strong> - Add meta tags and structured data</li>
<li><strong>Deploy your blog</strong> - Use <code>vango build</code> to create a production build</li>
</ol>

<div class="bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-blue-900 dark:text-blue-200 mb-2">💡 Pro Tip</p>
<p class="text-blue-800 dark:text-blue-300">Vango's development server includes hot-reloading for both Go code and CSS. Make changes and see them instantly without refreshing!</p>
</div>

<h2>Join the Community</h2>

<p>Ready to dive deeper? Join our growing community:</p>

<ul>
<li>📚 <a href="https://vango.dev/docs" class="text-blue-600 dark:text-blue-400 hover:underline">Documentation</a></li>
<li>💬 <a href="https://discord.gg/vango" class="text-blue-600 dark:text-blue-400 hover:underline">Discord Community</a></li>
<li>⭐ <a href="https://github.com/recera/vango" class="text-blue-600 dark:text-blue-400 hover:underline">GitHub Repository</a></li>
</ul>

<p>Happy coding with Vango! 🚀</p>
</div>` + "`" + `,
		},
		{
			Slug:        "customizing-your-vango-blog",
			Title:       "Customizing Your Vango Blog: Themes, Layouts, and Components",
			Date:        time.Date(2024, 12, 18, 14, 30, 0, 0, time.UTC),
			Author:      "Marcus Johnson",
			AuthorImage: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=150&h=150&fit=crop",
			Tags:        []string{"tutorial", "customization", "components", "design"},
			Excerpt:     "Take your Vango blog to the next level by learning how to customize themes, create reusable components, and build stunning layouts with Tailwind CSS.",
			HeroImage:   "https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?w=1200&h=600&fit=crop",
			ReadingTime: 10,
			Published:   true,
			Content: ` + "`" + `
<div class="prose prose-lg dark:prose-invert max-w-none">
<p class="lead">Your Vango blog is up and running - fantastic! Now let's make it truly yours. In this guide, we'll explore how to customize every aspect of your blog, from colors and typography to creating custom components and layouts.</p>

<img src="https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?w=800&h=400&fit=crop" alt="Design workspace" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Understanding the Theme System</h2>

<p>Vango blogs use Tailwind CSS for styling, which provides incredible flexibility. Your theme configuration lives in <code>tailwind.config.js</code>:</p>

<pre><code class="language-javascript">module.exports = {
  content: ["./app/**/*.go", "./content/**/*.md"],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#eff6ff',
          500: '#3b82f6',
          900: '#1e3a8a',
        }
      },
      fontFamily: {
        'serif': ['Merriweather', 'serif'],
        'sans': ['Inter', 'sans-serif'],
      }
    }
  }
}</code></pre>

<h2>Creating Custom Components</h2>

<p>Components in Vango are composable Go functions. Let's create a custom author bio component:</p>

<pre><code class="language-go">// app/components/author_bio.go
package components

import (
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/vex/functional"
)

func AuthorBio(name, bio, image string) *vdom.VNode {
    return functional.Div(
        functional.Class("flex items-center space-x-4 p-6 bg-gray-50 dark:bg-gray-800 rounded-xl"),
        
        // Author image
        functional.Img(functional.MergeProps(
            functional.Src(image),
            functional.Alt(name),
            functional.Class("w-20 h-20 rounded-full object-cover"),
        )),
        
        // Author info
        functional.Div(nil,
            functional.H3(
                functional.Class("font-bold text-lg dark:text-white"),
                functional.Text(name),
            ),
            functional.P(
                functional.Class("text-gray-600 dark:text-gray-300"),
                functional.Text(bio),
            ),
        ),
    )
}</code></pre>

<h2>Building Custom Layouts</h2>

<p>Layouts wrap your content and provide consistent structure. Here's how to create a custom layout:</p>

<pre><code class="language-go">// app/layouts/blog_layout.go
func BlogLayout(content *vdom.VNode) *vdom.VNode {
    return functional.Div(nil,
        Header(),        // Navigation
        HeroSection(),   // Hero banner
        functional.Main(
            functional.Class("container mx-auto px-4 py-12"),
            content,     // Page content
        ),
        Newsletter(),    // Email signup
        Footer(),        // Site footer
    )
}</code></pre>

<img src="https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=800&h=400&fit=crop" alt="Code editor" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Advanced Styling Techniques</h2>

<h3>1. Custom CSS with Scoped Styles</h3>

<p>While Tailwind handles most styling needs, you can add custom CSS when needed:</p>

<pre><code class="language-css">/* styles/custom.css */
.blog-hero {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
}

.blog-hero::before {
  content: '';
  position: absolute;
  width: 200%;
  height: 200%;
  background: url('pattern.svg') repeat;
  opacity: 0.1;
  animation: float 20s infinite linear;
}

@keyframes float {
  from { transform: translate(-50%, -50%) rotate(0deg); }
  to { transform: translate(-50%, -50%) rotate(360deg); }
}</code></pre>

<h3>2. Dynamic Themes</h3>

<p>Implement theme switching beyond just dark mode:</p>

<pre><code class="language-go">type Theme struct {
    Name      string
    Primary   string
    Secondary string
    Accent    string
}

var themes = map[string]Theme{
    "ocean": {
        Name:      "Ocean",
        Primary:   "#006994",
        Secondary: "#00a8cc",
        Accent:    "#00d2ff",
    },
    "forest": {
        Name:      "Forest",
        Primary:   "#2d5016",
        Secondary: "#3e7c17",
        Accent:    "#75b740",
    },
}

func ApplyTheme(themeName string) {
    theme := themes[themeName]
    // Apply CSS variables dynamically
}</code></pre>

<h2>Creating Interactive Elements</h2>

<p>Add interactivity with reactive components:</p>

<pre><code class="language-go">func InteractiveTabs(tabs []Tab) *vdom.VNode {
    activeTab := reactive.Signal(0)
    
    return functional.Div(nil,
        // Tab headers
        functional.Div(
            functional.Class("flex border-b"),
            ...tabs.map(func(tab Tab, index int) *vdom.VNode {
                return functional.Button(
                    functional.MergeProps(
                        functional.Class(getTabClass(index == activeTab.Get())),
                        functional.OnClick(func() { activeTab.Set(index) }),
                    ),
                    functional.Text(tab.Title),
                )
            }),
        ),
        
        // Tab content
        functional.Div(
            functional.Class("p-4"),
            tabs[activeTab.Get()].Content,
        ),
    )
}</code></pre>

<div class="bg-green-50 dark:bg-green-900/20 border-l-4 border-green-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-green-900 dark:text-green-200 mb-2">✨ Component Library</p>
<p class="text-green-800 dark:text-green-300">Check out the Vango component library for pre-built components like modals, tooltips, and more at <a href="https://vango.dev/components">vango.dev/components</a></p>
</div>

<h2>Responsive Design Best Practices</h2>

<p>Ensure your blog looks great on all devices:</p>

<pre><code class="language-go">functional.Div(
    functional.Class(
        "grid gap-6 " +
        "grid-cols-1 " +           // Mobile: 1 column
        "sm:grid-cols-2 " +         // Tablet: 2 columns
        "lg:grid-cols-3 " +         // Desktop: 3 columns
        "xl:grid-cols-4",           // Wide: 4 columns
    ),
    // Grid items
)</code></pre>

<h2>Performance Optimization</h2>

<p>Keep your blog fast with these techniques:</p>

<ol>
<li><strong>Lazy Loading Images</strong> - Use the <code>loading="lazy"</code> attribute</li>
<li><strong>Code Splitting</strong> - Vango automatically splits your WASM modules</li>
<li><strong>Caching</strong> - Configure proper cache headers in production</li>
<li><strong>Minification</strong> - Use <code>vango build --optimize</code> for production</li>
</ol>

<h2>Adding Custom Functionality</h2>

<h3>Search Feature</h3>

<p>Add search functionality to your blog:</p>

<pre><code class="language-go">func SearchPosts(query string, posts []BlogPost) []BlogPost {
    query = strings.ToLower(query)
    var results []BlogPost
    
    for _, post := range posts {
        if strings.Contains(strings.ToLower(post.Title), query) ||
           strings.Contains(strings.ToLower(post.Content), query) ||
           containsTag(post.Tags, query) {
            results = append(results, post)
        }
    }
    
    return results
}</code></pre>

<h3>Related Posts</h3>

<p>Show related content to keep readers engaged:</p>

<pre><code class="language-go">func GetRelatedPosts(current BlogPost, all []BlogPost) []BlogPost {
    var related []BlogPost
    
    for _, post := range all {
        if post.Slug == current.Slug {
            continue
        }
        
        similarity := calculateSimilarity(current.Tags, post.Tags)
        if similarity > 0.3 {
            related = append(related, post)
        }
        
        if len(related) >= 3 {
            break
        }
    }
    
    return related
}</code></pre>

<img src="https://images.unsplash.com/photo-1522542550221-31fd19575a2d?w=800&h=400&fit=crop" alt="Design elements" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Deployment Considerations</h2>

<p>When you're ready to deploy your customized blog:</p>

<pre><code class="language-bash"># Build for production
vango build --optimize

# The output will be in the dist/ directory
ls -la dist/
# app.wasm (your compiled Go code)
# index.html
# styles.css
# public/ (static assets)</code></pre>

<h2>What's Next?</h2>

<p>With these customization techniques, you can create a unique blog that stands out. Here are some ideas to explore:</p>

<ul>
<li>📊 Add analytics with privacy-focused solutions</li>
<li>💬 Implement a comment system</li>
<li>📧 Create an email newsletter integration</li>
<li>🔍 Add full-text search with highlighting</li>
<li>🌍 Implement internationalization (i18n)</li>
</ul>

<p>Remember, the beauty of Vango is that everything is just Go code. You have the full power of the language at your fingertips!</p>

<div class="bg-purple-50 dark:bg-purple-900/20 border-l-4 border-purple-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-purple-900 dark:text-purple-200 mb-2">🎨 Design Resources</p>
<p class="text-purple-800 dark:text-purple-300">Find free images at <a href="https://unsplash.com">Unsplash</a>, icons at <a href="https://heroicons.com">Heroicons</a>, and color palettes at <a href="https://coolors.co">Coolors</a>.</p>
</div>

<p>Happy customizing! Share your creations with the community on Discord. We'd love to see what you build! 🎨</p>
</div>` + "`" + `,
		},
		{
			Slug:        "deploying-vango-blog",
			Title:       "Deploy Your Vango Blog: From Local to Production",
			Date:        time.Date(2024, 12, 20, 9, 15, 0, 0, time.UTC),
			Author:      "Elena Rodriguez",
			AuthorImage: "https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=150&h=150&fit=crop",
			Tags:        []string{"deployment", "devops", "production", "hosting"},
			Excerpt:     "Ready to share your blog with the world? Learn how to deploy your Vango blog to various platforms including Vercel, Netlify, and traditional VPS hosting.",
			HeroImage:   "https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop",
			ReadingTime: 12,
			Published:   true,
			Content: ` + "`" + `
<div class="prose prose-lg dark:prose-invert max-w-none">
<p class="lead">You've built an amazing blog with Vango, and now it's time to share it with the world. This comprehensive guide will walk you through deploying your Vango blog to various hosting platforms, from serverless to traditional hosting.</p>

<img src="https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&h=400&fit=crop" alt="Cloud infrastructure" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Building for Production</h2>

<p>Before deploying, you need to create an optimized production build:</p>

<pre><code class="language-bash"># Create production build
vango build --optimize

# Output structure
dist/
├── index.html          # Entry point
├── app.wasm           # Compiled Go application (~2-5MB)
├── wasm_exec.js       # Go WASM runtime
├── styles.css         # Compiled CSS
└── public/            # Static assets</code></pre>

<h2>Deployment Options</h2>

<h3>Option 1: Vercel (Recommended for Simplicity)</h3>

<p>Vercel offers excellent support for static sites with WASM:</p>

<pre><code class="language-bash"># Install Vercel CLI
npm i -g vercel

# Deploy
cd my-blog
vango build --optimize
cd dist
vercel

# Follow the prompts to complete deployment</code></pre>

<p>Create a <code>vercel.json</code> for custom configuration:</p>

<pre><code class="language-json">{
  "buildCommand": "vango build --optimize",
  "outputDirectory": "dist",
  "headers": [
    {
      "source": "/(.*).wasm",
      "headers": [
        {
          "key": "Content-Type",
          "value": "application/wasm"
        }
      ]
    }
  ]
}</code></pre>

<h3>Option 2: Netlify</h3>

<p>Netlify is another excellent choice for static hosting:</p>

<pre><code class="language-toml"># netlify.toml
[build]
  command = "vango build --optimize"
  publish = "dist"

[[headers]]
  for = "/*.wasm"
  [headers.values]
    Content-Type = "application/wasm"
    
[[headers]]
  for = "/*"
  [headers.values]
    X-Frame-Options = "SAMEORIGIN"
    X-Content-Type-Options = "nosniff"
    X-XSS-Protection = "1; mode=block"</code></pre>

<img src="https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=800&h=400&fit=crop" alt="Server room" class="rounded-lg shadow-xl my-8" loading="lazy">

<h3>Option 3: GitHub Pages</h3>

<p>Deploy directly from your GitHub repository:</p>

<pre><code class="language-yaml"># .github/workflows/deploy.yml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: '1.22'
          
      - name: Install Vango
        run: go install github.com/recera/vango/cmd/vango@latest
        
      - name: Build
        run: vango build --optimize
        
      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist</code></pre>

<h3>Option 4: Traditional VPS (nginx)</h3>

<p>For more control, deploy to your own server:</p>

<pre><code class="language-nginx"># /etc/nginx/sites-available/vango-blog
server {
    listen 80;
    listen [::]:80;
    server_name yourblog.com;
    
    root /var/www/vango-blog;
    index index.html;
    
    # WASM mime type
    location ~ \.wasm$ {
        add_header Content-Type application/wasm;
    }
    
    # Compression
    gzip on;
    gzip_types text/plain text/css application/javascript application/wasm;
    
    # Caching
    location ~* \.(wasm|js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }
}</code></pre>

<h2>Performance Optimization</h2>

<h3>1. CDN Configuration</h3>

<p>Use a CDN for global distribution:</p>

<pre><code class="language-javascript">// Cloudflare Workers example
addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
  const url = new URL(request.url)
  
  // Cache WASM files aggressively
  if (url.pathname.endsWith('.wasm')) {
    const response = await fetch(request)
    const headers = new Headers(response.headers)
    headers.set('Cache-Control', 'public, max-age=31536000')
    headers.set('Content-Type', 'application/wasm')
    
    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: headers
    })
  }
  
  return fetch(request)
}</code></pre>

<h3>2. Preloading Critical Resources</h3>

<p>Add preload hints to your HTML:</p>

<pre><code class="language-html">&lt;!-- index.html --&gt;
&lt;link rel="preload" href="/app.wasm" as="fetch" crossorigin&gt;
&lt;link rel="preload" href="/wasm_exec.js" as="script"&gt;
&lt;link rel="preload" href="/styles.css" as="style"&gt;</code></pre>

<div class="bg-yellow-50 dark:bg-yellow-900/20 border-l-4 border-yellow-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-yellow-900 dark:text-yellow-200 mb-2">⚡ Performance Tip</p>
<p class="text-yellow-800 dark:text-yellow-300">WASM files can be large. Enable Brotli compression on your server to reduce file size by up to 30%!</p>
</div>

<h2>Environment Configuration</h2>

<p>Manage different environments with configuration:</p>

<pre><code class="language-go">// config/config.go
package config

import "os"

type Config struct {
    BaseURL    string
    APIEndpoint string
    Analytics  string
    IsProd     bool
}

func Load() *Config {
    env := os.Getenv("VANGO_ENV")
    
    if env == "production" {
        return &Config{
            BaseURL:    "https://yourblog.com",
            APIEndpoint: "https://api.yourblog.com",
            Analytics:  "UA-XXXXXXXX-X",
            IsProd:     true,
        }
    }
    
    return &Config{
        BaseURL:    "http://localhost:5173",
        APIEndpoint: "http://localhost:8080",
        Analytics:  "",
        IsProd:     false,
    }
}</code></pre>

<h2>Monitoring and Analytics</h2>

<h3>1. Error Tracking</h3>

<p>Implement error tracking for production:</p>

<pre><code class="language-go">func trackError(err error) {
    if config.IsProd {
        // Send to error tracking service
        js.Global().Get("console").Call("error", err.Error())
        
        // Send to analytics
        js.Global().Get("gtag").Call("event", "exception", map[string]interface{}{
            "description": err.Error(),
            "fatal": false,
        })
    }
}</code></pre>

<h3>2. Performance Monitoring</h3>

<p>Track Core Web Vitals:</p>

<pre><code class="language-javascript">// Add to index.html
import {getCLS, getFID, getLCP} from 'web-vitals';

function sendToAnalytics(metric) {
  // Send to your analytics endpoint
  const body = JSON.stringify(metric);
  
  if (navigator.sendBeacon) {
    navigator.sendBeacon('/analytics', body);
  }
}

getCLS(sendToAnalytics);
getFID(sendToAnalytics);
getLCP(sendToAnalytics);</code></pre>

<img src="https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800&h=400&fit=crop" alt="Analytics dashboard" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Security Best Practices</h2>

<h3>1. Content Security Policy</h3>

<p>Add CSP headers for security:</p>

<pre><code class="language-html">&lt;meta http-equiv="Content-Security-Policy" 
      content="default-src 'self'; 
               script-src 'self' 'wasm-unsafe-eval'; 
               style-src 'self' 'unsafe-inline';
               img-src 'self' data: https:;
               connect-src 'self' https://api.yourblog.com;"&gt;</code></pre>

<h3>2. HTTPS Configuration</h3>

<p>Always use HTTPS in production. With Let's Encrypt:</p>

<pre><code class="language-bash"># Install Certbot
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d yourblog.com -d www.yourblog.com

# Auto-renewal
sudo certbot renew --dry-run</code></pre>

<h2>Continuous Deployment</h2>

<p>Set up automated deployments with GitHub Actions:</p>

<pre><code class="language-yaml">name: Deploy Blog

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.22
    
    - name: Install dependencies
      run: |
        go install github.com/recera/vango/cmd/vango@latest
        npm ci
        
    - name: Build
      run: |
        vango build --optimize
        
    - name: Run tests
      run: |
        go test ./...
        
    - name: Deploy to production
      if: github.ref == 'refs/heads/main'
      run: |
        # Your deployment script here
        echo "Deploying to production..."</code></pre>

<h2>Backup and Recovery</h2>

<p>Don't forget to backup your content:</p>

<pre><code class="language-bash">#!/bin/bash
# backup.sh

# Backup content and configuration
tar -czf backup-$(date +%Y%m%d).tar.gz \
  content/ \
  vango.json \
  tailwind.config.js

# Upload to S3 (example)
aws s3 cp backup-*.tar.gz s3://your-backup-bucket/

# Keep only last 30 days of backups
find . -name "backup-*.tar.gz" -mtime +30 -delete</code></pre>

<div class="bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-blue-900 dark:text-blue-200 mb-2">🚀 Launch Checklist</p>
<ul class="text-blue-800 dark:text-blue-300 space-y-2">
<li>✅ Production build tested locally</li>
<li>✅ HTTPS configured</li>
<li>✅ Analytics installed</li>
<li>✅ Error tracking setup</li>
<li>✅ Backup strategy in place</li>
<li>✅ SEO meta tags added</li>
<li>✅ Performance optimized</li>
<li>✅ Security headers configured</li>
</ul>
</div>

<h2>Post-Deployment</h2>

<p>After deploying, monitor these metrics:</p>

<ul>
<li><strong>Performance:</strong> Page load time, Time to Interactive</li>
<li><strong>SEO:</strong> Google PageSpeed Insights score</li>
<li><strong>Uptime:</strong> Use services like UptimeRobot</li>
<li><strong>Traffic:</strong> Google Analytics or privacy-focused alternatives</li>
<li><strong>Errors:</strong> Browser console errors, 404s</li>
</ul>

<h2>Conclusion</h2>

<p>Congratulations! Your Vango blog is now live. Remember to:</p>

<ol>
<li>Regularly update dependencies for security</li>
<li>Monitor performance and optimize as needed</li>
<li>Backup your content regularly</li>
<li>Engage with your readers through comments and social media</li>
</ol>

<p>The Vango community is here to help if you run into issues. Share your deployed blog in our Discord channel - we love seeing what you build!</p>

<p>Happy blogging! 🎉</p>
</div>` + "`" + `,
		},
		{
			Slug:        "advanced-vango-patterns",
			Title:       "Advanced Vango Patterns: State Management and Performance",
			Date:        time.Date(2024, 12, 22, 16, 45, 0, 0, time.UTC),
			Author:      "Alex Kim",
			AuthorImage: "https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=150&h=150&fit=crop",
			Tags:        []string{"advanced", "performance", "state-management", "patterns"},
			Excerpt:     "Master advanced Vango patterns including global state management, performance optimization, code splitting, and reactive programming techniques.",
			HeroImage:   "https://images.unsplash.com/photo-1515879218367-8466d910aaa4?w=1200&h=600&fit=crop",
			ReadingTime: 15,
			Published:   true,
			Content: ` + "`" + `
<div class="prose prose-lg dark:prose-invert max-w-none">
<p class="lead">Ready to level up your Vango skills? This guide explores advanced patterns and techniques that will help you build performant, scalable applications with sophisticated state management and optimal user experiences.</p>

<img src="https://images.unsplash.com/photo-1515879218367-8466d910aaa4?w=800&h=400&fit=crop" alt="Advanced programming" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Global State Management</h2>

<p>For complex applications, you need robust state management. Let's build a global store pattern:</p>

<pre><code class="language-go">// app/lib/store/store.go
package store

import (
    "sync"
    "github.com/recera/vango/pkg/reactive"
)

type AppState struct {
    User        *User
    Posts       []BlogPost
    Theme       string
    IsLoading   bool
}

type Store struct {
    state    reactive.Signal[*AppState]
    mu       sync.RWMutex
    reducers map[string]Reducer
}

type Reducer func(state *AppState, action Action) *AppState

type Action struct {
    Type    string
    Payload interface{}
}

var globalStore *Store

func init() {
    globalStore = &Store{
        state: reactive.Signal(&AppState{
            Theme: "light",
            Posts: []BlogPost{},
        }),
        reducers: make(map[string]Reducer),
    }
    
    // Register reducers
    globalStore.Register("SET_USER", setUserReducer)
    globalStore.Register("SET_POSTS", setPostsReducer)
    globalStore.Register("SET_THEME", setThemeReducer)
}

func Dispatch(action Action) {
    globalStore.mu.Lock()
    defer globalStore.mu.Unlock()
    
    if reducer, ok := globalStore.reducers[action.Type]; ok {
        newState := reducer(globalStore.state.Get(), action)
        globalStore.state.Set(newState)
    }
}

func Subscribe(callback func(*AppState)) func() {
    return globalStore.state.Subscribe(callback)
}</code></pre>

<h2>Performance Optimization Techniques</h2>

<h3>1. Virtual List for Large Data Sets</h3>

<p>Render thousands of items efficiently with virtualization:</p>

<pre><code class="language-go">func VirtualList(items []Item, itemHeight int) *vdom.VNode {
    scrollTop := reactive.Signal(0)
    containerHeight := reactive.Signal(600)
    
    // Calculate visible range
    startIndex := scrollTop.Get() / itemHeight
    endIndex := (scrollTop.Get() + containerHeight.Get()) / itemHeight
    
    // Only render visible items
    visibleItems := items[startIndex:min(endIndex+1, len(items))]
    
    return functional.Div(
        functional.MergeProps(
            functional.Class("virtual-list-container"),
            functional.StyleAttr(fmt.Sprintf("height: %dpx; overflow-y: auto", containerHeight.Get())),
            functional.OnScroll(func(e Event) {
                scrollTop.Set(e.Target.ScrollTop)
            }),
        ),
        
        // Spacer for scroll height
        functional.Div(
            functional.StyleAttr(fmt.Sprintf("height: %dpx", len(items)*itemHeight)),
        ),
        
        // Visible items
        functional.Div(
            functional.StyleAttr(fmt.Sprintf("transform: translateY(%dpx)", startIndex*itemHeight)),
            ...renderVisibleItems(visibleItems),
        ),
    )
}</code></pre>

<h3>2. Memoization for Expensive Computations</h3>

<pre><code class="language-go">// Memoize expensive functions
func Memoize[T any, R any](fn func(T) R) func(T) R {
    cache := make(map[T]R)
    mu := &sync.RWMutex{}
    
    return func(input T) R {
        mu.RLock()
        if result, ok := cache[input]; ok {
            mu.RUnlock()
            return result
        }
        mu.RUnlock()
        
        mu.Lock()
        defer mu.Unlock()
        
        result := fn(input)
        cache[input] = result
        return result
    }
}

// Usage
var processMarkdown = Memoize(func(content string) string {
    // Expensive markdown processing
    return markdown.ToHTML(content)
})</code></pre>

<img src="https://images.unsplash.com/photo-1504639725590-34d0984388bd?w=800&h=400&fit=crop" alt="Code architecture" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Advanced Reactive Patterns</h2>

<h3>1. Computed Properties with Dependencies</h3>

<pre><code class="language-go">type ComputedValue[T any] struct {
    compute      func() T
    value        T
    dependencies []reactive.Signal[any]
    isDirty      bool
}

func Computed[T any](compute func() T, deps ...reactive.Signal[any]) *ComputedValue[T] {
    c := &ComputedValue[T]{
        compute:      compute,
        dependencies: deps,
        isDirty:      true,
    }
    
    // Subscribe to dependencies
    for _, dep := range deps {
        dep.Subscribe(func(_ any) {
            c.isDirty = true
        })
    }
    
    return c
}

func (c *ComputedValue[T]) Get() T {
    if c.isDirty {
        c.value = c.compute()
        c.isDirty = false
    }
    return c.value
}</code></pre>

<h3>2. Async State Management</h3>

<pre><code class="language-go">type AsyncState[T any] struct {
    data    reactive.Signal[*T]
    loading reactive.Signal[bool]
    error   reactive.Signal[error]
}

func UseAsync[T any](fetcher func() (*T, error)) *AsyncState[T] {
    state := &AsyncState[T]{
        data:    reactive.Signal[*T](nil),
        loading: reactive.Signal(true),
        error:   reactive.Signal[error](nil),
    }
    
    go func() {
        data, err := fetcher()
        state.loading.Set(false)
        
        if err != nil {
            state.error.Set(err)
        } else {
            state.data.Set(data)
        }
    }()
    
    return state
}

// Usage in component
func BlogPosts() *vdom.VNode {
    posts := UseAsync(func() (*[]BlogPost, error) {
        return fetchPosts()
    })
    
    if posts.loading.Get() {
        return LoadingSpinner()
    }
    
    if err := posts.error.Get(); err != nil {
        return ErrorMessage(err)
    }
    
    return PostList(*posts.data.Get())
}</code></pre>

<h2>Code Splitting and Lazy Loading</h2>

<p>Split your application into chunks for faster initial load:</p>

<pre><code class="language-go">// Route-based code splitting
type LazyRoute struct {
    path      string
    loader    func() (*vdom.VNode, error)
    component *vdom.VNode
    loaded    bool
}

func LazyLoad(path string, loader func() (*vdom.VNode, error)) *LazyRoute {
    return &LazyRoute{
        path:   path,
        loader: loader,
    }
}

func (r *LazyRoute) Render() *vdom.VNode {
    if !r.loaded {
        go func() {
            component, err := r.loader()
            if err == nil {
                r.component = component
                r.loaded = true
                // Trigger re-render
                ForceUpdate()
            }
        }()
        
        return LoadingSpinner()
    }
    
    return r.component
}</code></pre>

<div class="bg-orange-50 dark:bg-orange-900/20 border-l-4 border-orange-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-orange-900 dark:text-orange-200 mb-2">🔥 Performance Tip</p>
<p class="text-orange-800 dark:text-orange-300">Use the browser's Performance API to measure and optimize critical rendering paths. Aim for Time to Interactive (TTI) under 3 seconds on 3G connections.</p>
</div>

<h2>WebAssembly Optimization</h2>

<h3>1. Minimize WASM Size</h3>

<pre><code class="language-bash"># Build flags for smaller WASM
vango build \
  --optimize \
  --no-debug \
  --gc-sections \
  --compress

# Further optimization with wasm-opt
wasm-opt -Oz -o app.min.wasm app.wasm</code></pre>

<h3>2. Efficient Memory Management</h3>

<pre><code class="language-go">// Pool objects to reduce GC pressure
var nodePool = sync.Pool{
    New: func() interface{} {
        return &vdom.VNode{}
    },
}

func GetNode() *vdom.VNode {
    return nodePool.Get().(*vdom.VNode)
}

func PutNode(node *vdom.VNode) {
    // Reset node
    node.Kind = 0
    node.Tag = ""
    node.Props = nil
    node.Kids = nil
    node.Text = ""
    
    nodePool.Put(node)
}</code></pre>

<h2>Testing Strategies</h2>

<h3>1. Component Testing</h3>

<pre><code class="language-go">func TestBlogCard(t *testing.T) {
    post := BlogPost{
        Title: "Test Post",
        Excerpt: "Test excerpt",
        Slug: "test-post",
    }
    
    component := BlogCard(post)
    
    // Test structure
    assert.Equal(t, "article", component.Tag)
    assert.Contains(t, component.Props["class"], "blog-card")
    
    // Test content
    title := findByText(component, "Test Post")
    assert.NotNil(t, title)
    
    // Test interactions
    link := findByTag(component, "a")
    assert.Equal(t, "/blog/test-post", link.Props["href"])
}</code></pre>

<h3>2. Integration Testing</h3>

<pre><code class="language-go">func TestBlogNavigation(t *testing.T) {
    // Setup test environment
    app := setupTestApp()
    
    // Navigate to blog
    app.NavigateTo("/blog")
    
    // Verify posts are displayed
    posts := app.FindAll(".blog-card")
    assert.Greater(t, len(posts), 0)
    
    // Click on first post
    app.Click(posts[0].Find("a"))
    
    // Verify navigation
    assert.Equal(t, "/blog/test-post", app.CurrentPath())
    assert.NotNil(t, app.Find(".blog-post"))
}</code></pre>

<img src="https://images.unsplash.com/photo-1571171637578-41bc2dd41cd2?w=800&h=400&fit=crop" alt="Development workflow" class="rounded-lg shadow-xl my-8" loading="lazy">

<h2>Real-time Features</h2>

<p>Add real-time capabilities with WebSockets:</p>

<pre><code class="language-go">// WebSocket connection manager
type WSManager struct {
    conn      js.Value
    listeners map[string][]func(data interface{})
}

func NewWebSocket(url string) *WSManager {
    ws := &WSManager{
        listeners: make(map[string][]func(data interface{})),
    }
    
    ws.conn = js.Global().Get("WebSocket").New(url)
    
    ws.conn.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        data := args[0].Get("data").String()
        ws.handleMessage(data)
        return nil
    }))
    
    return ws
}

func (ws *WSManager) On(event string, handler func(data interface{})) {
    ws.listeners[event] = append(ws.listeners[event], handler)
}

func (ws *WSManager) Emit(event string, data interface{}) {
    message := map[string]interface{}{
        "event": event,
        "data":  data,
    }
    
    json, _ := json.Marshal(message)
    ws.conn.Call("send", string(json))
}

// Usage: Live comment system
func LiveComments() *vdom.VNode {
    comments := reactive.Signal([]Comment{})
    
    ws := NewWebSocket("wss://your-blog.com/ws")
    
    ws.On("new_comment", func(data interface{}) {
        comment := data.(Comment)
        current := comments.Get()
        comments.Set(append(current, comment))
    })
    
    return CommentList(comments.Get())
}</code></pre>

<h2>SEO and Meta Tags</h2>

<p>Optimize for search engines with dynamic meta tags:</p>

<pre><code class="language-go">func SetMetaTags(post BlogPost) {
    document := js.Global().Get("document")
    head := document.Get("head")
    
    // Update title
    document.Set("title", post.Title + " | My Blog")
    
    // Update meta description
    updateMetaTag("description", post.Excerpt)
    
    // Open Graph tags
    updateMetaTag("og:title", post.Title)
    updateMetaTag("og:description", post.Excerpt)
    updateMetaTag("og:image", post.HeroImage)
    updateMetaTag("og:url", "https://myblog.com/blog/" + post.Slug)
    
    // Twitter Card
    updateMetaTag("twitter:card", "summary_large_image")
    updateMetaTag("twitter:title", post.Title)
    updateMetaTag("twitter:description", post.Excerpt)
    updateMetaTag("twitter:image", post.HeroImage)
}

func updateMetaTag(name, content string) {
    document := js.Global().Get("document")
    
    selector := fmt.Sprintf("meta[name='%s'], meta[property='%s']", name, name)
    meta := document.Call("querySelector", selector)
    
    if meta.IsNull() {
        meta = document.Call("createElement", "meta")
        if strings.HasPrefix(name, "og:") {
            meta.Set("property", name)
        } else {
            meta.Set("name", name)
        }
        document.Get("head").Call("appendChild", meta)
    }
    
    meta.Set("content", content)
}</code></pre>

<div class="bg-green-50 dark:bg-green-900/20 border-l-4 border-green-500 p-6 my-8 rounded-r-lg">
<p class="font-semibold text-green-900 dark:text-green-200 mb-2">🎯 Best Practice</p>
<p class="text-green-800 dark:text-green-300">Always profile before optimizing. Use the browser's DevTools Performance tab to identify actual bottlenecks rather than guessing.</p>
</div>

<h2>Conclusion</h2>

<p>These advanced patterns will help you build sophisticated Vango applications that are:</p>

<ul>
<li>✅ Performant at scale</li>
<li>✅ Maintainable with clear state management</li>
<li>✅ Optimized for user experience</li>
<li>✅ SEO-friendly despite being a SPA</li>
<li>✅ Real-time capable</li>
</ul>

<p>Remember, not every application needs all these patterns. Start simple and add complexity only when needed. The beauty of Vango is that you can progressively enhance your application as requirements grow.</p>

<p>Keep experimenting, keep building, and share your discoveries with the community! 🚀</p>
</div>` + "`" + `,
		},
	}
}
`
	
	return WriteFile(filepath.Join(config.Directory, "app/lib/markdown/loader.go"), content)
}

func (t *BlogTemplate) createBlogTypes(_ *ProjectConfig) error {
	// This is now included in the loader.go file above
	return nil
}

func (t *BlogTemplate) createBlogIndex(config *ProjectConfig) error {
	content := `package routes

import (
	"fmt"
	"strings"
	
	"` + config.Module + `/app/lib/markdown"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// BlogIndex renders the blog homepage with all posts
func BlogIndex(posts []markdown.BlogPost) *vdom.VNode {
	// Get featured post (most recent)
	var featuredPost *markdown.BlogPost
	if len(posts) > 0 {
		featuredPost = &posts[0]
	}
	
	// Get other posts
	var otherPosts []markdown.BlogPost
	if len(posts) > 1 {
		otherPosts = posts[1:]
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-800 transition-colors duration-200"),
	),
		// Navigation Header
		BlogHeader(),
		
		// Hero Section with Featured Post
		HeroSection(featuredPost),
		
		// Blog Posts Grid
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16"),
		),
			// Section Title
			functional.Div(functional.MergeProps(
				functional.Class("text-center mb-12"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-3xl font-bold text-gray-900 dark:text-white mb-4"),
				), functional.Text("Latest Articles")),
				functional.P(functional.MergeProps(
					functional.Class("text-lg text-gray-600 dark:text-gray-400 max-w-2xl mx-auto"),
				), functional.Text("Explore our collection of tutorials, guides, and insights about building modern web applications with Vango.")),
			),
			
			// Posts Grid
			functional.Div(functional.MergeProps(
				functional.Class("grid gap-8 md:grid-cols-2 lg:grid-cols-3"),
			), createPostCards(otherPosts)...),
			
			// View All Link
			functional.Div(functional.MergeProps(
				functional.Class("text-center mt-12"),
			),
				functional.A(functional.MergeProps(
					functional.Href("/archive"),
					functional.Class("inline-flex items-center px-6 py-3 bg-gradient-to-r from-purple-600 to-blue-600 text-white font-semibold rounded-lg hover:from-purple-700 hover:to-blue-700 transition-all duration-200 transform hover:scale-105"),
					vdom.Props{"onclick": "event.preventDefault(); navigateTo('/archive')"},
				), 
					functional.Text("View All Posts"),
					functional.Span(functional.MergeProps(
						functional.Class("ml-2"),
					), functional.Text("→")),
				),
			),
		),
		
		// Newsletter Section
		NewsletterSection(),
		
		// Footer
		BlogFooter(),
	)
}

// HeroSection with featured post
func HeroSection(featured *markdown.BlogPost) *vdom.VNode {
	if featured == nil {
		return functional.Div(nil)
	}
	
	return functional.Section(functional.MergeProps(
		functional.Class("relative overflow-hidden bg-gradient-to-br from-purple-600 via-blue-600 to-purple-700 dark:from-purple-900 dark:via-blue-900 dark:to-purple-800"),
	),
		// Background Pattern
		functional.Div(functional.MergeProps(
			functional.Class("absolute inset-0 opacity-10"),
			functional.StyleAttr(` + "`" + `
				background-image: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.4'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
			` + "`" + `),
		)),
		
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("grid lg:grid-cols-2 gap-12 items-center"),
			),
				// Text Content
				functional.Div(functional.MergeProps(
					functional.Class("text-white"),
				),
					functional.Div(functional.MergeProps(
						functional.Class("inline-flex items-center px-3 py-1 bg-white/20 backdrop-blur-sm rounded-full text-sm font-medium mb-4"),
					),
						functional.Text("Featured Post"),
					),
					
					functional.H1(functional.MergeProps(
						functional.Class("text-4xl md:text-5xl font-bold mb-6"),
					), functional.Text(featured.Title)),
					
					functional.P(functional.MergeProps(
						functional.Class("text-xl text-purple-100 mb-8 leading-relaxed"),
					), functional.Text(featured.Excerpt)),
					
					functional.Div(functional.MergeProps(
						functional.Class("flex flex-wrap items-center gap-6 mb-8"),
					),
						// Author
						functional.Div(functional.MergeProps(
							functional.Class("flex items-center"),
						),
							functional.Img(functional.MergeProps(
								functional.Src(featured.AuthorImage),
								functional.Alt(featured.Author),
								functional.Class("w-12 h-12 rounded-full mr-3 border-2 border-white/50"),
							)),
							functional.Div(nil,
								functional.P(functional.MergeProps(
									functional.Class("font-semibold"),
								), functional.Text(featured.Author)),
								functional.P(functional.MergeProps(
									functional.Class("text-sm text-purple-200"),
								), functional.Text(featured.Date.Format("Jan 2, 2006"))),
							),
						),
						
						// Reading time
						functional.Div(functional.MergeProps(
							functional.Class("flex items-center text-purple-200"),
						),
							functional.Text(fmt.Sprintf("📖 %d min read", featured.ReadingTime)),
						),
					),
					
					functional.A(functional.MergeProps(
						functional.Href("/blog/" + featured.Slug),
						functional.Class("inline-flex items-center px-6 py-3 bg-white text-purple-700 font-semibold rounded-lg hover:bg-purple-50 transition-all duration-200 transform hover:scale-105"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/blog/" + featured.Slug + "')"},
					),
						functional.Text("Read Article"),
						functional.Span(functional.MergeProps(
							functional.Class("ml-2"),
						), functional.Text("→")),
					),
				),
				
				// Featured Image
				functional.Div(functional.MergeProps(
					functional.Class("relative"),
				),
					functional.Img(functional.MergeProps(
						functional.Src(featured.HeroImage),
						functional.Alt(featured.Title),
						functional.Class("rounded-2xl shadow-2xl w-full h-auto"),
						vdom.Props{"loading": "lazy"},
					)),
					
					// Tags
					functional.Div(functional.MergeProps(
						functional.Class("absolute bottom-4 left-4 flex flex-wrap gap-2"),
					), createTagBadges(featured.Tags)...),
				),
			),
		),
	)
}

// createPostCards creates blog post cards
func createPostCards(posts []markdown.BlogPost) []*vdom.VNode {
	cards := make([]*vdom.VNode, 0, len(posts))
	
	for _, post := range posts {
		card := functional.Article(functional.MergeProps(
			functional.Class("group bg-white dark:bg-gray-800 rounded-2xl shadow-lg overflow-hidden hover:shadow-2xl transition-all duration-300 hover:-translate-y-2"),
		),
			// Post Image
			functional.Div(functional.MergeProps(
				functional.Class("relative h-48 overflow-hidden"),
			),
				functional.Img(functional.MergeProps(
					functional.Src(post.HeroImage),
					functional.Alt(post.Title),
					functional.Class("w-full h-full object-cover group-hover:scale-110 transition-transform duration-300"),
					vdom.Props{"loading": "lazy"},
				)),
				
				// Category Badge
				functional.Div(functional.MergeProps(
					functional.Class("absolute top-4 left-4"),
				),
					functional.Span(functional.MergeProps(
						functional.Class("px-3 py-1 bg-white/90 dark:bg-gray-900/90 backdrop-blur-sm text-purple-600 dark:text-purple-400 text-sm font-semibold rounded-full"),
					), functional.Text(strings.Title(post.Tags[0]))),
				),
			),
			
			// Content
			functional.Div(functional.MergeProps(
				functional.Class("p-6"),
			),
				// Meta
				functional.Div(functional.MergeProps(
					functional.Class("flex items-center text-sm text-gray-500 dark:text-gray-400 mb-3"),
				),
					functional.Span(nil, functional.Text(post.Date.Format("Jan 2, 2006"))),
					functional.Span(functional.MergeProps(
						functional.Class("mx-2"),
					), functional.Text("•")),
					functional.Span(nil, functional.Text(fmt.Sprintf("%%d min read", post.ReadingTime))),
				),
				
				// Title
				functional.H3(functional.MergeProps(
					functional.Class("text-xl font-bold text-gray-900 dark:text-white mb-3 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors"),
				),
					functional.A(functional.MergeProps(
						functional.Href("/blog/" + post.Slug),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/blog/" + post.Slug + "')"},
					), functional.Text(post.Title)),
				),
				
				// Excerpt
				functional.P(functional.MergeProps(
					functional.Class("text-gray-600 dark:text-gray-300 mb-4 line-clamp-3"),
				), functional.Text(post.Excerpt)),
				
				// Footer
				functional.Div(functional.MergeProps(
					functional.Class("flex items-center justify-between"),
				),
					// Author
					functional.Div(functional.MergeProps(
						functional.Class("flex items-center"),
					),
						functional.Img(functional.MergeProps(
							functional.Src(post.AuthorImage),
							functional.Alt(post.Author),
							functional.Class("w-8 h-8 rounded-full mr-2"),
						)),
						functional.Span(functional.MergeProps(
							functional.Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
						), functional.Text(post.Author)),
					),
					
					// Read More
					functional.A(functional.MergeProps(
						functional.Href("/blog/" + post.Slug),
						functional.Class("text-purple-600 dark:text-purple-400 hover:text-purple-700 dark:hover:text-purple-300 font-medium text-sm inline-flex items-center"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/blog/" + post.Slug + "')"},
					), 
						functional.Text("Read →"),
					),
				),
			),
		)
		
		cards = append(cards, card)
	}
	
	return cards
}

// createTagBadges creates tag badges
func createTagBadges(tags []string) []*vdom.VNode {
	badges := make([]*vdom.VNode, 0, len(tags))
	
	for _, tag := range tags {
		badge := functional.Span(functional.MergeProps(
			functional.Class("px-3 py-1 bg-white/20 backdrop-blur-sm text-white text-xs font-medium rounded-full"),
		), functional.Text("#" + tag))
		badges = append(badges, badge)
	}
	
	return badges
}
`
	
	return WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), content)
}

func (t *BlogTemplate) createBlogPostPage(config *ProjectConfig) error {
	content := `package routes

import (
	"fmt"
	"strings"
	
	"` + config.Module + `/app/lib/markdown"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// BlogPostPage renders an individual blog post
func BlogPostPage(post markdown.BlogPost) *vdom.VNode {
	// Get related posts
	relatedPosts := markdown.GetRelatedPosts(post, 3)
	
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-white dark:bg-gray-900 transition-colors duration-200"),
	),
		// Navigation Header
		BlogHeader(),
		
		// Hero Image
		functional.Div(functional.MergeProps(
			functional.Class("relative h-96 overflow-hidden"),
		),
			functional.Img(functional.MergeProps(
				functional.Src(post.HeroImage),
				functional.Alt(post.Title),
				functional.Class("w-full h-full object-cover"),
			)),
			functional.Div(functional.MergeProps(
				functional.Class("absolute inset-0 bg-gradient-to-t from-black/60 to-transparent"),
			)),
			
			// Title Overlay
			functional.Div(functional.MergeProps(
				functional.Class("absolute bottom-0 left-0 right-0 p-8"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("max-w-4xl mx-auto"),
				),
					functional.H1(functional.MergeProps(
						functional.Class("text-4xl md:text-5xl font-bold text-white mb-4"),
					), functional.Text(post.Title)),
					
					// Meta
					functional.Div(functional.MergeProps(
						functional.Class("flex flex-wrap items-center gap-4 text-white/90"),
					),
						// Author
						functional.Div(functional.MergeProps(
							functional.Class("flex items-center"),
						),
							functional.Img(functional.MergeProps(
								functional.Src(post.AuthorImage),
								functional.Alt(post.Author),
								functional.Class("w-10 h-10 rounded-full mr-2 border-2 border-white/50"),
							)),
							functional.Span(nil, functional.Text(post.Author)),
						),
						
						functional.Span(nil, functional.Text("•")),
						functional.Span(nil, functional.Text(post.Date.Format("January 2, 2006"))),
						functional.Span(nil, functional.Text("•")),
						functional.Span(nil, functional.Text(fmt.Sprintf("%%d min read", post.ReadingTime))),
					),
				),
			),
		),
		
		// Article Content
		functional.Article(functional.MergeProps(
			functional.Class("max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
		),
			// Tags
			functional.Div(functional.MergeProps(
				functional.Class("flex flex-wrap gap-2 mb-8"),
			), createTagElements(post.Tags)...),
			
			// Content (raw HTML from markdown)
			functional.Div(functional.MergeProps(
				functional.Class("article-content"),
				vdom.Props{"innerHTML": post.Content},
			)),
			
			// Author Bio
			AuthorBio(post.Author, post.AuthorImage),
			
			// Share Buttons
			ShareButtons(post),
		),
		
		// Related Posts
		functional.Div(functional.MergeProps(
			functional.Class("bg-gray-50 dark:bg-gray-800 py-16"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-2xl font-bold text-gray-900 dark:text-white mb-8"),
				), functional.Text("Related Articles")),
				
				functional.Div(functional.MergeProps(
					functional.Class("grid md:grid-cols-3 gap-8"),
				), createPostCards(relatedPosts)...),
			),
		),
		
		// Footer
		BlogFooter(),
	)
}

// AuthorBio component
func AuthorBio(author string, image string) *vdom.VNode {
	bios := map[string]string{
		"Sarah Chen": "Sarah is a senior software engineer with 10+ years of experience building scalable web applications. She's passionate about Go, WebAssembly, and modern web development.",
		"Marcus Johnson": "Marcus is a UI/UX designer turned developer who loves creating beautiful, functional interfaces. He specializes in design systems and component architecture.",
		"Elena Rodriguez": "Elena is a DevOps engineer focused on cloud infrastructure and deployment automation. She helps teams ship faster and more reliably.",
		"Alex Kim": "Alex is a performance engineer obsessed with making web applications fast. He contributes to several open-source projects including Vango.",
	}
	
	bio, exists := bios[author]
	if !exists {
		bio = "A passionate developer and contributor to the Vango community."
	}
	
	return functional.Div(functional.MergeProps(
		functional.Class("my-12 p-6 bg-gray-50 dark:bg-gray-800 rounded-xl"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("flex items-start space-x-4"),
		),
			functional.Img(functional.MergeProps(
				functional.Src(image),
				functional.Alt(author),
				functional.Class("w-20 h-20 rounded-full"),
			)),
			
			functional.Div(nil,
				functional.H3(functional.MergeProps(
					functional.Class("font-bold text-lg text-gray-900 dark:text-white mb-2"),
				), functional.Text("About " + author)),
				
				functional.P(functional.MergeProps(
					functional.Class("text-gray-600 dark:text-gray-300"),
				), functional.Text(bio)),
				
				// Social Links
				functional.Div(functional.MergeProps(
					functional.Class("flex gap-4 mt-4"),
				),
					functional.A(functional.MergeProps(
						functional.Href("https://twitter.com"),
						functional.Class("text-gray-400 hover:text-blue-500"),
						functional.Target("_blank"),
						vdom.Props{"rel": "noopener noreferrer"},
					), functional.Text("Twitter")),
					
					functional.A(functional.MergeProps(
						functional.Href("https://github.com"),
						functional.Class("text-gray-400 hover:text-gray-900 dark:hover:text-white"),
						functional.Target("_blank"),
						vdom.Props{"rel": "noopener noreferrer"},
					), functional.Text("GitHub")),
					
					functional.A(functional.MergeProps(
						functional.Href("https://linkedin.com"),
						functional.Class("text-gray-400 hover:text-blue-700"),
						functional.Target("_blank"),
						vdom.Props{"rel": "noopener noreferrer"},
					), functional.Text("LinkedIn")),
				),
			),
		),
	)
}

// ShareButtons component
func ShareButtons(post markdown.BlogPost) *vdom.VNode {
	url := "https://yourblog.com/blog/" + post.Slug
	title := strings.ReplaceAll(post.Title, " ", "%20")
	
	return functional.Div(functional.MergeProps(
		functional.Class("my-8 py-8 border-t border-b border-gray-200 dark:border-gray-700"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("flex items-center justify-between"),
		),
			functional.P(functional.MergeProps(
				functional.Class("text-gray-600 dark:text-gray-400 font-medium"),
			), functional.Text("Share this article:")),
			
			functional.Div(functional.MergeProps(
				functional.Class("flex gap-4"),
			),
				// Twitter
				functional.A(functional.MergeProps(
					functional.Href("https://twitter.com/intent/tweet?text=" + title + "&url=" + url),
					functional.Target("_blank"),
					vdom.Props{"rel": "noopener noreferrer"},
					functional.Class("px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"),
				), functional.Text("Twitter")),
				
				// LinkedIn
				functional.A(functional.MergeProps(
					functional.Href("https://www.linkedin.com/sharing/share-offsite/?url=" + url),
					functional.Target("_blank"),
					vdom.Props{"rel": "noopener noreferrer"},
					functional.Class("px-4 py-2 bg-blue-700 text-white rounded-lg hover:bg-blue-800 transition-colors"),
				), functional.Text("LinkedIn")),
				
				// Copy Link
				functional.Button(functional.MergeProps(
					functional.Class("px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"),
					vdom.Props{"onclick": "navigator.clipboard.writeText('" + url + "'); alert('Link copied!')"},
				), functional.Text("Copy Link")),
			),
		),
	)
}

// createTagElements for the post page
func createTagElements(tags []string) []*vdom.VNode {
	elements := make([]*vdom.VNode, 0, len(tags))
	
	for _, tag := range tags {
		elem := functional.A(functional.MergeProps(
			functional.Href("/archive?tag=" + tag),
			functional.Class("px-3 py-1 text-sm bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 rounded-full hover:bg-purple-200 dark:hover:bg-purple-800 transition-colors"),
			vdom.Props{"onclick": "event.preventDefault(); navigateTo('/archive?tag=" + tag + "')"},
		), functional.Text("#" + tag))
		elements = append(elements, elem)
	}
	
	return elements
}
`
	
	return WriteFile(filepath.Join(config.Directory, "app/routes/blog.go"), content)
}

func (t *BlogTemplate) createBlogComponents(config *ProjectConfig) error {
	// Header component
	headerContent := fmt.Sprintf(`package routes

import (
	"fmt"
	"%s/app/lib/markdown"
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/functional"
)

// BlogHeader renders the navigation header
func BlogHeader() *vdom.VNode {
	return functional.Header(functional.MergeProps(
		functional.Class("sticky top-0 z-50 bg-white/80 dark:bg-gray-900/80 backdrop-blur-md border-b border-gray-200 dark:border-gray-800"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("flex justify-between items-center h-16"),
			),
				// Logo
				functional.A(functional.MergeProps(
					functional.Href("/"),
					functional.Class("text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 dark:from-purple-400 dark:to-blue-400 text-transparent bg-clip-text"),
					vdom.Props{"onclick": "event.preventDefault(); navigateTo('/')"},
				), functional.Text("VangoBlog")),
				
				// Navigation
				functional.Nav(functional.MergeProps(
					functional.Class("hidden md:flex items-center space-x-8"),
				),
					functional.A(functional.MergeProps(
						functional.Href("/"),
						functional.Class("text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400 transition-colors"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/')"},
					), functional.Text("Home")),
					
					functional.A(functional.MergeProps(
						functional.Href("/archive"),
						functional.Class("text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400 transition-colors"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/archive')"},
					), functional.Text("Archive")),
					
					functional.A(functional.MergeProps(
						functional.Href("/about"),
						functional.Class("text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400 transition-colors"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/about')"},
					), functional.Text("About")),
					
					functional.A(functional.MergeProps(
						functional.Href("https://github.com/recera/vango"),
						functional.Target("_blank"),
						vdom.Props{"rel": "noopener noreferrer"},
						functional.Class("text-gray-700 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400 transition-colors"),
					), functional.Text("GitHub")),
				),
				
				// Dark Mode Toggle
				functional.Button(functional.MergeProps(
					functional.Class("p-2 rounded-lg bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"),
					vdom.Props{"onclick": "toggleDarkMode()"},
				),
					functional.Span(functional.MergeProps(
						functional.Class("text-xl"),
					), functional.Text("🌙")),
				),
			),
		),
	)
}

// BlogFooter renders the site footer
func BlogFooter() *vdom.VNode {
	return functional.Footer(functional.MergeProps(
		functional.Class("bg-gray-900 text-white"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12"),
		),
			functional.Div(functional.MergeProps(
				functional.Class("grid md:grid-cols-4 gap-8"),
			),
				// About
				functional.Div(nil,
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold mb-4"),
					), functional.Text("VangoBlog")),
					functional.P(functional.MergeProps(
						functional.Class("text-gray-400"),
					), functional.Text("A modern blog built with Vango, the Go-native frontend framework. Learn, build, and share your journey.")),
				),
				
				// Quick Links
				functional.Div(nil,
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold mb-4"),
					), functional.Text("Quick Links")),
					functional.Ul(functional.MergeProps(
						functional.Class("space-y-2"),
					),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("/"),
								functional.Class("text-gray-400 hover:text-white transition-colors"),
								vdom.Props{"onclick": "event.preventDefault(); navigateTo('/')"},
							), functional.Text("Home")),
						),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("/archive"),
								functional.Class("text-gray-400 hover:text-white transition-colors"),
								vdom.Props{"onclick": "event.preventDefault(); navigateTo('/archive')"},
							), functional.Text("Archive")),
						),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("/about"),
								functional.Class("text-gray-400 hover:text-white transition-colors"),
								vdom.Props{"onclick": "event.preventDefault(); navigateTo('/about')"},
							), functional.Text("About")),
						),
					),
				),
				
				// Resources
				functional.Div(nil,
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold mb-4"),
					), functional.Text("Resources")),
					functional.Ul(functional.MergeProps(
						functional.Class("space-y-2"),
					),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("https://vango.dev/docs"),
								functional.Target("_blank"),
								vdom.Props{"rel": "noopener noreferrer"},
								functional.Class("text-gray-400 hover:text-white transition-colors"),
							), functional.Text("Documentation")),
						),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("https://github.com/recera/vango"),
								functional.Target("_blank"),
								vdom.Props{"rel": "noopener noreferrer"},
								functional.Class("text-gray-400 hover:text-white transition-colors"),
							), functional.Text("GitHub")),
						),
						functional.Li(nil,
							functional.A(functional.MergeProps(
								functional.Href("https://discord.gg/vango"),
								functional.Target("_blank"),
								vdom.Props{"rel": "noopener noreferrer"},
								functional.Class("text-gray-400 hover:text-white transition-colors"),
							), functional.Text("Discord")),
						),
					),
				),
				
				// Newsletter
				functional.Div(nil,
					functional.H3(functional.MergeProps(
						functional.Class("text-lg font-semibold mb-4"),
					), functional.Text("Stay Updated")),
					functional.P(functional.MergeProps(
						functional.Class("text-gray-400 mb-4"),
					), functional.Text("Get the latest Vango tutorials and updates.")),
					functional.Form(functional.MergeProps(
						functional.Class("flex"),
						vdom.Props{"onsubmit": "event.preventDefault(); alert('Newsletter subscription coming soon!')"},
					),
						functional.Input(functional.MergeProps(
							functional.Class("flex-1 px-4 py-2 bg-gray-800 text-white rounded-l-lg focus:outline-none focus:ring-2 focus:ring-purple-600"),
							vdom.Props{
								"type": "email",
								"placeholder": "your@email.com",
								"required": "true",
							},
						)),
						functional.Button(functional.MergeProps(
							functional.Class("px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-r-lg transition-colors"),
							vdom.Props{"type": "submit"},
						), functional.Text("Subscribe")),
					),
				),
			),
			
			// Copyright
			functional.Div(functional.MergeProps(
				functional.Class("mt-8 pt-8 border-t border-gray-800 text-center text-gray-400"),
			),
				functional.P(nil, functional.Text("© 2024 VangoBlog. Built with Vango and ❤️")),
			),
		),
	)
}

// NewsletterSection component
func NewsletterSection() *vdom.VNode {
	return functional.Section(functional.MergeProps(
		functional.Class("bg-gradient-to-r from-purple-600 to-blue-600 py-16"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center"),
		),
			functional.H2(functional.MergeProps(
				functional.Class("text-3xl font-bold text-white mb-4"),
			), functional.Text("Stay in the Loop")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-xl text-purple-100 mb-8"),
			), functional.Text("Get weekly updates on the latest Vango tutorials, tips, and community news.")),
			
			functional.Form(functional.MergeProps(
				functional.Class("max-w-md mx-auto flex flex-col sm:flex-row gap-4"),
				vdom.Props{"onsubmit": "event.preventDefault(); alert('Thank you for subscribing!')"},
			),
				functional.Input(functional.MergeProps(
					functional.Class("flex-1 px-4 py-3 rounded-lg text-gray-900 focus:outline-none focus:ring-4 focus:ring-white/50"),
					vdom.Props{
						"type": "email",
						"placeholder": "Enter your email",
						"required": "true",
					},
				)),
				
				functional.Button(functional.MergeProps(
					functional.Class("px-6 py-3 bg-white text-purple-600 font-semibold rounded-lg hover:bg-purple-50 transition-colors"),
					vdom.Props{"type": "submit"},
				), functional.Text("Subscribe")),
			),
			
			functional.P(functional.MergeProps(
				functional.Class("text-sm text-purple-200 mt-4"),
			), functional.Text("No spam, unsubscribe anytime.")),
		),
	)
}

// Additional page components

// AboutPage renders the about page
func AboutPage() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-white dark:bg-gray-900"),
	),
		BlogHeader(),
		
		functional.Div(functional.MergeProps(
			functional.Class("max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-16"),
		),
			functional.H1(functional.MergeProps(
				functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
			), functional.Text("About VangoBlog")),
			
			functional.Div(functional.MergeProps(
				functional.Class("prose prose-lg dark:prose-invert max-w-none"),
			),
				functional.P(nil, functional.Text("Welcome to VangoBlog - your comprehensive resource for learning and mastering Vango, the revolutionary Go-native frontend framework.")),
				
				functional.H2(nil, functional.Text("Our Mission")),
				functional.P(nil, functional.Text("We believe that web development should be simple, type-safe, and performant. That's why we created this blog - to help developers like you build amazing web applications using Go and WebAssembly.")),
				
				functional.H2(nil, functional.Text("What You'll Find Here")),
				functional.Ul(nil,
					functional.Li(nil, functional.Text("Step-by-step tutorials for beginners")),
					functional.Li(nil, functional.Text("Advanced patterns and techniques")),
					functional.Li(nil, functional.Text("Real-world project examples")),
					functional.Li(nil, functional.Text("Performance optimization guides")),
					functional.Li(nil, functional.Text("Community showcases and success stories")),
				),
				
				functional.H2(nil, functional.Text("Join Our Community")),
				functional.P(nil, functional.Text("Connect with thousands of developers who are building the future of web development with Vango. Share your projects, get help, and contribute to the ecosystem.")),
			),
		),
		
		BlogFooter(),
	)
}

// ArchivePage renders the archive page with all posts
func ArchivePage(posts []markdown.BlogPost) *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-white dark:bg-gray-900"),
	),
		BlogHeader(),
		
		functional.Div(functional.MergeProps(
			functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16"),
		),
			functional.H1(functional.MergeProps(
				functional.Class("text-4xl font-bold text-gray-900 dark:text-white mb-8"),
			), functional.Text("Blog Archive")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-lg text-gray-600 dark:text-gray-400 mb-12"),
			), functional.Text("Browse all our articles and tutorials.")),
			
			// Posts list
			functional.Div(functional.MergeProps(
				functional.Class("space-y-8"),
			), createArchiveList(posts)...),
		),
		
		BlogFooter(),
	)
}

// createArchiveList creates a list view of posts for the archive
func createArchiveList(posts []markdown.BlogPost) []*vdom.VNode {
	items := make([]*vdom.VNode, 0, len(posts))
	
	for _, post := range posts {
		item := functional.Article(functional.MergeProps(
			functional.Class("flex gap-6 p-6 bg-gray-50 dark:bg-gray-800 rounded-lg hover:shadow-lg transition-shadow"),
		),
			// Date
			functional.Div(functional.MergeProps(
				functional.Class("flex-shrink-0 text-center"),
			),
				functional.Div(functional.MergeProps(
					functional.Class("text-3xl font-bold text-purple-600 dark:text-purple-400"),
				), functional.Text(post.Date.Format("02"))),
				functional.Div(functional.MergeProps(
					functional.Class("text-sm text-gray-600 dark:text-gray-400"),
				), functional.Text(post.Date.Format("Jan"))),
				functional.Div(functional.MergeProps(
					functional.Class("text-sm text-gray-500 dark:text-gray-500"),
				), functional.Text(post.Date.Format("2006"))),
			),
			
			// Content
			functional.Div(functional.MergeProps(
				functional.Class("flex-1"),
			),
				functional.H2(functional.MergeProps(
					functional.Class("text-xl font-bold text-gray-900 dark:text-white mb-2"),
				),
					functional.A(functional.MergeProps(
						functional.Href("/blog/" + post.Slug),
						functional.Class("hover:text-purple-600 dark:hover:text-purple-400 transition-colors"),
						vdom.Props{"onclick": "event.preventDefault(); navigateTo('/blog/" + post.Slug + "')"},
					), functional.Text(post.Title)),
				),
				
				functional.P(functional.MergeProps(
					functional.Class("text-gray-600 dark:text-gray-300 mb-3"),
				), functional.Text(post.Excerpt)),
				
				functional.Div(functional.MergeProps(
					functional.Class("flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400"),
				),
					functional.Span(nil, functional.Text("By " + post.Author)),
					functional.Span(nil, functional.Text("•")),
					functional.Span(nil, functional.Text(fmt.Sprintf("%%d min read", post.ReadingTime))),
				),
			),
		)
		
		items = append(items, item)
	}
	
	return items
}

// NotFound renders a 404 page
func NotFound() *vdom.VNode {
	return functional.Div(functional.MergeProps(
		functional.Class("min-h-screen bg-white dark:bg-gray-900 flex items-center justify-center"),
	),
		functional.Div(functional.MergeProps(
			functional.Class("text-center"),
		),
			functional.H1(functional.MergeProps(
				functional.Class("text-6xl font-bold text-gray-900 dark:text-white mb-4"),
			), functional.Text("404")),
			
			functional.P(functional.MergeProps(
				functional.Class("text-xl text-gray-600 dark:text-gray-400 mb-8"),
			), functional.Text("Oops! The page you're looking for doesn't exist.")),
			
			functional.A(functional.MergeProps(
				functional.Href("/"),
				functional.Class("inline-flex items-center px-6 py-3 bg-purple-600 text-white font-semibold rounded-lg hover:bg-purple-700 transition-colors"),
				vdom.Props{"onclick": "event.preventDefault(); navigateTo('/')"},
			), functional.Text("Go Back Home")),
		),
	)
}
`, config.Module)
	
	return WriteFile(filepath.Join(config.Directory, "app/routes/blog_components.go"), headerContent)
}

func (t *BlogTemplate) createComprehensiveSamplePosts(config *ProjectConfig) error {
	// Sample posts are now generated in the markdown loader
	// We'll create markdown files that would be processed by a build step
	
	// Create a README for the content directory
	readmeContent := `# Blog Content

Place your blog posts as markdown files in the ` + "`content/posts/`" + ` directory.

## File Format

Each markdown file should have front matter in YAML format:

` + "```yaml" + `
---
title: "Your Post Title"
date: 2024-12-20
author: "Your Name"
author_image: "https://example.com/avatar.jpg"
tags: ["tag1", "tag2"]
excerpt: "A brief description of your post"
hero_image: "https://example.com/hero.jpg"
published: true
---

Your markdown content here...
` + "```" + `

## Naming Convention

Use the format: ` + "`YYYY-MM-DD-slug.md`" + `

Example: ` + "`2024-12-20-getting-started-with-vango.md`" + `

## Adding Images

You can use any public image URL or place images in ` + "`public/images/`" + ` and reference them as ` + "`/images/your-image.jpg`" + `.

## Publishing

Set ` + "`published: true`" + ` in the front matter to make a post visible on the blog.
`
	
	return WriteFile(filepath.Join(config.Directory, "content/README.md"), readmeContent)
}

func (t *BlogTemplate) createEnhancedTailwindConfig(config *ProjectConfig) error {
	tailwindConfig := `module.exports = {
  content: [
    "./app/**/*.go",
    "./content/**/*.md",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        gray: {
          850: '#18202F',
          950: '#0a0e1a',
        },
      },
      typography: (theme) => ({
        DEFAULT: {
          css: {
            color: theme('colors.gray.700'),
            a: {
              color: theme('colors.purple.600'),
              '&:hover': {
                color: theme('colors.purple.700'),
              },
            },
            'h1,h2,h3,h4': {
              color: theme('colors.gray.900'),
            },
            code: {
              color: theme('colors.purple.600'),
              backgroundColor: theme('colors.purple.50'),
              borderRadius: theme('borderRadius.md'),
              paddingLeft: theme('spacing.1'),
              paddingRight: theme('spacing.1'),
            },
            'code::before': {
              content: '""',
            },
            'code::after': {
              content: '""',
            },
            pre: {
              backgroundColor: theme('colors.gray.900'),
              color: theme('colors.gray.100'),
            },
            blockquote: {
              borderLeftColor: theme('colors.purple.600'),
              color: theme('colors.gray.600'),
            },
          },
        },
        dark: {
          css: {
            color: theme('colors.gray.300'),
            a: {
              color: theme('colors.purple.400'),
              '&:hover': {
                color: theme('colors.purple.300'),
              },
            },
            'h1,h2,h3,h4': {
              color: theme('colors.gray.100'),
            },
            code: {
              color: theme('colors.purple.400'),
              backgroundColor: theme('colors.gray.800'),
            },
            pre: {
              backgroundColor: theme('colors.gray.900'),
              color: theme('colors.gray.100'),
            },
            blockquote: {
              borderLeftColor: theme('colors.purple.400'),
              color: theme('colors.gray.400'),
            },
          },
        },
      }),
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}`
	
	if err := WriteFile(filepath.Join(config.Directory, "tailwind.config.js"), tailwindConfig); err != nil {
		return err
	}
	
	// Create package.json
	packageJSON := fmt.Sprintf(`{
  "name": "%s-blog",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "build-css": "tailwindcss -i ./styles/input.css -o ./public/styles.css --minify",
    "watch-css": "tailwindcss -i ./styles/input.css -o ./public/styles.css --watch"
  },
  "devDependencies": {
    "@tailwindcss/typography": "^0.5.10",
    "tailwindcss": "^3.4.0"
  }
}`, config.Name)
	
	return WriteFile(filepath.Join(config.Directory, "package.json"), packageJSON)
}

func (t *BlogTemplate) createEnhancedStyles(config *ProjectConfig) error {
	inputCSS := `@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  html {
    scroll-behavior: smooth;
  }
  
  ::selection {
    @apply bg-purple-200 dark:bg-purple-800 text-purple-900 dark:text-purple-100;
  }
}

@layer components {
  /* Button styles */
  .btn {
    @apply px-4 py-2 rounded-lg font-medium transition-all duration-200 inline-flex items-center justify-center;
  }
  
  .btn-primary {
    @apply bg-gradient-to-r from-purple-600 to-blue-600 text-white hover:from-purple-700 hover:to-blue-700 transform hover:scale-105;
  }
  
  .btn-secondary {
    @apply bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600;
  }
  
  /* Card styles */
  .card {
    @apply bg-white dark:bg-gray-800 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300;
  }
  
  /* Article content styles */
  .article-content {
    @apply prose prose-lg dark:prose-invert max-w-none;
  }
  
  .article-content img {
    @apply rounded-lg shadow-xl my-8;
  }
  
  .article-content pre {
    @apply bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto;
  }
  
  .article-content code {
    @apply text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-gray-800 px-1 py-0.5 rounded;
  }
  
  .article-content pre code {
    @apply bg-transparent p-0 text-gray-100;
  }
  
  .article-content blockquote {
    @apply border-l-4 border-purple-500 pl-4 italic;
  }
  
  .article-content a {
    @apply text-purple-600 dark:text-purple-400 hover:underline;
  }
  
  /* Loading animation */
  .loading-spinner {
    @apply inline-block w-8 h-8 border-4 border-purple-600 border-t-transparent rounded-full animate-spin;
  }
}

@layer utilities {
  /* Line clamp for text truncation */
  .line-clamp-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .line-clamp-3 {
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  /* Gradient text */
  .gradient-text {
    @apply bg-gradient-to-r from-purple-600 to-blue-600 dark:from-purple-400 dark:to-blue-400 text-transparent bg-clip-text;
  }
  
  /* Animations */
  @keyframes fade-in {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  .animate-fade-in {
    animation: fade-in 0.5s ease-out;
  }
  
  @keyframes slide-up {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  .animate-slide-up {
    animation: slide-up 0.6s ease-out;
  }
}

/* Custom scrollbar */
.dark ::-webkit-scrollbar {
  width: 12px;
}

.dark ::-webkit-scrollbar-track {
  @apply bg-gray-900;
}

.dark ::-webkit-scrollbar-thumb {
  @apply bg-gray-700 rounded-full;
}

.dark ::-webkit-scrollbar-thumb:hover {
  @apply bg-gray-600;
}

/* Code syntax highlighting (you can extend this) */
.hljs {
  @apply bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto;
}

.hljs-keyword {
  @apply text-purple-400;
}

.hljs-string {
  @apply text-green-400;
}

.hljs-comment {
  @apply text-gray-500 italic;
}

.hljs-function {
  @apply text-blue-400;
}

.hljs-number {
  @apply text-orange-400;
}

/* Print styles */
@media print {
  .no-print {
    display: none;
  }
  
  .article-content {
    max-width: 100%;
  }
}
`
	
	return WriteFile(filepath.Join(config.Directory, "styles/input.css"), inputCSS)
}