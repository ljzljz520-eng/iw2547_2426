package cryptoiface

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidKey  = errors.New("envelope key must contain at least 16 bytes")
	ErrTagMismatch = errors.New("envelope authentication tag mismatch")
)

type Codec struct {
	encryptionKey     []byte
	authenticationKey []byte
}

func NewCodec(key []byte) (*Codec, error) {
	if len(key) < 16 {
		return nil, ErrInvalidKey
	}
	left := sha256.Sum256(append([]byte("room-snapshot/encryption/"), key...))
	right := sha256.Sum256(append([]byte("room-snapshot/authentication/"), key...))
	return &Codec{encryptionKey: left[:], authenticationKey: right[:]}, nil
}

func (codec *Codec) Seal(batchID string, minuteUnix int64, plaintext []byte) ([]byte, []byte, error) {
	if batchID == "" {
		return nil, nil, errors.New("batch id is required")
	}
	nonce := deriveNonce(batchID, minuteUnix)
	ciphertext := xorStream(codec.encryptionKey, nonce, plaintext)
	tag := codec.computeTag(batchID, minuteUnix, ciphertext)
	return ciphertext, tag, nil
}

func (codec *Codec) Open(batchID string, minuteUnix int64, ciphertext, tag []byte) ([]byte, error) {
	expected := codec.computeTag(batchID, minuteUnix, ciphertext)
	if !hmac.Equal(expected, tag) {
		return nil, ErrTagMismatch
	}
	return xorStream(codec.encryptionKey, deriveNonce(batchID, minuteUnix), ciphertext), nil
}

func (codec *Codec) computeTag(batchID string, minuteUnix int64, ciphertext []byte) []byte {
	mac := hmac.New(sha256.New, codec.authenticationKey)
	var stamp [8]byte
	binary.BigEndian.PutUint64(stamp[:], uint64(minuteUnix))
	mac.Write([]byte(batchID))
	mac.Write([]byte{0})
	mac.Write(stamp[:])
	mac.Write(ciphertext)
	return mac.Sum(nil)
}

func deriveNonce(batchID string, minuteUnix int64) []byte {
	digest := sha256.New()
	digest.Write([]byte(batchID))
	var stamp [8]byte
	binary.BigEndian.PutUint64(stamp[:], uint64(minuteUnix))
	digest.Write(stamp[:])
	return digest.Sum(nil)[:16]
}

func xorStream(key, nonce, input []byte) []byte {
	output := make([]byte, len(input))
	counter := uint64(0)
	for offset := 0; offset < len(input); {
		mac := hmac.New(sha256.New, key)
		mac.Write(nonce)
		var value [8]byte
		binary.BigEndian.PutUint64(value[:], counter)
		mac.Write(value[:])
		block := mac.Sum(nil)
		for index := 0; index < len(block) && offset < len(input); index++ {
			output[offset] = input[offset] ^ block[index]
			offset++
		}
		counter++
	}
	return output
}

func (codec *Codec) Verify(batchID string, minuteUnix int64, ciphertext, tag []byte) error {
	if _, err := codec.Open(batchID, minuteUnix, ciphertext, tag); err != nil {
		return fmt.Errorf("verify batch %s: %w", batchID, err)
	}
	return nil
}
