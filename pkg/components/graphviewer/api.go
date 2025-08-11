package graphviewer

// API provides a public API to control the viewer when a ref is stored by the app.
// Users can capture the controller via a ref callback and keep it around.
type API interface {
    FitGraph(padding float64)
    Reset()
    FocusNode(id string, scale float64)
    ExportPNG() string
}

// Note: Controller implements API in WASM builds only


