package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"syscall/js"
	"time"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

const bufferSize = 1024

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

var vertices = []float32{
	0.0, 0.6, 0, 1, 1, 0, 0, 1, -0.5, -0.6, 0, 1, 0, 1, 0, 1, 0.5, -0.6, 0, 1, 0,
	0, 1, 1,
}

func runRender(device wasmgpu.GPUDevice, context wasmgpu.GPUCanvasContext) error {
	shaderBytes, err := loadFile("/static/shaders/render.wgsl")
	if err != nil {
		return fmt.Errorf("loading shader: %v", err)
	}
	shaderModule := device.CreateShaderModule(wasmgpu.GPUShaderModuleDescriptor{
		Code: string(shaderBytes),
	})

	vertexBuffer := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:  wasmgpu.GPUSize64(len(vertices) * 4),
		Usage: wasmgpu.GPUBufferUsageFlagsVertex | wasmgpu.GPUBufferUsageFlagsCopyDst,
	})

	device.Queue().WriteBuffer(vertexBuffer, 0, asByteSlice(vertices))

	vertexBuffers := []wasmgpu.GPUVertexBufferLayout{
		{
			Attributes: []wasmgpu.GPUVertexAttribute{
				{
					ShaderLocation: 0, // position
					Offset:         0,
					Format:         wasmgpu.GPUVertexFormatFloat32x4,
				},
				{
					ShaderLocation: 1, // color
					Offset:         16,
					Format:         wasmgpu.GPUVertexFormatFloat32x4,
				},
			},
			ArrayStride: 32,
			StepMode:    opt.V(wasmgpu.GPUVertexStepModeVertex),
		},
	}

	pipelineDescriptor := wasmgpu.GPURenderPipelineDescriptor{
		// Layout: "auto",
		Vertex: wasmgpu.GPUVertexState{
			Module:     shaderModule,
			EntryPoint: "vertex_main",
			Buffers:    vertexBuffers,
		},
		Fragment: opt.V(wasmgpu.GPUFragmentState{
			Module:     shaderModule,
			EntryPoint: "fragment_main",
			Targets: []wasmgpu.GPUColorTargetState{
				{
					//Format: navigator.gpu.getPreferredCanvasFormat(),
					Format: wasmgpu.GPUTextureFormatBGRA8Unorm,
				},
			},
		}),
		Primitive: opt.V(wasmgpu.GPUPrimitiveState{
			Topology: opt.V(wasmgpu.GPUPrimitiveTopologyTriangleList),
		}),
	}
	renderPipeline := device.CreateRenderPipeline(pipelineDescriptor)

	commandEncoder := device.CreateCommandEncoder()

	renderPassDescriptor := wasmgpu.GPURenderPassDescriptor{
		ColorAttachments: []wasmgpu.GPURenderPassColorAttachment{
			{
				View: context.GetCurrentTexture().CreateView(),
				ClearValue: opt.V(wasmgpu.GPUColor{
					R: 0.0,
					G: 0.5,
					B: 1.0,
					A: 1.0,
				}),
				LoadOp:  wasmgpu.GPULoadOpClear,
				StoreOp: wasmgpu.GPUStoreOPStore,
			},
		},
	}

	renderPass := commandEncoder.BeginRenderPass(renderPassDescriptor)
	renderPass.SetPipeline(renderPipeline)
	renderPass.SetVertexBuffer(0, vertexBuffer, opt.Unspecified[wasmgpu.GPUSize64](), opt.Unspecified[wasmgpu.GPUSize64]())
	renderPass.Draw(3, opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32]())
	renderPass.End()

	device.Queue().Submit([]wasmgpu.GPUCommandBuffer{
		commandEncoder.Finish(),
	})
	return nil
}

