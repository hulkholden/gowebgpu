package static

import "embed"

//go:embed *.css *.gz *.js *.map *.scss *.svg *.wasm
var FS embed.FS
