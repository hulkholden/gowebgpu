package engine

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/hulkholden/gowebgpu/client/browser"
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/wasmgpu"
)

func InitRenderCallback(update func()) {
	frame := js.FuncOf(func(this js.Value, args []js.Value) any {
		update()
		InitRenderCallback(update)
		return nil
	})
	browser.Window().RequestAnimationFrame(frame)
}

func LoadShaderModule(device wasmgpu.GPUDevice, url string, structs []wgsltypes.Struct) (wasmgpu.GPUShaderModule, error) {
	bytes, err := loadFile(url)
	if err != nil {
		return wasmgpu.GPUShaderModule{}, fmt.Errorf("loading shader: %v", err)
	}
	return InitShaderModule(device, string(bytes), structs), nil
}

func InitShaderModule(device wasmgpu.GPUDevice, code string, structs []wgsltypes.Struct) wasmgpu.GPUShaderModule {
	defs := make([]string, len(structs))
	for i, s := range structs {
		defs[i] = s.ToWGSL()
	}
	prologue := strings.Join(defs, "\n")

	return device.CreateShaderModule(wasmgpu.GPUShaderModuleDescriptor{
		Code: prologue + "\n" + code,
	})
}

func loadFile(url string) ([]byte, error) {
	res, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %q", res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %v", err)
	}
	return data, nil
}
