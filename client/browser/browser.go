package browser

import "syscall/js"

type HTMLWindow struct{ jsValue js.Value }

func Window() HTMLWindow {
	return HTMLWindow{js.Global().Get("window")}
}
func (w HTMLWindow) RequestAnimationFrame(fn js.Func) { w.jsValue.Call("requestAnimationFrame", fn) }
