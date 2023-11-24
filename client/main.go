package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"syscall/js"
	"time"

	"github.com/hulkholden/gowebgpu/client/browser"
	"github.com/hulkholden/gowebgpu/common/vmath"
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

const (
	numParticles = 3000

	float32Size = 4

	particleSize = 4 * float32Size
	vertexSize   = 2 * float32Size
)

type SimParams struct {
	deltaT        float32
	rule1Distance float32
	rule2Distance float32
	rule3Distance float32
	rule1Scale    float32
	rule2Scale    float32
	rule3Scale    float32
}

type Vector2 struct {
	x, y float32
}

type Particle struct {
	pos vmath.V2
	vel vmath.V2
}

var (
	simParamsStruct = wgsltypes.MustNew[SimParams]("SimParams")
	particleStruct  = wgsltypes.MustNew[Particle]("Particle")
)

// https://webgpu.github.io/webgpu-samples/samples/computeBoids
func runComputeBoids(device wasmgpu.GPUDevice, context wasmgpu.GPUCanvasContext) error {
	spriteShaderModule, err := loadShaderModule(device, "/static/shaders/render.wgsl", nil)
	if err != nil {
		return fmt.Errorf("loading shader: %v", err)
	}

	renderPipelineDescriptor := wasmgpu.GPURenderPipelineDescriptor{
		// Layout: "auto",
		Vertex: wasmgpu.GPUVertexState{
			Module:     spriteShaderModule,
			EntryPoint: "vertex_main",
			Buffers: []wasmgpu.GPUVertexBufferLayout{
				{
					// instanced particles buffer
					ArrayStride: particleSize,
					StepMode:    opt.V(wasmgpu.GPUVertexStepModeInstance),
					Attributes: []wasmgpu.GPUVertexAttribute{
						makeGPUVertexAttribute(0, wasmgpu.GPUVertexFormatFloat32x2, 0),             // position
						makeGPUVertexAttribute(1, wasmgpu.GPUVertexFormatFloat32x2, 2*float32Size), // velocity
					},
				},
				{
					// vertex buffer
					ArrayStride: vertexSize,
					StepMode:    opt.V(wasmgpu.GPUVertexStepModeVertex),
					Attributes: []wasmgpu.GPUVertexAttribute{
						makeGPUVertexAttribute(2, wasmgpu.GPUVertexFormatFloat32x2, 0), // position
					},
				},
			},
		},
		Fragment: opt.V(wasmgpu.GPUFragmentState{
			Module:     spriteShaderModule,
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
	renderPipeline := device.CreateRenderPipeline(renderPipelineDescriptor)

	simParams := SimParams{
		deltaT:        0.04,
		rule1Distance: 0.1,
		rule2Distance: 0.025,
		rule3Distance: 0.025,
		rule1Scale:    0.02,
		rule2Scale:    0.05,
		rule3Scale:    0.005,
	}
	structDefinitions := []wgsltypes.Struct{
		simParamsStruct,
		particleStruct,
	}

	// Compute
	updateSpritesShaderModule, err := loadShaderModule(device, "/static/shaders/compute.wgsl", structDefinitions)
	if err != nil {
		return fmt.Errorf("loading shader: %v", err)
	}

	computePipelineDescriptor := wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     updateSpritesShaderModule,
			EntryPoint: "main",
			// Doesn't seem to work: https://bugs.chromium.org/p/dawn/issues/detail?id=2255
			// Constants: opt.V(wasmgpu.GPUProgrammableStageConstants{
			// 	"contant_u32": 13,
			// }),
		},
	}
	computePipeline := device.CreateComputePipeline(computePipelineDescriptor)

	renderPassDescriptor := wasmgpu.GPURenderPassDescriptor{
		ColorAttachments: []wasmgpu.GPURenderPassColorAttachment{
			{
				View:       context.GetCurrentTexture().CreateView(),
				ClearValue: opt.V(wasmgpu.GPUColor{R: 0.0, G: 0.0, B: 0.0, A: 1.0}),
				LoadOp:     wasmgpu.GPULoadOpClear,
				StoreOp:    wasmgpu.GPUStoreOPStore,
			},
		},
	}

	computePassDescriptor := wasmgpu.GPUComputePassDescriptor{}

	vertexBufferData := []float32{
		-0.01, -0.02, 0.01,
		-0.02, 0.0, 0.02,
	}

	spriteVertexBuffer := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(len(vertexBufferData) * float32Size),
		Usage:            wasmgpu.GPUBufferUsageFlagsVertex,
		MappedAtCreation: opt.V(true),
	})
	setFloat32Array(float32ArrayCtor.New(spriteVertexBuffer.GetMappedRange(0, 0)), vertexBufferData)
	spriteVertexBuffer.Unmap()

	simParamBuffer := initUniformBuffer(device, simParams)

	// TODO: add sim params to GUI.

	initialParticleData := initParticleData(numParticles)

	particleBuffers := make([]wasmgpu.GPUBuffer, 2)
	for i := 0; i < 2; i++ {
		particleBuffers[i] = device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
			Size:             wasmgpu.GPUSize64(len(initialParticleData) * float32Size),
			Usage:            wasmgpu.GPUBufferUsageFlagsVertex | wasmgpu.GPUBufferUsageFlagsStorage,
			MappedAtCreation: opt.V(true),
		})
		setFloat32Array(float32ArrayCtor.New(particleBuffers[i].GetMappedRange(0, 0)), initialParticleData)
		particleBuffers[i].Unmap()
	}

	particleBindGroups := make([]wasmgpu.GPUBindGroup, 2)
	for i := 0; i < 2; i++ {
		particleBindGroups[i] = device.CreateBindGroup(wasmgpu.GPUBindGroupDescriptor{
			Layout: computePipeline.GetBindGroupLayout(0),
			Entries: makeGPUBindingGroupEntries(
				wasmgpu.GPUBufferBinding{Buffer: simParamBuffer.buffer},
				wasmgpu.GPUBufferBinding{Buffer: particleBuffers[i]},
				wasmgpu.GPUBufferBinding{Buffer: particleBuffers[(i+1)%2]},
			),
		})
	}

	t := 0
	update := func() {
		renderPassDescriptor.ColorAttachments[0].View = context.GetCurrentTexture().CreateView()
		commandEncoder := device.CreateCommandEncoder()

		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(computePipeline)
			passEncoder.SetBindGroup(0, particleBindGroups[t%2], nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32((numParticles+63)/64), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginRenderPass(renderPassDescriptor)
			passEncoder.SetPipeline(renderPipeline)
			passEncoder.SetVertexBuffer(0, particleBuffers[(t+1)%2], opt.Unspecified[wasmgpu.GPUSize64](), opt.Unspecified[wasmgpu.GPUSize64]())
			passEncoder.SetVertexBuffer(1, spriteVertexBuffer, opt.Unspecified[wasmgpu.GPUSize64](), opt.Unspecified[wasmgpu.GPUSize64]())
			passEncoder.Draw(3, opt.V(wasmgpu.GPUSize32(numParticles)), opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32]())
			passEncoder.End()
		}

		device.Queue().Submit([]wasmgpu.GPUCommandBuffer{
			commandEncoder.Finish(),
		})

		t++
	}

	initRenderCallback(update)
	return nil
}

