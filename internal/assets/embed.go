package assets

import _ "embed"

//go:embed bootstrap.js
var BootstrapJS []byte

//go:embed wasm_exec.js
var WasmExecJS []byte
