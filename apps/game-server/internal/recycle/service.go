package recycle

import (
	"context"
	"time"
)

type Repository interface {
	PostRecycle(context.Context, Intent, Rule) (Receipt, error)
	LoadRecycle(context.Context, string) (Receipt, error)
	ReconcileRecycle(context.Context) (ReconciliationReport, error)
}

type Service struct {
	repo     Repository
	registry Registry
}

func NewService(repo Repository, registry Registry) *Service {
	return &Service{repo: repo, registry: registry}
}

func (s *Service) Recycle(ctx context.Context, intent Intent) (Receipt, error) {
	if s == nil || s.repo == nil {
		return Receipt{}, ErrInvalidIntent
	}
	if !intent.Valid() {
		return Receipt{}, ErrInvalidIntent
	}
	intent.RequestedAt = intent.RequestedAt.UTC().Truncate(time.Microsecond)
	previous, err := s.repo.LoadRecycle(ctx, intent.OperationID)
	if err == nil {
		if previous.PlayerID != intent.PlayerID || previous.ItemInstanceID != intent.ItemInstanceID ||
			previous.ItemRevision != intent.ExpectedItemRevision || previous.RuleID != intent.RuleID ||
			!previous.RequestedAt.Equal(intent.RequestedAt) {
			return Receipt{}, ErrConflict
		}
		return previous, nil
	}
	if err != ErrNotFound {
		return Receipt{}, err
	}
	rule, err := s.registry.Resolve(intent.RuleID)
	if err != nil {
		return Receipt{}, err
	}
	return s.repo.PostRecycle(ctx, intent, rule)
}

func (s *Service) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	if s == nil || s.repo == nil {
		return ReconciliationReport{}, ErrInvalidIntent
	}
	return s.repo.ReconcileRecycle(ctx)
}
