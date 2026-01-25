package xtime

import "time"

// IsToday checks if the given time is the same calendar day as today.
func IsToday(t time.Time) bool {
	return SameDay(t, time.Now())
}

// SameDay checks if two times are on the same calendar day.
func SameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// BeforeDay checks if time a is on a calendar day before time b.
func BeforeDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	if y1 != y2 {
		return y1 < y2
	}
	if m1 != m2 {
		return m1 < m2
	}
	return d1 < d2
}

// StartOfDay returns midnight (00:00:00) of the given time's date in its location.
func StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// LastDayOfMonth returns the last day of the month containing t.
func LastDayOfMonth(t time.Time) time.Time {
	firstOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	return firstOfNextMonth.AddDate(0, 0, -1)
}
