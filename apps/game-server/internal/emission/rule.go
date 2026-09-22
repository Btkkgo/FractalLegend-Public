package emission

import (
	"errors"
	"math"
)

var (
	ErrUnknownRule   = errors.New("black iron emission rule is not active")
	ErrInvalidAmount = errors.New("invalid eligible spend amount")
	ErrOverflow      = errors.New("black iron emission capacity overflow")
)

// DevelopmentRuleVersion is an explicit test/development fixture. No
// production emission ratio is active or finalized in G17.
const DevelopmentRuleVersion = "DEV_G17_1_TO_1"

func Calculate(version string, eligibleSpend int64) (int64, error) {
	if version != DevelopmentRuleVersion {
		return 0, ErrUnknownRule
	}
	if eligibleSpend <= 0 {
		return 0, ErrInvalidAmount
	}
	return eligibleSpend, nil
}

func AddCapacity(current, amount int64) (int64, error) {
	if current < 0 || amount < 0 || current > math.MaxInt64-amount {
		return 0, ErrOverflow
	}
	return current + amount, nil
}
