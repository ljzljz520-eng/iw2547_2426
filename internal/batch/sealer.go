package batch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/model"
)

type Sealer struct{ codec *cryptoiface.Codec }

func NewSealer(codec *cryptoiface.Codec) *Sealer { return &Sealer{codec: codec} }

func DeterministicBatchID(minute time.Time, details []model.SnapshotDetail) string {
	digest := sha256.New()
	digest.Write([]byte(model.NormalizeMinute(minute).Format(time.RFC3339)))
	for _, detail := range details {
		digest.Write([]byte(fmt.Sprintf("/%d/%s", detail.Sequence, detail.Kind)))
		if detail.Reading != nil {
			digest.Write([]byte(detail.Reading.ID))
		}
		if detail.Alert != nil {
			digest.Write([]byte(detail.Alert.ID))
		}
	}
	return "batch-" + hex.EncodeToString(digest.Sum(nil)[:8])
}

func (sealer *Sealer) Seal(minute time.Time, details []model.SnapshotDetail) (model.SnapshotBatch, error) {
	minute = model.NormalizeMinute(minute)
	plain, err := model.EncodeDetails(details)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	id := DeterministicBatchID(minute, details)
	ciphertext, tag, err := sealer.codec.Seal(id, minute.Unix(), plain)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	value := model.SnapshotBatch{ID: id, Minute: minute, DetailCount: len(details), Details: append([]model.SnapshotDetail(nil), details...), Payload: ciphertext, Tag: tag, Status: "sealed"}
	if err := model.ValidateBatch(value); err != nil {
		return model.SnapshotBatch{}, err
	}
	return value, nil
}

func (sealer *Sealer) Open(value model.SnapshotBatch) ([]model.SnapshotDetail, error) {
	plain, err := sealer.codec.Open(value.ID, value.Minute.Unix(), value.Payload, value.Tag)
	if err != nil {
		return nil, err
	}
	details, err := model.DecodeDetails(plain)
	if err != nil {
		return nil, err
	}
	if len(details) != value.DetailCount {
		return nil, fmt.Errorf("detail count: expected %d got %d", value.DetailCount, len(details))
	}
	return details, nil
}

func (sealer *Sealer) Frame(value model.SnapshotBatch) ([]byte, error) {
	return cryptoiface.MarshalCEnvelope(cryptoiface.CEnvelope{Version: cryptoiface.ABIVersion, BatchID: value.ID, MinuteUnix: value.Minute.Unix(), Ciphertext: value.Payload, Tag: value.Tag})
}
func (sealer *Sealer) OpenFrame(frame []byte) (model.SnapshotBatch, error) {
	value, err := cryptoiface.UnmarshalCEnvelope(frame)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	plain, err := sealer.codec.Open(value.BatchID, value.MinuteUnix, value.Ciphertext, value.Tag)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	details, err := model.DecodeDetails(plain)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	return model.SnapshotBatch{ID: value.BatchID, Minute: time.Unix(value.MinuteUnix, 0).UTC(), DetailCount: len(details), Details: details, Payload: value.Ciphertext, Tag: value.Tag, Status: "sealed"}, nil
}
