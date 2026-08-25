package model

import (
	"testing"
	"time"
)

func TestDetailRoundTrip(t *testing.T) {
	minute := time.Date(2026, 8, 26, 1, 2, 0, 0, time.UTC)
	reading := SensorReading{ID: "r1", RecordedMinute: minute, TemperatureMilliC: 23000, CurrentMilliAmp: 12000}
	input := []SnapshotDetail{{Kind: DetailReading, Reading: &reading}}
	encoded, err := EncodeDetails(input)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeDetails(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].Reading.ID != "r1" {
		t.Fatalf("decoded=%+v", decoded)
	}
}
func TestPolicyEvaluation(t *testing.T) {
	minute := time.Date(2026, 8, 26, 1, 2, 0, 0, time.UTC)
	reading := SensorReading{ID: "r", RecordedMinute: minute, TemperatureMilliC: 35000, CurrentMilliAmp: 12000}
	value := SnapshotBatch{ID: "b", Minute: minute, DetailCount: 1, Details: []SnapshotDetail{{Kind: DetailReading, Reading: &reading}}, Status: "sealed"}
	result, err := EvaluatePolicy(DefaultRoomPolicy("room-a"), value)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted {
		t.Fatal("expected rejection")
	}
}
