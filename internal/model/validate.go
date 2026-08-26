package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidReading = errors.New("invalid sensor reading")
	ErrInvalidAlert   = errors.New("invalid alert summary")
	ErrInvalidBatch   = errors.New("invalid snapshot batch")
)

func NormalizeMinute(value time.Time) time.Time { return value.UTC().Truncate(time.Minute) }

func ValidateReading(value SensorReading) error {
	if strings.TrimSpace(value.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidReading)
	}
	if value.RecordedMinute.IsZero() {
		return fmt.Errorf("%w: minute is required", ErrInvalidReading)
	}
	if value.RecordedMinute != NormalizeMinute(value.RecordedMinute) {
		return fmt.Errorf("%w: timestamp must be minute aligned", ErrInvalidReading)
	}
	if value.TemperatureMilliC < -40000 || value.TemperatureMilliC > 125000 {
		return fmt.Errorf("%w: temperature is outside sensor range", ErrInvalidReading)
	}
	if value.CurrentMilliAmp < 0 || value.CurrentMilliAmp > 250000 {
		return fmt.Errorf("%w: current is outside circuit range", ErrInvalidReading)
	}
	if value.Sequence < 0 {
		return fmt.Errorf("%w: sequence must be non-negative", ErrInvalidReading)
	}
	return nil
}

func ValidateAlert(value AlertSummary) error {
	if strings.TrimSpace(value.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidAlert)
	}
	if value.RecordedMinute.IsZero() {
		return fmt.Errorf("%w: minute is required", ErrInvalidAlert)
	}
	if value.RecordedMinute != NormalizeMinute(value.RecordedMinute) {
		return fmt.Errorf("%w: timestamp must be minute aligned", ErrInvalidAlert)
	}
	switch value.Severity {
	case "info", "warning", "critical":
	default:
		return fmt.Errorf("%w: unsupported severity", ErrInvalidAlert)
	}
	if strings.TrimSpace(value.Code) == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidAlert)
	}
	if len(value.Message) > 512 {
		return fmt.Errorf("%w: message is too long", ErrInvalidAlert)
	}
	if value.Sequence < 0 {
		return fmt.Errorf("%w: sequence must be non-negative", ErrInvalidAlert)
	}
	return nil
}

func ValidateDetail(value SnapshotDetail) error {
	if value.Sequence < 0 {
		return fmt.Errorf("%w: negative detail sequence", ErrInvalidBatch)
	}
	switch value.Kind {
	case DetailReading:
		if value.Reading == nil || value.Alert != nil {
			return fmt.Errorf("%w: reading detail shape", ErrInvalidBatch)
		}
		if err := ValidateReading(*value.Reading); err != nil {
			return err
		}
	case DetailAlert:
		if value.Alert == nil || value.Reading != nil {
			return fmt.Errorf("%w: alert detail shape", ErrInvalidBatch)
		}
		if err := ValidateAlert(*value.Alert); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unknown detail kind", ErrInvalidBatch)
	}
	return nil
}

func ValidateBatch(value SnapshotBatch) error {
	if strings.TrimSpace(value.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidBatch)
	}
	if value.Minute.IsZero() || value.Minute != NormalizeMinute(value.Minute) {
		return fmt.Errorf("%w: invalid minute", ErrInvalidBatch)
	}
	if value.DetailCount != len(value.Details) {
		return fmt.Errorf("%w: detail count mismatch", ErrInvalidBatch)
	}
	if len(value.Details) == 0 {
		return fmt.Errorf("%w: details are required", ErrInvalidBatch)
	}
	for index, detail := range value.Details {
		if detail.Sequence != index {
			return fmt.Errorf("%w: sequence %d is out of order", ErrInvalidBatch, detail.Sequence)
		}
		if err := ValidateDetail(detail); err != nil {
			return err
		}
		if detail.Reading != nil && detail.Reading.RecordedMinute != value.Minute {
			return fmt.Errorf("%w: reading minute mismatch", ErrInvalidBatch)
		}
		if detail.Alert != nil && detail.Alert.RecordedMinute != value.Minute {
			return fmt.Errorf("%w: alert minute mismatch", ErrInvalidBatch)
		}
	}
	switch value.Status {
	case "sealed", "verified", "rejected":
	default:
		return fmt.Errorf("%w: invalid status", ErrInvalidBatch)
	}
	return nil
}
