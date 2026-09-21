package contribution

// ValidateAvailableSpend is an internal rule check. A future spend producer
// must execute this check and its debit atomically; G14 exposes no writer.
func ValidateAvailableSpend(account Account, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if account.ReviewRequired || account.Balance < 0 || account.RecoveryDebt < 0 || (account.RecoveryDebt > 0 && account.Balance > 0) {
		return ErrManualReview
	}
	if account.RecoveryDebt > 0 {
		return ErrRecoveryHold
	}
	if account.Balance < amount {
		return ErrInsufficient
	}
	return nil
}
