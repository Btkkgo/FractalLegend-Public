package rules

import "time"

func CooldownReady(last, now time.Time, interval time.Duration) bool {
	return last.IsZero() || !now.Before(last.Add(interval))
}
