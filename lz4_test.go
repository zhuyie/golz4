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
