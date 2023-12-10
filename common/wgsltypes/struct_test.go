package wgsltypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hulkholden/gowebgpu/common/vmath"
)

type testStruct struct {
	vec4 vmath.V4

	vec3 vmath.V3
	pad0 uint32

	vec2 vmath.V2

	f32Val    float32
	int32Val  int32
	uint32Val uint32

	atomicInt32Val  int32  `atomic:"true"`
	atomicUint32Val uint32 `atomic:"true"`

	arrayInt32Val [2]int32
}

func TestNewStruct(t *testing.T) {
	got, err := NewStruct[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("NewStruct() = %v, want nil error", err)
	}
	want := Struct{
		Name: "testStruct",
		Size: 68,
		Fields: []string{
			"vec4",
			"vec3",
			"pad0",
			"vec2",
			"f32Val",
			"int32Val",
			"uint32Val",
			"atomicInt32Val",
			"atomicUint32Val",
			"arrayInt32Val",
		},
		FieldMap: map[string]Field{
			"vec4": {
				Name:     "vec4",
				Offset:   0,
				WGSLType: Type{Name: "vec4<f32>", AlignOf: 16, SizeOf: 16},
			},
			"vec3": {
				Name:     "vec3",
				Offset:   16,
				WGSLType: Type{Name: "vec3<f32>", AlignOf: 16, SizeOf: 12},
			},
			"pad0": {
				Name:     "pad0",
				Offset:   28,
				WGSLType: Type{Name: "u32", AlignOf: 4, SizeOf: 4},
			},
			"vec2": {
				Name:     "vec2",
				Offset:   32,
				WGSLType: Type{Name: "vec2<f32>", AlignOf: 8, SizeOf: 8},
			},
			"f32Val": {
				Name:     "f32Val",
				Offset:   40,
				WGSLType: Type{Name: "f32", AlignOf: 4, SizeOf: 4},
			},
			"int32Val": {
				Name:     "int32Val",
				Offset:   44,
				WGSLType: Type{Name: "i32", AlignOf: 4, SizeOf: 4},
			},
			"uint32Val": {
				Name:     "uint32Val",
				Offset:   48,
				WGSLType: Type{Name: "u32", AlignOf: 4, SizeOf: 4},
			},
			"atomicInt32Val": {
				Name:     "atomicInt32Val",
				Offset:   52,
				WGSLType: Type{Name: "atomic<i32>", AlignOf: 4, SizeOf: 4},
			},
			"atomicUint32Val": {
				Name:     "atomicUint32Val",
				Offset:   56,
				WGSLType: Type{Name: "atomic<u32>", AlignOf: 4, SizeOf: 4},
			},
			"arrayInt32Val": {
				Name:     "arrayInt32Val",
				Offset:   60,
				WGSLType: Type{Name: "array<i32, 2>", AlignOf: 4, SizeOf: 8},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}

func TestToWGSL(t *testing.T) {
	s, err := NewStruct[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("NewStruct() failed unexpectedly: %v", err)
	}

	got := s.ToWGSL()
	want := `struct testStruct {
  vec4 : vec4<f32>,
  vec3 : vec3<f32>,
  pad0 : u32,
  vec2 : vec2<f32>,
  f32Val : f32,
  int32Val : i32,
  uint32Val : u32,
  atomicInt32Val : atomic<i32>,
  atomicUint32Val : atomic<u32>,
  arrayInt32Val : array<i32, 2>,
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}
