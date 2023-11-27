package wgsltypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hulkholden/gowebgpu/common/vmath"
)

type testStruct struct {
	f32Val    float32
	int32Val  int32
	uint32Val uint32
	vec2      vmath.V2
	vec3      vmath.V3
	vec4      vmath.V4
}

func TestNewStruct(t *testing.T) {
	got, err := NewStruct[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("NewStruct() = %v, want nil error", err)
	}
	want := Struct{
		Name:   "testStruct",
		Size:   48,
		Fields: []string{"f32Val", "int32Val", "uint32Val", "vec2", "vec3", "vec4"},
		FieldMap: map[string]Field{
			"f32Val": {
				Name:     "f32Val",
				Offset:   0,
				WGSLType: Type{Name: "f32", AlignOf: 4, SizeOf: 4},
			},
			"int32Val": {
				Name:     "int32Val",
				Offset:   4,
				WGSLType: Type{Name: "i32", AlignOf: 4, SizeOf: 4},
			},
			"uint32Val": {
				Name:     "uint32Val",
				Offset:   8,
				WGSLType: Type{Name: "u32", AlignOf: 4, SizeOf: 4},
			},
			"vec2": {
				Name:     "vec2",
				Offset:   12,
				WGSLType: Type{Name: "vec2<f32>", AlignOf: 8, SizeOf: 8},
			},
			"vec3": {
				Name:     "vec3",
				Offset:   20,
				WGSLType: Type{Name: "vec3<f32>", AlignOf: 16, SizeOf: 12},
			},
			"vec4": {
				Name:     "vec4",
				Offset:   32,
				WGSLType: Type{Name: "vec4<f32>", AlignOf: 16, SizeOf: 16},
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
  f32Val : f32,
  int32Val : i32,
  uint32Val : u32,
  vec2 : vec2<f32>,
  vec3 : vec3<f32>,
  vec4 : vec4<f32>,
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}
