package batch

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidBatchFile = errors.New("invalid batch file")
var fileMagic = [8]byte{'R', 'S', 'N', 'A', 'P', '0', '0', '1'}

type FileWriter struct {
	writer *bufio.Writer
	count  int
	closed bool
}

func NewFileWriter(writer io.Writer) (*FileWriter, error) {
	buffer := bufio.NewWriter(writer)
	if _, err := buffer.Write(fileMagic[:]); err != nil {
		return nil, err
	}
	return &FileWriter{writer: buffer}, nil
}
func (writer *FileWriter) WriteFrame(frame []byte) error {
	if writer.closed {
		return errors.New("batch writer is closed")
	}
	if len(frame) == 0 {
		return errors.New("empty frame")
	}
	if len(frame) > 64<<20 {
		return errors.New("frame exceeds 64 MiB")
	}
	if err := binary.Write(writer.writer, binary.BigEndian, uint32(len(frame))); err != nil {
		return err
	}
	if _, err := writer.writer.Write(frame); err != nil {
		return err
	}
	writer.count++
	return nil
}
func (writer *FileWriter) Close() error {
	if writer.closed {
		return nil
	}
	writer.closed = true
	if err := binary.Write(writer.writer, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	return writer.writer.Flush()
}
func (writer *FileWriter) Count() int { return writer.count }

type FileReader struct {
	reader *bufio.Reader
	done   bool
	index  int
}

func NewFileReader(reader io.Reader) (*FileReader, error) {
	buffer := bufio.NewReader(reader)
	header := make([]byte, len(fileMagic))
	if _, err := io.ReadFull(buffer, header); err != nil {
		return nil, fmt.Errorf("%w: header", ErrInvalidBatchFile)
	}
	if string(header) != string(fileMagic[:]) {
		return nil, fmt.Errorf("%w: magic", ErrInvalidBatchFile)
	}
	return &FileReader{reader: buffer}, nil
}
func (reader *FileReader) Next() ([]byte, error) {
	if reader.done {
		return nil, io.EOF
	}
	var size uint32
	if err := binary.Read(reader.reader, binary.BigEndian, &size); err != nil {
		return nil, fmt.Errorf("%w: frame size", ErrInvalidBatchFile)
	}
	if size == 0 {
		reader.done = true
		if _, err := reader.reader.Peek(1); err != io.EOF {
			return nil, fmt.Errorf("%w: trailing data", ErrInvalidBatchFile)
		}
		return nil, io.EOF
	}
	if size > 64<<20 {
		return nil, fmt.Errorf("%w: oversized frame", ErrInvalidBatchFile)
	}
	frame := make([]byte, size)
	if _, err := io.ReadFull(reader.reader, frame); err != nil {
		return nil, fmt.Errorf("%w: frame %d", ErrInvalidBatchFile, reader.index)
	}
	reader.index++
	return frame, nil
}
func ReadAll(reader io.Reader) ([][]byte, error) {
	source, err := NewFileReader(reader)
	if err != nil {
		return nil, err
	}
	var frames [][]byte
	for {
		frame, err := source.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
