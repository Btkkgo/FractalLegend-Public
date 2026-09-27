package miningpower

import "context"

type Service struct {
	repo        Repository
	ruleVersion string
	production  bool
}

func NewService(repo Repository, ruleVersion string, production bool) *Service {
	return &Service{repo: repo, ruleVersion: ruleVersion, production: production}
}
func (s *Service) Validate(ctx context.Context, principal Principal, intent ActionIntent) (ValidationResult, error) {
	if s == nil {
		return ValidationResult{}, ErrUnavailable
	}
	if s.production {
		return ValidationResult{Status: StatusNotEligible, ReasonCode: "PRODUCTION_NOT_APPROVED"}, nil
	}
	if err := ValidateIntent(intent); err != nil {
		return ValidationResult{Status: StatusInvalid, ReasonCode: "INVALID_IDENTITY"}, nil
	}
	if !ValidID(principal.PlayerID, 128) || !ValidID(principal.AccountID, 128) {
		return ValidationResult{Status: StatusNotEligible, ReasonCode: "INVALID_PRINCIPAL"}, nil
	}
	if s.repo == nil {
		return ValidationResult{}, ErrUnavailable
	}
	return s.repo.ValidateMiningActivity(ctx, principal, intent, s.ruleVersion)
}
