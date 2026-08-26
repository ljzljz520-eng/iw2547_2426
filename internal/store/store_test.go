package store

import (
	"example.com/roomsnapshot/internal/model"
	"testing"
	"time"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := t.TempDir() + "/snapshot.db"
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	minute := time.Date(2026, 8, 26, 3, 4, 0, 0, time.UTC)
	reading := model.SensorReading{ID: "reading-persisted", RecordedMinute: minute, TemperatureMilliC: 24000, CurrentMilliAmp: 11000}
	if err := first.SaveReading(reading); err != nil {
		t.Fatal(err)
	}
	if err := first.SaveReport(model.ThroughputReport{ID: "report-persisted", CreatedAt: minute, BatchSize: 8, DurationNanos: 100, VerifiedCount: 8}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, err := second.Reading(reading.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TemperatureMilliC != reading.TemperatureMilliC {
		t.Fatalf("reading=%+v", got)
	}
	reports, err := second.ListReports()
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports=%d", len(reports))
	}
}
func TestStoreStatistics(t *testing.T) {
	value, err := Open(t.TempDir() + "/s.db")
	if err != nil {
		t.Fatal(err)
	}
	defer value.Close()
	stats, err := value.Statistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Batches != 0 {
		t.Fatalf("stats=%+v", stats)
	}
}
