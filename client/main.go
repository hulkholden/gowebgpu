package main

import (
	"log"
	"syscall/js"
	"time"

	"github.com/hulkholden/gowebgpu/client/examples/battle"
	"github.com/mokiat/wasmgpu"
)

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

	if err := battle.Run(device, context); err != nil {
		log.Printf("runRender() failed: %v", err)
	}

	<-make(chan bool)
}
