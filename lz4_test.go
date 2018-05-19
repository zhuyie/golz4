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
	plainText := []byte("This is a test string")
	t.Logf("plainText = %v", plainText)

	compressed := Compress(nil, plainText)
	if len(compressed) == 0 {
		t.Error("Compress failed")
	}
	t.Logf("compressed = %v", compressed)

	var err error
	decompressed := make([]byte, 0, len(plainText))
	decompressed, err = Decompress(decompressed, compressed)
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
	}
	t.Logf("decompressed = %v", decompressed)

	if !bytes.Equal(decompressed, plainText) {
		t.Error("decompressed != plainText")
	}
}
