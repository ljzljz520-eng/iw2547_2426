package report

import (
	"fmt"
	"time"

	"example.com/roomsnapshot/internal/model"
)

func FixtureDetails(minute time.Time, size int) []model.SnapshotDetail {
	minute = model.NormalizeMinute(minute)
	details := make([]model.SnapshotDetail, 0, size)
	for index := 0; index < size; index++ {
		if index%4 == 3 {
			alert := model.AlertSummary{ID: fmt.Sprintf("alert-%04d", index), RecordedMinute: minute, Severity: severity(index), Code: fmt.Sprintf("ROOM-%03d", index%17), Message: fmt.Sprintf("deterministic condition %d", index), Sequence: index}
			details = append(details, model.SnapshotDetail{Sequence: index, Kind: model.DetailAlert, Alert: &alert})
		} else {
			reading := model.SensorReading{ID: fmt.Sprintf("reading-%04d", index), RecordedMinute: minute, TemperatureMilliC: 22000 + (index%19)*50, CurrentMilliAmp: 10000 + (index%13)*25, Sequence: index}
			details = append(details, model.SnapshotDetail{Sequence: index, Kind: model.DetailReading, Reading: &reading})
		}
	}
	return details
}
func severity(index int) string {
	switch index % 3 {
	case 0:
		return "info"
	case 1:
		return "warning"
	default:
		return "critical"
	}
}
func Summarize(values []model.ThroughputReport) map[string]int64 {
	result := map[string]int64{"smallest": 0, "largest": 0, "total": 0}
	if len(values) == 0 {
		return result
	}
	result["smallest"] = values[0].DurationNanos
	for _, value := range values {
		if value.DurationNanos < result["smallest"] {
			result["smallest"] = value.DurationNanos
		}
		if value.DurationNanos > result["largest"] {
			result["largest"] = value.DurationNanos
		}
		result["total"] += value.DurationNanos
	}
	return result
}
