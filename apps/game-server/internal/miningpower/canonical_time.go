package miningpower

import "time"

// CanonicalTime fixes the authoritative representation without changing the
// instant or its precision. Zero remains zero; process monotonic data is removed.
func CanonicalTime(t time.Time) time.Time { return t.UTC().Round(0) }

// Canonical returns a value copy for use at construction and persistence decode
// boundaries, before an immutable fact is published or compared.
func (s ActivitySession) Canonical() ActivitySession {
	s.OpenedAt = CanonicalTime(s.OpenedAt)
	s.ExpiresAt = CanonicalTime(s.ExpiresAt)
	s.CreatedAt = CanonicalTime(s.CreatedAt)
	s.UpdatedAt = CanonicalTime(s.UpdatedAt)
	return s
}

func (s SourceEvent) Canonical() SourceEvent {
	s.ObservedAt = CanonicalTime(s.ObservedAt)
	s.ExpiresAt = CanonicalTime(s.ExpiresAt)
	return s
}

func (b BlockContext) Canonical() BlockContext {
	b.StartedAt = CanonicalTime(b.StartedAt)
	b.ScheduledEndAt = CanonicalTime(b.ScheduledEndAt)
	b.ValidatedAt = CanonicalTime(b.ValidatedAt)
	return b
}

func (s MiningBlockValidationSnapshot) Canonical() MiningBlockValidationSnapshot {
	s.StartedAt = CanonicalTime(s.StartedAt)
	s.ScheduledEndAt = CanonicalTime(s.ScheduledEndAt)
	s.ValidatedAt = CanonicalTime(s.ValidatedAt)
	return s
}

func (a ValidatedMiningActivity) Canonical() ValidatedMiningActivity {
	a.AcceptedAt = CanonicalTime(a.AcceptedAt)
	a.ObservedAt = CanonicalTime(a.ObservedAt)
	a.ExpiresAt = CanonicalTime(a.ExpiresAt)
	a.ValidationSnapshot = a.ValidationSnapshot.Canonical()
	return a
}

func (p MiningParticipant) Canonical() MiningParticipant {
	p.CreatedAt = CanonicalTime(p.CreatedAt)
	p.UpdatedAt = CanonicalTime(p.UpdatedAt)
	return p
}
