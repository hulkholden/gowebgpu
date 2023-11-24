package structexporter

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testStruct struct {
	foo float32
	bar float32
}

func TestNew(t *testing.T) {
	got, err := New[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("Export() = %v, want nil error", err)
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
				Name:   "bar",
				Offset: 4,
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
	s, err := New[testStruct]("testStruct")
	if err != nil {
		t.Fatalf("New() failed unexpectedly: %v", err)
	}

	got := s.ToWGSL()
	want := `struct testStruct {
  foo : f32,
  bar : f32,
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff mismatch (-want +got):\n%s", diff)
	}
}
