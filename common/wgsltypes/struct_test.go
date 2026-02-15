package wgsltypes

import (
	"reflect"
	"strings"
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

	arrayInt32Val        [2]int32
	runtimeArrayInt32Val [2]int32 `runtimeArray:"true"`
}

func TestNewStruct(t *testing.T) {
	got, err := RegisterStruct[testStruct]()
	if err != nil {
		t.Fatalf("NewStruct() = %v, want nil error", err)
	}
	want := Struct{
		Name:   "testStruct",
		GoName: "github.com/hulkholden/gowebgpu/common/wgsltypes.testStruct",
		Size:   76,
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
			"runtimeArrayInt32Val",
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
			"runtimeArrayInt32Val": {
				Name:     "runtimeArrayInt32Val",
				Offset:   68,
				WGSLType: Type{Name: "array<i32>", AlignOf: 4, SizeOf: 8},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}

func TestToWGSL(t *testing.T) {
	s, err := RegisterStruct[testStruct]()
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
  runtimeArrayInt32Val : array<i32>,
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}

func TestRegisterStructRejectsNonStruct(t *testing.T) {
	_, err := RegisterStruct[int]()
	if err == nil {
		t.Fatal("RegisterStruct[int]() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "not a struct") {
		t.Errorf("error = %q, want it to mention 'not a struct'", err)
	}
}

type unsupportedFieldStruct struct {
	val float64 // float64 is not mapped to any WGSL type
}

func TestRegisterStructRejectsUnsupportedType(t *testing.T) {
	_, err := RegisterStruct[unsupportedFieldStruct]()
	if err == nil {
		t.Fatal("RegisterStruct[unsupportedFieldStruct]() succeeded, want error for float64 field")
	}
	if !strings.Contains(err.Error(), "unhandled type") {
		t.Errorf("error = %q, want it to mention 'unhandled type'", err)
	}
}

func TestValidateOffset(t *testing.T) {
	tests := []struct {
		name    string
		offset  uintptr
		alignOf int
		wantErr bool
	}{
		{name: "aligned", offset: 16, alignOf: 16, wantErr: false},
		{name: "zero offset", offset: 0, alignOf: 8, wantErr: false},
		{name: "misaligned", offset: 5, alignOf: 4, wantErr: true},
		{name: "misaligned vec3", offset: 4, alignOf: 16, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			field := reflect.StructField{
				Name:   "testField",
				Offset: tc.offset,
			}
			wgslType := Type{Name: "test", AlignOf: tc.alignOf, SizeOf: 4}
			err := validateOffset(field, wgslType)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateOffset(offset=%d, align=%d) error = %v, wantErr = %v", tc.offset, tc.alignOf, err, tc.wantErr)
			}
		})
	}
}

func TestMustOffsetOf(t *testing.T) {
	s, err := RegisterStruct[testStruct]()
	if err != nil {
		t.Fatalf("RegisterStruct failed: %v", err)
	}

	// Known field should return its offset.
	got := s.MustOffsetOf("f32Val")
	if got != 40 {
		t.Errorf("MustOffsetOf('f32Val') = %d, want 40", got)
	}

	got = s.MustOffsetOf("vec4")
	if got != 0 {
		t.Errorf("MustOffsetOf('vec4') = %d, want 0", got)
	}
}

func TestMustOffsetOfPanicsOnUnknown(t *testing.T) {
	s, err := RegisterStruct[testStruct]()
	if err != nil {
		t.Fatalf("RegisterStruct failed: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustOffsetOf('nonexistent') did not panic")
		}
	}()
	s.MustOffsetOf("nonexistent")
}

func TestMustRegisterStructPanicsOnError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustRegisterStruct[int]() did not panic")
		}
	}()
	MustRegisterStruct[int]()
}

func TestStructString(t *testing.T) {
	s, err := RegisterStruct[testStruct]()
	if err != nil {
		t.Fatalf("RegisterStruct failed: %v", err)
	}

	str := s.String()
	// Should contain the Go name and size.
	if !strings.Contains(str, string(s.GoName)) {
		t.Errorf("String() = %q, want it to contain GoName %q", str, s.GoName)
	}
	if !strings.Contains(str, "76") {
		t.Errorf("String() = %q, want it to contain size '76'", str)
	}
	// Should mention field names.
	if !strings.Contains(str, "vec4") {
		t.Errorf("String() = %q, want it to contain field 'vec4'", str)
	}
}

// simpleStruct is a minimal struct for testing basic registration.
type simpleStruct struct {
	x float32
	y float32
}

func TestRegisterSimpleStruct(t *testing.T) {
	s, err := RegisterStruct[simpleStruct]()
	if err != nil {
		t.Fatalf("RegisterStruct[simpleStruct]() = %v", err)
	}
	if s.Name != "simpleStruct" {
		t.Errorf("Name = %q, want 'simpleStruct'", s.Name)
	}
	if len(s.Fields) != 2 {
		t.Errorf("len(Fields) = %d, want 2", len(s.Fields))
	}
	if s.Size != 8 {
		t.Errorf("Size = %d, want 8", s.Size)
	}
}

// nestedInner is a struct that will be used as a field in another struct.
type nestedInner struct {
	a float32
	b float32
}

type nestedOuter struct {
	inner nestedInner
}

func TestRegisterNestedStruct(t *testing.T) {
	// Register the inner struct first so it's in the registry.
	_, err := RegisterStruct[nestedInner]()
	if err != nil {
		t.Fatalf("RegisterStruct[nestedInner]() = %v", err)
	}

	// Now register the outer struct which references the inner.
	s, err := RegisterStruct[nestedOuter]()
	if err != nil {
		t.Fatalf("RegisterStruct[nestedOuter]() = %v", err)
	}
	innerField := s.FieldMap["inner"]
	if innerField.WGSLType.Name != "nestedInner" {
		t.Errorf("nested field WGSL type = %q, want 'nestedInner'", innerField.WGSLType.Name)
	}
}
