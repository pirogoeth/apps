package util

import (
	"testing"
	"time"
)

func TestRoundDateToYear(t *testing.T) {
	// Test for rounding a date to the nearest year
	// Create a date that is not rounded to the nearest year
	date := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	// Round the date to the nearest year
	roundedDate := RoundDateToYear(date)
	// Check if the rounded date is the expected value
	expectedDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if roundedDate != expectedDate {
		t.Errorf("expected %v, got %v", expectedDate, roundedDate)
	}
}

func TestRoundDateToHalfYear(t *testing.T) {
	t.Run("in half 1", func(t *testing.T) {
		date := time.Date(2020, 4, 15, 0, 0, 0, 0, time.UTC)
		roundedDate := RoundDateToHalfYear(date)
		expectedDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		if roundedDate != expectedDate {
			t.Errorf("expected %v, got %v", expectedDate, roundedDate)
		}
	})

	t.Run("in half 2", func(t *testing.T) {
		date := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
		roundedDate := RoundDateToHalfYear(date)
		expectedDate := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
		if roundedDate != expectedDate {
			t.Errorf("expected %v, got %v", expectedDate, roundedDate)
		}
	})
}

func TestRoundToMonth(t *testing.T) {
	date := time.Date(2020, 8, 15, 0, 0, 0, 0, time.UTC)
	roundedDate := RoundDateToMonth(date)
	expectedDate := time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)
	if roundedDate != expectedDate {
		t.Errorf("expected %v, got %v", expectedDate, roundedDate)
	}
}
