package verify

import (
	"fmt"
	"time"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/model"
)

type Verifier struct {
	sealer *batch.Sealer
	now    func() time.Time
}

func New(sealer *batch.Sealer, now func() time.Time) *Verifier {
	if now == nil {
		now = func() time.Time { return time.Unix(0, 0).UTC() }
	}
	return &Verifier{sealer: sealer, now: now}
}

func (verifier *Verifier) VerifyBatch(value model.SnapshotBatch) model.VerificationRun {
	run := model.VerificationRun{ID: "verify-" + value.ID, BatchID: value.ID, CheckedAt: verifier.now(), Valid: true}
	details, err := verifier.sealer.Open(value)
	if err != nil {
		run.Valid = false
		run.Failure = err.Error()
		return run
	}
	results := make([]model.VerificationDetail, 0, len(details))
	for index, detail := range details {
		result := model.VerificationDetail{BatchID: value.ID, Sequence: detail.Sequence, Kind: detail.Kind, Valid: true, Message: "authenticated and ordered"}
		if detail.Sequence != index {
			result.Valid = false
			result.Message = fmt.Sprintf("expected sequence %d", index)
			run.Valid = false
		}
		if err := model.ValidateDetail(detail); err != nil {
			result.Valid = false
			result.Message = err.Error()
			run.Valid = false
		}
		results = append(results, result)
	}
	if len(results) > 0 {
		run.Details = results[:len(results)-1]
	} else {
		run.Details = results
	}
	return run
}

func (verifier *Verifier) VerifyFrame(frame []byte) model.VerificationRun {
	value, err := verifier.sealer.OpenFrame(frame)
	if err != nil {
		return model.VerificationRun{ID: "verify-invalid", CheckedAt: verifier.now(), Valid: false, Failure: err.Error()}
	}
	return verifier.VerifyBatch(value)
}
