package service

import (
	"bytes"
	"fmt"
	"time"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/store"
	"example.com/roomsnapshot/internal/verify"
)

type Service struct {
	store    *store.Store
	sealer   *batch.Sealer
	verifier *verify.Verifier
	reporter *report.Runner
	now      func() time.Time
}

func New(repository *store.Store, sealer *batch.Sealer, verifier *verify.Verifier, reporter *report.Runner, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Unix(0, 0).UTC() }
	}
	return &Service{store: repository, sealer: sealer, verifier: verifier, reporter: reporter, now: now}
}

type CaptureRequest struct {
	Minute   time.Time             `json:"minute"`
	Readings []model.SensorReading `json:"readings"`
	Alerts   []model.AlertSummary  `json:"alerts"`
}

func (service *Service) Capture(request CaptureRequest) (model.SnapshotBatch, error) {
	minute := model.NormalizeMinute(request.Minute)
	if minute.IsZero() {
		return model.SnapshotBatch{}, fmt.Errorf("capture minute is required")
	}
	collector := batch.NewMinuteCollector(minute)
	for _, reading := range request.Readings {
		reading.RecordedMinute = minute
		if err := collector.AddReading(reading); err != nil {
			return model.SnapshotBatch{}, err
		}
		if err := service.store.SaveReading(reading); err != nil {
			return model.SnapshotBatch{}, err
		}
	}
	for _, alert := range request.Alerts {
		alert.RecordedMinute = minute
		if err := collector.AddAlert(alert); err != nil {
			return model.SnapshotBatch{}, err
		}
		if err := service.store.SaveAlert(alert); err != nil {
			return model.SnapshotBatch{}, err
		}
	}
	details := collector.Details()
	value, err := service.sealer.Seal(minute, details)
	if err != nil {
		return model.SnapshotBatch{}, err
	}
	if err := service.store.SaveBatch(value); err != nil {
		return model.SnapshotBatch{}, err
	}
	return value, nil
}

func (service *Service) ExportBatchFile(ids []string) ([]byte, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("batch ids are required")
	}
	var output bytes.Buffer
	writer, err := batch.NewFileWriter(&output)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		value, err := service.store.Batch(id)
		if err != nil {
			return nil, err
		}
		frame, err := service.sealer.Frame(value)
		if err != nil {
			return nil, err
		}
		if err := writer.WriteFrame(frame); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
func (service *Service) VerifyBatchFile(data []byte) (verify.FileResult, error) {
	result, err := service.verifier.VerifyFile(bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	for _, run := range result.Runs {
		if err := service.store.SaveVerification(run); err != nil {
			return result, err
		}
		if run.BatchID != "" {
			value, err := service.store.Batch(run.BatchID)
			if err == nil {
				if run.Valid {
					value.Status = "verified"
				} else {
					value.Status = "rejected"
				}
				if saveErr := service.store.SaveBatch(value); saveErr != nil {
					return result, saveErr
				}
			}
		}
	}
	return result, nil
}
func (service *Service) VerifyLatest() ([]model.VerificationDetail, error) {
	latest, err := service.store.LatestBatch()
	if err != nil {
		return nil, err
	}
	file, err := service.ExportBatchFile([]string{latest.ID})
	if err != nil {
		return nil, err
	}
	result, err := service.VerifyBatchFile(file)
	if err != nil {
		return nil, err
	}
	return verify.Flatten(result), nil
}
func (service *Service) GenerateReports(sizes []int) ([]model.ThroughputReport, error) {
	values, err := service.reporter.Run(sizes)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		if err := service.store.SaveReport(value); err != nil {
			return nil, err
		}
	}
	return values, nil
}
func (service *Service) Reports() ([]model.ThroughputReport, error) {
	return service.store.ListReports()
}
func (service *Service) Batches() ([]model.SnapshotBatch, error) { return service.store.ListBatches() }
func (service *Service) Statistics() (store.Statistics, error)   { return service.store.Statistics() }
