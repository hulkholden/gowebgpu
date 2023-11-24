package vmath

type V4 struct {
	X, Y, Z, W float32
}

func NewV4(x, y, z, w float32) V4 { return V4{X: x, Y: y, Z: z, W: w} }
