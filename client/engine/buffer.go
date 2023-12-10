package engine

import (
	"syscall/js"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

func initBuffer(device wasmgpu.GPUDevice, usage wasmgpu.GPUBufferUsageFlags, data []byte, initContents bool, opts ...BufferOption) wasmgpu.GPUBuffer {
	desc := wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(len(data)),
		Usage:            usage,
		MappedAtCreation: opt.V(initContents),
	}
	for _, opt := range opts {
		opt(&desc)
	}
	buffer := device.CreateBuffer(desc)
	if initContents {
		js.CopyBytesToJS(uint8ArrayCtor.New(buffer.GetMappedRange(0, 0)), data)
		buffer.Unmap()
	}
	return buffer
}

type StorageBuffer struct {
	buffer wasmgpu.GPUBuffer
	size   int
}

func InitStorageBufferStruct[T any](device wasmgpu.GPUDevice, value T, opts ...BufferOption) StorageBuffer {
	data := structAsByteSlice(value)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsStorage, data, true, opts...)
	return StorageBuffer{
		buffer: buffer,
		size:   len(data),
	}
}

func InitStorageBufferSlice[T any](device wasmgpu.GPUDevice, values []T, opts ...BufferOption) StorageBuffer {
	data := sliceAsBytesSlice(values)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsStorage, data, true, opts...)
	return StorageBuffer{
		buffer: buffer,
		size:   len(data),
	}
}

func (b StorageBuffer) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}

func (b StorageBuffer) BufferSize() wasmgpu.GPUSize64 {
	return wasmgpu.GPUSize64(b.size)
}

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	buffer wasmgpu.GPUBuffer
}

func InitUniformBuffer[T any](device wasmgpu.GPUDevice, value T, opts ...BufferOption) UniformBuffer {
	data := structAsByteSlice(value)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsUniform, data, true, opts...)
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

type DebugBuffer[T any] struct {
	buffer wasmgpu.GPUBuffer
	size   int
}

func InitDebugBuffer[T any](device wasmgpu.GPUDevice, values []T, opts ...BufferOption) DebugBuffer[T] {
	data := sliceAsBytesSlice(values)
	buffer := initBuffer(device, wasmgpu.GPUBufferUsageFlagsMapRead|wasmgpu.GPUBufferUsageFlagsCopyDst, data, false, opts...)
	return DebugBuffer[T]{
		buffer: buffer,
		size:   len(data),
	}
}

func (b DebugBuffer[T]) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}

func (b DebugBuffer[T]) BufferSize() wasmgpu.GPUSize64 {
	return wasmgpu.GPUSize64(b.size)
}

func (b DebugBuffer[T]) ReadAsync(callback func(data []T)) {
	promise := b.buffer.MapAsync(wasmgpu.GPUMapModeFlagsRead, 0, b.BufferSize())
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		ab := b.buffer.GetMappedRange(0, b.BufferSize())
		abCopy := ab.Call("slice")
		b.buffer.Unmap()

		bytes := make([]byte, b.size)
		numBytes := js.CopyBytesToGo(bytes, uint8ArrayCtor.New(abCopy))
		typedData := byteSliceAsStructSlice[T](bytes[:numBytes])
		callback(typedData)
		return nil
	}))
}
