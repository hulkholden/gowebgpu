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
	maxFreeIDsCount = numParticles

	initialVelScale = 100.0

	shipFireCooldown = 3.0

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

	time   float32
	deltaT float32

	avoidDistance float32
	cMassDistance float32
	cVelDistance  float32
	cMassScale    float32
	avoidScale    float32
	cVelScale     float32

	maxMissileAge        float32
	missileCollisionDist float32

	// boundaryBounceFactor is the velocity preserved after colliding with the boundary.
	boundaryBounceFactor float32

	maxShipSpeed     float32
	shipFireCooldown float32

	maxMissileSpeed  float32
	maxMissileAcc    float32
	maxMissileAngAcc float32

	// TODO: need to ensure struct is multiple of alignment size (8 for V2).
	// pad uint32
}

const kParticleFlagHit = 1

type Body struct {
	pos        vmath.V2
	vel        vmath.V2
	angle      float32
	angularVel float32
}

type Particle struct {
	metadata uint32
	flags    uint32
	col      uint32
	debugVal float32
}

type Ship struct {
	// TODO: store time of next shot rather than tracking the cooldown (which needs updating each frame).
	cooldown float32
	pad      uint32
}

type Missile struct {
	// TODO: compress these down. Use 16 bits for each?
	targetIdx int32
	age       float32
}

func (p Particle) BodyType() BodyType {
	return BodyType((p.metadata >> 8) & 0xff)
}

func (p Particle) Team() Team {
	return Team(p.metadata & 0xff)
}

