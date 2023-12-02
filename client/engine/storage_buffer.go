package engine

import (
	"syscall/js"
	"unsafe"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

type StorageBuffer struct {
	buffer wasmgpu.GPUBuffer
}

func InitStorageBuffer[T any](device wasmgpu.GPUDevice, values []T, opts ...BufferOption) StorageBuffer {
	// TODO: use Struct to get this?
	byteLen := int(unsafe.Sizeof(values[0])) * len(values)

	desc := wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(byteLen),
		Usage:            wasmgpu.GPUBufferUsageFlagsStorage,
		MappedAtCreation: opt.V(true),
	}
	for _, opt := range opts {
		opt(&desc)
	}
	buffer := device.CreateBuffer(desc)
	js.CopyBytesToJS(uint8ArrayCtor.New(buffer.GetMappedRange(0, 0)), sliceAsBytesSlice(values))
	buffer.Unmap()

	return StorageBuffer{
		buffer: buffer,
	}
}

func (b StorageBuffer) Buffer() wasmgpu.GPUBuffer {
	return b.buffer
}
