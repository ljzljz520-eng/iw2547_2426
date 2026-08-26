package report

import (
	"fmt"
	"sort"

	"example.com/roomsnapshot/internal/model"
)

type Comparison struct {
	BaselineSize     int     `json:"baseline_size"`
	CandidateSize    int     `json:"candidate_size"`
	BaselineNanos    int64   `json:"baseline_nanos"`
	CandidateNanos   int64   `json:"candidate_nanos"`
	NanosPerDetail   float64 `json:"nanos_per_detail"`
	RelativeSpeed    float64 `json:"relative_speed"`
	VerifiedFraction float64 `json:"verified_fraction"`
}

func Compare(values []model.ThroughputReport) ([]Comparison, error) {
	if len(values) < 2 {
		return nil, fmt.Errorf("at least two throughput reports are required")
	}
	ordered := append([]model.ThroughputReport(nil), values...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].BatchSize < ordered[j].BatchSize
	})
	baseline := ordered[0]
	if baseline.BatchSize <= 0 || baseline.DurationNanos <= 0 {
		return nil, fmt.Errorf("invalid baseline report")
	}
	baselineRate := float64(baseline.DurationNanos) / float64(baseline.BatchSize)
	result := make([]Comparison, 0, len(ordered)-1)
	for _, candidate := range ordered[1:] {
		if candidate.BatchSize <= 0 || candidate.DurationNanos <= 0 {
			return nil, fmt.Errorf("invalid candidate report %s", candidate.ID)
		}
		candidateRate := float64(candidate.DurationNanos) / float64(candidate.BatchSize)
		verifiedFraction := float64(candidate.VerifiedCount) / float64(candidate.BatchSize)
		result = append(result, Comparison{
			BaselineSize:     baseline.BatchSize,
			CandidateSize:    candidate.BatchSize,
			BaselineNanos:    baseline.DurationNanos,
			CandidateNanos:   candidate.DurationNanos,
			NanosPerDetail:   candidateRate,
			RelativeSpeed:    baselineRate / candidateRate,
			VerifiedFraction: verifiedFraction,
		})
	}
	return result, nil
}

type Trend string

const (
	TrendImproving Trend = "improving"
	TrendStable    Trend = "stable"
	TrendDeclining Trend = "declining"
)

func ClassifyTrend(comparisons []Comparison, tolerance float64) Trend {
	if len(comparisons) == 0 {
		return TrendStable
	}
	if tolerance < 0 {
		tolerance = 0
	}
	improvements := 0
	declines := 0
	for _, comparison := range comparisons {
		if comparison.RelativeSpeed > 1+tolerance {
			improvements++
		}
		if comparison.RelativeSpeed < 1-tolerance {
			declines++
		}
	}
	if improvements > declines {
		return TrendImproving
	}
	if declines > improvements {
		return TrendDeclining
	}
	return TrendStable
}

func BestBatchSize(values []model.ThroughputReport) (int, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("reports are required")
	}
	bestSize := 0
	bestRate := float64(0)
	for _, value := range values {
		if value.BatchSize <= 0 || value.DurationNanos <= 0 {
			return 0, fmt.Errorf("invalid report %s", value.ID)
		}
		rate := float64(value.DurationNanos) / float64(value.BatchSize)
		if bestSize == 0 || rate < bestRate {
			bestSize = value.BatchSize
			bestRate = rate
		}
	}
	return bestSize, nil
}
