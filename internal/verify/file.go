package verify

import (
	"errors"
	"io"

	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/model"
)

type FileResult struct {
	Runs         []model.VerificationRun `json:"runs"`
	FrameCount   int                     `json:"frame_count"`
	ValidCount   int                     `json:"valid_count"`
	InvalidCount int                     `json:"invalid_count"`
}

func (verifier *Verifier) VerifyFile(source io.Reader) (FileResult, error) {
	reader, err := batch.NewFileReader(source)
	if err != nil {
		return FileResult{}, err
	}
	var result FileResult
	for {
		frame, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return result, err
		}
		run := verifier.VerifyFrame(frame)
		result.Runs = append(result.Runs, run)
		result.FrameCount++
		if run.Valid {
			result.ValidCount++
		} else {
			result.InvalidCount++
		}
	}
	return result, nil
}
func Flatten(result FileResult) []model.VerificationDetail {
	count := 0
	for _, run := range result.Runs {
		count += len(run.Details)
	}
	details := make([]model.VerificationDetail, 0, count)
	for _, run := range result.Runs {
		details = append(details, run.Details...)
	}
	return details
}
func AllValid(result FileResult) bool {
	return result.FrameCount > 0 && result.InvalidCount == 0 && result.ValidCount == result.FrameCount
}