func runCompute(device wasmgpu.GPUDevice) error {
	shaderBytes, err := loadFile("/static/shaders/compute.wgsl")
	if err != nil {
		return fmt.Errorf("loading shader: %v", err)
	}

	shaderModule := device.CreateShaderModule(wasmgpu.GPUShaderModuleDescriptor{
		Code: string(shaderBytes),
	})

	output := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:  bufferSize,
		Usage: wasmgpu.GPUBufferUsageFlagsStorage | wasmgpu.GPUBufferUsageFlagsCopySrc,
	})
	stagingBuffer := device.CreateBuffer((wasmgpu.GPUBufferDescriptor{
		Size:  bufferSize,
		Usage: wasmgpu.GPUBufferUsageFlagsMapRead | wasmgpu.GPUBufferUsageFlagsCopyDst,
	}))
	bindGroupLayout := device.CreateBindGroupLayout(wasmgpu.GPUBindGroupLayoutDescriptor{
		Entries: []wasmgpu.GPUBindGroupLayoutEntry{
			{
				Binding:    0,
				Visibility: wasmgpu.GPUShaderStageFlagsCompute,
				Buffer: opt.V(wasmgpu.GPUBufferBindingLayout{
					Type: opt.V(wasmgpu.GPUBufferBindingTypeStorage),
				}),
			},
		},
	})
	bindGroup := device.CreateBindGroup(wasmgpu.GPUBindGroupDescriptor{
		Layout: bindGroupLayout,
		Entries: []wasmgpu.GPUBindGroupEntry{
			{
				Binding: 0,
				Resource: wasmgpu.GPUBufferBinding{
					Buffer: output,
				},
			},
		},
	})
	pipelineDescriptor := wasmgpu.GPUComputePipelineDescriptor{
		Layout: opt.V(device.CreatePipelineLayout(wasmgpu.GPUPipelineLayoutDescriptor{
			BindGroupLayouts: []wasmgpu.GPUBindGroupLayout{bindGroupLayout},
		})),
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     shaderModule,
			EntryPoint: "main",
			// Doesn't seem to work: https://bugs.chromium.org/p/dawn/issues/detail?id=2255
			// Constants: opt.V(wasmgpu.GPUProgrammableStageConstants{
			// 	"contant_u32": 13,
			// }),
		},
	}
	computePipeline := device.CreateComputePipeline(pipelineDescriptor)
	passDescriptor := wasmgpu.GPUComputePassDescriptor{}
	commandEncoder := device.CreateCommandEncoder()
	passEncoder := commandEncoder.BeginComputePass(opt.V(passDescriptor))
	passEncoder.SetPipeline(computePipeline)
	passEncoder.SetBindGroup(0, bindGroup, nil)
	numFloats := bufferSize / 4
	numWorkgroups := (numFloats + 63) / 64
	passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(numWorkgroups), 0, 0)
	passEncoder.End()

	commandEncoder.CopyBufferToBuffer(output, 0, stagingBuffer, 0, bufferSize)

	device.Queue().Submit([]wasmgpu.GPUCommandBuffer{commandEncoder.Finish()})

	fmt.Printf("Calling MapAsync\n")
	promise := stagingBuffer.MapAsync(wasmgpu.GPUMapModeFlagsRead, 0, bufferSize)
	wait := make(chan any)
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		fmt.Printf("got an arg: %v\n", args[0])
		wait <- nil
		return nil
	}))
	<-wait
	fmt.Printf("MapAsync returned\n")

	ab := stagingBuffer.GetMappedRange(0, bufferSize)
	abCopy := ab.Call("slice")
	stagingBuffer.Unmap()

	u8 := js.Global().Get("Uint8Array").New(abCopy)
	bytes := make([]byte, 1024)
	numBytes := js.CopyBytesToGo(bytes, u8)
	fmt.Printf("Go got: %v, %v\n", bytes, numBytes)

	for i := 0; i+3 < len(bytes); i += 4 {
		bits := binary.LittleEndian.Uint32(bytes[i : i+4])
		f := math.Float32frombits(bits)
		fmt.Printf("%d: %f\n", i/4, f)
	}

	return nil
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

	if err := runRender(device, context); err != nil {
		log.Printf("runRender() failed: %v", err)
	}

	if err := runCompute(device); err != nil {
		log.Printf("runCompute() failed: %v", err)
	}

	exportSolve()
	<-make(chan bool)
}
