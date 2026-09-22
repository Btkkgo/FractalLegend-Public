package emission

import (
	"errors"
	"math"
	"testing"
)

// A missing or malformed server rule must never create capacity.
func TestDevelopmentRuleCalculatesCapacityAndFailsClosed(t *testing.T) {
	got, err := Calculate(DevelopmentRuleVersion, 7)
	if err != nil || got != 7 {
		t.Fatalf("development capacity=%d error=%v", got, err)
	}
	for _, version := range []string{"", "UNKNOWN", "PRODUCTION_V1"} {
		if _, err := Calculate(version, 7); !errors.Is(err, ErrUnknownRule) {
			t.Fatalf("version %q: %v", version, err)
		}
	}
	for _, amount := range []int64{0, -1} {
		if _, err := Calculate(DevelopmentRuleVersion, amount); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %d: %v", amount, err)
		}
	}
	if _, err := AddCapacity(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow: %v", err)
	}
}
