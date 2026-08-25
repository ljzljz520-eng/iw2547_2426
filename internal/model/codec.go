package model

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrMalformedPayload = errors.New("malformed snapshot payload")

func EncodeDetails(details []SnapshotDetail) ([]byte, error) {
	var output bytes.Buffer
	if err := binary.Write(&output, binary.BigEndian, uint32(len(details))); err != nil {
		return nil, err
	}
	for _, detail := range details {
		if err := ValidateDetail(detail); err != nil {
			return nil, err
		}
		if err := binary.Write(&output, binary.BigEndian, uint32(detail.Sequence)); err != nil {
			return nil, err
		}
		switch detail.Kind {
		case DetailReading:
			output.WriteByte(1)
			writeString(&output, detail.Reading.ID)
			binary.Write(&output, binary.BigEndian, detail.Reading.RecordedMinute.Unix())
			binary.Write(&output, binary.BigEndian, int64(detail.Reading.TemperatureMilliC))
			binary.Write(&output, binary.BigEndian, int64(detail.Reading.CurrentMilliAmp))
		case DetailAlert:
			output.WriteByte(2)
			writeString(&output, detail.Alert.ID)
			binary.Write(&output, binary.BigEndian, detail.Alert.RecordedMinute.Unix())
			writeString(&output, detail.Alert.Severity)
			writeString(&output, detail.Alert.Code)
			writeString(&output, detail.Alert.Message)
		default:
			return nil, fmt.Errorf("%w: unknown kind", ErrMalformedPayload)
		}
	}
	return output.Bytes(), nil
}

func DecodeDetails(payload []byte) ([]SnapshotDetail, error) {
	reader := bytes.NewReader(payload)
	var count uint32
	if err := binary.Read(reader, binary.BigEndian, &count); err != nil {
		return nil, fmt.Errorf("%w: count", ErrMalformedPayload)
	}
	if count > 100000 {
		return nil, fmt.Errorf("%w: excessive detail count", ErrMalformedPayload)
	}
	details := make([]SnapshotDetail, 0, count)
	for index := uint32(0); index < count; index++ {
		var sequence uint32
		if err := binary.Read(reader, binary.BigEndian, &sequence); err != nil {
			return nil, fmt.Errorf("%w: sequence", ErrMalformedPayload)
		}
		kind, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("%w: kind", ErrMalformedPayload)
		}
		switch kind {
		case 1:
			id, err := readString(reader)
			if err != nil {
				return nil, err
			}
			var stamp, temperature, current int64
			if binary.Read(reader, binary.BigEndian, &stamp) != nil || binary.Read(reader, binary.BigEndian, &temperature) != nil || binary.Read(reader, binary.BigEndian, &current) != nil {
				return nil, fmt.Errorf("%w: reading", ErrMalformedPayload)
			}
			reading := SensorReading{ID: id, RecordedMinute: time.Unix(stamp, 0).UTC(), TemperatureMilliC: int(temperature), CurrentMilliAmp: int(current), Sequence: int(sequence)}
			details = append(details, SnapshotDetail{Sequence: int(sequence), Kind: DetailReading, Reading: &reading})
		case 2:
			id, err := readString(reader)
			if err != nil {
				return nil, err
			}
			var stamp int64
			if binary.Read(reader, binary.BigEndian, &stamp) != nil {
				return nil, fmt.Errorf("%w: alert time", ErrMalformedPayload)
			}
			severity, err := readString(reader)
			if err != nil {
				return nil, err
			}
			code, err := readString(reader)
			if err != nil {
				return nil, err
			}
			message, err := readString(reader)
			if err != nil {
				return nil, err
			}
			alert := AlertSummary{ID: id, RecordedMinute: time.Unix(stamp, 0).UTC(), Severity: severity, Code: code, Message: message, Sequence: int(sequence)}
			details = append(details, SnapshotDetail{Sequence: int(sequence), Kind: DetailAlert, Alert: &alert})
		default:
			return nil, fmt.Errorf("%w: kind %d", ErrMalformedPayload, kind)
		}
	}
	if reader.Len() != 0 {
		return nil, fmt.Errorf("%w: trailing bytes", ErrMalformedPayload)
	}
	return details, nil
}

func writeString(writer io.Writer, value string) {
	binary.Write(writer, binary.BigEndian, uint32(len(value)))
	io.WriteString(writer, value)
}
func readString(reader *bytes.Reader) (string, error) {
	var size uint32
	if binary.Read(reader, binary.BigEndian, &size) != nil || size > uint32(reader.Len()) {
		return "", fmt.Errorf("%w: string", ErrMalformedPayload)
	}
	value := make([]byte, size)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return string(value), nil
}
