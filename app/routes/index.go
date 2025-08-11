package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// IndexPage is the home page with navigation
func IndexPage() *vdom.VNode {
	// Using the Builder API (Layer 1 VEX)
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			// Navigation bar
			builder.Nav().
				Class("bg-white dark:bg-gray-800 shadow-md").
				Children(
					builder.Div().
						Class("container mx-auto px-6 py-4").
						Children(
							builder.Div().
								Class("flex items-center justify-between").
								Children(
									// Logo
									builder.H1().
										Class("text-2xl font-bold text-gray-900 dark:text-white").
										Text("🚀 Vango").
										Build(),
									
									// Navigation links
									builder.Div().
										Class("flex items-center space-x-6").
										Children(
											builder.A().
												Href("/").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("Home").
												Build(),
											builder.A().
												Href("/about").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("About").
												Build(),
											builder.A().
												Href("/counter").
												Class("text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors").
												Text("Counter").
												Build(),
											
											// Dark mode toggle
											builder.Button().
												Class("p-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors").
												Attr("onclick", "toggleDarkMode()").
												Title("Toggle dark mode").
												Text("🌙").
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
			
			// Main content
			builder.Main().
				Class("container mx-auto px-6 py-12").
				Children(
					// Hero section
					builder.Section().
						Class("text-center mb-16").
						Children(
							builder.H1().
								Class("text-5xl font-bold text-gray-900 dark:text-white mb-6").
								Text("Welcome to Vango").
								Build(),
							builder.P().
								Class("text-xl text-gray-600 dark:text-gray-300 max-w-3xl mx-auto mb-8").
								Text("Build blazing-fast web applications with Go and WebAssembly. Experience the power of server-driven components and reactive state management.").
								Build(),
							
							// CTA buttons
							builder.Div().
								Class("flex justify-center space-x-4").
								Children(
									builder.Button().
										Class("px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium shadow-lg").
										Attr("onclick", "navigateTo('/about')").
										Text("Learn More").
										Build(),
									builder.A().
										Href("https://github.com/recera/vango").
										Target("_blank").
										Class("px-6 py-3 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors font-medium shadow-lg").
										Text("View on GitHub").
										Build(),
								).Build(),
						).Build(),
				).Build(),
		).Build()
}
