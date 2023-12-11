package battle

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hulkholden/gowebgpu/client/engine"
	"github.com/hulkholden/gowebgpu/common/vmath"
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
	"github.com/mroth/weightedrand/v2"

	_ "embed"
)

const (
	numParticles = 500

	maxContactCount = 1024

	initialVelScale = 100.0

	enableDebugBuffer = false
)

type ARGB uint32

const (
	NiceRed    ARGB = 0xfffc0335
	NiceBlue   ARGB = 0xff035efc
	NiceOrange ARGB = 0xfffc8803
	NicePurple ARGB = 0xff891cb8
	Magenta    ARGB = 0xffff00ff
)

type SimParams struct {
	minBound vmath.V2
	maxBound vmath.V2

	deltaT        float32
	avoidDistance float32
	cMassDistance float32
	cVelDistance  float32
	cMassScale    float32
	avoidScale    float32
	cVelScale     float32

	maxMissileAge float32

	// boundaryBounceFactor is the velocity preserved after colliding with the boundary.
	boundaryBounceFactor float32

	maxSpeed  float32
	maxAcc    float32
	maxAngAcc float32

	// TODO: need to ensure struct is multiple of alignment size (8 for V2).
	// pad uint32
}

type Particle struct {
	pos        vmath.V2
	vel        vmath.V2
	angle      float32
	angularVel float32
	col        uint32
	metadata   uint32

	// TODO: compress these down. Pack age in the metadata word?
	targetIdx int32
	age       float32

	debugVal float32
	pad      float32
}

func (p Particle) BodyType() BodyType {
	return BodyType((p.metadata >> 8) & 0xff)
}

func (p Particle) Team() Team {
	return Team(p.metadata & 0xff)
}

type Contact struct {
	aIdx uint32
	bIdx uint32
}

type ContactsContainer struct {
	count    uint32 `atomic:"true"`
	pad      uint32
	elements [maxContactCount]Contact `runtimeArray:"true"`
}

type Team uint8

var teamColMap = map[Team]ARGB{
	0: NiceRed,
	1: NiceBlue,
	2: NicePurple,
	3: NiceOrange,
}

func (t Team) Color() ARGB {
	if col, ok := teamColMap[t]; ok {
		return col
	}
	return Magenta
}

type BodyType uint8

// TODO: need a way to expose this to the shader.
const (
	BodyTypeNone BodyType = iota
	BodyTypeShip
	BodyTypeMissile
)

func makeMeta(bodyType BodyType, team Team) uint32 {
	return uint32(bodyType)<<8 | uint32(team)
}

type Vertex struct {
	pos vmath.V2
}

var (
	simParamsStruct         = wgsltypes.MustRegisterStruct[SimParams]()
	particleStruct          = wgsltypes.MustRegisterStruct[Particle]()
	vertexStruct            = wgsltypes.MustRegisterStruct[Vertex]()
	contactStruct           = wgsltypes.MustRegisterStruct[Contact]()
	contactsContainerStruct = wgsltypes.MustRegisterStruct[ContactsContainer]()
)

//go:embed compute.wgsl
var computeShaderCode string

//go:embed render.wgsl
var renderShaderCode string

