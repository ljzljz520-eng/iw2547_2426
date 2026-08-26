package service

import (
	"fmt"
	"sort"
	"time"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/report"
)

type Dashboard struct {
	GeneratedAt     time.Time                `json:"generated_at"`
	Statistics      map[string]int64         `json:"statistics"`
	Catalog         batch.Catalog            `json:"catalog"`
	ReportSummary   map[string]int64         `json:"report_summary"`
	RecommendedSize int                      `json:"recommended_size"`
	RecentBatches   []batch.CatalogEntry     `json:"recent_batches"`
	PolicyResults   []model.PolicyEvaluation `json:"policy_results"`
}

func (service *Service) Dashboard(roomID string, limit int) (Dashboard, error) {
	if roomID == "" {
		return Dashboard{}, fmt.Errorf("room id is required")
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	batches, err := service.store.ListBatches()
	if err != nil {
		return Dashboard{}, err
	}
	catalog, err := batch.BuildCatalog(batches)
	if err != nil {
		return Dashboard{}, err
	}
	reports, err := service.store.ListReports()
	if err != nil {
		return Dashboard{}, err
	}
	statistics, err := service.store.Statistics()
	if err != nil {
		return Dashboard{}, err
	}
	result := Dashboard{
		GeneratedAt: service.now(),
		Statistics: map[string]int64{
			"readings":      int64(statistics.Readings),
			"alerts":        int64(statistics.Alerts),
			"batches":       int64(statistics.Batches),
			"reports":       int64(statistics.Reports),
			"verifications": int64(statistics.Verifications),
			"file_bytes":    statistics.FileBytes,
		},
		Catalog:       catalog,
		ReportSummary: report.Summarize(reports),
	}
	if len(reports) > 0 {
		result.RecommendedSize, err = report.BestBatchSize(reports)
		if err != nil {
			return Dashboard{}, err
		}
	}
	if len(catalog.Entries) > limit {
		result.RecentBatches = append([]batch.CatalogEntry(nil), catalog.Entries[len(catalog.Entries)-limit:]...)
	} else {
		result.RecentBatches = append([]batch.CatalogEntry(nil), catalog.Entries...)
	}
	policy := model.DefaultRoomPolicy(roomID)
	for _, value := range batches {
		evaluation, evaluateErr := model.EvaluatePolicy(policy, value)
		if evaluateErr != nil {
			return Dashboard{}, evaluateErr
		}
		result.PolicyResults = append(result.PolicyResults, evaluation)
	}
	return result, nil
}

type Incident struct {
	Minute   time.Time `json:"minute"`
	BatchID  string    `json:"batch_id"`
	Severity string    `json:"severity"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Sequence int       `json:"sequence"`
}

func (service *Service) Incidents(from, to time.Time) ([]Incident, error) {
	alerts, err := service.QueryAlerts(AlertFilter{From: from, To: to})
	if err != nil {
		return nil, err
	}
	result := make([]Incident, 0, len(alerts))
	for _, alert := range alerts {
		if alert.Severity == "info" {
			continue
		}
		batchValue, batchErr := service.store.BatchByMinute(alert.RecordedMinute)
		if batchErr != nil {
			continue
		}
		result = append(result, Incident{
			Minute:   alert.RecordedMinute,
			BatchID:  batchValue.ID,
			Severity: alert.Severity,
			Code:     alert.Code,
			Message:  alert.Message,
			Sequence: alert.Sequence,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Minute.Equal(result[j].Minute) {
			return result[i].Sequence < result[j].Sequence
		}
		return result[i].Minute.Before(result[j].Minute)
	})
	return result, nil
}
