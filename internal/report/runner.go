package report

import (
	"fmt"
	"time"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/verify"
)

type Runner struct {
	sealer   *batch.Sealer
	verifier *verify.Verifier
	epoch    time.Time
}

func NewRunner(sealer *batch.Sealer, verifier *verify.Verifier, epoch time.Time) *Runner {
	return &Runner{sealer: sealer, verifier: verifier, epoch: model.NormalizeMinute(epoch)}
}
func (runner *Runner) Run(sizes []int) ([]model.ThroughputReport, error) {
	if len(sizes) == 0 {
		return nil, fmt.Errorf("batch sizes are required")
	}
	reports := make([]model.ThroughputReport, 0, len(sizes))
	for _, size := range sizes {
		if size <= 0 || size > 10000 {
			return nil, fmt.Errorf("unsupported batch size %d", size)
		}
		details := FixtureDetails(runner.epoch, size)
		start := deterministicCost(details, false)
		value, err := runner.sealer.Seal(runner.epoch, details)
		if err != nil {
			return nil, err
		}
		run := runner.verifier.VerifyBatch(value)
		finish := deterministicCost(details, true)
		if !run.Valid {
			return nil, fmt.Errorf("verification failed: %s", run.Failure)
		}
		reports = append(reports, model.ThroughputReport{ID: fmt.Sprintf("report-%04d", size), CreatedAt: runner.epoch, BatchSize: size, DurationNanos: finish - start, VerifiedCount: len(run.Details)})
	}
	return reports, nil
}
func deterministicCost(details []model.SnapshotDetail, verified bool) int64 {
	cost := int64(1000)
	for _, detail := range details {
		cost += int64(41 + detail.Sequence*3)
		switch detail.Kind {
		case model.DetailReading:
			cost += 73
		case model.DetailAlert:
			cost += 107
		}
		if verified {
			cost += 211
		}
	}
	return cost
}
