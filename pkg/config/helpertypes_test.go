package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		// Standard time.Duration string formats
		{
			name:     "seconds with s suffix",
			input:    `"5s"`,
			expected: 5 * time.Second,
			wantErr:  false,
		},
		{
			name:     "minutes with m suffix",
			input:    `"10m"`,
			expected: 10 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours with h suffix",
			input:    `"2h"`,
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "milliseconds with ms suffix",
			input:    `"500ms"`,
			expected: 500 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "microseconds with μs suffix",
			input:    `"100μs"`,
			expected: 100 * time.Microsecond,
			wantErr:  false,
		},
		{
			name:     "nanoseconds with ns suffix",
			input:    `"1000ns"`,
			expected: 1000 * time.Nanosecond,
			wantErr:  false,
		},
		{
			name:     "complex duration",
			input:    `"1h30m45s"`,
			expected: 1*time.Hour + 30*time.Minute + 45*time.Second,
			wantErr:  false,
		},
		{
			name:     "zero duration",
			input:    `"0s"`,
			expected: 0,
			wantErr:  false,
		},

		// Plain numeric inputs (should be treated as seconds)
		{
			name:     "plain integer as seconds",
			input:    `"30"`,
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "plain float as seconds",
			input:    `"2.5"`,
			expected: time.Duration(2.5 * float64(time.Second)),
			wantErr:  false,
		},
		{
			name:     "zero as plain number",
			input:    `"0"`,
			expected: 0,
			wantErr:  false,
		},

		// Numeric JSON values (without quotes) - should also work
		{
			name:     "raw JSON number as seconds",
			input:    `30`,
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "raw JSON float as seconds",
			input:    `2.5`,
			expected: time.Duration(2.5 * float64(time.Second)),
			wantErr:  false,
		},

		// Error cases
		{
			name:    "invalid duration string",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			input:   `"5s`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var td TimeDuration
			err := json.Unmarshal([]byte(tt.input), &td)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if td.Duration != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, td.Duration)
			}
		})
	}
}

func TestTimeDuration_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{
			name:     "zero duration",
			duration: 0,
		},
		{
			name:     "positive duration",
			duration: 5 * time.Second,
		},
		{
			name:     "complex duration",
			duration: 1*time.Hour + 30*time.Minute + 45*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := TimeDuration{Duration: tt.duration}
			unwrapped := td.Unwrap()

			if unwrapped != tt.duration {
				t.Errorf("expected %v, got %v", tt.duration, unwrapped)
			}
		})
	}
}

func TestTimeDuration_JSONRoundTrip(t *testing.T) {
	// Test struct containing TimeDuration
	type Config struct {
		Timeout TimeDuration `json:"timeout"`
	}

	tests := []struct {
		name     string
		jsonStr  string
		expected time.Duration
	}{
		{
			name:     "duration string format",
			jsonStr:  `{"timeout": "5s"}`,
			expected: 5 * time.Second,
		},
		{
			name:     "numeric string format",
			jsonStr:  `{"timeout": "30"}`,
			expected: 30 * time.Second,
		},
		{
			name:     "raw numeric format",
			jsonStr:  `{"timeout": 45}`,
			expected: 45 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.jsonStr), &config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config.Timeout.Duration != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, config.Timeout.Duration)
			}

			// Test that Unwrap() returns the correct value
			if config.Timeout.Unwrap() != tt.expected {
				t.Errorf("Unwrap() expected %v, got %v", tt.expected, config.Timeout.Unwrap())
			}
		})
	}
}

func TestTimeDuration_ZeroValue(t *testing.T) {
	var td TimeDuration
	
	// Zero value should be zero duration
	if td.Duration != 0 {
		t.Errorf("expected zero duration, got %v", td.Duration)
	}
	
	if td.Unwrap() != 0 {
		t.Errorf("Unwrap() expected zero duration, got %v", td.Unwrap())
	}
}