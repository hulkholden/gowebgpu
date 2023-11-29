package battle

import (
	"math/rand"

	"github.com/hulkholden/gowebgpu/client/engine"
	"github.com/hulkholden/gowebgpu/common/vmath"
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"

	_ "embed"
)

const (
	numParticles = 20000
	numTeams     = 3
)

type ARGB uint32

const (
	NiceRed    ARGB = 0xfffc0335
	NiceBlue   ARGB = 0xff035efc
	NiceOrange ARGB = 0xfffc8803
	NicePurple ARGB = 0xff891cb8

	Magenta = 0xffff00ff
)

type SimParams struct {
	deltaT        float32
	avoidDistance float32
	cMassDistance float32
	cVelDistance  float32
	cMassScale    float32
	avoidScale    float32
	cVelScale     float32
}

type Particle struct {
	pos  vmath.V2
	vel  vmath.V2
	col  uint32
	team uint32
}

type Vertex struct {
	pos vmath.V2
}

var (
	simParamsStruct = wgsltypes.MustNewStruct[SimParams]("SimParams")
	particleStruct  = wgsltypes.MustNewStruct[Particle]("Particle")
	vertexStruct    = wgsltypes.MustNewStruct[Vertex]("Vertex")
)

//go:embed compute.wgsl
var computeShaderCode string

//go:embed render.wgsl
var renderShaderCode string

// https://webgpu.github.io/webgpu-samples/samples/computeBoids
func Run(device wasmgpu.GPUDevice, context wasmgpu.GPUCanvasContext) error {
	simParams := SimParams{
		deltaT:        0.04,
		avoidDistance: 0.025,
		cMassDistance: 0.1,
		cVelDistance:  0.025,
		cMassScale:    0.02,
		avoidScale:    0.05,
		cVelScale:     0.005,
	}
	simParamBuffer := engine.InitUniformBuffer(device, simParams)
	// TODO: add sim params to GUI.

	const boidScale = 0.5
	vertexBufferData := []float32{
		-0.01 * boidScale, -0.02 * boidScale,
		0.01 * boidScale, -0.02 * boidScale,
		0.0 * boidScale, 0.02 * boidScale,
	}
	spriteVertexBuffer := engine.InitStorageBuffer(device, vertexBufferData)

	initialParticleData := initParticleData(numParticles)
	particleBuffers := []engine.StorageBuffer{
		engine.InitStorageBuffer(device, initialParticleData),
		engine.InitStorageBuffer(device, initialParticleData),
	}

	// TODO: Figure out a nice way to retreive these from VertexBuffers.
	const particleBufferIdx = 0
	const vertexBufferIdx = 1

	bufDefs := []engine.BufferDescriptor{
		{Struct: &particleStruct, Instanced: true},
		{Struct: &vertexStruct},
	}
	vtxAttrs := []engine.VertexAttribute{
		{BufferIndex: particleBufferIdx, FieldName: "pos"},
		{BufferIndex: particleBufferIdx, FieldName: "vel"},
		{BufferIndex: particleBufferIdx, FieldName: "col"},
		{BufferIndex: vertexBufferIdx, FieldName: "pos"},
	}
	vertexBuffers := engine.NewVertexBuffers(bufDefs, vtxAttrs)

	spriteShaderModule := engine.InitShaderModule(device, renderShaderCode, nil)
	renderPipelineDescriptor := wasmgpu.GPURenderPipelineDescriptor{
		// Layout: "auto",
		Vertex: wasmgpu.GPUVertexState{
			Module:     spriteShaderModule,
			EntryPoint: "vertex_main",
			Buffers:    vertexBuffers.Layout,
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

	structDefinitions := []wgsltypes.Struct{
		simParamsStruct,
		particleStruct,
	}

	// Compute
	updateSpritesShaderModule := engine.InitShaderModule(device, computeShaderCode, structDefinitions)
	computePipelineDescriptor := wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     updateSpritesShaderModule,
			EntryPoint: "main",
		},
	}
	computePipeline := device.CreateComputePipeline(computePipelineDescriptor)

	particleBindGroups := make([]wasmgpu.GPUBindGroup, 2)
	for i := 0; i < 2; i++ {
		particleBindGroups[i] = device.CreateBindGroup(wasmgpu.GPUBindGroupDescriptor{
			Layout: computePipeline.GetBindGroupLayout(0),
			Entries: engine.MakeGPUBindingGroupEntries(
				wasmgpu.GPUBufferBinding{Buffer: simParamBuffer.Buffer()},
				wasmgpu.GPUBufferBinding{Buffer: particleBuffers[i].Buffer()},
				wasmgpu.GPUBufferBinding{Buffer: particleBuffers[(i+1)%2].Buffer()},
			),
		})
	}

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

	t := 0
	update := func() {
		renderPassDescriptor.ColorAttachments[0].View = context.GetCurrentTexture().CreateView()
		commandEncoder := device.CreateCommandEncoder()

		// Flip the buffer used for rendering.
		vertexBuffers.Buffers[particleBufferIdx] = particleBuffers[(t+1)%2].Buffer()
		vertexBuffers.Buffers[vertexBufferIdx] = spriteVertexBuffer.Buffer()

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
			vertexBuffers.Bind(passEncoder)
			passEncoder.Draw(3, opt.V(wasmgpu.GPUSize32(numParticles)), opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32]())
			passEncoder.End()
		}

		device.Queue().Submit([]wasmgpu.GPUCommandBuffer{
			commandEncoder.Finish(),
		})

		t++
	}

	engine.InitRenderCallback(update)
	return nil
}

func initParticleData(n int) []Particle {
	data := make([]Particle, n)
	for i := 0; i < n; i++ {
		data[i].pos.X = 2 * (rand.Float32() - 0.5)
		data[i].pos.Y = 2 * (rand.Float32() - 0.5)
		data[i].vel.X = 2 * (rand.Float32() - 0.5) * 0.1
		data[i].vel.Y = 2 * (rand.Float32() - 0.5) * 0.1

		data[i].team = rand.Uint32() % numTeams
		switch data[i].team {
		case 0:
			data[i].col = uint32(NiceRed)
		case 1:
			data[i].col = uint32(NicePurple)
		case 2:
			data[i].col = uint32(NiceBlue)
		case 3:
			data[i].col = uint32(NiceOrange)
		default:
			data[i].col = Magenta
		}
	}
	return data
}
