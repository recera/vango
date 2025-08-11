package routes

import (
	"github.com/recera/vango/pkg/vango/vdom"
	"github.com/recera/vango/pkg/vex/builder"
)

// CounterPage demonstrates an interactive counter component
func CounterPage() *vdom.VNode {
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
						Class("max-w-md mx-auto").
						Children(
							builder.Div().
								Class("bg-white dark:bg-gray-800 rounded-lg shadow-xl p-8").
								Children(
									builder.H1().
										Class("text-3xl font-bold text-center text-gray-900 dark:text-white mb-8").
										Text("Interactive Counter").
										Build(),
									
									// Counter display
									builder.Div().
										Class("text-center mb-8").
										Children(
											builder.Div().
												ID("counter-value").
												Class("text-6xl font-bold text-blue-600 dark:text-blue-400").
												Text("0").
												Build(),
										).Build(),
									
									// Button container
									builder.Div().
										Class("flex gap-4 justify-center").
										Children(
											// Decrement button
											builder.Button().
												Class("px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors font-semibold").
												Attr("onclick", "updateCounter(-1)").
												Text("− Decrement").
												Build(),
											
											// Reset button  
											builder.Button().
												Class("px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors font-semibold").
												Attr("onclick", "updateCounter(0)").
												Text("↺ Reset").
												Build(),
											
											// Increment button
											builder.Button().
												Class("px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600 transition-colors font-semibold").
												Attr("onclick", "updateCounter(1)").
												Text("+ Increment").
												Build(),
										).Build(),
									
									// Info box
									builder.Div().
										Class("mt-8 p-4 bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 rounded").
										Children(
											builder.P().
												Class("text-sm text-blue-800 dark:text-blue-200").
												Text("This counter demonstrates client-side interactivity in Vango. The state is managed locally in the WASM application.").
												Build(),
										).Build(),
								).Build(),
						).Build(),
				).Build(),
		).Build()
}