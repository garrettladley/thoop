package xtime

import "time"

// IsToday checks if the given time is the same calendar day as today.
func IsToday(t time.Time) bool {
	now := time.Now()
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
