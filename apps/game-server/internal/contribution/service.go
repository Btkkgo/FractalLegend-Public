package contribution

import "context"

type Service struct {
	repo   Repository
	policy EligibilityPolicy
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAccount(ctx context.Context, playerID string) (Account, error) {
	if s == nil || s.repo == nil {
		return Account{}, ErrUnavailable
	}
	if playerID == "" || len(playerID) > 128 {
		return Account{}, ErrInvalidAccount
	}
	return s.repo.CreateContributionAccount(ctx, playerID)
}

func (s *Service) Account(ctx context.Context, playerID string) (Account, error) {
	if s == nil || s.repo == nil {
		return Account{}, ErrUnavailable
	}
	return s.repo.LoadContributionAccount(ctx, playerID)
}

func (s *Service) PostSystemSpend(ctx context.Context, request SpendRequest) (PostingResult, error) {
	if s == nil || s.repo == nil {
		return PostingResult{}, ErrUnavailable
	}
	if request.PlayerID == "" || request.PlayerFBAccountID == "" || request.SystemFBAccountID == "" || request.PlayerFBAccountID == request.SystemFBAccountID || request.SourceID == "" || len(request.SourceID) > 128 || len(request.PlayerID) > 128 || len(request.PlayerFBAccountID) > 128 || len(request.SystemFBAccountID) > 128 {
		return PostingResult{}, ErrInvalidSource
	}
	amount, err := s.policy.Evaluate(request.Source, request.EligibleSpend, request.RuleVersion)
	if err != nil {
		return PostingResult{}, err
	}
	if amount == 0 {
		return PostingResult{}, ErrIneligibleSource
	}
	return s.repo.PostContributionSystemSpend(ctx, request)
}

func (s *Service) Entries(ctx context.Context, playerID string) ([]Entry, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ContributionEntries(ctx, playerID)
}

func (s *Service) AuditEvents(ctx context.Context, playerID string) ([]AuditEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ContributionAuditEvents(ctx, playerID)
}

func (s *Service) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	if s == nil || s.repo == nil {
		return ReconciliationReport{}, ErrUnavailable
	}
	return s.repo.ReconcileContribution(ctx)
}