// https://webgpu.github.io/webgpu-samples/samples/computeBoids
func Run(device wasmgpu.GPUDevice, context wasmgpu.GPUCanvasContext) error {
	simParams := SimParams{
		// TODO: get from the canvas
		minBound: vmath.NewV2(-1000, -1000),
		maxBound: vmath.NewV2(+1000, +1000),

		deltaT: 1 / 50.0,

		avoidDistance: 25.0,
		cMassDistance: 100,
		cVelDistance:  25.0,

		cMassScale: 0.02,
		avoidScale: 0.05,
		cVelScale:  0.005,

		maxMissileAge: 10.0,

		maxSpeed:  100.0,
		maxAcc:    150.0,
		maxAngAcc: 16.0,

		boundaryBounceFactor: 0.95,
	}
	simParamBuffer := engine.InitUniformBuffer(device, simParams, engine.WithCopyDstUsage())
	// TODO: add sim params to GUI.

	const boidScale = 500
	vertexBufferData := []float32{
		-0.01 * boidScale, -0.02 * boidScale,
		+0.01 * boidScale, -0.02 * boidScale,
		+0.00 * boidScale, +0.02 * boidScale,
	}
	spriteVertexBuffer := engine.InitStorageBufferSlice(device, vertexBufferData, engine.WithVertexUsage())

	initialParticleData := initParticleData(numParticles, simParams)
	particleBufferOpts := []engine.BufferOption{
		engine.WithVertexUsage(),
	}
	if enableDebugBuffer {
		particleBufferOpts = append(particleBufferOpts, engine.WithCopySrcUsage())
	}
	particleBuffers := []engine.GPUBuffer{
		engine.InitStorageBufferSlice(device, initialParticleData, particleBufferOpts...),
		engine.InitStorageBufferSlice(device, initialParticleData, particleBufferOpts...),
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
		{BufferIndex: particleBufferIdx, FieldName: "angle"},
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
		contactStruct,
		contactsContainerStruct,
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

	var debugBuffer engine.DebugBuffer[Particle]
	if enableDebugBuffer {
		debugData := make([]Particle, 2)
		debugBuffer = engine.InitDebugBuffer(device, debugData)
	}

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
		if enableDebugBuffer {
			commandEncoder.CopyBufferToBuffer(particleBuffers[(t+1)%2].Buffer(), 0, debugBuffer.Buffer(), 0, debugBuffer.BufferSize())
		}

		device.Queue().Submit([]wasmgpu.GPUCommandBuffer{
			commandEncoder.Finish(),
		})

		if enableDebugBuffer {
			debugBuffer.ReadAsync(func(particles []Particle) {
				fmt.Printf("Paticles[1]]: %+v\n", particles[1])
			})
		}

		t++
	}

	engine.InitRenderCallback(update)
	return nil
}

func initParticleData(n int, params SimParams) []Particle {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	type particleChoice struct {
		bodyType BodyType
		team     Team
	}

	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(particleChoice{BodyTypeShip, 0}, 100),
		weightedrand.NewChoice(particleChoice{BodyTypeMissile, 1}, 5),
		weightedrand.NewChoice(particleChoice{BodyTypeMissile, 2}, 1), // AntiMissile
	)

	data := make([]Particle, n)
	for i := 0; i < n; i++ {
		data[i].pos = randomLocation(r, params)
		data[i].vel = randomVelocity(r)
		data[i].angle = 2 * (rand.Float32() - 0.5) * 3.141
		data[i].angularVel = (rand.Float32() - 0.5) * 1

		choice := chooser.Pick()

		// For debugging single missile.
		// if i == 0 {
		// 	data[0].pos = vmath.NewV2(0, 0)
		// 	choice.bodyType = BodyTypeShip
		// 	choice.team = 0
		// } else {
		// 	choice.bodyType = BodyTypeMissile
		// 	choice.team = 1
		// 	data[i].vel = vmath.NewV2(0, 0)
		// }

		data[i].metadata = makeMeta(choice.bodyType, choice.team)
		data[i].col = uint32(choice.team.Color())
		data[i].targetIdx = -1
	}
	return data
}

func randomLocation(r *rand.Rand, params SimParams) vmath.V2 {
	x := (r.Float32() * (params.maxBound.X - params.minBound.X)) + params.minBound.X
	y := (r.Float32() * (params.maxBound.Y - params.minBound.Y)) + params.minBound.Y
	return vmath.NewV2(x, y)
}

func randomVelocity(r *rand.Rand) vmath.V2 {
	x := 2 * (rand.Float32() - 0.5) * initialVelScale
	y := 2 * (rand.Float32() - 0.5) * initialVelScale
	return vmath.NewV2(x, y)
}
