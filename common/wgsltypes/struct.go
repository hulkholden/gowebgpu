package wgsltypes

import (
	"fmt"
	"reflect"
	"strings"
)

// TypeName is the name of a WGSL type.
type TypeName string

// goToTypeMap maps Go types to WGSL types.
var goToTypeMap = map[string]TypeName{
	// Builtin types.
	"float32": "f32",
	"int32":   "i32",
	"uint32":  "u32",
	// Custom types.
	"github.com/hulkholden/gowebgpu/common/vmath.V2": "vec2<f32>",
	"github.com/hulkholden/gowebgpu/common/vmath.V3": "vec3<f32>",
	"github.com/hulkholden/gowebgpu/common/vmath.V4": "vec4<f32>",
}

var typeMap = map[TypeName]Type{
	"f32":       {Name: "f32", AlignOf: 4, SizeOf: 4},
	"i32":       {Name: "i32", AlignOf: 4, SizeOf: 4},
	"u32":       {Name: "u32", AlignOf: 4, SizeOf: 4},
	"vec2<f32>": {Name: "vec2<f32>", AlignOf: 8, SizeOf: 8},
	"vec3<f32>": {Name: "vec3<f32>", AlignOf: 16, SizeOf: 12},
	"vec4<f32>": {Name: "vec4<f32>", AlignOf: 16, SizeOf: 16},
}

type Type struct {
	// Name of the WGSL type.
	Name TypeName
	// Alignment of the WGSL type (see https://www.w3.org/TR/WGSL/#alignof).
	AlignOf int
	// Size if the WGSL type (see https://www.w3.org/TR/WGSL/#sizeof).
	SizeOf int
}

// A Struct provides information about a Go struct.
type Struct struct {
	// Name is the name of the struct as it appears in Go.
	Name string
	// Size of the structure, in bytes.
	Size int

	// Fields is a slice of the struct's fields, in declaration order.
	Fields []string
	// FieldMap maps field names to Fields.
	FieldMap map[string]Field
}

// A Field provides information about a particular field in a Go struct.
type Field struct {
	// Name is the name of the field in the Go struct.
	Name string

	// Offset is the offset (in bytes) of the field in the Go struct.
	Offset uintptr

	// WGSLType is the corresponding WGSL type to use.
	WGSLType Type
}

func MustNewStruct[T any](name string) Struct {
	s, err := NewStruct[T](name)
	if err != nil {
		panic(fmt.Sprintf("exporting %q: %v", name, err))
	}
	return s
}

func NewStruct[T any](name string) (Struct, error) {
	var t T
	structType := reflect.TypeOf(t)
	if structType.Kind() != reflect.Struct {
		return Struct{}, fmt.Errorf("provided type is not a struct")
	}

	s := Struct{
		Name:     name,
		Size:     int(structType.Size()),
		FieldMap: make(map[string]Field),
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type.Name()
		if path := field.Type.PkgPath(); path != "" {
			fieldType = path + "." + fieldType
		}
		wgslTypeName, ok := goToTypeMap[fieldType]
		if !ok {
			return Struct{}, fmt.Errorf("unhandled Go type: %q", fieldType)
		}
		wgslType, ok := typeMap[wgslTypeName]
		if !ok {
			return Struct{}, fmt.Errorf("unhandled WGSL type: %q", wgslTypeName)
		}
		s.Fields = append(s.Fields, field.Name)
		s.FieldMap[field.Name] = Field{
			Name:     field.Name,
			Offset:   field.Offset,
			WGSLType: wgslType,
		}
	}
	return s, nil
}

func (s Struct) String() string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("struct %q, size %d\n", s.Name, s.Size))
	for idx, fName := range s.Fields {
		f := s.FieldMap[fName]
		output.WriteString(fmt.Sprintf("  %d: %s at offset %d\n", idx, f.Name, f.Offset))
	}
	return output.String()
}

// ToWGSL returns a string representing the Go struct as a WGSL struct definition.
func (s Struct) ToWGSL() string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("struct %s {\n", s.Name))
	for _, fieldName := range s.Fields {
		f := s.FieldMap[fieldName]
		output.WriteString(fmt.Sprintf("  %s : %s,\n", fieldName, f.WGSLType.Name))
	}
	output.WriteString("}\n")
	return output.String()
}

// MustOffsetOf returns the offset of the specified field.
// Panics if the field is not found.
func (s *Struct) MustOffsetOf(fieldName string) int {
	field, ok := s.FieldMap[fieldName]
	if !ok {
		panic("unknown field: " + fieldName)
	}
	return int(field.Offset)
}
