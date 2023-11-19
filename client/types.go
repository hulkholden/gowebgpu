package main

import (
	"runtime"
	"unsafe"
)

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

// asByteSlice reinterprets the provided slice of data as a []byte.
// See https://github.com/golang/go/issues/32402.
func asByteSlice[T numeric](data []T) []byte {
	if len(data) == 0 {
		return nil
	}
	bytePtr := (*byte)(unsafe.Pointer(&data[0]))
	byteLen := len(data) * int(unsafe.Sizeof(T(0)))
	bytes := unsafe.Slice(bytePtr, byteLen)
	runtime.KeepAlive(data)
	return bytes
}
