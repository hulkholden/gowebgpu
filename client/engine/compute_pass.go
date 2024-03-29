package engine

import (
	"github.com/hulkholden/gowebgpu/common/wgsltypes"
	"github.com/mokiat/gog/opt"
	"github.com/mokiat/wasmgpu"
)

type ComputePass func(commandEncoder wasmgpu.GPUCommandEncoder)

type ComputePassBuffer interface {
	StructDefs() []wgsltypes.Struct
	MakeBindGroupLayoutEntry(idx int) wasmgpu.GPUBindGroupLayoutEntry
	MakeBindingGroupEntry(idx int) wasmgpu.GPUBindGroupEntry
}

type ComputePassFactory struct {
	device                wasmgpu.GPUDevice
	computeShaderModule   wasmgpu.GPUShaderModule
	computePassDescriptor wasmgpu.GPUComputePassDescriptor

	layout           wasmgpu.GPUPipelineLayout
	bindGroupEntries []wasmgpu.GPUBindGroupEntry
}

func NewComputePassFactory(device wasmgpu.GPUDevice, computeShaderCode string, extraStructDefinitions []wgsltypes.Struct, buffers []ComputePassBuffer) ComputePassFactory {
	structDefinitions := []wgsltypes.Struct{}
	for _, b := range buffers {
		structDefinitions = append(structDefinitions, b.StructDefs()...)
	}
	structDefinitions = append(structDefinitions, extraStructDefinitions...)

	computeShaderModule := InitShaderModule(device, computeShaderCode, structDefinitions)

	bindGroupEntries := make([]wasmgpu.GPUBindGroupEntry, len(buffers))
	bindGroupLayoutEntries := make([]wasmgpu.GPUBindGroupLayoutEntry, len(buffers))
	for i, b := range buffers {
		bindGroupEntries[i] = b.MakeBindingGroupEntry(i)
		bindGroupLayoutEntries[i] = b.MakeBindGroupLayoutEntry(i)
	}

	layout := device.CreatePipelineLayout(wasmgpu.GPUPipelineLayoutDescriptor{
		BindGroupLayouts: []wasmgpu.GPUBindGroupLayout{
			device.CreateBindGroupLayout(wasmgpu.GPUBindGroupLayoutDescriptor{
				Entries: bindGroupLayoutEntries,
			}),
		},
	})
	return ComputePassFactory{
		device:                device,
		layout:                layout,
		computeShaderModule:   computeShaderModule,
		bindGroupEntries:      bindGroupEntries,
		computePassDescriptor: wasmgpu.GPUComputePassDescriptor{},
	}
}

func (cpf ComputePassFactory) InitPass(entryPoint string, numWorkgroups int) ComputePass {
	pipeline := cpf.device.CreateComputePipeline(wasmgpu.GPUComputePipelineDescriptor{
		Layout: opt.V(cpf.layout),
		Compute: wasmgpu.GPUProgrammableStage{
			Module:     cpf.computeShaderModule,
			EntryPoint: entryPoint,
		},
	})
	bindGroup := cpf.device.CreateBindGroup(wasmgpu.GPUBindGroupDescriptor{
		Layout:  pipeline.GetBindGroupLayout(0),
		Entries: cpf.bindGroupEntries,
	})
	return func(commandEncoder wasmgpu.GPUCommandEncoder) {
		passEncoder := commandEncoder.BeginComputePass(opt.V(cpf.computePassDescriptor))
		passEncoder.SetPipeline(pipeline)
		passEncoder.SetBindGroup(0, bindGroup, nil)
		passEncoder.DispatchWorkgroups(wasmgpu.GPUSize32(numWorkgroups), 0, 0)
		passEncoder.End()
	}
}
