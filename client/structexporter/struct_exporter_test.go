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
	tests := []struct {
		name        string
		structName  string
		structValue any
		want        Struct
	}{
		{
			name:        "simple",
			structName:  "testStruct",
			structValue: testStruct{},
			want: Struct{
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
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := New(tc.structName, tc.structValue)
			if err != nil {
				t.Fatalf("Export() = %v, want nil error", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("diff mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToWGSL(t *testing.T) {
	tests := []struct {
		name        string
		structName  string
		structValue any
		want        string
	}{
		{
			name:        "simple",
			structName:  "testStruct",
			structValue: testStruct{},
			want: `struct testStruct {
  foo : f32,
  bar : f32,
}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := New(tc.structName, tc.structValue)
			if err != nil {
				t.Fatalf("New() failed unexpectedly: %v", err)
			}

			got := s.ToWGSL()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("diff mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
