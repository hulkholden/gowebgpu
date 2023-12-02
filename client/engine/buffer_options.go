package engine

import "github.com/mokiat/wasmgpu"

type BufferOption func(d *wasmgpu.GPUBufferDescriptor)

func WithVertexUsage() BufferOption {
	return func(d *wasmgpu.GPUBufferDescriptor) {
		d.Usage |= wasmgpu.GPUBufferUsageFlagsVertex
	}
}

func WithCopySrcUsage() BufferOption {
	return func(d *wasmgpu.GPUBufferDescriptor) {
		d.Usage |= wasmgpu.GPUBufferUsageFlagsCopySrc
	}
}

func WithCopyDstUsage() BufferOption {
	return func(d *wasmgpu.GPUBufferDescriptor) {
		d.Usage |= wasmgpu.GPUBufferUsageFlagsCopyDst
	}
}
