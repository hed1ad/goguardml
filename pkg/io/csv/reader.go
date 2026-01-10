// Package csv provides CSV file reading for tabular data.
package csv

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
)

// Reader reads data from CSV files.
type Reader struct {
	file      *os.File
	reader    *csv.Reader
	hasHeader bool
	headers   []string
}

// Option configures a CSV reader.
type Option func(*Reader)

// WithHeader indicates the CSV has a header row.
func WithHeader(has bool) Option {
	return func(r *Reader) {
		r.hasHeader = has
	}
}

// NewReader creates a new CSV reader.
func NewReader(filename string, opts ...Option) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	r := &Reader{
		file:      file,
		reader:    csv.NewReader(file),
		hasHeader: true,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Read header if present
	if r.hasHeader {
		headers, err := r.reader.Read()
		if err != nil {
			file.Close()
			return nil, err
		}
		r.headers = headers
	}

	return r, nil
}

// Headers returns the column headers.
func (r *Reader) Headers() []string {
	return r.headers
}

// Read returns all data as a 2D float slice.
func (r *Reader) Read() ([][]float64, error) {
	var data [][]float64

	for {
		record, err := r.reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		row, err := parseRow(record)
		if err != nil {
			continue // Skip malformed rows
		}
		data = append(data, row)
	}

	return data, nil
}

// Stream returns a channel of rows for real-time processing.
func (r *Reader) Stream(ctx context.Context) (<-chan []float64, error) {
	out := make(chan []float64, 100)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				record, err := r.reader.Read()
				if err == io.EOF {
					return
				}
				if err != nil {
					continue
				}

				row, err := parseRow(record)
				if err != nil {
					continue
				}

				select {
				case out <- row:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// Close releases resources.
func (r *Reader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// parseRow converts string slice to float slice.
func parseRow(record []string) ([]float64, error) {
	if len(record) == 0 {
		return nil, errors.New("empty row")
	}

	row := make([]float64, len(record))
	for i, val := range record {
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, err
		}
		row[i] = f
	}
	return row, nil
}
