package util

import "time"

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
