package cryptoiface

import (
	"bytes"
	"testing"
)

func TestCEnvelopeRoundTrip(t *testing.T) {
	codec, err := NewCodec([]byte("0123456789abcdef0123456789"))
	if err != nil {
		t.Fatal(err)
	}
	frame, err := SealToCBuffer(codec, "batch-a", 1234, []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	id, stamp, plain, err := OpenFromCBuffer(codec, frame)
	if err != nil {
		t.Fatal(err)
	}
	if id != "batch-a" || stamp != 1234 || !bytes.Equal(plain, []byte("payload")) {
		t.Fatalf("round trip failed")
	}
}
func TestTagRejectsMutation(t *testing.T) {
	codec, _ := NewCodec([]byte("0123456789abcdef0123456789"))
	ciphertext, tag, _ := codec.Seal("b", 1, []byte("payload"))
	ciphertext[0] ^= 1
	if _, err := codec.Open("b", 1, ciphertext, tag); err == nil {
		t.Fatal("expected tag failure")
	}
}
