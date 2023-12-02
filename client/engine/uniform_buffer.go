package engine

import (
	"syscall/js"
	"unsafe"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	buffer wasmgpu.GPUBuffer
}

func InitUniformBuffer[T any](device wasmgpu.GPUDevice, value T, opts ...BufferOption) UniformBuffer {
	// TODO: use Struct to get this?
	byteLen := unsafe.Sizeof(value)

	desc := wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(byteLen),
		Usage:            wasmgpu.GPUBufferUsageFlagsUniform,
		MappedAtCreation: opt.V(true),
	}
	for _, opt := range opts {
		opt(&desc)
	}

	buffer := device.CreateBuffer(desc)
	js.CopyBytesToJS(uint8ArrayCtor.New(buffer.GetMappedRange(0, 0)), structAsByteSlice(value))
	buffer.Unmap()

	return UniformBuffer{
		device: device,
		buffer: buffer,
	}
}

// TODO: make UniformBuffer generic so we can use `value T` here?
func (b UniformBuffer) UpdateBuffer(bytes []byte) {
	b.device.Queue().WriteBuffer(b.buffer, 0, bytes)
}

func (b UniformBuffer) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}
