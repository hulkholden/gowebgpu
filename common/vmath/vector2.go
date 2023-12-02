package vmath

import (
	"fmt"

	"github.com/hulkholden/gowebgpu/common/math32"
)

type V2 struct {
	X, Y float32
}

func NewV2(x, y float32) V2 { return V2{X: x, Y: y} }

func NewV2FromAngle(a float32) V2 {
	s, c := math32.SinCos(a)
	return NewV2(-s, +c)
}

func (v V2) String() string {
	return fmt.Sprintf("{%f, %f}", v.X, v.Y)
}

func (v V2) Add(w V2) V2             { return V2{X: v.X + w.X, Y: v.Y + w.Y} }
func (v V2) Sub(w V2) V2             { return V2{X: v.X - w.X, Y: v.Y - w.Y} }
func (v V2) Distance(w V2) float32   { return math32.Sqrt(v.DistanceSq(w)) }
func (v V2) DistanceSq(w V2) float32 { return v.Sub(w).LengthSq() }
func (v V2) Negate() V2              { return V2{X: -v.X, Y: -v.Y} }
func (v V2) Scale(s float32) V2      { return V2{X: v.X * s, Y: v.Y * s} }
func (v V2) Dot(w V2) float32        { return v.X*w.X + v.Y*w.Y }
func (v V2) Cross(w V2) float32      { return v.X*w.Y - v.Y*w.X }
func (v V2) LengthSq() float32       { return v.Dot(v) }
func (v V2) Length() float32         { return math32.Sqrt(v.LengthSq()) }
func (v V2) ToAngle() float32        { return math32.Atan2(-v.X, v.Y) }
func (v V2) Lerp(w V2, f float32) V2 { return v.Scale(1 - f).Add(w.Scale(f)) }

func (v V2) Normal() (V2, float32) {
	d := v.Length()
	return v.Scale(1 / d), d
}

func (v V2) Rotate(a float32) V2 {
	s, c := math32.SinCos(a)
	return V2{X: c*v.X - s*v.Y, Y: s*v.X + c*v.Y}
}

func (v V2) Min(w V2) V2 {
	return V2{X: math32.Min(v.X, w.X), Y: math32.Min(v.Y, w.Y)}
}

func (v V2) Max(w V2) V2 {
	return V2{X: math32.Max(v.X, w.X), Y: math32.Max(v.Y, w.Y)}
}
