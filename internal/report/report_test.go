package report

import (
	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/verify"
	"testing"
	"time"
)

func TestRunnerProducesOrderedSizes(t *testing.T) {
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	values, err := NewRunner(sealer, verify.New(sealer, nil), epoch).Run([]int{4, 16})
	if err != nil {
		t.Fatal(err)
	}
	if values[0].BatchSize != 4 || values[1].BatchSize != 16 {
		t.Fatalf("values=%+v", values)
	}
}
func TestBestBatchSize(t *testing.T) {
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	values, _ := NewRunner(sealer, verify.New(sealer, nil), epoch).Run([]int{4, 16})
	size, err := BestBatchSize(values)
	if err != nil {
		t.Fatal(err)
	}
	if size == 0 {
		t.Fatal("missing size")
	}
}
