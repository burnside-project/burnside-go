// Package epoch provides types and utilities for CDC epoch ordering and parsing.
package epoch

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Epoch represents a CDC batch boundary.
type Epoch struct {
	Number    int64
	Timestamp time.Time
}

// After returns true if e is after other.
func (e Epoch) After(other Epoch) bool {
	return e.Number > other.Number
}

// ParseFromFilename extracts the epoch number from a filename like "epoch=000042.parquet".
func ParseFromFilename(name string) (int64, error) {
	base := filepath.Base(name)

	// Strip extension
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Handle "epoch=000042" format
	if strings.HasPrefix(base, "epoch=") {
		numStr := strings.TrimPrefix(base, "epoch=")
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse epoch from %q: %w", name, err)
		}
		return n, nil
	}

	return 0, fmt.Errorf("unrecognized epoch filename format: %q", name)
}

// FormatFilename returns the standard epoch filename: "epoch=000042.parquet".
func FormatFilename(number int64) string {
	return fmt.Sprintf("epoch=%06d.parquet", number)
}

// FilterAfter returns only epochs with numbers strictly greater than afterEpoch,
// sorted in ascending order.
func FilterAfter(files []string, afterEpoch int64) ([]string, error) {
	type parsed struct {
		path   string
		number int64
	}

	var results []parsed
	for _, f := range files {
		n, err := ParseFromFilename(f)
		if err != nil {
			continue // skip non-epoch files
		}
		if n > afterEpoch {
			results = append(results, parsed{path: f, number: n})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].number < results[j].number
	})

	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.path
	}
	return out, nil
}
