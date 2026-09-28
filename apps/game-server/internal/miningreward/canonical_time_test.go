package miningreward

import (
	"testing"
	"time"
)

func TestG21CanonicalTimeMicrosecondContract(t *testing.T) {
	base := time.Date(2026, 9, 28, 12, 0, 0, 123456000, time.UTC)
	for _, tc := range []struct {
		name        string
		input, want time.Time
	}{
		{"lower_submicrosecond", base.Add(time.Nanosecond), base},
		{"upper_submicrosecond_not_rounded", base.Add(999 * time.Nanosecond), base},
		{"next_microsecond", base.Add(time.Microsecond), time.Date(2026, 9, 28, 12, 0, 0, 123457000, time.UTC)},
		{"same_instant_offset", time.Date(2026, 9, 28, 20, 0, 0, 123456999, time.FixedZone("+0800", 8*3600)), base},
		{"different_instant", base.Add(time.Second), time.Date(2026, 9, 28, 12, 0, 1, 123456000, time.UTC)},
		{"zero", time.Time{}, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalTime(tc.input)
			if got != tc.want || got.Location() != time.UTC || CanonicalTime(got) != got {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
		})
	}
	now := time.Now()
	got := CanonicalTime(now)
	if got != got.Round(0) || got.Nanosecond()%1000 != 0 {
		t.Fatal("monotonic clock or submicroseconds survived")
	}
}
