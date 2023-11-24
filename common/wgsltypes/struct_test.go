package wgsltypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hulkholden/gowebgpu/common/vmath"
)

type testStruct struct {
	foo  float32
	vec2 vmath.V2
	vec3 vmath.V3
	vec4 vmath.V4
	bar  float32
}

func TestNewStruct(t *testing.T) {
	got, err := NewStruct[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("NewStruct() = %v, want nil error", err)
	}
	want := Struct{
		Name: "testStruct",
		Fields: []Field{
			{
				Name:   "foo",
				Offset: 0,
				WGSLType: wgslType{
					Name: "f32",
				},
			}, {
				Name:   "vec2",
				Offset: 4,
				WGSLType: wgslType{
					Name: "vec2<f32>",
				},
			}, {
				Name:   "vec3",
				Offset: 12,
				WGSLType: wgslType{
					Name: "vec3<f32>",
				},
			}, {
				Name:   "vec4",
				Offset: 24,
				WGSLType: wgslType{
					Name: "vec4<f32>",
				},
			}, {
				Name:   "bar",
				Offset: 40,
				WGSLType: wgslType{
					Name: "f32",
				},
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
  foo : f32,
  vec2 : vec2<f32>,
  vec3 : vec3<f32>,
  vec4 : vec4<f32>,
  bar : f32,
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}
