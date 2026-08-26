package batch

import (
	"fmt"
	"sort"
	"time"

	"example.com/roomsnapshot/internal/model"
)

type CatalogEntry struct {
	BatchID       string    `json:"batch_id"`
	Minute        time.Time `json:"minute"`
	DetailCount   int       `json:"detail_count"`
	ReadingCount  int       `json:"reading_count"`
	AlertCount    int       `json:"alert_count"`
	CriticalCount int       `json:"critical_count"`
	Status        string    `json:"status"`
}

type Catalog struct {
	Entries        []CatalogEntry `json:"entries"`
	FirstMinute    time.Time      `json:"first_minute"`
	LastMinute     time.Time      `json:"last_minute"`
	TotalDetails   int            `json:"total_details"`
	TotalReadings  int            `json:"total_readings"`
	TotalAlerts    int            `json:"total_alerts"`
	CriticalAlerts int            `json:"critical_alerts"`
}

func BuildCatalog(batches []model.SnapshotBatch) (Catalog, error) {
	result := Catalog{Entries: make([]CatalogEntry, 0, len(batches))}
	for _, value := range batches {
		if err := model.ValidateBatch(value); err != nil {
			return Catalog{}, fmt.Errorf("catalog batch %s: %w", value.ID, err)
		}
		entry := CatalogEntry{
			BatchID:     value.ID,
			Minute:      value.Minute,
			DetailCount: value.DetailCount,
			Status:      value.Status,
		}
		for _, detail := range value.Details {
			switch detail.Kind {
			case model.DetailReading:
				entry.ReadingCount++
			case model.DetailAlert:
				entry.AlertCount++
				if detail.Alert.Severity == "critical" {
					entry.CriticalCount++
				}
			}
		}
		result.Entries = append(result.Entries, entry)
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		return result.Entries[i].Minute.Before(result.Entries[j].Minute)
	})
	for index, entry := range result.Entries {
		if index == 0 {
			result.FirstMinute = entry.Minute
		}
		result.LastMinute = entry.Minute
		result.TotalDetails += entry.DetailCount
		result.TotalReadings += entry.ReadingCount
		result.TotalAlerts += entry.AlertCount
		result.CriticalAlerts += entry.CriticalCount
	}
	return result, nil
}

func (catalog Catalog) Entry(batchID string) (CatalogEntry, bool) {
	for _, entry := range catalog.Entries {
		if entry.BatchID == batchID {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}

func (catalog Catalog) Between(from, to time.Time) []CatalogEntry {
	entries := make([]CatalogEntry, 0)
	for _, entry := range catalog.Entries {
		if !from.IsZero() && entry.Minute.Before(from) {
			continue
		}
		if !to.IsZero() && entry.Minute.After(to) {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func (catalog Catalog) StatusCounts() map[string]int {
	counts := make(map[string]int)
	for _, entry := range catalog.Entries {
		counts[entry.Status]++
	}
	return counts
}

func (catalog Catalog) AlertRatio() float64 {
	if catalog.TotalDetails == 0 {
		return 0
	}
	return float64(catalog.TotalAlerts) / float64(catalog.TotalDetails)
}
