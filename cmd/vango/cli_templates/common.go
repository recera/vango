package cli_templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createCommonFiles creates files common to all templates
func createCommonFiles(config *ProjectConfig) error {
	// Create necessary directories first
	dirs := []string{
		"public",
		"styles",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(config.Directory, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create go.mod
	if err := createGoMod(config); err != nil {
		return err
	}

	// Create README.md
	readmeContent := generateReadme(config)
	if err := WriteFile(filepath.Join(config.Directory, "README.md"), readmeContent); err != nil {
		return err
	}

	// Create .gitignore
	if err := createGitignore(config); err != nil {
		return err
	}

	// Create vango.json
	if err := createVangoConfig(config); err != nil {
		return err
	}

	// Create index.html
	if err := createIndexHTML(config); err != nil {
		return err
	}

	// Create base styles
	if err := createBaseStyles(config); err != nil {
		return err
	}

	// Create default layout and error pages for SSR
	if config.RoutingStrategy == "file-based" {
		if err := createDefaultLayout(config); err != nil {
			return err
		}
		if err := createErrorPages(config); err != nil {
			return err
		}
	}

	return nil
}

// createGoMod creates the go.mod file
func createGoMod(config *ProjectConfig) error {
	content := fmt.Sprintf(`module %s

go 1.22
`, config.Module)

	return WriteFile(filepath.Join(config.Directory, "go.mod"), content)
}

// generateReadme generates README.md content
func generateReadme(config *ProjectConfig) string {
	featuresSection := ""
	if len(config.Features) > 0 {
		featuresSection = "\n## Features\n\n"
		for _, feature := range config.Features {
			featuresSection += fmt.Sprintf("- ✅ %s\n", feature)
		}
	}

	return fmt.Sprintf(`# %s

 A [Vango](https://github.com/recera/vango) application.
 %s
 ## Getting Started

 ### Development

 Run the development server:

 `+"```bash"+`
 vango dev
 `+"```"+`

 The application will be available at http://localhost:%d

 **Note:** For client-side routing to work properly when directly accessing routes like `+"`/about`"+` or `+"`/counter`"+`, the development server currently requires navigating from the home page first. This is a known limitation that will be addressed in future updates.

 ### Features Demonstrated

 - **Three VEX Syntax Layers**: Examples of Functional, Builder, and Template syntax
 - **Multiple Pages**: Home, About, and Counter pages with client-side routing
 - **Dark/Light Mode**: Toggle between themes with localStorage persistence
 - **Interactive Components**: Counter demo with client-side state management
 - **Reusable Components**: Card, Navigation, Footer, and FeatureItem components

 ### Build

 Create a production build:

 `+"```bash"+`
 vango build
 `+"```"+`

 ### Project Structure

 `+"```"+`
 %s/
 ├── app/
 │   ├── main.go          # Application entry point
 │   ├── routes/          # Page components
 │   ├── components/      # Reusable components
 │   └── layouts/         # Layout components
 ├── public/              # Static assets
 ├── styles/              # CSS files
 └── vango.json           # Configuration
 `+"```"+`

 ## Learn More

 - [Vango Documentation](https://vango.dev/docs)
 - [Examples](https://github.com/recera/vango/tree/main/examples)
 - [Discord Community](https://discord.gg/vango)

 ## License

 MIT`,
		config.Name,
		featuresSection,
		config.Port,
		config.Name,
	)
}

// createGitignore creates the .gitignore file
func createGitignore(config *ProjectConfig) error {
	content := `# Binaries
*.wasm
*.exe
*.dll
*.so
*.dylib

# Test binary
*.test

# Output directories
/dist/
/public/app.wasm
/public/styles.css

# Dependency directories
vendor/`

	if config.UseTailwind {
		content += `
node_modules/
package-lock.json
yarn.lock
pnpm-lock.yaml`
	}

	content += `

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Environment
.env
.env.local`

	return WriteFile(filepath.Join(config.Directory, ".gitignore"), content)
}

// createVangoConfig creates the vango.json configuration file
func createVangoConfig(config *ProjectConfig) error {
	var features []string

	// Build features based on configuration
	if config.UseTailwind {
		features = append(features, fmt.Sprintf(`
  "styling": {
    "tailwind": {
      "enabled": true,
      "strategy": "%s",
      "autoDownload": true,
      "config": "tailwind.config.js",
      "input": "styles/input.css",
      "output": "public/styles.css"
    }
  }`, config.TailwindStrategy))
	}

	featuresStr := ""
	if len(features) > 0 {
		featuresStr = "," + strings.Join(features, ",")
	}

	content := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "dev": {
    "port": %d,
    "host": "localhost",
    "open": %t
  },
  "build": {
    "output": "dist",
    "optimize": true
  }%s
}`, config.Name, config.Port, config.OpenBrowser, featuresStr)

	return WriteFile(filepath.Join(config.Directory, "vango.json"), content)
}

// createIndexHTML creates the public/index.html file
func createIndexHTML(config *ProjectConfig) error {
	content := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + config.Name + `</title>
    <link rel="stylesheet" href="/styles/base.css">`

	if config.UseTailwind {
		// Link to the compiled Tailwind CSS output
		// The dev server will compile this from styles/input.css
		content += `
    <link rel="stylesheet" href="/styles.css">`
	}

	if config.DarkMode {
		// Add dark mode initialization script
		content += `
    <script>
        // Initialize dark mode from localStorage or system preference
        (function() {
            const darkMode = localStorage.getItem('darkMode');
            if (darkMode === 'true' || 
                (darkMode === null && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
                document.documentElement.classList.add('dark');
            }
        })();
        
        // Global dark mode toggle function
        window.toggleDarkMode = function() {
            const isDark = document.documentElement.classList.contains('dark');
            if (isDark) {
                document.documentElement.classList.remove('dark');
                localStorage.setItem('darkMode', 'false');
            } else {
                document.documentElement.classList.add('dark');
                localStorage.setItem('darkMode', 'true');
            }
        };
    </script>`
	}

	content += `
    <script src="/vango/bootstrap.js" defer></script>
    <script>
        // Handle initial route for direct navigation
        window.addEventListener('DOMContentLoaded', function() {
            // The WASM app will handle routing once loaded
            // This ensures proper routing even when accessing URLs directly
        });
    </script>
</head>
<body>
    <div id="app">
        <div class="min-h-screen flex items-center justify-center">
            <div class="text-center">
                <div class="text-4xl mb-4">⚡</div>
                <p class="text-gray-600">Loading Vango...</p>
            </div>
        </div>
    </div>
</body>
</html>`

	return WriteFile(filepath.Join(config.Directory, "public/index.html"), content)
}

// createBaseStyles creates the base CSS file
func createBaseStyles(config *ProjectConfig) error {
	content := `/* Reset and base styles */
* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

html {
	scroll-behavior: smooth;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
	line-height: 1.6;
	color: #333;
	background-color: #fafafa;
	-webkit-font-smoothing: antialiased;
	-moz-osx-font-smoothing: grayscale;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: 2rem;
}

/* Typography */
h1, h2, h3, h4, h5, h6 {
	font-weight: 600;
	line-height: 1.25;
}

a {
	color: #667eea;
	text-decoration: none;
	transition: color 0.2s;
}

a:hover {
	color: #5a67d8;
}

code {
	font-family: 'Courier New', Courier, monospace;
	background: #f1f5f9;
	padding: 0.125rem 0.25rem;
	border-radius: 0.25rem;
	font-size: 0.875em;
}

/* Loading state */
.vango-loading {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 100vh;
	font-size: 1.25rem;
	color: #667eea;
}

/* Responsive design */
@media (max-width: 768px) {
	.container {
		padding: 1rem;
	}
}`

	return WriteFile(filepath.Join(config.Directory, "styles/base.css"), content)
}

// createTailwindConfig creates Tailwind configuration files
func createTailwindConfig(config *ProjectConfig) error {
	// Create tailwind.config.js, but don't overwrite if a template already created one
	tailwindConfig := `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{go,html,js}",
    "./public/**/*.html",
  ],`

	if config.DarkMode {
		tailwindConfig += `
  darkMode: 'class',`
	}

	tailwindConfig += `
  theme: {
    extend: {
      colors: {
        'vango-blue': '#3b82f6',
        'vango-dark': '#1e293b',
      },
    },
  },
  plugins: [],
}`

	twPath := filepath.Join(config.Directory, "tailwind.config.js")
	if _, err := os.Stat(twPath); os.IsNotExist(err) {
		if err := WriteFile(twPath, tailwindConfig); err != nil {
			return err
		}
	}

	// Only create a default input.css if the template didn't provide one
	if _, err := os.Stat(filepath.Join(config.Directory, "styles/input.css")); os.IsNotExist(err) {
		inputCSS := `@tailwind base;
@tailwind components;
@tailwind utilities;

@layer components {
  .btn {
    @apply px-4 py-2 rounded-lg font-medium transition-all duration-200;
  }
  
  .btn-primary {
    @apply bg-blue-600 text-white hover:bg-blue-700 active:scale-95;
  }
  
  .btn-secondary {
    @apply bg-gray-600 text-white hover:bg-gray-700 active:scale-95;
  }
  
  .card {
    @apply bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6;
  }
}`
		if err := WriteFile(filepath.Join(config.Directory, "styles/input.css"), inputCSS); err != nil {
			return err
		}
	}

	// Create package.json only for npm strategy and when it doesn't exist.
	if strings.ToLower(config.TailwindStrategy) == "npm" {
		if _, err := os.Stat(filepath.Join(config.Directory, "package.json")); os.IsNotExist(err) {
			packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "build-css": "tailwindcss -i ./styles/input.css -o ./public/styles.css --minify",
    "watch-css": "tailwindcss -i ./styles/input.css -o ./public/styles.css --watch"
  },
  "devDependencies": {
    "tailwindcss": "^3.4.0"
  }
}`, config.Name)
			return WriteFile(filepath.Join(config.Directory, "package.json"), packageJSON)
		}
	}
	return nil
}

