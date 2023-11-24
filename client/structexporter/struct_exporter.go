package structexporter

import (
	"fmt"
	"reflect"
	"strings"
)

var goToWGSLTypeMap = map[string]wgslType{
	"float32": {Name: "f32"},
}

type wgslType struct {
	Name string
}

// A Struct provides information about a Go struct.
type Struct struct {
	// Name is the name of the struct as it appears in Go.
	Name string
	// Fields is a list of fields of the struct, in order.
	Fields []Field
}

// A Field provides information about a particular field in a Go struct.
type Field struct {
	// Name is the name of the field in the Go struct.
	Name string

	// Offset is the offset (in bytes) of the field in the Go struct.
	Offset uintptr

	// WGSLType is the corresponding WGSL type to use.
	WGSLType wgslType
}

func MustNew[T any](name string) Struct {
	s, err := New[T](name)
	if err != nil {
		panic(fmt.Sprintf("exporting %q: %v", name, err))
	}
	return s
}

func New[T any](name string) (Struct, error) {
	var t T
	structType := reflect.TypeOf(t)
	if structType.Kind() != reflect.Struct {
		return Struct{}, fmt.Errorf("provided type is not a struct")
	}

	s := Struct{
		Name: name,
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		wgslType, ok := goToWGSLTypeMap[field.Type.Name()]
		if !ok {
			return Struct{}, fmt.Errorf("unhandled Go type: %q", field.Type)
		}
		s.Fields = append(s.Fields, Field{
			Name:     field.Name,
			Offset:   field.Offset,
			WGSLType: wgslType,
		})
	}
	return s, nil
}

// ToWGSL returns a string representing the Go struct as a WGSL struct definition.
func (s Struct) ToWGSL() string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("struct %s {\n", s.Name))
	for _, f := range s.Fields {
		output.WriteString(fmt.Sprintf("  %s : %s,\n", f.Name, f.WGSLType.Name))
	}
	output.WriteString("}\n")
	return output.String()
}
