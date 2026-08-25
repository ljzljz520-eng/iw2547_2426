package cryptoiface

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const ABIVersion uint16 = 1

var ErrInvalidFrame = errors.New("invalid c envelope frame")

type CEnvelope struct {
	Version    uint16
	BatchID    string
	MinuteUnix int64
	Ciphertext []byte
	Tag        []byte
}

func MarshalCEnvelope(value CEnvelope) ([]byte, error) {
	if value.Version != ABIVersion {
		return nil, fmt.Errorf("%w: version", ErrInvalidFrame)
	}
	if value.BatchID == "" || len(value.BatchID) > 65535 {
		return nil, fmt.Errorf("%w: batch id", ErrInvalidFrame)
	}
	if len(value.Ciphertext) > int(^uint32(0)) || len(value.Tag) > 65535 {
		return nil, fmt.Errorf("%w: component size", ErrInvalidFrame)
	}
	size := 2 + 2 + len(value.BatchID) + 8 + 4 + len(value.Ciphertext) + 2 + len(value.Tag)
	frame := make([]byte, size)
	offset := 0
	binary.BigEndian.PutUint16(frame[offset:], value.Version)
	offset += 2
	binary.BigEndian.PutUint16(frame[offset:], uint16(len(value.BatchID)))
	offset += 2
	copy(frame[offset:], value.BatchID)
	offset += len(value.BatchID)
	binary.BigEndian.PutUint64(frame[offset:], uint64(value.MinuteUnix))
	offset += 8
	binary.BigEndian.PutUint32(frame[offset:], uint32(len(value.Ciphertext)))
	offset += 4
	copy(frame[offset:], value.Ciphertext)
	offset += len(value.Ciphertext)
	binary.BigEndian.PutUint16(frame[offset:], uint16(len(value.Tag)))
	offset += 2
	copy(frame[offset:], value.Tag)
	return frame, nil
}

func UnmarshalCEnvelope(frame []byte) (CEnvelope, error) {
	var value CEnvelope
	if len(frame) < 18 {
		return value, fmt.Errorf("%w: truncated header", ErrInvalidFrame)
	}
	offset := 0
	value.Version = binary.BigEndian.Uint16(frame[offset:])
	offset += 2
	if value.Version != ABIVersion {
		return value, fmt.Errorf("%w: version %d", ErrInvalidFrame, value.Version)
	}
	idSize := int(binary.BigEndian.Uint16(frame[offset:]))
	offset += 2
	if idSize == 0 || offset+idSize+14 > len(frame) {
		return value, fmt.Errorf("%w: batch id", ErrInvalidFrame)
	}
	value.BatchID = string(frame[offset : offset+idSize])
	offset += idSize
	value.MinuteUnix = int64(binary.BigEndian.Uint64(frame[offset:]))
	offset += 8
	payloadSize := int(binary.BigEndian.Uint32(frame[offset:]))
	offset += 4
	if payloadSize < 0 || offset+payloadSize+2 > len(frame) {
		return value, fmt.Errorf("%w: payload", ErrInvalidFrame)
	}
	value.Ciphertext = append([]byte(nil), frame[offset:offset+payloadSize]...)
	offset += payloadSize
	tagSize := int(binary.BigEndian.Uint16(frame[offset:]))
	offset += 2
	if tagSize == 0 || offset+tagSize != len(frame) {
		return value, fmt.Errorf("%w: tag", ErrInvalidFrame)
	}
	value.Tag = append([]byte(nil), frame[offset:]...)
	return value, nil
}

func SealToCBuffer(codec *Codec, batchID string, minuteUnix int64, plaintext []byte) ([]byte, error) {
	ciphertext, tag, err := codec.Seal(batchID, minuteUnix, plaintext)
	if err != nil {
		return nil, err
	}
	return MarshalCEnvelope(CEnvelope{Version: ABIVersion, BatchID: batchID, MinuteUnix: minuteUnix, Ciphertext: ciphertext, Tag: tag})
}
func OpenFromCBuffer(codec *Codec, frame []byte) (string, int64, []byte, error) {
	value, err := UnmarshalCEnvelope(frame)
	if err != nil {
		return "", 0, nil, err
	}
	plain, err := codec.Open(value.BatchID, value.MinuteUnix, value.Ciphertext, value.Tag)
	return value.BatchID, value.MinuteUnix, plain, err
}
