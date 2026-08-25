package verify_test

import (
	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/verify"
	"testing"
	"time"
)

func TestVerifierAcceptsAuthenticatedBatch(t *testing.T) {
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	minute := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	value, err := sealer.Seal(minute, report.FixtureDetails(minute, 3))
	if err != nil {
		t.Fatal(err)
	}
	run := verify.New(sealer, func() time.Time { return minute }).VerifyBatch(value)
	if !run.Valid {
		t.Fatalf("run=%+v", run)
	}
}
func TestVerifierRejectsTagMutation(t *testing.T) {
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	minute := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	value, _ := sealer.Seal(minute, report.FixtureDetails(minute, 2))
	value.Tag[0] ^= 1
	run := verify.New(sealer, nil).VerifyBatch(value)
	if run.Valid {
		t.Fatal("expected invalid run")
	}
}
