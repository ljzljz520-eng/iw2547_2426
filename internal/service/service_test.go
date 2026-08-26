package service

import (
	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/store"
	"example.com/roomsnapshot/internal/verify"
	"testing"
	"time"
)

func testService(t *testing.T) *Service {
	repository, err := store.Open(t.TempDir() + "/service.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	verifier := verify.New(sealer, nil)
	return New(repository, sealer, verifier, report.NewRunner(sealer, verifier, epoch), func() time.Time { return epoch })
}
func TestCaptureAndQuery(t *testing.T) {
	service := testService(t)
	minute := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	reading := model.SensorReading{ID: "r", TemperatureMilliC: 23000, CurrentMilliAmp: 10000}
	value, err := service.Capture(CaptureRequest{Minute: minute, Readings: []model.SensorReading{reading}})
	if err != nil {
		t.Fatal(err)
	}
	values, err := service.QueryBatches(BatchFilter{Status: "sealed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != value.ID {
		t.Fatalf("values=%+v", values)
	}
}
func TestDashboard(t *testing.T) {
	service := testService(t)
	dashboard, err := service.Dashboard("room-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.GeneratedAt.IsZero() {
		t.Fatal("missing generated time")
	}
}
