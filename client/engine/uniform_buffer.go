package engine

import (
	"unsafe"

	"github.com/mokiat/wasmgpu"
)

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	buffer wasmgpu.GPUBuffer
}

func InitUniformBuffer[T any](device wasmgpu.GPUDevice, values T, opts ...BufferOption) UniformBuffer {
	// TODO: use Struct to get this?
	byteLen := unsafe.Sizeof(values)

	desc := wasmgpu.GPUBufferDescriptor{
		Size:  wasmgpu.GPUSize64(byteLen),
		Usage: wasmgpu.GPUBufferUsageFlagsUniform,
	}
	for _, opt := range opts {
		opt(&desc)
	}

	buffer := device.CreateBuffer(desc)
	b := UniformBuffer{
		device: device,
		buffer: buffer,
	}
	b.updateBuffer(structAsByteSlice(values))
	return b
}

func (b UniformBuffer) updateBuffer(bytes []byte) {
	b.device.Queue().WriteBuffer(b.buffer, 0, bytes)
}

func (b UniformBuffer) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}
