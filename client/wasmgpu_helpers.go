package main

import "github.com/mokiat/wasmgpu"

func makeGPUVertexAttribute(shaderLocation wasmgpu.GPUIndex32, format wasmgpu.GPUVertexFormat, offset wasmgpu.GPUSize64) wasmgpu.GPUVertexAttribute {
	return wasmgpu.GPUVertexAttribute{
		ShaderLocation: shaderLocation,
		Format:         wasmgpu.GPUVertexFormatFloat32x2,
		Offset:         offset,
	}
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
