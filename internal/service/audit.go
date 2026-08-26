package service

import (
	"fmt"
	"sort"
	"time"

	"example.com/roomsnapshot/internal/model"
)

type AuditIssue struct {
	BatchID  string    `json:"batch_id"`
	Minute   time.Time `json:"minute"`
	Sequence int       `json:"sequence"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
}

type AuditResult struct {
	StartedAt     time.Time    `json:"started_at"`
	CompletedAt   time.Time    `json:"completed_at"`
	BatchCount    int          `json:"batch_count"`
	DetailCount   int          `json:"detail_count"`
	SealedCount   int          `json:"sealed_count"`
	VerifiedCount int          `json:"verified_count"`
	RejectedCount int          `json:"rejected_count"`
	IssueCount    int          `json:"issue_count"`
	Issues        []AuditIssue `json:"issues"`
}

func (service *Service) Audit() (AuditResult, error) {
	result := AuditResult{
		StartedAt: service.now(),
		Issues:    make([]AuditIssue, 0),
	}
	values, err := service.store.ListBatches()
	if err != nil {
		return result, err
	}
	result.BatchCount = len(values)
	seenIDs := make(map[string]time.Time, len(values))
	seenMinutes := make(map[time.Time]string, len(values))
	for _, value := range values {
		result.DetailCount += value.DetailCount
		switch value.Status {
		case "sealed":
			result.SealedCount++
		case "verified":
			result.VerifiedCount++
		case "rejected":
			result.RejectedCount++
		default:
			result.Issues = append(result.Issues, AuditIssue{
				BatchID:  value.ID,
				Minute:   value.Minute,
				Sequence: -1,
				Code:     "unknown-status",
				Message:  "batch status is not recognized",
			})
		}
		if previousMinute, exists := seenIDs[value.ID]; exists {
			result.Issues = append(result.Issues, AuditIssue{
				BatchID:  value.ID,
				Minute:   value.Minute,
				Sequence: -1,
				Code:     "duplicate-id",
				Message:  fmt.Sprintf("batch id also used at %s", previousMinute.Format(time.RFC3339)),
			})
		} else {
			seenIDs[value.ID] = value.Minute
		}
		if previousID, exists := seenMinutes[value.Minute]; exists && previousID != value.ID {
			result.Issues = append(result.Issues, AuditIssue{
				BatchID:  value.ID,
				Minute:   value.Minute,
				Sequence: -1,
				Code:     "duplicate-minute",
				Message:  fmt.Sprintf("minute also assigned to %s", previousID),
			})
		} else {
			seenMinutes[value.Minute] = value.ID
		}
		if err := model.ValidateBatch(value); err != nil {
			result.Issues = append(result.Issues, AuditIssue{
				BatchID:  value.ID,
				Minute:   value.Minute,
				Sequence: -1,
				Code:     "invalid-batch",
				Message:  err.Error(),
			})
			continue
		}
		for index, detail := range value.Details {
			if detail.Sequence != index {
				result.Issues = append(result.Issues, AuditIssue{
					BatchID:  value.ID,
					Minute:   value.Minute,
					Sequence: detail.Sequence,
					Code:     "sequence-gap",
					Message:  fmt.Sprintf("expected detail sequence %d", index),
				})
			}
			if detail.Reading != nil && detail.Reading.RecordedMinute != value.Minute {
				result.Issues = append(result.Issues, AuditIssue{
					BatchID:  value.ID,
					Minute:   value.Minute,
					Sequence: detail.Sequence,
					Code:     "reading-minute",
					Message:  "reading minute differs from batch minute",
				})
			}
			if detail.Alert != nil && detail.Alert.RecordedMinute != value.Minute {
				result.Issues = append(result.Issues, AuditIssue{
					BatchID:  value.ID,
					Minute:   value.Minute,
					Sequence: detail.Sequence,
					Code:     "alert-minute",
					Message:  "alert minute differs from batch minute",
				})
			}
		}
	}
	sort.SliceStable(result.Issues, func(i, j int) bool {
		if result.Issues[i].Minute.Equal(result.Issues[j].Minute) {
			return result.Issues[i].Sequence < result.Issues[j].Sequence
		}
		return result.Issues[i].Minute.Before(result.Issues[j].Minute)
	})
	result.IssueCount = len(result.Issues)
	result.CompletedAt = service.now()
	return result, nil
}
