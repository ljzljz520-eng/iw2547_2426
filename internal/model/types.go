package model

import "time"

type DetailKind string

const (
	DetailReading DetailKind = "reading"
	DetailAlert   DetailKind = "alert"
)

type SensorReading struct {
	ID                string    `json:"id"`
	RecordedMinute    time.Time `json:"recorded_minute"`
	TemperatureMilliC int       `json:"temperature_milli_c"`
	CurrentMilliAmp   int       `json:"current_milli_amp"`
	Sequence          int       `json:"sequence"`
}

type AlertSummary struct {
	ID             string    `json:"id"`
	RecordedMinute time.Time `json:"recorded_minute"`
	Severity       string    `json:"severity"`
	Code           string    `json:"code"`
	Message        string    `json:"message"`
	Sequence       int       `json:"sequence"`
}

type SnapshotDetail struct {
	Sequence int            `json:"sequence"`
	Kind     DetailKind     `json:"kind"`
	Reading  *SensorReading `json:"reading,omitempty"`
	Alert    *AlertSummary  `json:"alert,omitempty"`
}

type SnapshotBatch struct {
	ID          string           `json:"id"`
	Minute      time.Time        `json:"minute"`
	DetailCount int              `json:"detail_count"`
	Details     []SnapshotDetail `json:"details"`
	Payload     []byte           `json:"payload"`
	Tag         []byte           `json:"tag"`
	Status      string           `json:"status"`
}

type ThroughputReport struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	BatchSize     int       `json:"batch_size"`
	DurationNanos int64     `json:"duration_nanos"`
	VerifiedCount int       `json:"verified_count"`
}

type VerificationDetail struct {
	BatchID  string     `json:"batch_id"`
	Sequence int        `json:"sequence"`
	Kind     DetailKind `json:"kind"`
	Valid    bool       `json:"valid"`
	Message  string     `json:"message"`
}

type VerificationRun struct {
	ID        string               `json:"id"`
	BatchID   string               `json:"batch_id"`
	CheckedAt time.Time            `json:"checked_at"`
	Valid     bool                 `json:"valid"`
	Details   []VerificationDetail `json:"details"`
	Failure   string               `json:"failure,omitempty"`
}
