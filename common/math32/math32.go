package math32

import (
	"math"
	"math/rand"
)

const (
	Pi      = float32(math.Pi)
	TwoPi   = float32(2. * Pi)
	Epsilon = math.SmallestNonzeroFloat32
)

func NaN() float32 {
	return float32(math.NaN())
}

func Abs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func MagnitudeAndSign(x float32) (float32, float32) {
	if x < 0 {
		return -x, -1
	} else if x > 0 {
		return +x, +1
	}
	return 0, 0
}

func Floor(x float32) float32 {
	return float32(math.Floor(float64(x)))
}

func Sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

func Sin(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

func Cos(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

func SinCos(x float32) (float32, float32) {
	return Sin(x), Cos(x)
}

func Atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func Pow(x, y float32) float32 {
	return float32(math.Pow(float64(x), float64(y)))
}

func Mod(x, y float32) float32 {
	return float32(math.Mod(float64(x), float64(y)))
}

func Modf(x float32) (float32, float32) {
	i, f := math.Modf(float64(x))
	return float32(i), float32(f)
}

func Min(x, y float32) float32 {
	if x < y {
		return x
	}
	return y
}

func Max(x, y float32) float32 {
	if x > y {
		return x
	}
	return y
}

func Clamp(x, min, max float32) float32 {
	return Max(Min(x, max), min)
}

func Lerp(a, b, f float32) float32 {
	return a*(1-f) + b*f
}

func RadiansFromDegrees(degrees float32) float32 {
	return (degrees * TwoPi) / 360
}

func DegreesFromRadians(radians float32) float32 {
	return (radians * 360) / TwoPi
}

// NormalizeAngle normalizes an angle and ensures it lies in the range [-Pi,+Pi]
func NormalizeAngle(a float32) float32 {
	n := Mod(a+Pi, TwoPi)
	// Handle negative results - Mod returns a result with the same sign as a.
	if n < 0 {
		n += 2 * Pi
	}
	return float32(n - Pi)
}

func AngleDiff(from, to float32) float32 {
	return NormalizeAngle(to - from)
}

type UniformRangedValue struct {
	Min, Max float32
}

func (v UniformRangedValue) Get(r *rand.Rand) float32 {
	return v.Min + (rand.Float32() * (v.Max - v.Min))
}

type NormallyDistributedValue struct {
	Mean, StdDev float32
}

func (v NormallyDistributedValue) Get(r *rand.Rand) float32 {
	return v.Mean + float32(r.NormFloat64())*v.StdDev
}
