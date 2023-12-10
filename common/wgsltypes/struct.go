package wgsltypes

import (
	"fmt"
	"reflect"
	"strings"
)

// TypeName is the name of a WGSL type.
type TypeName string

// GoTypeName is the fully qualified name of a Go type.
type GoTypeName string

// goToTypeMap maps Go types to WGSL types.
var goToTypeMap = map[GoTypeName]TypeName{
	// Builtin types.
	"float32": "f32",
	"int32":   "i32",
	"uint32":  "u32",
	// Custom types.
	"github.com/hulkholden/gowebgpu/common/vmath.V2": "vec2<f32>",
	"github.com/hulkholden/gowebgpu/common/vmath.V3": "vec3<f32>",
	"github.com/hulkholden/gowebgpu/common/vmath.V4": "vec4<f32>",
}

var builtinTypeMap = map[TypeName]Type{
	"f32":       {Name: "f32", AlignOf: 4, SizeOf: 4},
	"i32":       {Name: "i32", AlignOf: 4, SizeOf: 4},
	"u32":       {Name: "u32", AlignOf: 4, SizeOf: 4},
	"vec2<f32>": {Name: "vec2<f32>", AlignOf: 8, SizeOf: 8},
	"vec3<f32>": {Name: "vec3<f32>", AlignOf: 16, SizeOf: 12},
	"vec4<f32>": {Name: "vec4<f32>", AlignOf: 16, SizeOf: 16},
}

// registeredGoStructs stores all the Go types that have been registered.
var registeredGoStructs = map[GoTypeName]Struct{}

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
	// Name is the name of the struct as it appears in WGSL.
	Name TypeName
	// GoName is the name of the struct as it appears in Go.
	GoName GoTypeName
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

func MustRegisterStruct[T any]() Struct {
	s, err := RegisterStruct[T]()
	if err != nil {
		var zero T
		panic(fmt.Sprintf("exporting %T: %v", zero, err))
	}
	return s
}

func RegisterStruct[T any]() (Struct, error) {
	var t T
	structType := reflect.TypeOf(t)

	if structType.Kind() != reflect.Struct {
		return Struct{}, fmt.Errorf("provided type is not a struct")
	}

	s := Struct{
		Name:     TypeName(structType.Name()),
		GoName:   GoTypeName(structType.PkgPath() + "." + structType.Name()),
		Size:     int(structType.Size()),
		FieldMap: make(map[string]Field),
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		arrayLen := 0
		fieldType := field.Type
		runtimeArray := false
		if fieldType.Kind() == reflect.Array {
			arrayLen = fieldType.Len()
			fieldType = fieldType.Elem()
			runtimeArray = field.Tag.Get("runtimeArray") == "true"

			// TODO: if this is a runtimeArray then verify it's the last field in the struct.
		}

		fieldTypeName := makeGoTypeName(fieldType)
		isAtomic := field.Tag.Get("atomic") == "true"
		wgslType, ok := lookupWGSLType(fieldTypeName, isAtomic, arrayLen, runtimeArray)
		if !ok {
			return Struct{}, fmt.Errorf("unhandled type: %q", fieldType.String())
		}

		if err := validateOffset(field, wgslType); err != nil {
			return Struct{}, err
		}

		s.Fields = append(s.Fields, field.Name)
		s.FieldMap[field.Name] = Field{
			Name:     field.Name,
			Offset:   field.Offset,
			WGSLType: wgslType,
		}
	}

	registeredGoStructs[s.GoName] = s
	return s, nil
}

// TODO: add test coverage for this.
func validateOffset(field reflect.StructField, wgslType Type) error {
	if (field.Offset % uintptr(wgslType.AlignOf)) != 0 {
		return fmt.Errorf("incompatible offset for field %q: Go offset is %d but wgsl requires aligment of %d bytes for fields of type %q", field.Name, field.Offset, wgslType.AlignOf, wgslType.Name)
	}
	return nil
}

func (s Struct) String() string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("struct %q, size %d\n", s.GoName, s.Size))
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

func makeGoTypeName(t reflect.Type) GoTypeName {
	if path := t.PkgPath(); path != "" {
		return GoTypeName(path + "." + t.Name())
	}
	return GoTypeName(t.Name())
}

func lookupWGSLType(goTypeName GoTypeName, isAtomic bool, arrayLen int, runtimeArray bool) (Type, bool) {
	wgslType, ok := lookupPrimitiveWGSLType(goTypeName)
	if !ok {
		return Type{}, false
	}

	// If the field has an atomic tag then treat is as atomic<T>.
	if isAtomic {
		wgslType = makeAtomic(wgslType)
	}
	// If the field is an array then treat is as array<T,N>.
	if arrayLen > 0 {
		if runtimeArray {
			wgslType = makeRuntimeArray(wgslType, arrayLen)
		} else {
			wgslType = makeArray(wgslType, arrayLen)
		}
	}

	return wgslType, true
}

func lookupPrimitiveWGSLType(goTypeName GoTypeName) (Type, bool) {
	if tn, ok := goToTypeMap[goTypeName]; ok {
		if wgslType, ok := builtinTypeMap[tn]; ok {
			return wgslType, true
		}
		return Type{}, false
	}
	if s, ok := registeredGoStructs[goTypeName]; ok {
		wgslType := Type{
			Name:    s.Name,
			AlignOf: s.Size, // TODO: this should be max(fieldAlignment)
			SizeOf:  s.Size,
		}
		return wgslType, true
	}
	fmt.Printf("Couldn't find %q -> %v\n", goTypeName, registeredGoStructs)
	return Type{}, false
}

func makeAtomic(t Type) Type {
	return Type{
		Name:    TypeName(fmt.Sprintf("atomic<%s>", t.Name)),
		AlignOf: t.AlignOf,
		SizeOf:  t.SizeOf,
	}
}

func makeRuntimeArray(t Type, n int) Type {
	return Type{
		Name:    TypeName(fmt.Sprintf("array<%s>", t.Name)),
		AlignOf: t.AlignOf,
		SizeOf:  t.SizeOf * n,
	}
}

func makeArray(t Type, n int) Type {
	return Type{
		Name:    TypeName(fmt.Sprintf("array<%s, %d>", t.Name, n)),
		AlignOf: t.AlignOf,
		SizeOf:  t.SizeOf * n,
	}
}
