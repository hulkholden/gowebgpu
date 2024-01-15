package engine

import (
	"fmt"

	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

var vertexFormatTypeMap = map[wgsltypes.TypeName]wasmgpu.GPUVertexFormat{
	"f32":       wasmgpu.GPUVertexFormatFloat32,
	"i32":       wasmgpu.GPUVertexFormatSint32,
	"u32":       wasmgpu.GPUVertexFormatUint32,
	"vec2<f32>": wasmgpu.GPUVertexFormatFloat32x2,
	"vec3<f32>": wasmgpu.GPUVertexFormatFloat32x3,
	"vec4<f32>": wasmgpu.GPUVertexFormatFloat32x4,
}

func makeGPUVertexAttribute(shaderLocation int, s wgsltypes.Struct, fieldName string) wasmgpu.GPUVertexAttribute {
	field, ok := s.FieldMap[fieldName]
	if !ok {
		panic(fmt.Sprintf("field %s.%s does not exist", s.GoName, fieldName))
	}
	return wasmgpu.GPUVertexAttribute{
		ShaderLocation: wasmgpu.GPUIndex32(shaderLocation),
		Format:         mustFormatFromFieldType(field.WGSLType.Name),
		Offset:         wasmgpu.GPUSize64(s.MustOffsetOf(fieldName)),
	}
}

func mustFormatFromFieldType(fieldType wgsltypes.TypeName) wasmgpu.GPUVertexFormat {
	format, ok := vertexFormatTypeMap[fieldType]
	if !ok {
		panic("unhandled wgsltype: " + fieldType)
	}
	return format
}

type BufferDescriptor struct {
	Struct *wgsltypes.Struct
	// Instanced specifices whether the buffer is stepped as a vertex or instance buffer.
	Instanced bool
}

type VertexAttribute struct {
	BufferIndex int
	FieldName   string
}

type VertexBuffers struct {
	Layout  []wasmgpu.GPUVertexBufferLayout
	Buffers []wasmgpu.GPUBuffer
}

func NewVertexBuffers(bufDefs []BufferDescriptor, vtxAttrs []VertexAttribute) *VertexBuffers {
	result := make([]wasmgpu.GPUVertexBufferLayout, len(bufDefs))
	for idx, bd := range bufDefs {
		stepMode := wasmgpu.GPUVertexStepModeVertex
		if bd.Instanced {
			stepMode = wasmgpu.GPUVertexStepModeInstance
		}
		result[idx] = wasmgpu.GPUVertexBufferLayout{
			ArrayStride: wasmgpu.GPUSize64(bd.Struct.Size),
			StepMode:    opt.V(stepMode),
		}
	}

	for idx, a := range vtxAttrs {
		if a.BufferIndex >= len(result) {
			panic("buffer index out of bounds")
		}
		attribute := makeGPUVertexAttribute(idx, *bufDefs[a.BufferIndex].Struct, a.FieldName)
		result[a.BufferIndex].Attributes = append(result[a.BufferIndex].Attributes, attribute)
	}

	return &VertexBuffers{
		Layout:  result,
		Buffers: make([]wasmgpu.GPUBuffer, len(result)),
	}
}

func (v *VertexBuffers) Bind(passEncoder wasmgpu.GPURenderPassEncoder) {
	unspecified := opt.Unspecified[wasmgpu.GPUSize64]()
	for idx, buffer := range v.Buffers {
		passEncoder.SetVertexBuffer(wasmgpu.GPUIndex32(idx), buffer, unspecified, unspecified)
	}
}
