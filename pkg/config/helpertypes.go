package config

import (
	"strconv"
	"time"
)

// TimeDuration is a encoding/json-compatible wrapper around time.Duration
type TimeDuration struct {
	time.Duration
}

func (d *TimeDuration) UnmarshalJSON(b []byte) error {
	// Try to handle raw numeric JSON values first (e.g., 30, 2.5)
	if b[0] != '"' {
		// Parse as float64 and treat as seconds
		if f, err := strconv.ParseFloat(string(b), 64); err == nil {
			d.Duration = time.Duration(f * float64(time.Second))
			return nil
		}
	}

	// Unquote the string (e.g., "5s" → 5s, "30" → 30)
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	// First try to parse as a time.Duration string
	duration, err := time.ParseDuration(s)
	if err == nil {
		d.Duration = duration
		return nil
	}

	// If that fails, try to parse as a plain number (treat as seconds)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		d.Duration = time.Duration(f * float64(time.Second))
		return nil
	}

	// If both fail, return the original duration parsing error
	return err
}

func (d *TimeDuration) Unwrap() time.Duration {
	return d.Duration
}
