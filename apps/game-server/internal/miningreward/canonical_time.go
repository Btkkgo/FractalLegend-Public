package miningreward

import "time"

// CanonicalTime is the G21 economic persistence contract: UTC, no monotonic
// clock, and microsecond truncation (never rounding). Apply before hashing,
// persistence, serialization, or comparison; G20 evidence keeps its own contract.
func CanonicalTime(t time.Time) time.Time {
	return t.UTC().Round(0).Truncate(time.Microsecond)
}
