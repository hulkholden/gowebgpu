package vmath

type V3 struct {
	X, Y, Z float32
}

func NewV3(x, y, z float32) V3 { return V3{X: x, Y: y, Z: z} }
