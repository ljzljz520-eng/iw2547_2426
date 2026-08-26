package service

import (
	"sort"
	"strings"
	"time"

	"example.com/roomsnapshot/internal/model"
)

type BatchFilter struct {
	From           time.Time
	To             time.Time
	Status         string
	MinimumDetails int
}

func (service *Service) QueryBatches(filter BatchFilter) ([]model.SnapshotBatch, error) {
	values, err := service.store.ListBatches()
	if err != nil {
		return nil, err
	}
	result := make([]model.SnapshotBatch, 0, len(values))
	for _, value := range values {
		if !filter.From.IsZero() && value.Minute.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && value.Minute.After(filter.To) {
			continue
		}
		if filter.Status != "" && value.Status != filter.Status {
			continue
		}
		if value.DetailCount < filter.MinimumDetails {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Minute.After(result[j].Minute) })
	return result, nil
}

type AlertFilter struct {
	Severity   string
	CodePrefix string
	From       time.Time
	To         time.Time
}

func (service *Service) QueryAlerts(filter AlertFilter) ([]model.AlertSummary, error) {
	values, err := service.store.ListAlerts()
	if err != nil {
		return nil, err
	}
	result := make([]model.AlertSummary, 0, len(values))
	for _, value := range values {
		if filter.Severity != "" && value.Severity != filter.Severity {
			continue
		}
		if filter.CodePrefix != "" && !strings.HasPrefix(value.Code, filter.CodePrefix) {
			continue
		}
		if !filter.From.IsZero() && value.RecordedMinute.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && value.RecordedMinute.After(filter.To) {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RecordedMinute.Equal(result[j].RecordedMinute) {
			return result[i].Sequence < result[j].Sequence
		}
		return result[i].RecordedMinute.Before(result[j].RecordedMinute)
	})
	return result, nil
}

type MinuteSummary struct {
	Minute             time.Time `json:"minute"`
	ReadingCount       int       `json:"reading_count"`
	AlertCount         int       `json:"alert_count"`
	CriticalCount      int       `json:"critical_count"`
	MinimumTemperature int       `json:"minimum_temperature"`
	MaximumTemperature int       `json:"maximum_temperature"`
	AverageCurrent     int       `json:"average_current"`
}

func (service *Service) SummarizeMinute(minute time.Time) (MinuteSummary, error) {
	readings, err := service.store.ReadingsByMinute(minute)
	if err != nil {
		return MinuteSummary{}, err
	}
	alerts, err := service.store.AlertsByMinute(minute)
	if err != nil {
		return MinuteSummary{}, err
	}
	result := MinuteSummary{Minute: model.NormalizeMinute(minute), ReadingCount: len(readings), AlertCount: len(alerts)}
	totalCurrent := 0
	for index, value := range readings {
		if index == 0 || value.TemperatureMilliC < result.MinimumTemperature {
			result.MinimumTemperature = value.TemperatureMilliC
		}
		if index == 0 || value.TemperatureMilliC > result.MaximumTemperature {
			result.MaximumTemperature = value.TemperatureMilliC
		}
		totalCurrent += value.CurrentMilliAmp
	}
	if len(readings) > 0 {
		result.AverageCurrent = totalCurrent / len(readings)
	}
	for _, value := range alerts {
		if value.Severity == "critical" {
			result.CriticalCount++
		}
	}
	return result, nil
}
