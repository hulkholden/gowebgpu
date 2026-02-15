package vmath

import (
	"testing"

	"github.com/hulkholden/gowebgpu/common/math32"
)

const eps = 1e-5

func approxEqualF(a, b, epsilon float32) bool {
	return math32.Abs(a-b) < epsilon
}

func approxEqualV2(a, b V2) bool {
	return approxEqualF(a.X, b.X, eps) && approxEqualF(a.Y, b.Y, eps)
}

func TestV2Add(t *testing.T) {
	tests := []struct {
		name string
		v, w V2
		want V2
	}{
		{name: "basic", v: NewV2(1, 2), w: NewV2(3, 4), want: NewV2(4, 6)},
		{name: "zero", v: NewV2(1, 2), w: NewV2(0, 0), want: NewV2(1, 2)},
		{name: "negative", v: NewV2(1, 2), w: NewV2(-1, -2), want: NewV2(0, 0)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.Add(tc.w)
			if got != tc.want {
				t.Errorf("(%v).Add(%v) = %v, want %v", tc.v, tc.w, got, tc.want)
			}
		})
	}
}

func TestV2Sub(t *testing.T) {
	tests := []struct {
		name string
		v, w V2
		want V2
	}{
		{name: "basic", v: NewV2(5, 7), w: NewV2(3, 4), want: NewV2(2, 3)},
		{name: "self", v: NewV2(3, 4), w: NewV2(3, 4), want: NewV2(0, 0)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.Sub(tc.w)
			if got != tc.want {
				t.Errorf("(%v).Sub(%v) = %v, want %v", tc.v, tc.w, got, tc.want)
			}
		})
	}
}

func TestV2Scale(t *testing.T) {
	v := NewV2(3, 4)
	got := v.Scale(2)
	want := NewV2(6, 8)
	if got != want {
		t.Errorf("(%v).Scale(2) = %v, want %v", v, got, want)
	}
}

func TestV2Negate(t *testing.T) {
	v := NewV2(3, -4)
	got := v.Negate()
	want := NewV2(-3, 4)
	if got != want {
		t.Errorf("(%v).Negate() = %v, want %v", v, got, want)
	}
}

func TestV2Dot(t *testing.T) {
	tests := []struct {
		name string
		v, w V2
		want float32
	}{
		{name: "parallel", v: NewV2(1, 0), w: NewV2(3, 0), want: 3},
		{name: "perpendicular", v: NewV2(1, 0), w: NewV2(0, 1), want: 0},
		{name: "general", v: NewV2(2, 3), w: NewV2(4, 5), want: 23},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.Dot(tc.w)
			if got != tc.want {
				t.Errorf("(%v).Dot(%v) = %v, want %v", tc.v, tc.w, got, tc.want)
			}
		})
	}
}

func TestV2Cross(t *testing.T) {
	tests := []struct {
		name string
		v, w V2
		want float32
	}{
		{name: "parallel", v: NewV2(1, 0), w: NewV2(2, 0), want: 0},
		{name: "perpendicular", v: NewV2(1, 0), w: NewV2(0, 1), want: 1},
		{name: "anti-perpendicular", v: NewV2(0, 1), w: NewV2(1, 0), want: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.Cross(tc.w)
			if got != tc.want {
				t.Errorf("(%v).Cross(%v) = %v, want %v", tc.v, tc.w, got, tc.want)
			}
		})
	}
}

func TestV2LengthAndLengthSq(t *testing.T) {
	// 3-4-5 triangle
	v := NewV2(3, 4)
	if got := v.LengthSq(); got != 25 {
		t.Errorf("(%v).LengthSq() = %v, want 25", v, got)
	}
	if got := v.Length(); got != 5 {
		t.Errorf("(%v).Length() = %v, want 5", v, got)
	}
}

func TestV2DistanceAndDistanceSq(t *testing.T) {
	a := NewV2(1, 1)
	b := NewV2(4, 5) // diff = (3,4), distance = 5
	if got := a.DistanceSq(b); got != 25 {
		t.Errorf("(%v).DistanceSq(%v) = %v, want 25", a, b, got)
	}
	if got := a.Distance(b); got != 5 {
		t.Errorf("(%v).Distance(%v) = %v, want 5", a, b, got)
	}
}

func TestV2Normal(t *testing.T) {
	v := NewV2(3, 4)
	norm, length := v.Normal()
	if !approxEqualF(length, 5, eps) {
		t.Errorf("(%v).Normal() length = %v, want 5", v, length)
	}
	if !approxEqualV2(norm, NewV2(0.6, 0.8)) {
		t.Errorf("(%v).Normal() direction = %v, want {0.6, 0.8}", v, norm)
	}
	// Verify the result is actually unit length.
	if !approxEqualF(norm.Length(), 1, eps) {
		t.Errorf("(%v).Normal() result has length %v, want 1", v, norm.Length())
	}
}

func TestV2Rotate(t *testing.T) {
	v := NewV2(1, 0)
	tests := []struct {
		name  string
		angle float32
		want  V2
	}{
		{name: "0 degrees", angle: 0, want: NewV2(1, 0)},
		{name: "90 degrees", angle: math32.Pi / 2, want: NewV2(0, 1)},
		{name: "180 degrees", angle: math32.Pi, want: NewV2(-1, 0)},
		{name: "360 degrees", angle: math32.TwoPi, want: NewV2(1, 0)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := v.Rotate(tc.angle)
			if !approxEqualV2(got, tc.want) {
				t.Errorf("(%v).Rotate(%v) = %v, want %v", v, tc.angle, got, tc.want)
			}
		})
	}
}

func TestV2AngleRoundTrip(t *testing.T) {
	angles := []float32{0, math32.Pi / 4, math32.Pi / 2, -math32.Pi / 4}
	for _, a := range angles {
		v := NewV2FromAngle(a)
		got := v.ToAngle()
		if !approxEqualF(got, a, eps) {
			t.Errorf("NewV2FromAngle(%v).ToAngle() = %v", a, got)
		}
	}
}

func TestV2Lerp(t *testing.T) {
	a := NewV2(0, 0)
	b := NewV2(10, 20)
	tests := []struct {
		name string
		f    float32
		want V2
	}{
		{name: "f=0", f: 0, want: NewV2(0, 0)},
		{name: "f=1", f: 1, want: NewV2(10, 20)},
		{name: "f=0.5", f: 0.5, want: NewV2(5, 10)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := a.Lerp(b, tc.f)
			if !approxEqualV2(got, tc.want) {
				t.Errorf("(%v).Lerp(%v, %v) = %v, want %v", a, b, tc.f, got, tc.want)
			}
		})
	}
}

func TestV2MinMax(t *testing.T) {
	a := NewV2(1, 4)
	b := NewV2(3, 2)

	gotMin := a.Min(b)
	wantMin := NewV2(1, 2)
	if gotMin != wantMin {
		t.Errorf("(%v).Min(%v) = %v, want %v", a, b, gotMin, wantMin)
	}

	gotMax := a.Max(b)
	wantMax := NewV2(3, 4)
	if gotMax != wantMax {
		t.Errorf("(%v).Max(%v) = %v, want %v", a, b, gotMax, wantMax)
	}
}

func TestV2String(t *testing.T) {
	v := NewV2(1.5, 2.5)
	got := v.String()
	if got == "" {
		t.Error("V2.String() returned empty string")
	}
}
