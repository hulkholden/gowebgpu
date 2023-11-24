package vmath

type V2 struct {
	X, Y float32
}

func NewV2(x, y float32) V2 { return V2{X: x, Y: y} }
