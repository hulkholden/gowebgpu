package main

import (
	"syscall/js"
	"unsafe"

	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

type StorageBuffer struct {
	buffer wasmgpu.GPUBuffer
}

func initStorageBuffer[T any](device wasmgpu.GPUDevice, values []T) StorageBuffer {
	// TODO: use Struct?
	byteLen := int(unsafe.Sizeof(values[0])) * len(values)
	buffer := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:             wasmgpu.GPUSize64(byteLen),
		Usage:            wasmgpu.GPUBufferUsageFlagsVertex | wasmgpu.GPUBufferUsageFlagsStorage,
		MappedAtCreation: opt.V(true),
	})
	js.CopyBytesToJS(uint8ArrayCtor.New(buffer.GetMappedRange(0, 0)), sliceAsBytesSlice(values))
	buffer.Unmap()

	return StorageBuffer{
		buffer: buffer,
	}
}
