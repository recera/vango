package cli_templates

//WIP
import (
	"path/filepath"
)

func init() {
	Register("todo", &TodoTemplate{})
}

// TodoTemplate generates a todo list application
type TodoTemplate struct{}

func (t *TodoTemplate) Name() string {
	return "todo"
}

func (t *TodoTemplate) Description() string {
	return "Todo list application"
}

func (t *TodoTemplate) Generate(config *ProjectConfig) error {
	// Create main.go for todo app
	if err := t.createMainFile(config); err != nil {
		return err
	}

	// Create todo route
	if err := t.createTodoRoute(config); err != nil {
		return err
	}

	return nil
}

func (t *TodoTemplate) createMainFile(config *ProjectConfig) error {
	content := `package main

import (
	"log"

	"github.com/recera/vango/pkg/vango"
)

func main() {
	app := vango.New()
	app.Title = "Todo App"
	
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}`

	return WriteFile(filepath.Join(config.Directory, "app/main.go"), content)
}

func (t *TodoTemplate) createTodoRoute(config *ProjectConfig) error {
	content := `package routes

import (
	"github.com/recera/vango/pkg/reactive"
	"github.com/recera/vango/pkg/vango"
	"github.com/recera/vango/pkg/vango/vdom"
)

type Todo struct {
	ID        int
	Text      string
	Completed bool
}

func Index() vango.Component {
	return vango.FC(func(ctx *vango.Context) *vdom.VNode {
		// State
		todos := reactive.CreateState([]Todo{})
		inputText := reactive.CreateState("")
		nextID := reactive.CreateState(1)
		
		// Add todo
		addTodo := func(e vango.Event) {
			text := inputText.Get()
			if text == "" {
				return
			}
			
			todos.Update(func(list []Todo) []Todo {
				return append(list, Todo{
					ID:        nextID.Get(),
					Text:      text,
					Completed: false,
				})
			})
			
			nextID.Update(func(id int) int { return id + 1 })
			inputText.Set("")
		}
		
		// Toggle todo
		toggleTodo := func(id int) func(vango.Event) {
			return func(e vango.Event) {
				todos.Update(func(list []Todo) []Todo {
					for i := range list {
						if list[i].ID == id {
							list[i].Completed = !list[i].Completed
						}
					}
					return list
				})
			}
		}
		
		// Delete todo
		deleteTodo := func(id int) func(vango.Event) {
			return func(e vango.Event) {
				todos.Update(func(list []Todo) []Todo {
					filtered := make([]Todo, 0, len(list))
					for _, todo := range list {
						if todo.ID != id {
							filtered = append(filtered, todo)
						}
					}
					return filtered
				})
			}
		}
		
		// Render todo items
		todoItems := make([]*vdom.VNode, 0, len(todos.Get()))
		for _, todo := range todos.Get() {
			todoClass := "todo-item"
			if todo.Completed {
				todoClass += " completed"
			}
			
			todoItems = append(todoItems, vdom.NewElement("li", vdom.Props{
				"key":   todo.ID,
				"class": todoClass,
			},
				vdom.NewElement("input", vdom.Props{
					"type":     "checkbox",
					"checked":  todo.Completed,
					"onChange": toggleTodo(todo.ID),
				}),
				vdom.NewElement("span", nil, vdom.NewText(todo.Text)),
				vdom.NewElement("button", vdom.Props{
					"onClick": deleteTodo(todo.ID),
					"class":   "delete-btn",
				}, vdom.NewText("Delete")),
			))
		}
		
		// Render
		return vdom.NewElement("div", vdom.Props{
			"class": "todo-container",
		},
			vdom.NewElement("h1", nil, vdom.NewText("Todo List")),
			vdom.NewElement("div", vdom.Props{
				"class": "todo-input",
			},
				vdom.NewElement("input", vdom.Props{
					"type":        "text",
					"value":       inputText.Get(),
					"placeholder": "What needs to be done?",
					"onInput": func(e vango.Event) {
						inputText.Set(e.Target.Value)
					},
					"onKeyDown": func(e vango.Event) {
						if e.Key == "Enter" {
							addTodo(e)
						}
					},
				}),
				vdom.NewElement("button", vdom.Props{
					"onClick": addTodo,
					"class":   "add-btn",
				}, vdom.NewText("Add")),
			),
			vdom.NewElement("ul", vdom.Props{
				"class": "todo-list",
			}, todoItems...),
		)
	})
}`

	return WriteFile(filepath.Join(config.Directory, "app/routes/index.go"), content)
}
