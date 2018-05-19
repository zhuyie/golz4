package golz4

import (
	"bytes"
	"testing"
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

func TestCompress(t *testing.T) {
	plaintext := []byte("AAAAAAAAAAAAAAAA")
	t.Logf("plaintext = %v", plaintext)

	compressed, err := Compress(nil, plaintext)
	if err != nil {
		t.Errorf("Compress failed: %v", err)
	}
	t.Logf("compressed = %v", compressed)

	decompressed := make([]byte, 0, len(plaintext))
	decompressed, err = Decompress(decompressed, compressed)
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}
	t.Logf("decompressed = %v", decompressed)

	if !bytes.Equal(decompressed, plaintext) {
		t.Error("decompressed != plaintext")
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
