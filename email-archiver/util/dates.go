package util

import (
	"fmt"
	"strings"
	"time"
)

const (
	RoundYearly     = "yearly"
	RoundSemiYearly = "semiyearly"
	RoundMonthly    = "monthly"
)

type DatePartitioner func(time.Time) time.Time

func GetDatePartitioner(name string) (DatePartitioner, error) {
	switch strings.ToLower(name) {
	case RoundYearly:
		return RoundDateToYear, nil
	case RoundSemiYearly:
		return RoundDateToHalfYear, nil
	case RoundMonthly:
		return RoundDateToMonth, nil
	default:
		return nil, fmt.Errorf("unknown name for DatePartitioner: `%s`", name)
	}
}

func RoundDateToYear(date time.Time) time.Time {
	return time.Date(date.Year(), 1, 1, 0, 0, 0, 0, date.Location())
}

func RoundDateToHalfYear(date time.Time) time.Time {
	halfYear := time.Date(date.Year(), 6, 1, 0, 0, 0, 0, date.Location())
	if date.After(halfYear) {
		return halfYear
	}

	return RoundDateToYear(date)
}

func RoundDateToMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}
