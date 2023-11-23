package main

import "github.com/mokiat/wasmgpu"

type UniformBuffer struct {
	device wasmgpu.GPUDevice
	values []float32
	buffer wasmgpu.GPUBuffer
}

func initUniformBuffer(device wasmgpu.GPUDevice, values []float32) UniformBuffer {
	buffer := device.CreateBuffer(wasmgpu.GPUBufferDescriptor{
		Size:  wasmgpu.GPUSize64(len(values) * float32Size),
		Usage: wasmgpu.GPUBufferUsageFlagsUniform | wasmgpu.GPUBufferUsageFlagsCopyDst,
	})

	b := UniformBuffer{
		device: device,
		values: values,
		buffer: buffer,
	}
	b.updateBuffer()
	return b
}

func (b UniformBuffer) updateBuffer() {
	b.device.Queue().WriteBuffer(b.buffer, 0, asByteSlice(b.values))
}
