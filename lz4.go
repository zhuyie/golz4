package golz4

// #cgo CFLAGS: -Wno-deprecated-declarations
// #include "lz4/lib/lz4.h"
// #include "lz4/lib/lz4.c"
import "C"

import (
	"errors"
	"unsafe"
)

const (
	LZ4_MAX_INPUT_SIZE = 0x7E000000
)

// VersionNumber returns the library version number.
// see macro LZ4_VERSION_NUMBER.
func VersionNumber() int {
	return int(C.LZ4_versionNumber())
}

// VersionString returns the library version string.
// see macro LZ4_LIB_VERSION.
func VersionString() string {
	return C.GoString(C.LZ4_versionString())
}

// CompressBound returns the maximum size that LZ4 compression may output
// in a "worst case" scenario (input data not compressible).
// see macro LZ4_COMPRESSBOUND.
func CompressBound(inputSize int) int {
	if inputSize > LZ4_MAX_INPUT_SIZE {
		return 0
	}
	return inputSize + inputSize/255 + 16
}

// Compress appends compressed src to dst and returns the result.
func Compress(dst, src []byte) []byte {
	if len(src) == 0 {
		return dst
	}

	dstLen := len(dst)
	if cap(dst) >= dstLen+len(src) {
		// Fast path
		dst = dst[:cap(dst)]
		result := C.LZ4_compress_default(
			(*C.char)(unsafe.Pointer(&src[0])),
			(*C.char)(unsafe.Pointer(&dst[dstLen])),
			C.int(len(src)),
			C.int(cap(dst)-dstLen))
		compressedSize := int(result)
		if compressedSize > 0 {
			return dst[:dstLen+compressedSize]
		}
	}

	// Slow path
	compressBound := CompressBound(len(src))
	for cap(dst)-dstLen < compressBound {
		dst = append(dst, 0)
	}
	dst = dst[:cap(dst)]
	result := C.LZ4_compress_default(
		(*C.char)(unsafe.Pointer(&src[0])),
		(*C.char)(unsafe.Pointer(&dst[dstLen])),
		C.int(len(src)),
		C.int(cap(dst)-dstLen))
	// Compression is guaranteed to succeed if 'dstCapacity' >= LZ4_compressBound(srcSize)
	compressedSize := int(result)
	return dst[:dstLen+compressedSize]
}

// Decompress appends decompressed src to dst and returns the result.
func Decompress(dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return dst, nil
	}
	dstLen := len(dst)
	if cap(dst) == dstLen {
		return dst, errors.New("dst is not large enough")
	}
	dst = dst[:cap(dst)]
	result := C.LZ4_decompress_safe(
		(*C.char)(unsafe.Pointer(&src[0])),
		(*C.char)(unsafe.Pointer(&dst[dstLen])),
		C.int(len(src)),
		C.int(cap(dst)-dstLen))
	decompressedSize := int(result)
	if decompressedSize >= 0 {
		return dst[:dstLen+decompressedSize], nil
	}
	return dst[:dstLen], errors.New("dst is not large enough, or src data is malformed")
}
