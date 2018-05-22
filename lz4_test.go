package golz4

import (
	"bytes"
	"runtime"
	"testing"
)

var (
	plaintext0 = []byte("aldkjflakdjf.,asdfjlkjlakjdskkkkkkkkkkkkkk")
	plaintext1 = []byte("AAAAAAAAAAAAAAAA")
	plaintext2 = []byte("1234567890")
)

func TestVersion(t *testing.T) {
	num := VersionNumber()
	if num != 10802 {
		t.Errorf("expect %v, got %v", 10802, num)
	}
	str := VersionString()
	if str != "1.8.2" {
		t.Errorf("expect %v, got %v", "1.8.2", str)
	}
}

func TestCompressBound(t *testing.T) {
	n := CompressBound(255)
	if n != 272 {
		t.Errorf("Expect %v, got %v", 272, n)
	}

	n = CompressBound(MaxInputSize + 1)
	if n != 0 {
		t.Errorf("Expect %v, got %v", 0, n)
	}
}

func TestCompress(t *testing.T) {
	var l [3]int
	decompressed := make([]byte, 0, len(plaintext0))

	compressed, err := Compress(nil, plaintext0)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	l[0] = len(compressed)

	compressed, err = Compress(compressed, plaintext1)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	l[1] = len(compressed)

	compressed, err = Compress(compressed, plaintext2)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	l[2] = len(compressed)

	decompressed, err = Decompress(decompressed, compressed[:l[0]])
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed, plaintext0) {
		t.Error("decompressed != plaintext0")
	}

	decompressed = decompressed[0:0]
	decompressed, err = Decompress(decompressed, compressed[l[0]:l[1]])
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed, plaintext1) {
		t.Error("decompressed != plaintext1")
	}

	decompressed = decompressed[0:0]
	decompressed, err = Decompress(decompressed, compressed[l[1]:l[2]])
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed, plaintext2) {
		t.Error("decompressed != plaintext1")
	}
}

func TestCompressError(t *testing.T) {
	var err error

	// src is empty
	_, err = Compress(nil, nil)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	_, err = Decompress(nil, nil)
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}

	src := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	compressed, err := Compress(nil, src)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}

	// dst is full
	dst := make([]byte, 1, 1)
	_, err = Decompress(dst, src)
	if err == nil {
		t.Errorf("Expect error, got nil")
	}

	// dst not large enough
	dst2 := make([]byte, 0, 2)
	_, err = Decompress(dst2, compressed)
	if err == nil {
		t.Errorf("Expect error, got nil")
	}
}

func BenchmarkCompressFast(b *testing.B) {
	var err error
	dst := make([]byte, 0, CompressBound(len(plaintext0)))
	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		dst, err = Compress(dst, plaintext0)
		if err != nil {
			b.Errorf("Compress error: %v", err)
		}
		dst = dst[0:0]
	}
}

func BenchmarkCompressSlow(b *testing.B) {
	var err error
	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		_, err = Compress(nil, plaintext0)
		if err != nil {
			b.Errorf("Compress error: %v", err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	compressed, err := Compress(nil, plaintext0)
	if err != nil {
		b.Errorf("Compress error: %v", err)
	}

	dst := make([]byte, 0, len(plaintext0))
	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		dst, err = Decompress(dst, compressed)
		if err != nil {
			b.Errorf("Decompress error: %v", err)
		}
		dst = dst[0:0]
	}
}

func TestContinueCompress(t *testing.T) {
	cc := NewContinueCompress(32*1024, 4096)

	err := cc.Write([]byte("abcdefghijklmnoptrstuvwxyz"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	err = cc.Write([]byte("1234"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	_, err = cc.Process(nil)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	for i := 0; i < 5000; i++ {
		err = cc.Write([]byte("abcdefghijklmnoptrstuvwxyz1234"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		_, err = cc.Process(nil)
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}
	}

	processTimes, totalSrcLen, totalCompressedLen := cc.Stats()
	if processTimes != 5001 {
		t.Errorf("Expect %v, got %v", 5001, processTimes)
	}
	ratio := float64(totalCompressedLen) / float64(totalSrcLen) * 100.0
	t.Logf("totalSrcLen=%v, totalCompressedLen=%v, ratio=%.1f%%", totalSrcLen, totalCompressedLen, ratio)

	// Let finalizer run...
	cc = nil
	runtime.GC()
}

func TestContinueCompressError(t *testing.T) {
	cc := NewContinueCompress(0, 0)
	defer cc.Release()

	_, err := cc.Process(nil)
	if err != ErrNoData {
		t.Errorf("Expect %v, got %v", ErrSrcTooLarge, err)
	}

	err = cc.Write(nil)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	largeMsg := make([]byte, 257) // default maxMessageSize is 256
	err = cc.Write(largeMsg)
	if err != ErrSrcTooLarge {
		t.Errorf("Expect %v, got %v", ErrSrcTooLarge, err)
	}
}

func BenchmarkContinueCompress(b *testing.B) {
	cc := NewContinueCompress(32*1024, 4096)
	defer cc.Release()

	var err error
	dst := make([]byte, 0, CompressBound(len(plaintext0)))

	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		err = cc.Write(plaintext0)
		if err != nil {
			b.Errorf("Write error: %v", err)
		}
		dst, err = cc.Process(dst)
		if err != nil {
			b.Errorf("Process error: %v", err)
		}
		dst = dst[0:0]
	}
}
