package roomsnapshot_test

import (
	"testing"
	"time"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/service"
	"example.com/roomsnapshot/internal/store"
	"example.com/roomsnapshot/internal/verify"
)

func workflowService(t *testing.T) *service.Service {
	t.Helper()
	repository, err := store.Open(t.TempDir() + "/workflow.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	codec, err := cryptoiface.NewCodec([]byte("workflow-contract-key-2026"))
	if err != nil {
		t.Fatal(err)
	}
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 14, 52, 0, 0, time.UTC)
	verifier := verify.New(sealer, func() time.Time { return epoch })
	return service.New(repository, sealer, verifier, report.NewRunner(sealer, verifier, epoch), func() time.Time { return epoch })
}

func TestBusinessChain09(t *testing.T) {
	application := workflowService(t)
	minute := time.Date(2026, 8, 26, 14, 52, 0, 0, time.UTC)
	details := report.FixtureDetails(minute, 9)
	readings := make([]model.SensorReading, 0)
	alerts := make([]model.AlertSummary, 0)
	for _, detail := range details {
		if detail.Reading != nil {
			readings = append(readings, *detail.Reading)
		} else {
			alerts = append(alerts, *detail.Alert)
		}
	}
	value, err := application.Capture(service.CaptureRequest{Minute: minute, Readings: readings, Alerts: alerts})
	if err != nil {
		t.Fatal(err)
	}
	file, err := application.ExportBatchFile([]string{value.ID})
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.VerifyBatchFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 1 {
		t.Fatalf("runs=%d", len(result.Runs))
	}
	got := result.Runs[0].Details
	if len(got) != len(details) {
		t.Fatalf("details=%d want=%d", len(got), len(details))
	}
	for index, detail := range got {
		if detail.Sequence != index {
			t.Fatalf("detail %d sequence=%d", index, detail.Sequence)
		}
	}
}

func TestWorkflowVerifyBatchFile(t *testing.T) {
	application := workflowService(t)
	minute := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	details := report.FixtureDetails(minute, 4)
	readings := []model.SensorReading{*details[0].Reading, *details[1].Reading, *details[2].Reading}
	alerts := []model.AlertSummary{*details[3].Alert}
	value, err := application.Capture(service.CaptureRequest{Minute: minute, Readings: readings, Alerts: alerts})
	if err != nil {
		t.Fatal(err)
	}
	file, err := application.ExportBatchFile([]string{value.ID})
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.VerifyBatchFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidCount != 1 || !result.Runs[0].Valid {
		t.Fatalf("result=%+v", result)
	}
}
func TestWorkflowThroughputReport(t *testing.T) {
	values, err := workflowService(t).GenerateReports([]int{4, 16, 64})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 {
		t.Fatalf("reports=%d", len(values))
	}
	if values[0].DurationNanos >= values[2].DurationNanos {
		t.Fatalf("durations not ordered")
	}
}
