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
	initialShipCount = 2000
	maxParticleCount = 4000

	maxContactCount = 1024
	maxFreeIDsCount = maxParticleCount

	initialVelScale = 100.0

	shipShotCooldown = 5.0

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
	shipShotCooldown float32

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
	nextShotTime float32
	targetIdx    int32
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

var (
	bodyStruct     = wgsltypes.MustRegisterStruct[Body]()
	particleStruct = wgsltypes.MustRegisterStruct[Particle]()
	contactStruct  = wgsltypes.MustRegisterStruct[Contact]()
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
		shipShotCooldown: shipShotCooldown,

		maxMissileSpeed:  150.0,
		maxMissileAcc:    150.0,
		maxMissileAngAcc: 16.0,

		boundaryBounceFactor: 0.95,
	}
	simParamBuffer := engine.InitUniformBuffer(device, simParams, engine.WithCopyDstUsage())
	// TODO: add sim params to GUI.

	bodyData, particleData, shipData, missileData, freeIDs := initParticleData(initialShipCount, maxParticleCount, simParams)
	particleBufferOpts := []engine.BufferOption{engine.WithVertexUsage()}
	if enableDebugBuffer {
		particleBufferOpts = append(particleBufferOpts, engine.WithCopySrcUsage())
	}
	bodyBuffer := engine.InitStorageBufferSlice(device, bodyData, particleBufferOpts...)
	particleBuffer := engine.InitStorageBufferSlice(device, particleData, particleBufferOpts...)
	shipsBuffer := engine.InitStorageBufferSlice(device, shipData, particleBufferOpts...)
	missilesBuffer := engine.InitStorageBufferSlice(device, missileData, particleBufferOpts...)
	accelerationsBuffer := engine.InitStorageBufferSlice(device, make([]Acceleration, maxParticleCount))
	contactsBuffer := engine.InitStorageBufferStruct(device, ContactsContainer{}, engine.WithCopyDstUsage(), engine.WithCopySrcUsage())
	freeIDsBuffer := engine.InitStorageBufferStruct(device, freeIDs, engine.WithCopyDstUsage(), engine.WithCopySrcUsage())

	// TODO: Figure out a nice way to retreive these from VertexBuffers.
	const bodyBufferIdx = 0
	const particleBufferIdx = 1

	// TODO: invert this so we maintain a slice of buffers (like for compute) and get the structs from them.
	bufDefs := []engine.BufferDescriptor{
		{Struct: &bodyStruct, Instanced: true},
		{Struct: &particleStruct, Instanced: true},
	}
	vtxAttrs := []engine.VertexAttribute{
		{BufferIndex: bodyBufferIdx, FieldName: "pos"},
		{BufferIndex: bodyBufferIdx, FieldName: "angle"},
		{BufferIndex: particleBufferIdx, FieldName: "metadata"},
		{BufferIndex: particleBufferIdx, FieldName: "col"},
	}
	vertexBuffers := engine.NewVertexBuffers(bufDefs, vtxAttrs)
	// TODO: pass into constructor?
	vertexBuffers.Buffers[bodyBufferIdx] = bodyBuffer.Buffer()
	vertexBuffers.Buffers[particleBufferIdx] = particleBuffer.Buffer()

	spriteShaderModule := engine.InitShaderModule(device, renderShaderCode, nil)
	fragmentState := opt.V(wasmgpu.GPUFragmentState{
		Module:     spriteShaderModule,
		EntryPoint: "fragment_main",
		Targets: []wasmgpu.GPUColorTargetState{
			{
				//Format: navigator.gpu.getPreferredCanvasFormat(),
				Format: wasmgpu.GPUTextureFormatBGRA8Unorm,
			},
		},
	})
	primitiveState := opt.V(wasmgpu.GPUPrimitiveState{
		Topology: opt.V(wasmgpu.GPUPrimitiveTopologyTriangleList),
	})
	shipRenderPipeline := device.CreateRenderPipeline(wasmgpu.GPURenderPipelineDescriptor{
		Vertex: wasmgpu.GPUVertexState{
			Module:     spriteShaderModule,
			EntryPoint: "vertex_main_ship",
			Buffers:    vertexBuffers.Layout,
		},
		Fragment:  fragmentState,
		Primitive: primitiveState,
	})
	missileRenderPipeline := device.CreateRenderPipeline(wasmgpu.GPURenderPipelineDescriptor{
		Vertex: wasmgpu.GPUVertexState{
			Module:     spriteShaderModule,
			EntryPoint: "vertex_main_missile",
			Buffers:    vertexBuffers.Layout,
		},
		Fragment:  fragmentState,
		Primitive: primitiveState,
	})

	// TODO: figure out how to tie this order to the @bindings specified in the wgsl.
	buffers := []engine.ComputePassBuffer{
		simParamBuffer,
		bodyBuffer,
		particleBuffer,
		shipsBuffer,
		missilesBuffer,
		accelerationsBuffer,
		contactsBuffer,
		freeIDsBuffer,
	}
	// TODO: ideally wgsltypes should return a set of all the reachable types from the buffer.
	extraStructDefinitions := []wgsltypes.Struct{
		contactStruct,
	}

	// Compute
	cpf := engine.NewComputePassFactory(device, computeShaderCode, extraStructDefinitions, buffers)

	// TODO: this is hard-coded in the shader. Ideally should be passed in somehow.
	workgroupSize := 64
	numParticleWorkgroups := (maxParticleCount + (workgroupSize - 1)) / workgroupSize
	computePasses := []engine.ComputePass{
		cpf.InitPass("computeAcceleration", numParticleWorkgroups),
		cpf.InitPass("applyAcceleration", numParticleWorkgroups),
		cpf.InitPass("computeCollisions", numParticleWorkgroups),
		cpf.InitPass("applyCollisions", 1),
		cpf.InitPass("updateMissileLifecycle", numParticleWorkgroups),
		cpf.InitPass("selectTargets", numParticleWorkgroups),
		cpf.InitPass("spawnMissiles", numParticleWorkgroups),
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

		for _, pass := range computePasses {
			pass(commandEncoder)
		}

		{
			passEncoder := commandEncoder.BeginRenderPass(renderPassDescriptor)

			passEncoder.SetPipeline(shipRenderPipeline)
			vertexBuffers.Bind(passEncoder)
			passEncoder.Draw(3, opt.V(wasmgpu.GPUSize32(maxParticleCount)), opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32]())

			passEncoder.SetPipeline(missileRenderPipeline)
			vertexBuffers.Bind(passEncoder)
			passEncoder.Draw(9, opt.V(wasmgpu.GPUSize32(maxParticleCount)), opt.Unspecified[wasmgpu.GPUSize32](), opt.Unspecified[wasmgpu.GPUSize32]())

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

func initParticleData(numShips, maxParticles int, params SimParams) ([]Body, []Particle, []Ship, []Missile, FreeIDsContainer) {
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

	bs := make([]Body, maxParticles)
	ps := make([]Particle, maxParticles)
	ss := make([]Ship, maxParticles)
	ms := make([]Missile, maxParticles)
	fids := FreeIDsContainer{}
	for i := 0; i < numShips; i++ {
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
		ms[i].targetIdx = -1

		ss[i].nextShotTime = rand.Float32() * params.shipShotCooldown
		ss[i].targetIdx = -1
	}
	fids.count = uint32(maxParticles - numShips)
	for i := numShips; i < maxParticles; i++ {
		fids.elements[i-numShips] = uint32(i)
	}
	return bs, ps, ss, ms, fids
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