func initParticleData(n int) []float32 {
	data := make([]float32, n*4)
	for i := 0; i < n; i++ {
		data[i*4+0] = 2 * (rand.Float32() - 0.5)
		data[i*4+1] = 2 * (rand.Float32() - 0.5)
		data[i*4+2] = 2 * (rand.Float32() - 0.5) * 0.1
		data[i*4+3] = 2 * (rand.Float32() - 0.5) * 0.1
	}
	return data
}

func initRenderCallback(update func()) {
	frame := js.FuncOf(func(this js.Value, args []js.Value) any {
		update()
		initRenderCallback(update)
		return nil
	})
	browser.Window().RequestAnimationFrame(frame)
}

func loadShaderModule(device wasmgpu.GPUDevice, url string, structs []wgsltypes.Struct) (wasmgpu.GPUShaderModule, error) {
	bytes, err := loadFile(url)
	if err != nil {
		return wasmgpu.GPUShaderModule{}, fmt.Errorf("loading shader: %v", err)
	}

	defs := make([]string, len(structs))
	for i, s := range structs {
		defs[i] = s.ToWGSL()
	}
	prologue := strings.Join(defs, "\n")

	return device.CreateShaderModule(wasmgpu.GPUShaderModuleDescriptor{
		Code: prologue + "\n" + string(bytes),
	}), nil
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

	if err := runComputeBoids(device, context); err != nil {
		log.Printf("runRender() failed: %v", err)
	}

	<-make(chan bool)
}
