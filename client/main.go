package main

import (
	"log"
	"syscall/js"
	"time"

	"github.com/hulkholden/gowebgpu/client/examples/battle"
	"github.com/hulkholden/gowebgpu/client/examples/boids"
	"github.com/mokiat/wasmgpu"
)

type runFunc func(device wasmgpu.GPUDevice, context wasmgpu.GPUCanvasContext) error

var examples = map[string]runFunc{
	"battle": battle.Run,
	"boids":  boids.Run,
}

// waitForExports waits until the JS which initializes the globals has finished running.
func waitForExports() {
	for {
		if fn := js.Global().Get("getContext"); !fn.IsUndefined() {
			return
		}
		log.Printf("getContext is still undefined")
		// TODO: slower backoff.
		time.Sleep(1 * time.Second)
	}
}

func main() {
	log.Println("Started client!")

	waitForExports()

	jsContext := js.Global().Call("getContext")
	jsDevice := js.Global().Call("getDevice")
	context := wasmgpu.NewCanvasContext(jsContext)
	device := wasmgpu.NewDevice(jsDevice)

	const defaultExample = "battle"
	example := defaultExample
	if jsExample := js.Global().Call("getExample"); !jsExample.IsNull() {
		example = jsExample.String()
	}
	run, ok := examples[example]
	if !ok {
		run = examples[defaultExample]
	}
	err := run(device, context)
	if err != nil {
		log.Printf("runRender() failed: %v", err)
		if fn := js.Global().Get("showError"); !fn.IsUndefined() {
			fn.Invoke("Run error: " + err.Error())
		}
	}

	<-make(chan bool)
}
