package engine

import (
	"runtime"
	"syscall/js"
	"unsafe"
)

var (
	objectCtor       = js.Global().Get("Object")
	arrayBufferCtor  = js.Global().Get("ArrayBuffer")
	uint8ArrayCtor   = js.Global().Get("Uint8Array")
	float32ArrayCtor = js.Global().Get("Float32Array")
)

// sliceAsBytesSlice reinterprets the provided slice of data as a []byte.
// See https://github.com/golang/go/issues/32402.
func sliceAsBytesSlice[T any](data []T) []byte {
	if len(data) == 0 {
		return nil
	}
	var zero T
	bytePtr := (*byte)(unsafe.Pointer(&data[0]))
	byteLen := len(data) * int(unsafe.Sizeof(zero))
	bytes := unsafe.Slice(bytePtr, byteLen)
	runtime.KeepAlive(data)
	return bytes
}

// structAsByteSlice reinterprets the provided sturct as a byte[].
func structAsByteSlice[T any](data T) []byte {
	bytePtr := (*byte)(unsafe.Pointer(&data))
	byteLen := unsafe.Sizeof(data)
	bytes := unsafe.Slice(bytePtr, byteLen)
	runtime.KeepAlive(data)
	return bytes
}

// ByteSliceAsStructSlice reinterprets the provided sturct as a byte[].
func ByteSliceAsStructSlice[T any](data []byte) []T {
	ptr := (*T)(unsafe.Pointer(&data[0]))
	var zero T
	l := uintptr(len(data)) / unsafe.Sizeof(zero)
	s := unsafe.Slice(ptr, l)
	runtime.KeepAlive(data)
	return s
}

func setFloat32Array(f32arr js.Value, values []float32) {
	// Ideally we could all something like: `f32arr.Call("set", vertexBufferData)`
	// but the js only handles []any.
	for i := 0; i < len(values); i++ {
		f32arr.SetIndex(i, values[i])
	}
}
