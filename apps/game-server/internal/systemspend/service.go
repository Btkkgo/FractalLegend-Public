package systemspend

import (
	"context"
	"encoding/json"

	"fractallegend/game-server/internal/contribution"
)

type Service struct {
	repo     Repository
	registry Registry
	policy   contribution.EligibilityPolicy
}

func NewService(repo Repository, registry Registry) *Service {
	return &Service{repo: repo, registry: registry}
}

func (s *Service) Post(ctx context.Context, intent Intent) (SystemSpend, error) {
	if s == nil || s.repo == nil {
		return SystemSpend{}, ErrUnavailable
	}
	if !validID(intent.OperationID, 120) || !validID(intent.ProducerReference, 128) ||
		!validID(intent.PlayerID, 128) || !validID(intent.PlayerFBAccountID, 128) ||
		!validID(intent.SystemFBAccountID, 128) || intent.PlayerFBAccountID == intent.SystemFBAccountID ||
		intent.FBAmount <= 0 {
		return SystemSpend{}, ErrInvalidIntent
	}
	metadata, err := json.Marshal(intent.Metadata)
	if err != nil || len(metadata) > 2048 {
		return SystemSpend{}, ErrInvalidIntent
	}
	producer, err := s.registry.Resolve(intent.ProducerType)
	if err != nil {
		return SystemSpend{}, err
	}
	resolved := ResolvedIntent{Intent: intent, Eligible: producer.Eligible,
		RuleVersion: producer.AllowedRuleVersion, Refundable: producer.Refundable,
		PartialRefundAllowed: producer.PartialRefundAllowed}
	if producer.Eligible {
		resolved.ContributionAmount, err = s.policy.Evaluate(contribution.SourceSystemService, intent.FBAmount, producer.AllowedRuleVersion)
		if err != nil {
			return SystemSpend{}, err
		}
	}
	return s.repo.PostSystemSpend(ctx, resolved)
}

func (s *Service) Load(ctx context.Context, operationID string) (SystemSpend, error) {
	if s == nil || s.repo == nil {
		return SystemSpend{}, ErrUnavailable
	}
	if !validID(operationID, 120) {
		return SystemSpend{}, ErrInvalidIntent
	}
	return s.repo.LoadSystemSpend(ctx, operationID)
}

func (s *Service) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	if s == nil || s.repo == nil {
		return ReconciliationReport{}, ErrUnavailable
	}
	return s.repo.ReconcileSystemSpends(ctx)
}

func validID(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, c := range value {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' || c == '/') {
			return false
		}
	}
	return true
}
