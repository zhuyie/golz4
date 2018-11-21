package golz4

/*
#cgo CFLAGS: -O3 -Wno-deprecated-declarations
#include "lz4/lib/lz4.h"
#include "lz4/lib/lz4.c"
#include <stdlib.h>

static int LZ4_decompress_safe_continue_and_memcpy(
	LZ4_streamDecode_t* stream,
	const char* src, int srcSize,
	char* dstBuf, int dstBufCapacity,
	char* dst)
{
	// decompress to ringbuffer
	int result = LZ4_decompress_safe_continue(stream, src, dstBuf, srcSize, dstBufCapacity);
	if (result > 0) {
		// copy decompressed data to dst
		memcpy(dst, dstBuf, (size_t)result);
	}
	return result;
}
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

const (
	// MaxInputSize is the max supported input size. see macro LZ4_MAX_INPUT_SIZE.
	MaxInputSize = 0x7E000000 // 2 113 929 216 bytes

	minDictionarySize = 4096
	minMessageSize    = 256
)

// Error codes
var (
	ErrSrcTooLarge = errors.New("src is too large")
	ErrCompress    = errors.New("compress: dst is not large enough")
	ErrDecompress  = errors.New("decompress: dst is not large enough, or src data is malformed")
	ErrNoData      = errors.New("no data")
	ErrInternal    = errors.New("internal error")
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
	if inputSize <= 0 || inputSize > MaxInputSize {
		return 0
	}
	return inputSize + inputSize/255 + 16
}

// Compress compresses `src` into `dst`.
func Compress(dst, src []byte) (n int, err error) {
	if len(src) == 0 {
		return 0, nil
	}
	if len(src) > MaxInputSize {
		return 0, ErrSrcTooLarge
	}
	var result C.int
	if len(dst) > 0 {
		result = C.LZ4_compress_default(
			(*C.char)(unsafe.Pointer(&src[0])),
			(*C.char)(unsafe.Pointer(&dst[0])),
			C.int(len(src)),
			C.int(len(dst)))
	}
	if result <= 0 {
		return 0, ErrCompress
	}
	return int(result), nil
}

// Decompress decompresses `src` into `dst`.
func Decompress(dst, src []byte) (n int, err error) {
	if len(src) == 0 {
		return 0, nil
	}
	var result C.int
	if len(dst) > 0 {
		result = C.LZ4_decompress_safe(
			(*C.char)(unsafe.Pointer(&src[0])),
			(*C.char)(unsafe.Pointer(&dst[0])),
			C.int(len(src)),
			C.int(len(dst)))
	}
	if result <= 0 {
		return 0, ErrDecompress
	}
	return int(result), nil
}

// ContinueCompress implements streaming compression.
type ContinueCompress struct {
	dictionarySize     int
	maxMessageSize     int
	lz4Stream          *C.LZ4_stream_t
	ringBuffer         []byte
	msgLen             int
	processTimes       int64
	totalSrcLen        int64
	totalCompressedLen int64
}

// NewContinueCompress returns a new ContinueCompress object.
// Call Release when the ContinueCompress is no longer needed.
func NewContinueCompress(dictionarySize, maxMessageSize int) *ContinueCompress {
	if dictionarySize < minDictionarySize {
		dictionarySize = minDictionarySize
	}
	if maxMessageSize < minMessageSize {
		maxMessageSize = minMessageSize
	}
	if maxMessageSize > MaxInputSize {
		maxMessageSize = MaxInputSize
	}
	cc := &ContinueCompress{
		dictionarySize: dictionarySize,
		maxMessageSize: maxMessageSize,
		lz4Stream:      C.LZ4_createStream(),
		ringBuffer:     make([]byte, 0, dictionarySize+maxMessageSize),
	}
	runtime.SetFinalizer(cc, freeContinueCompress)
	return cc
}

func freeContinueCompress(v interface{}) {
	v.(*ContinueCompress).Release()
}

// Release releases all the resources occupied by cc.
// cc cannot be used after the release.
func (cc *ContinueCompress) Release() {
	if cc.lz4Stream != nil {
		C.LZ4_freeStream(cc.lz4Stream)
		cc.lz4Stream = nil
	}
}

// DictionarySize returns the dictionary size.
func (cc *ContinueCompress) DictionarySize() int {
	return cc.dictionarySize
}

// MaxMessageSize returns the max message size.
func (cc *ContinueCompress) MaxMessageSize() int {
	return cc.maxMessageSize
}

// Write writes src to cc.
func (cc *ContinueCompress) Write(src []byte) error {
	srcLen := len(src)
	if srcLen == 0 {
		return nil
	}
	if cc.msgLen+srcLen > cc.maxMessageSize {
		return ErrSrcTooLarge
	}
	if len(cc.ringBuffer)+srcLen > cap(cc.ringBuffer) {
		return ErrInternal
	}
	cc.ringBuffer = append(cc.ringBuffer, src...)
	cc.msgLen += srcLen
	return nil
}

// MsgLen returns the length of buffered data.
func (cc *ContinueCompress) MsgLen() int {
	return cc.msgLen
}

// Process compresses buffered data into `dst`.
// len(dst) should >= CompressBound(cc.MsgLen()).
func (cc *ContinueCompress) Process(dst []byte) (int, error) {
	if cc.msgLen == 0 {
		return 0, ErrNoData
	}
	if cc.msgLen > len(cc.ringBuffer) {
		return 0, ErrInternal
	}
	if len(dst) < CompressBound(cc.msgLen) {
		return 0, ErrDecompress
	}

	offset := len(cc.ringBuffer) - cc.msgLen
	result := C.LZ4_compress_fast_continue(
		cc.lz4Stream,
		(*C.char)(unsafe.Pointer(&cc.ringBuffer[offset])),
		(*C.char)(unsafe.Pointer(&dst[0])),
		C.int(cc.msgLen),
		C.int(len(dst)),
		1)
	if result <= 0 {
		return 0, ErrInternal
	}

	// Update stats
	cc.processTimes++
	cc.totalSrcLen += int64(cc.msgLen)
	cc.totalCompressedLen += int64(result)

	// Add and wraparound the ringbuffer offset
	if len(cc.ringBuffer) >= cc.dictionarySize {
		cc.ringBuffer = cc.ringBuffer[0:0]
	}
	// Reset msgLen
	cc.msgLen = 0

	return int(result), nil
}

// Stats returns statistics data.
func (cc *ContinueCompress) Stats() (processTimes, totalSrcLen, totalCompressedLen int64) {
	return cc.processTimes, cc.totalSrcLen, cc.totalCompressedLen
}

// ContinueDecompress implements streaming decompression.
type ContinueDecompress struct {
	dictionarySize       int
	maxMessageSize       int
	lz4Stream            *C.LZ4_streamDecode_t
	ringBuffer           []byte
	offset               int
	processTimes         int64
	totalSrcLen          int64
	totalDecompressedLen int64
}

// NewContinueDecompress returns a new ContinueDecompress object.
//
// dictionarySize and maxMessageSize must be exactly the same as NewContinueCompress.
// see LZ4_decompress_*_continue() - Synchronized mode.
//
// Call Release when the ContinueDecompress is no longer needed.
func NewContinueDecompress(dictionarySize, maxMessageSize int) *ContinueDecompress {
	if dictionarySize < minDictionarySize {
		dictionarySize = minDictionarySize
	}
	if maxMessageSize < minMessageSize {
		maxMessageSize = minMessageSize
	}
	if maxMessageSize > MaxInputSize {
		maxMessageSize = MaxInputSize
	}
	cd := &ContinueDecompress{
		dictionarySize: dictionarySize,
		maxMessageSize: maxMessageSize,
		lz4Stream:      C.LZ4_createStreamDecode(),
		ringBuffer:     make([]byte, dictionarySize+maxMessageSize),
	}
	runtime.SetFinalizer(cd, freeContinueDecompress)
	return cd
}

func freeContinueDecompress(v interface{}) {
	v.(*ContinueDecompress).Release()
}

// Release releases all the resources occupied by cd.
// cd cannot be used after the release.
func (cd *ContinueDecompress) Release() {
	if cd.lz4Stream != nil {
		C.LZ4_freeStreamDecode(cd.lz4Stream)
		cd.lz4Stream = nil
	}
}

// DictionarySize returns the dictionary size.
func (cd *ContinueDecompress) DictionarySize() int {
	return cd.dictionarySize
}

// MaxMessageSize returns the max message size.
func (cd *ContinueDecompress) MaxMessageSize() int {
	return cd.maxMessageSize
}

// Process decompresses `src` into `dst`.
// len(dst) should >= uncompressed_length_of_src_data.
func (cd *ContinueDecompress) Process(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, ErrNoData
	}
	if cd.offset < 0 || cd.offset >= cd.dictionarySize {
		return 0, ErrInternal
	}

	// Decompress to ringbuffer, then copy to dst
	var result C.int
	if dstLen := len(dst); dstLen > 0 {
		if dstLen > cd.maxMessageSize {
			dstLen = cd.maxMessageSize
		}
		result = C.LZ4_decompress_safe_continue_and_memcpy(
			cd.lz4Stream,
			(*C.char)(unsafe.Pointer(&src[0])),
			C.int(len(src)),
			(*C.char)(unsafe.Pointer(&cd.ringBuffer[cd.offset])),
			C.int(dstLen),
			(*C.char)(unsafe.Pointer(&dst[0])))
	}
	if result <= 0 {
		return 0, ErrDecompress
	}

	// Update stats
	cd.processTimes++
	cd.totalSrcLen += int64(len(src))
	cd.totalDecompressedLen += int64(result)

	// Add and wraparound the ringbuffer offset
	cd.offset += int(result)
	if cd.offset >= cd.dictionarySize {
		cd.offset = 0
	}

	return int(result), nil
}

// Stats returns statistics data.
func (cd *ContinueDecompress) Stats() (processTimes, totalSrcLen, totalDecompressedLen int64) {
	return cd.processTimes, cd.totalSrcLen, cd.totalDecompressedLen
}
