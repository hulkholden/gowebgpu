package main

import (
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/wasmgpu"
)

var vertexFormatTypeMap = map[string]wasmgpu.GPUVertexFormat{
	"f32":       wasmgpu.GPUVertexFormatFloat32,
	"vec2<f32>": wasmgpu.GPUVertexFormatFloat32x2,
	"vec3<f32>": wasmgpu.GPUVertexFormatFloat32x3,
	"vec4<f32>": wasmgpu.GPUVertexFormatFloat32x4,
}

func makeGPUVertexAttribute(shaderLocation wasmgpu.GPUIndex32, s wgsltypes.Struct, fieldName string) wasmgpu.GPUVertexAttribute {
	return wasmgpu.GPUVertexAttribute{
		ShaderLocation: shaderLocation,
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
