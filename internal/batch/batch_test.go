package batch

import (
	"bytes"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/model"
	"testing"
	"time"
)

func TestCollectorOrdersDetails(t *testing.T) {
	minute := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	collector := NewMinuteCollector(minute)
	alert := model.AlertSummary{ID: "a", RecordedMinute: minute, Severity: "info", Code: "A", Sequence: 2}
	reading := model.SensorReading{ID: "r", RecordedMinute: minute, TemperatureMilliC: 22000, CurrentMilliAmp: 10, Sequence: 1}
	if err := collector.AddAlert(alert); err != nil {
		t.Fatal(err)
	}
	if err := collector.AddReading(reading); err != nil {
		t.Fatal(err)
	}
	details := collector.Details()
	if len(details) != 2 || details[0].Kind != model.DetailReading {
		t.Fatalf("details=%+v", details)
	}
}
func TestBatchFileRoundTrip(t *testing.T) {
	var value bytes.Buffer
	writer, err := NewFileWriter(&value)
	if err != nil {
		t.Fatal(err)
	}
	writer.WriteFrame([]byte("one"))
	writer.WriteFrame([]byte("two"))
	writer.Close()
	frames, err := ReadAll(&value)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames=%d", len(frames))
	}
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	_ = NewSealer(codec)
}
