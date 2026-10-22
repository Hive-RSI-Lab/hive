package system

import (
	"testing"
	"time"
)

func TestSchedules(t *testing.T) {
	start := time.Date(2026, 10, 22, 1, 0, 0, 0, time.Local)
	every, err := Parse("every 12h")
	if err != nil {
		t.Fatal(err)
	}
	if every.Due(start.Add(time.Hour), time.Time{}, start, false) {
		t.Fatal("due before the first interval")
	}
	if !every.Due(start.Add(13*time.Hour), time.Time{}, start, false) {
		t.Fatal("not due after the interval")
	}
	if !every.Due(start.Add(time.Minute), time.Time{}, start, true) {
		t.Fatal("at-start job not due")
	}
	daily, _ := Parse("daily 04:00")
	if daily.Due(start.Add(2*time.Hour), time.Time{}, start, false) {
		t.Fatal("daily due before its time")
	}
	if !daily.Due(start.Add(3*time.Hour+time.Minute), time.Time{}, start, false) {
		t.Fatal("daily not due after its time")
	}
	if daily.Due(start.Add(4*time.Hour), start.Add(3*time.Hour+30*time.Minute), start, false) {
		t.Fatal("daily ran twice")
	}
	if off, _ := Parse("off"); off.Due(start.Add(1000*time.Hour), time.Time{}, start, true) {
		t.Fatal("off ran")
	}
	if _, err := Parse("sometimes"); err == nil {
		t.Fatal("accepted a bad schedule")
	}
}
