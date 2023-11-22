package main

import "github.com/mokiat/wasmgpu"

func makeGPUVertexAttribute(shaderLocation wasmgpu.GPUIndex32, format wasmgpu.GPUVertexFormat, offset wasmgpu.GPUSize64) wasmgpu.GPUVertexAttribute {
	return wasmgpu.GPUVertexAttribute{
		ShaderLocation: shaderLocation,
		Format:         wasmgpu.GPUVertexFormatFloat32x2,
		Offset:         offset,
	}
}