// createDefaultLayout creates a minimal SSR layout
func createDefaultLayout(config *ProjectConfig) error {
	content := `package routes

import (
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/vex/functional"
)

// Layout wraps route content with a simple HTML shell
func Layout(content *vdom.VNode) *vdom.VNode {
    return functional.Div(functional.MergeProps(
        functional.Class("min-h-screen bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"),
    ),
        functional.Div(functional.MergeProps(
            functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6"),
        ),
            content,
        ),
    )
}`

	return WriteFile(filepath.Join(config.Directory, "app/routes/_layout.go"), content)
}

// createErrorPages creates default 404 and 500 error pages
func createErrorPages(config *ProjectConfig) error {
	// 404 page
	notFound := `package routes

import (
    "github.com/recera/vango/pkg/server"
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/vex/functional"
)

// Page renders the default 404 page
func Page(ctx server.Ctx) (*vdom.VNode, error) {
    return functional.Div(functional.MergeProps(
        functional.Class("min-h-screen bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"),
    ),
        functional.Div(functional.MergeProps(
            functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6"),
        ),
            functional.Div(functional.MergeProps(
                functional.Class("min-h-[60vh] flex items-center justify-center"),
            ),
                functional.Div(functional.MergeProps(
                    functional.Class("text-center"),
                ),
                    functional.H1(functional.MergeProps(
                        functional.Class("text-5xl font-bold mb-4"),
                    ), functional.Text("404")),
                    functional.P(nil, functional.Text("Page not found")),
                ),
            ),
        ),
    ), nil
}`

	if err := WriteFile(filepath.Join(config.Directory, "app/routes/_404.go"), notFound); err != nil {
		return err
	}

	// 500 page
	internal := `package routes

import (
    "github.com/recera/vango/pkg/server"
    "github.com/recera/vango/pkg/vango/vdom"
    "github.com/recera/vango/pkg/vex/functional"
)

// Page renders the default 500 page
func Page(ctx server.Ctx) (*vdom.VNode, error) {
    return functional.Div(functional.MergeProps(
        functional.Class("min-h-screen bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"),
    ),
        functional.Div(functional.MergeProps(
            functional.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6"),
        ),
            functional.Div(functional.MergeProps(
                functional.Class("min-h-[60vh] flex items-center justify-center"),
            ),
                functional.Div(functional.MergeProps(
                    functional.Class("text-center"),
                ),
                    functional.H1(functional.MergeProps(
                        functional.Class("text-5xl font-bold mb-4"),
                    ), functional.Text("500")),
                    functional.P(nil, functional.Text("Internal Server Error")),
                ),
            ),
        ),
    ), nil
}`

	return WriteFile(filepath.Join(config.Directory, "app/routes/_500.go"), internal)
}
