package static

import "embed"

//go:embed *.css *.gz *.js *.map *.scss *.svg *.wasm shaders
var FS embed.FS
