package main

import (
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

var vertexFormatTypeMap = map[wgsltypes.TypeName]wasmgpu.GPUVertexFormat{
	"f32":       wasmgpu.GPUVertexFormatFloat32,
	"vec2<f32>": wasmgpu.GPUVertexFormatFloat32x2,
	"vec3<f32>": wasmgpu.GPUVertexFormatFloat32x3,
	"vec4<f32>": wasmgpu.GPUVertexFormatFloat32x4,
}

func makeGPUVertexAttribute(shaderLocation int, s wgsltypes.Struct, fieldName string) wasmgpu.GPUVertexAttribute {
	return wasmgpu.GPUVertexAttribute{
		ShaderLocation: wasmgpu.GPUIndex32(shaderLocation),
		Format:         mustFormatFromFieldType(s.FieldMap[fieldName].WGSLType.Name),
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

func makeGPUBindingGroupEntries(resources ...wasmgpu.GPUBindingResource) []wasmgpu.GPUBindGroupEntry {
	entries := make([]wasmgpu.GPUBindGroupEntry, len(resources))
	for idx, resource := range resources {
		entries[idx] = wasmgpu.GPUBindGroupEntry{
			Binding:  wasmgpu.GPUIndex32(idx),
			Resource: resource,
		}
	}
	return entries
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

func newVertexBuffers(bufDefs []BufferDescriptor, vtxAttrs []VertexAttribute) *VertexBuffers {
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
