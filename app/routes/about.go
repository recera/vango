package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// AboutPage demonstrates more complex layouts and composition
func AboutPage() *vdom.VNode {
	return builder.Div().
		Class("min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800").
		Children(
			// Navigation bar (same as index)
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
					builder.Section().
						Class("max-w-4xl mx-auto").
						Children(
							builder.H1().
								Class("text-4xl font-bold text-gray-900 dark:text-white mb-8").
								Text("About Vango").
								Build(),
							
							builder.Div().
								Class("prose dark:prose-invert max-w-none").
								Children(
									builder.P().
										Class("text-lg text-gray-600 dark:text-gray-300 mb-6").
										Text("Vango is a revolutionary Go-native frontend framework that brings the power and simplicity of Go to web development.").
										Build(),
									
									builder.H2().
										Class("text-2xl font-semibold text-gray-900 dark:text-white mt-8 mb-4").
										Text("Key Features").
										Build(),
									
									builder.Ul().
										Class("list-disc list-inside space-y-2 text-gray-600 dark:text-gray-300").
										Children(
											builder.Li().Text("⚡ Blazing-fast performance with WebAssembly").Build(),
											builder.Li().Text("🔄 Server-driven components with live updates").Build(),
											builder.Li().Text("📦 Small bundle sizes thanks to TinyGo").Build(),
											builder.Li().Text("🛠️ Type-safe from backend to frontend").Build(),
											builder.Li().Text("🎨 Multiple styling options including Tailwind CSS").Build(),
										).Build(),
									
									builder.H2().
										Class("text-2xl font-semibold text-gray-900 dark:text-white mt-8 mb-4").
										Text("Architecture").
										Build(),
									
									builder.P().
										Class("text-gray-600 dark:text-gray-300 mb-6").
										Text("Vango uses a unique hybrid rendering approach that combines the best of server-side rendering (SSR) and client-side interactivity through WebAssembly.").
										Build(),
								).Build(),
						).Build(),
				).Build(),
		).Build()
}