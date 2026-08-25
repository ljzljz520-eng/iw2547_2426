package batch

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"example.com/roomsnapshot/internal/model"
)

type MinuteCollector struct {
	mu       sync.Mutex
	minute   time.Time
	readings []model.SensorReading
	alerts   []model.AlertSummary
}

func NewMinuteCollector(minute time.Time) *MinuteCollector {
	return &MinuteCollector{minute: model.NormalizeMinute(minute)}
}
func (collector *MinuteCollector) AddReading(value model.SensorReading) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	if err := model.ValidateReading(value); err != nil {
		return err
	}
	if value.RecordedMinute != collector.minute {
		return fmt.Errorf("reading belongs to another minute")
	}
	collector.readings = append(collector.readings, value)
	return nil
}
func (collector *MinuteCollector) AddAlert(value model.AlertSummary) error {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	if err := model.ValidateAlert(value); err != nil {
		return err
	}
	if value.RecordedMinute != collector.minute {
		return fmt.Errorf("alert belongs to another minute")
	}
	collector.alerts = append(collector.alerts, value)
	return nil
}
func (collector *MinuteCollector) Counts() (int, int) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	return len(collector.readings), len(collector.alerts)
}

func (collector *MinuteCollector) Details() []model.SnapshotDetail {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	readings := append([]model.SensorReading(nil), collector.readings...)
	alerts := append([]model.AlertSummary(nil), collector.alerts...)
	sort.SliceStable(readings, func(i, j int) bool { return readings[i].Sequence < readings[j].Sequence })
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].Sequence < alerts[j].Sequence })
	details := make([]model.SnapshotDetail, 0, len(readings)+len(alerts))
	ri, ai := 0, 0
	for ri < len(readings) || ai < len(alerts) {
		useReading := ai >= len(alerts) || (ri < len(readings) && readings[ri].Sequence <= alerts[ai].Sequence)
		if useReading {
			value := readings[ri]
			details = append(details, model.SnapshotDetail{Kind: model.DetailReading, Reading: &value})
			ri++
		} else {
			value := alerts[ai]
			details = append(details, model.SnapshotDetail{Kind: model.DetailAlert, Alert: &value})
			ai++
		}
	}
	for index := range details {
		details[index].Sequence = index
		if details[index].Reading != nil {
			details[index].Reading.Sequence = index
		}
		if details[index].Alert != nil {
			details[index].Alert.Sequence = index
		}
	}
	return details
}

func (collector *MinuteCollector) Reset(minute time.Time) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.minute = model.NormalizeMinute(minute)
	collector.readings = nil
	collector.alerts = nil
}
