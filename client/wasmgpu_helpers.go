package main

import (
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

var vertexFormatTypeMap = map[string]wasmgpu.GPUVertexFormat{
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

func mustFormatFromFieldType(fieldType string) wasmgpu.GPUVertexFormat {
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

type VertexAttribute struct {
	Struct    *wgsltypes.Struct
	FieldName string
}

type VertexBuffers struct {
	Layout  []wasmgpu.GPUVertexBufferLayout
	Buffers []wasmgpu.GPUBuffer
}

func newVertexBuffers(v []VertexAttribute) *VertexBuffers {
	m := map[*wgsltypes.Struct]int{}

	result := []wasmgpu.GPUVertexBufferLayout{}

	for idx, a := range v {
		bufIdx, ok := m[a.Struct]
		if !ok {
			bufIdx = len(result)
			m[a.Struct] = bufIdx

			// TODO: provide a way to declare the step mode.
			// I'm not sure if it should be specified on Struct, or if we need
			// another type to encapsulate Struct+StepMode.
			stepMode := wasmgpu.GPUVertexStepModeInstance
			if a.Struct.Name == "Vertex" {
				stepMode = wasmgpu.GPUVertexStepModeVertex
			}

			layout := wasmgpu.GPUVertexBufferLayout{
				ArrayStride: wasmgpu.GPUSize64(a.Struct.Size),
				StepMode:    opt.V(stepMode),
			}
			result = append(result, layout)
		}

		attribute := makeGPUVertexAttribute(idx, *a.Struct, a.FieldName)
		result[bufIdx].Attributes = append(result[bufIdx].Attributes, attribute)
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
