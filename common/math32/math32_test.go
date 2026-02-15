package math32

import (
	"math"
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClamp(t *testing.T) {
	tests := []struct {
		name           string
		x, min, max    float32
		want           float32
	}{
		{name: "in range", x: 5, min: 0, max: 10, want: 5},
		{name: "at min", x: 0, min: 0, max: 10, want: 0},
		{name: "at max", x: 10, min: 0, max: 10, want: 10},
		{name: "below min", x: -5, min: 0, max: 10, want: 0},
		{name: "above max", x: 15, min: 0, max: 10, want: 10},
		{name: "negative range", x: -3, min: -5, max: -1, want: -3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Clamp(tc.x, tc.min, tc.max)
			if got != tc.want {
				t.Errorf("Clamp(%v, %v, %v) = %v, want %v", tc.x, tc.min, tc.max, got, tc.want)
			}
		})
	}
}

func TestLerp(t *testing.T) {
	tests := []struct {
		name    string
		a, b, f float32
		want    float32
	}{
		{name: "f=0", a: 10, b: 20, f: 0, want: 10},
		{name: "f=1", a: 10, b: 20, f: 1, want: 20},
		{name: "f=0.5", a: 10, b: 20, f: 0.5, want: 15},
		{name: "f=0.25", a: 0, b: 100, f: 0.25, want: 25},
		{name: "negative values", a: -10, b: 10, f: 0.5, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Lerp(tc.a, tc.b, tc.f)
			if got != tc.want {
				t.Errorf("Lerp(%v, %v, %v) = %v, want %v", tc.a, tc.b, tc.f, got, tc.want)
			}
		})
	}
}

func TestMagnitudeAndSign(t *testing.T) {
	tests := []struct {
		name     string
		x        float32
		wantMag  float32
		wantSign float32
	}{
		{name: "positive", x: 5, wantMag: 5, wantSign: 1},
		{name: "negative", x: -3, wantMag: 3, wantSign: -1},
		{name: "zero", x: 0, wantMag: 0, wantSign: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotMag, gotSign := MagnitudeAndSign(tc.x)
			if gotMag != tc.wantMag || gotSign != tc.wantSign {
				t.Errorf("MagnitudeAndSign(%v) = (%v, %v), want (%v, %v)", tc.x, gotMag, gotSign, tc.wantMag, tc.wantSign)
			}
		})
	}
}

func approxEqual(a, b, epsilon float32) bool {
	return Abs(a-b) < epsilon
}

func TestNormalizeAngle(t *testing.T) {
	const eps = 1e-5
	tests := []struct {
		name    string
		a       float32
		want    float32
		absBound bool // if true, accept either +want or -want (Pi boundary)
	}{
		{name: "zero", a: 0, want: 0},
		{name: "positive pi", a: Pi, want: Pi, absBound: true},
		{name: "negative pi", a: -Pi, want: Pi, absBound: true},
		{name: "two pi wraps to zero", a: TwoPi, want: 0},
		{name: "negative two pi wraps to zero", a: -TwoPi, want: 0},
		{name: "three pi wraps to pi", a: 3 * Pi, want: Pi, absBound: true},
		{name: "half pi", a: Pi / 2, want: Pi / 2},
		{name: "negative half pi", a: -Pi / 2, want: -Pi / 2},
		{name: "large positive", a: 5 * Pi, want: Pi, absBound: true},
		{name: "large negative", a: -5 * Pi, want: Pi, absBound: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeAngle(tc.a)
			ok := false
			if tc.absBound {
				// At the Pi/-Pi boundary, either sign is acceptable.
				ok = approxEqual(Abs(got), tc.want, eps)
			} else {
				ok = approxEqual(got, tc.want, eps)
			}
			if !ok {
				t.Errorf("NormalizeAngle(%v) = %v, want %v", tc.a, got, tc.want)
			}
		})
	}

	// All results should be in the range [-Pi, Pi].
	for _, a := range []float32{0, 0.1, -0.1, 1, -1, 10, -10, 100, -100} {
		got := NormalizeAngle(a)
		if got < -Pi-eps || got > Pi+eps {
			t.Errorf("NormalizeAngle(%v) = %v, out of range [-Pi, Pi]", a, got)
		}
	}
}