type Acceleration struct {
	linearAcc  vmath.V2
	angularAcc float32
	pad        uint32
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

type FreeIDsContainer struct {
	count    uint32 `atomic:"true"`
	pad      uint32
	elements [maxFreeIDsCount]uint32 `runtimeArray:"true"`
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
	bodyStruct              = wgsltypes.MustRegisterStruct[Body]()
	particleStruct          = wgsltypes.MustRegisterStruct[Particle]()
	shipStruct              = wgsltypes.MustRegisterStruct[Ship]()
	missileStruct           = wgsltypes.MustRegisterStruct[Missile]()
	accelerationStruct      = wgsltypes.MustRegisterStruct[Acceleration]()
	vertexStruct            = wgsltypes.MustRegisterStruct[Vertex]()
	contactStruct           = wgsltypes.MustRegisterStruct[Contact]()
	contactsContainerStruct = wgsltypes.MustRegisterStruct[ContactsContainer]()
	freeIDsContainerStruct  = wgsltypes.MustRegisterStruct[FreeIDsContainer]()
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

		maxMissileAge:        10.0,
		missileCollisionDist: 10.0,

		maxShipSpeed:     100.0,
		shipFireCooldown: shipFireCooldown,

		maxMissileSpeed:  150.0,
		maxMissileAcc:    150.0,
		maxMissileAngAcc: 16.0,

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

	bodyData, particleData, shipData, missileData := initParticleData(numParticles, simParams)
	particleBufferOpts := []engine.BufferOption{engine.WithVertexUsage()}
	if enableDebugBuffer {
		particleBufferOpts = append(particleBufferOpts, engine.WithCopySrcUsage())
	}
	bodyBuffer := engine.InitStorageBufferSlice(device, bodyData, particleBufferOpts...)
	particleBuffer := engine.InitStorageBufferSlice(device, particleData, particleBufferOpts...)
	shipsBuffer := engine.InitStorageBufferSlice(device, shipData, particleBufferOpts...)
	missilesBuffer := engine.InitStorageBufferSlice(device, missileData, particleBufferOpts...)
	accelerationsBuffer := engine.InitStorageBufferSlice(device, make([]Acceleration, numParticles))
	contactsBuffer := engine.InitStorageBufferStruct(device, ContactsContainer{}, engine.WithCopyDstUsage(), engine.WithCopySrcUsage())
	freeIDsBuffer := engine.InitStorageBufferStruct(device, FreeIDsContainer{}, engine.WithCopyDstUsage(), engine.WithCopySrcUsage())

	// TODO: Figure out a nice way to retreive these from VertexBuffers.
	const bodyBufferIdx = 0
	const particleBufferIdx = 1
	const vertexBufferIdx = 2

	bufDefs := []engine.BufferDescriptor{
		{Struct: &bodyStruct, Instanced: true},
		{Struct: &particleStruct, Instanced: true},
		{Struct: &vertexStruct},
	}
	vtxAttrs := []engine.VertexAttribute{
		{BufferIndex: bodyBufferIdx, FieldName: "pos"},
		{BufferIndex: bodyBufferIdx, FieldName: "angle"},
		{BufferIndex: particleBufferIdx, FieldName: "metadata"},
		{BufferIndex: particleBufferIdx, FieldName: "col"},
		{BufferIndex: vertexBufferIdx, FieldName: "pos"},
	}
	vertexBuffers := engine.NewVertexBuffers(bufDefs, vtxAttrs)
	// TODO: pass into constructor?
	vertexBuffers.Buffers[bodyBufferIdx] = bodyBuffer.Buffer()
	vertexBuffers.Buffers[particleBufferIdx] = particleBuffer.Buffer()
	vertexBuffers.Buffers[vertexBufferIdx] = spriteVertexBuffer.Buffer()

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
		bodyStruct,
		particleStruct,
		shipStruct,
		missileStruct,
		accelerationStruct,
		contactStruct,
		contactsContainerStruct,
		freeIDsContainerStruct,
	}

	// Compute
	computeShaderModule := engine.InitShaderModule(device, computeShaderCode, structDefinitions)
	computeAccelerationPipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "computeAcceleration",
		},
	})
	applyAccelerationPipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "applyAcceleration",
		},
	})
	computeCollisionsPipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "computeCollisions",
		},
	})
	applyCollisionsPipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "applyCollisions",
		},
	})
	updateMissileLifecyclePipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "updateMissileLifecycle",
		},
	})
	spawnMissilesPipeline := device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		// Layout: "auto",
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     computeShaderModule,
			EntryPoint: "spawnMissiles",
		},
	})

	computeBindGroup := device.CreateBindGroup(wasmgpu.GPUBindGroupDescriptor{
		Layout: computeAccelerationPipeline.GetBindGroupLayout(0),
		Entries: engine.MakeGPUBindingGroupEntries(
			wasmgpu.GPUBufferBinding{Buffer: simParamBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: bodyBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: particleBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: shipsBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: missilesBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: accelerationsBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: contactsBuffer.Buffer()},
			wasmgpu.GPUBufferBinding{Buffer: freeIDsBuffer.Buffer()},
		),
	})

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

	update := func() {
		renderPassDescriptor.ColorAttachments[0].View = context.GetCurrentTexture().CreateView()
		commandEncoder := device.CreateCommandEncoder()

		simParams.time += simParams.deltaT
		simParamBuffer.UpdateBufferStruct(simParams)

		commandEncoder.ClearBuffer(contactsBuffer.Buffer(), 0, contactsBuffer.BufferSize())

		particleWorkgroups := (numParticles + 63) / 64
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(computeAccelerationPipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(particleWorkgroups), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(applyAccelerationPipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(particleWorkgroups), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(computeCollisionsPipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(particleWorkgroups), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(applyCollisionsPipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			// Runs with single worker.
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(1), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(updateMissileLifecyclePipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(particleWorkgroups), 0, 0)
			passEncoder.End()
		}
		{
			passEncoder := commandEncoder.BeginComputePass(opt.V(computePassDescriptor))
			passEncoder.SetPipeline(spawnMissilesPipeline)
			passEncoder.SetBindGroup(0, computeBindGroup, nil)
			passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(particleWorkgroups), 0, 0)
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
			commandEncoder.CopyBufferToBuffer(particleBuffer.Buffer(), 0, debugBuffer.Buffer(), 0, debugBuffer.BufferSize())
		}

		device.Queue().Submit([]wasmgpu.GPUCommandBuffer{
			commandEncoder.Finish(),
		})

		if enableDebugBuffer {
			debugBuffer.ReadAsync(func(particles []Particle) {
				fmt.Printf("Paticles[1]]: %+v\n", particles[1])
			})
		}
	}

	engine.InitRenderCallback(update)
	return nil
}

func initParticleData(n int, params SimParams) ([]Body, []Particle, []Ship, []Missile) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	type particleChoice struct {
		bodyType BodyType
		team     Team
	}

	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(particleChoice{BodyTypeShip, 0}, 100),
		weightedrand.NewChoice(particleChoice{BodyTypeShip, 1}, 100),
		weightedrand.NewChoice(particleChoice{BodyTypeMissile, 0}, 5),
		weightedrand.NewChoice(particleChoice{BodyTypeMissile, 1}, 5),
		// TODO: figure out a better way to represent anti-missile missiles.
		// weightedrand.NewChoice(particleChoice{BodyTypeMissile, 0}, 1),
	)

	bs := make([]Body, n)
	ps := make([]Particle, n)
	ss := make([]Ship, n)
	ms := make([]Missile, n)
	for i := 0; i < n; i++ {
		bs[i].pos = randomLocation(r, params)
		bs[i].vel = randomVelocity(r)
		bs[i].angle = 2 * (rand.Float32() - 0.5) * 3.141
		bs[i].angularVel = (rand.Float32() - 0.5) * 1

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

		ps[i].metadata = makeMeta(choice.bodyType, choice.team)
		ps[i].col = uint32(choice.team.Color())
		ss[i].cooldown = rand.Float32() * shipFireCooldown
		ms[i].targetIdx = -1
	}
	return bs, ps, ss, ms
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
