package model

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type RoomPolicy struct {
	RoomID                string   `json:"room_id"`
	MinimumTemperature    int      `json:"minimum_temperature"`
	MaximumTemperature    int      `json:"maximum_temperature"`
	MaximumCurrent        int      `json:"maximum_current"`
	CriticalAlertCodes    []string `json:"critical_alert_codes"`
	RequiredReadingCount  int      `json:"required_reading_count"`
	MaximumAlertCount     int      `json:"maximum_alert_count"`
	RejectUnknownSeverity bool     `json:"reject_unknown_severity"`
}

type PolicyFinding struct {
	Sequence int    `json:"sequence"`
	Code     string `json:"code"`
	Level    string `json:"level"`
	Message  string `json:"message"`
}

type PolicyEvaluation struct {
	RoomID       string          `json:"room_id"`
	Minute       time.Time       `json:"minute"`
	Accepted     bool            `json:"accepted"`
	ReadingCount int             `json:"reading_count"`
	AlertCount   int             `json:"alert_count"`
	Findings     []PolicyFinding `json:"findings"`
}

func ValidatePolicy(policy RoomPolicy) error {
	if strings.TrimSpace(policy.RoomID) == "" {
		return fmt.Errorf("room id is required")
	}
	if policy.MinimumTemperature >= policy.MaximumTemperature {
		return fmt.Errorf("minimum temperature must be below maximum")
	}
	if policy.MaximumCurrent <= 0 {
		return fmt.Errorf("maximum current must be positive")
	}
	if policy.RequiredReadingCount < 1 {
		return fmt.Errorf("at least one reading is required")
	}
	if policy.MaximumAlertCount < 0 {
		return fmt.Errorf("maximum alert count cannot be negative")
	}
	seen := make(map[string]struct{}, len(policy.CriticalAlertCodes))
	for _, code := range policy.CriticalAlertCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			return fmt.Errorf("critical alert code cannot be empty")
		}
		if _, exists := seen[code]; exists {
			return fmt.Errorf("duplicate critical alert code %s", code)
		}
		seen[code] = struct{}{}
	}
	return nil
}

func EvaluatePolicy(policy RoomPolicy, batch SnapshotBatch) (PolicyEvaluation, error) {
	if err := ValidatePolicy(policy); err != nil {
		return PolicyEvaluation{}, err
	}
	if err := ValidateBatch(batch); err != nil {
		return PolicyEvaluation{}, err
	}
	result := PolicyEvaluation{
		RoomID:   policy.RoomID,
		Minute:   batch.Minute,
		Accepted: true,
		Findings: make([]PolicyFinding, 0),
	}
	criticalCodes := make(map[string]struct{}, len(policy.CriticalAlertCodes))
	for _, code := range policy.CriticalAlertCodes {
		criticalCodes[code] = struct{}{}
	}
	for _, detail := range batch.Details {
		switch detail.Kind {
		case DetailReading:
			result.ReadingCount++
			reading := detail.Reading
			if reading.TemperatureMilliC < policy.MinimumTemperature {
				result.Findings = append(result.Findings, PolicyFinding{
					Sequence: detail.Sequence,
					Code:     "temperature-low",
					Level:    "warning",
					Message:  fmt.Sprintf("temperature %d below %d", reading.TemperatureMilliC, policy.MinimumTemperature),
				})
			}
			if reading.TemperatureMilliC > policy.MaximumTemperature {
				result.Accepted = false
				result.Findings = append(result.Findings, PolicyFinding{
					Sequence: detail.Sequence,
					Code:     "temperature-high",
					Level:    "critical",
					Message:  fmt.Sprintf("temperature %d above %d", reading.TemperatureMilliC, policy.MaximumTemperature),
				})
			}
			if reading.CurrentMilliAmp > policy.MaximumCurrent {
				result.Accepted = false
				result.Findings = append(result.Findings, PolicyFinding{
					Sequence: detail.Sequence,
					Code:     "current-high",
					Level:    "critical",
					Message:  fmt.Sprintf("current %d above %d", reading.CurrentMilliAmp, policy.MaximumCurrent),
				})
			}
		case DetailAlert:
			result.AlertCount++
			alert := detail.Alert
			_, criticalCode := criticalCodes[alert.Code]
			if alert.Severity == "critical" || criticalCode {
				result.Accepted = false
				result.Findings = append(result.Findings, PolicyFinding{
					Sequence: detail.Sequence,
					Code:     "critical-alert",
					Level:    "critical",
					Message:  alert.Code + ": " + alert.Message,
				})
			}
		}
	}
	if result.ReadingCount < policy.RequiredReadingCount {
		result.Accepted = false
		result.Findings = append(result.Findings, PolicyFinding{
			Sequence: -1,
			Code:     "readings-missing",
			Level:    "critical",
			Message:  fmt.Sprintf("expected %d readings, got %d", policy.RequiredReadingCount, result.ReadingCount),
		})
	}
	if result.AlertCount > policy.MaximumAlertCount {
		result.Findings = append(result.Findings, PolicyFinding{
			Sequence: -1,
			Code:     "alert-volume",
			Level:    "warning",
			Message:  fmt.Sprintf("alert count %d exceeds %d", result.AlertCount, policy.MaximumAlertCount),
		})
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		return result.Findings[i].Sequence < result.Findings[j].Sequence
	})
	return result, nil
}

func DefaultRoomPolicy(roomID string) RoomPolicy {
	return RoomPolicy{
		RoomID:                roomID,
		MinimumTemperature:    16000,
		MaximumTemperature:    32000,
		MaximumCurrent:        200000,
		CriticalAlertCodes:    []string{"FIRE", "POWER-LOSS", "COOLING-FAIL"},
		RequiredReadingCount:  1,
		MaximumAlertCount:     20,
		RejectUnknownSeverity: true,
	}
}
