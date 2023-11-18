package main

import (
	"log"
	"syscall/js"
)

func exportSolve() {
	fn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			// Log error?
			return nil
		}
		a0 := args[0].String()
		a1 := args[1].String()
		a2 := args[2].String()
		log.Printf("Solve(%q, %q, %q) = %v", a0, a1, a2)
		return 0
	})

	js.Global().Get("window").Set("solve", fn)
}

func main() {
	log.Println("Started client!")

	exportSolve()
	<-make(chan bool)
}
