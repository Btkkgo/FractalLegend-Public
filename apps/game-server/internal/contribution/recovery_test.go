package contribution

import (
	"errors"
	"testing"
)

func TestG14SpendGuardPrioritizesReviewDebtAndAvailableBalance(t *testing.T) {
	cases := []struct {
		name    string
		account Account
		amount  int64
		want    error
	}{
		{"manual review", Account{Balance: 100, ReviewRequired: true}, 1, ErrManualReview},
		{"recovery hold", Account{Balance: 0, RecoveryDebt: 80}, 1, ErrRecoveryHold},
		{"insufficient", Account{Balance: 20}, 21, ErrInsufficient},
		{"invalid amount", Account{Balance: 20}, 0, ErrInvalidAmount},
		{"available", Account{Balance: 20}, 20, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAvailableSpend(tc.account, tc.amount)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}
