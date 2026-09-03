package ocr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"", 0, false},
		{"500ms", 500 * time.Millisecond, false},
		{"10s", 10 * time.Second, false},
		{"2m", 2 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"2days", 48 * time.Hour, false},
		{"0.5d", 12 * time.Hour, false},
		{"120", 120 * time.Second, false},
		{"invalid_dur", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseDuration(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}
		})
	}
}

func TestParseRateInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"", 0, false},
		{"500ms", 500 * time.Millisecond, false},
		{"10s", 10 * time.Second, false},
		{"2m", 2 * time.Minute, false},
		{"30m", 30 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"120", 120 * time.Second, false},
		{"2/h", 30 * time.Minute, false},
		{"2/hour", 30 * time.Minute, false},
		{"30/m", 2 * time.Second, false},
		{"30/min", 2 * time.Second, false},
		{"10/s", 100 * time.Millisecond, false},
		{"100/d", time.Duration(float64(24*time.Hour) / 100.0), false},
		{"1/30m", 30 * time.Minute, false},
		{"invalid/unit", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseRateInterval(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}
		})
	}
}
