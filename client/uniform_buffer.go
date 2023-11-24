package main

import (
	"unsafe"

	"github.com/mokiat/wasmgpu"
)

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	buffer wasmgpu.GPUBuffer
}

func initUniformBuffer[T any](device wasmgpu.GPUDevice, values T) UniformBuffer {
	byteLen := unsafe.Sizeof(values)
	buffer := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:  wasmgpu.GPUSize64(byteLen),
		Usage: wasmgpu.GPUBufferUsageFlagsUniform | wasmgpu.GPUBufferUsageFlagsCopyDst,
	})

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
