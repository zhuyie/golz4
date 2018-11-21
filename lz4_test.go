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

func testCompress(t *testing.T, plaintext []byte) {
	compressed := make([]byte, CompressBound(len(plaintext)))
	n, err := Compress(compressed, plaintext)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	compressed = compressed[:n]
	t.Logf("compressed=%v, len=%v", compressed, len(compressed))

	decompressed := make([]byte, len(plaintext))
	n, err = Decompress(decompressed, compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	decompressed = decompressed[:n]
	t.Logf("decompressed=%v, len=%v", decompressed, len(decompressed))

	if !bytes.Equal(decompressed, plaintext) {
		t.Fatalf("decompressed != plaintext")
	}
}

func TestCompress(t *testing.T) {
	testCompress(t, plaintext0)
	testCompress(t, plaintext1)
	testCompress(t, plaintext2)
}

func TestCompressError(t *testing.T) {
	var err error

	// src is empty
	_, err = Compress(nil, nil)
	if err != nil {
		t.Errorf("Expect nil, got %v", err)
	}
	_, err = Decompress(nil, nil)
	if err != nil {
		t.Errorf("Expect nil, got %v", err)
	}

	src := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	// dst not large enough
	_, err = Compress(nil, src)
	if err != ErrCompress {
		t.Errorf("Expect ErrCompress, got %v", err)
	}
	compressed := make([]byte, CompressBound(len(src)))
	n, err := Compress(compressed, src)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	compressed = compressed[:n]

	// dst not large enough
	dst := make([]byte, 2)
	_, err = Decompress(dst, compressed)
	if err != ErrDecompress {
		t.Errorf("Expect ErrDecompress, got %v", err)
	}

	// src data is malformed
	compressed[0] = 0
	compressed[1] = 0
	dst2 := make([]byte, len(src))
	_, err = Decompress(dst2, compressed)
	if err != ErrDecompress {
		t.Errorf("Expect ErrDecompress, got %v", err)
	}
}

func BenchmarkCompress(b *testing.B) {
	dst := make([]byte, CompressBound(len(plaintext0)))
	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		_, err := Compress(dst, plaintext0)
		if err != nil {
			b.Errorf("Compress error: %v", err)
		}
	}
}

func BenchmarkDecompress(b *testing.B) {
	compressed := make([]byte, CompressBound(len(plaintext0)))
	n, err := Compress(compressed, plaintext0)
	if err != nil {
		b.Errorf("Compress error: %v", err)
	}
	compressed = compressed[:n]

	dst := make([]byte, len(plaintext0))
	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		_, err := Decompress(dst, compressed)
		if err != nil {
			b.Errorf("Decompress error: %v", err)
		}
	}
}

func TestContinueCompress(t *testing.T) {
	cc := NewContinueCompress(32*1024, 4096)
	cd := NewContinueDecompress(32*1024, 4096)

	var compressed []byte
	var allCompressed [][]byte
	var n int

	vvv := []byte("abcdefghijklmnoptrstuvwxyz")
	err := cc.Write(vvv)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if cc.MsgLen() != len(vvv) {
		t.Errorf("Except %v, got %v", len(vvv), cc.MsgLen())
	}
	err = cc.Write([]byte("1234"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	compressed = make([]byte, CompressBound(cc.MsgLen()))
	n, err = cc.Process(compressed)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}
	compressed = compressed[:n]
	allCompressed = append(allCompressed, compressed)

	for i := 0; i < 5000; i++ {
		err = cc.Write([]byte("abcdefghijklmnoptrstuvwxyz1234"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		compressed = make([]byte, CompressBound(cc.MsgLen()))
		n, err = cc.Process(compressed)
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}
		compressed = compressed[:n]
		allCompressed = append(allCompressed, compressed)
	}

	processTimes, totalSrcLen, totalCompressedLen := cc.Stats()
	if processTimes != 5001 {
		t.Errorf("Expect %v, got %v", 5001, processTimes)
	}
	ratio := float64(totalCompressedLen) / float64(totalSrcLen) * 100.0
	t.Logf("totalSrcLen=%v, totalCompressedLen=%v, ratio=%.1f%%", totalSrcLen, totalCompressedLen, ratio)

	decompressBuf := make([]byte, 4096)
	for _, compressed = range allCompressed {
		_, err = cd.Process(decompressBuf, compressed)
		if err != nil {
			t.Errorf("Process failed: %v", err)
		}
	}
	processTimes, totalSrcLen, totalDecompressedLen := cd.Stats()
	if processTimes != 5001 {
		t.Errorf("Expect %v, got %v", 5001, processTimes)
	}
	ratio = float64(totalSrcLen) / float64(totalDecompressedLen) * 100.0
	t.Logf("totalSrcLen=%v, totalDecompressedLen=%v, ratio=%.1f%%", totalSrcLen, totalDecompressedLen, ratio)

	// Let finalizer run...
	cc = nil
	cd = nil
	runtime.GC()
}

func TestContinueCompressMisc(t *testing.T) {
	cc := NewContinueCompress(32*1024, 4096)
	defer cc.Release()
	cd := NewContinueDecompress(32*1024, 4096)
	defer cd.Release()

	if cc.DictionarySize() != 32*1024 {
		t.Errorf("Except %v, got %v", 32*1024, cc.DictionarySize())
	}
	if cc.MaxMessageSize() != 4096 {
		t.Errorf("Except %v, got %v", 4096, cc.MaxMessageSize())
	}
	if cd.DictionarySize() != 32*1024 {
		t.Errorf("Except %v, got %v", 32*1024, cd.DictionarySize())
	}
	if cd.MaxMessageSize() != 4096 {
		t.Errorf("Except %v, got %v", 4096, cd.MaxMessageSize())
	}
}

func TestContinueCompressError(t *testing.T) {
	cc := NewContinueCompress(0, 0)
	defer cc.Release()

	_, err := cc.Process(nil)
	if err != ErrNoData {
		t.Errorf("Expect %v, got %v", ErrNoData, err)
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

func TestContinueDecompressError(t *testing.T) {
	cd := NewContinueDecompress(0, 0)
	defer cd.Release()

	_, err := cd.Process(nil, nil)
	if err != ErrNoData {
		t.Errorf("Expect %v, got %v", ErrNoData, err)
	}

	_, err = cd.Process(nil, []byte("a"))
	if err != ErrDecompress {
		t.Errorf("Expect %v, got %v", ErrDecompress, err)
	}
}

func BenchmarkContinueCompressDecompress(b *testing.B) {
	cc := NewContinueCompress(32*1024, 4096)
	defer cc.Release()

	cd := NewContinueDecompress(32*1024, 4096)
	defer cd.Release()

	var err error
	var n int
	compressed := make([]byte, CompressBound(len(plaintext0)))
	decompressed := make([]byte, 4096)

	b.SetBytes(int64(len(plaintext0)))
	for i := 0; i < b.N; i++ {
		err = cc.Write(plaintext0)
		if err != nil {
			b.Errorf("Write error: %v", err)
		}
		n, err = cc.Process(compressed)
		if err != nil {
			b.Errorf("Process error: %v", err)
		}

		n, err = cd.Process(decompressed, compressed[:n])
		if err != nil {
			b.Errorf("Process error: %v", err)
		}

		if i%100 == 0 {
			if !bytes.Equal(plaintext0, decompressed[:n]) {
				b.Error("decompressed != plaintext0")
			}
		}
	}
}
