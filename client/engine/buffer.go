package engine

import (
	"syscall/js"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

func initBuffer(device wasmgpu.GPUDevice, usage wasmgpu.GPUBufferUsageFlags, data []byte, opts ...BufferOption) wasmgpu.GPUBuffer {
	desc := wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(len(data)),
		Usage:            usage,
		MappedAtCreation: opt.V(true),
	}
	for _, opt := range opts {
		opt(&desc)
	}
	buffer := device.CreateBuffer(desc)
	js.CopyBytesToJS(uint8ArrayCtor.New(buffer.GetMappedRange(0, 0)), data)
	buffer.Unmap()
	return buffer
}

type StorageBuffer struct {
	buffer wasmgpu.GPUBuffer
}

func InitStorageBuffer[T any](device wasmgpu.GPUDevice, values []T, opts ...BufferOption) StorageBuffer {
	data := sliceAsBytesSlice(values)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsStorage, data, opts...)
	return StorageBuffer{
		buffer: buffer,
	}
}

func (b StorageBuffer) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	buffer wasmgpu.GPUBuffer
}

func InitUniformBuffer[T any](device wasmgpu.GPUDevice, value T, opts ...BufferOption) UniformBuffer {
	data := structAsByteSlice(value)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsUniform, data, opts...)
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