func TestAngleDiff(t *testing.T) {
	const eps = 1e-5
	tests := []struct {
		name     string
		from, to float32
		want     float32
	}{
		{name: "same angle", from: 0, to: 0, want: 0},
		{name: "quarter turn", from: 0, to: Pi / 2, want: Pi / 2},
		{name: "reverse quarter", from: Pi / 2, to: 0, want: -Pi / 2},
		{name: "half turn", from: 0, to: Pi, want: -Pi}, // Pi boundary maps to -Pi
		{name: "wrap around", from: 3 * Pi / 4, to: -3 * Pi / 4, want: Pi / 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AngleDiff(tc.from, tc.to)
			if !approxEqual(got, tc.want, eps) {
				t.Errorf("AngleDiff(%v, %v) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestRadiansDegreesRoundTrip(t *testing.T) {
	const eps = 1e-4
	tests := []float32{0, 45, 90, 180, 270, 360, -90}
	for _, deg := range tests {
		got := DegreesFromRadians(RadiansFromDegrees(deg))
		if !approxEqual(got, deg, eps) {
			t.Errorf("round trip for %v degrees: got %v", deg, got)
		}
	}
}

func TestRadiansFromDegrees(t *testing.T) {
	const eps = 1e-6
	tests := []struct {
		deg  float32
		want float32
	}{
		{deg: 0, want: 0},
		{deg: 180, want: Pi},
		{deg: 90, want: Pi / 2},
		{deg: 360, want: TwoPi},
	}
	for _, tc := range tests {
		got := RadiansFromDegrees(tc.deg)
		if !approxEqual(got, tc.want, eps) {
			t.Errorf("RadiansFromDegrees(%v) = %v, want %v", tc.deg, got, tc.want)
		}
	}
}

func TestMinMax(t *testing.T) {
	if got := Min(3, 5); got != 3 {
		t.Errorf("Min(3, 5) = %v, want 3", got)
	}
	if got := Min(5, 3); got != 3 {
		t.Errorf("Min(5, 3) = %v, want 3", got)
	}
	if got := Max(3, 5); got != 5 {
		t.Errorf("Max(3, 5) = %v, want 5", got)
	}
	if got := Max(5, 3); got != 5 {
		t.Errorf("Max(5, 3) = %v, want 5", got)
	}
	if got := Min(-1, -2); got != -2 {
		t.Errorf("Min(-1, -2) = %v, want -2", got)
	}
}

func TestAbs(t *testing.T) {
	if got := Abs(5); got != 5 {
		t.Errorf("Abs(5) = %v, want 5", got)
	}
	if got := Abs(-5); got != 5 {
		t.Errorf("Abs(-5) = %v, want 5", got)
	}
	if got := Abs(0); got != 0 {
		t.Errorf("Abs(0) = %v, want 0", got)
	}
}

func TestSinCos(t *testing.T) {
	const eps = 1e-6
	s, c := SinCos(0)
	if !approxEqual(s, 0, eps) || !approxEqual(c, 1, eps) {
		t.Errorf("SinCos(0) = (%v, %v), want (0, 1)", s, c)
	}
	s, c = SinCos(Pi / 2)
	if !approxEqual(s, 1, eps) || !approxEqual(c, 0, eps) {
		t.Errorf("SinCos(Pi/2) = (%v, %v), want (1, 0)", s, c)
	}
}

func TestNaN(t *testing.T) {
	n := NaN()
	if !math.IsNaN(float64(n)) {
		t.Errorf("NaN() = %v, want NaN", n)
	}
}

func TestUniformRangedValue(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	v := UniformRangedValue{Min: 10, Max: 20}

	for i := 0; i < 100; i++ {
		got := v.Get(r)
		if got < v.Min || got > v.Max {
			t.Errorf("UniformRangedValue{%v, %v}.Get() = %v, out of range", v.Min, v.Max, got)
		}
	}
}

func TestNormallyDistributedValue(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	v := NormallyDistributedValue{Mean: 100, StdDev: 10}

	var sum float32
	n := 10000
	for i := 0; i < n; i++ {
		sum += v.Get(r)
	}
	mean := sum / float32(n)
	// With 10000 samples, the mean should be close to 100.
	if !approxEqual(mean, v.Mean, 1.0) {
		t.Errorf("NormallyDistributedValue mean over %d samples = %v, want ~%v", n, mean, v.Mean)
	}
}

func TestUniformRangedValueUsesSeedRand(t *testing.T) {
	// This test verifies that UniformRangedValue.Get produces deterministic
	// results when given a seeded *rand.Rand. Two identically-seeded
	// generators should produce the same sequence.
	r1 := rand.New(rand.NewSource(99))
	r2 := rand.New(rand.NewSource(99))
	v := UniformRangedValue{Min: 0, Max: 100}

	results1 := make([]float32, 10)
	results2 := make([]float32, 10)
	for i := range results1 {
		results1[i] = v.Get(r1)
		results2[i] = v.Get(r2)
	}
	if diff := cmp.Diff(results1, results2); diff != "" {
		t.Errorf("UniformRangedValue.Get not deterministic with seeded rand (-r1 +r2):\n%s", diff)
	}
}
