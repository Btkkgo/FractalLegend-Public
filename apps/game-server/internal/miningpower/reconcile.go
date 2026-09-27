package miningpower

import (
	"math/big"
	"sort"
	"strconv"
	"time"
)

type participantKey struct{ player, block, session, rule string }
type profileKey struct{ rule, reference string }

func activityKey(a ValidatedMiningActivity) participantKey {
	return participantKey{a.PlayerID, a.BlockInstanceID, a.ActivitySessionID, a.RuleVersion}
}
func aggregateKey(p MiningParticipant) participantKey {
	return participantKey{p.PlayerID, p.BlockInstanceID, p.ActivitySessionID, p.RuleVersion}
}
func (k participantKey) label() string {
	return k.player + "/" + k.block + "/" + k.session + "/" + k.rule
}
func decimal(v int64) string { return strconv.FormatInt(v, 10) }
func ptr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func finding(identity, field, status, expected, actual string) ReconciliationFinding {
	f := ReconciliationFinding{Identity: identity, Field: field, Status: status, Expected: ptr(expected), Actual: ptr(actual)}
	e, eok := new(big.Int).SetString(expected, 10)
	a, aok := new(big.Int).SetString(actual, 10)
	if eok && aok {
		f.Delta = ptr(new(big.Int).Sub(a, e).String())
	}
	return f
}
func sortFindings(findings []ReconciliationFinding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Identity != b.Identity {
			return a.Identity < b.Identity
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Status < b.Status
	})
}
func reportStatus(findings []ReconciliationFinding) string {
	if len(findings) == 0 {
		return "PASS"
	}
	for _, f := range findings {
		if f.Status == "DUPLICATE" || f.Status == "UNKNOWN_RULE" || f.Status == "INVALID_REFERENCE" || f.Status == "OVERFLOW" || f.Status == "UNVERIFIABLE" {
			return "UNVERIFIABLE"
		}
	}
	return "MISMATCH"
}

// RebuildParticipants projects immutable facts into memory. It has no write
// dependency and never applies a correction to facts, aggregates or assets.
func RebuildParticipants(snapshot Snapshot) (RebuildResult, error) {
	r, _, err := rebuildParticipants(snapshot)
	return r, err
}

func rebuildParticipants(snapshot Snapshot) (RebuildResult, map[participantKey]bool, error) {
	r := RebuildResult{Participants: []MiningParticipant{}, Findings: []ReconciliationFinding{}}
	bad := func(id, field, status, expected, actual string) {
		r.Findings = append(r.Findings, finding(id, field, status, expected, actual))
	}
	rules := map[string]RuleManifest{}
	tools := map[profileKey]ToolProfile{}
	maps := map[profileKey]MapProfile{}
	sessions := map[string]ActivitySession{}
	sources := map[string]SourceEvent{}
	blocks := map[string]BlockContext{}
	ambiguousRules := map[string]bool{}
	ambiguousTools := map[profileKey]bool{}
	ambiguousMaps := map[profileKey]bool{}
	ambiguousSessions := map[string]bool{}
	ambiguousSources := map[string]bool{}
	ambiguousBlocks := map[string]bool{}
	for _, v := range snapshot.Rules {
		if _, ok := rules[v.Version]; ok {
			ambiguousRules[v.Version] = true
			bad(v.Version, "rule", "DUPLICATE", "unique", "duplicate")
		}
		rules[v.Version] = v
	}
	for _, v := range snapshot.Tools {
		k := profileKey{v.RuleVersion, v.Reference}
		if _, ok := tools[k]; ok {
			ambiguousTools[k] = true
			bad(v.Reference, "tool", "DUPLICATE", "unique", "duplicate")
		}
		tools[k] = v
	}
	for _, v := range snapshot.Maps {
		k := profileKey{v.RuleVersion, v.Reference}
		if _, ok := maps[k]; ok {
			ambiguousMaps[k] = true
			bad(v.Reference, "map", "DUPLICATE", "unique", "duplicate")
		}
		maps[k] = v
	}
	for _, v := range snapshot.Sessions {
		v = v.Canonical()
		if _, ok := sessions[v.ID]; ok {
			ambiguousSessions[v.ID] = true
			bad(v.ID, "session", "DUPLICATE", "unique", "duplicate")
		}
		sessions[v.ID] = v
	}
	sourceActivity := map[string]string{}
	for _, v := range snapshot.Sources {
		v = v.Canonical()
		if previous, repeated := sourceActivity[v.ActivityID]; repeated {
			ambiguousSources[previous] = true
			ambiguousSources[v.ID] = true
		}
		if _, ok := sources[v.ID]; ok || ambiguousSources[v.ID] {
			ambiguousSources[v.ID] = true
			bad(v.ID, "sourceIdentity", "DUPLICATE", "unique", "duplicate")
		}
		sources[v.ID] = v
		sourceActivity[v.ActivityID] = v.ID
	}
	for _, v := range snapshot.Blocks {
		v = v.Canonical()
		if _, ok := blocks[v.BlockInstanceID]; ok {
			ambiguousBlocks[v.BlockInstanceID] = true
			bad(v.BlockInstanceID, "block", "DUPLICATE", "unique", "duplicate")
		}
		blocks[v.BlockInstanceID] = v
	}
	aggregates := map[participantKey]MiningParticipant{}
	unverifiable := map[participantKey]bool{}
	poison := func(key participantKey) { unverifiable[key] = true; delete(aggregates, key) }
	// Pre-scan immutable identities so an ambiguity invalidates every affected
	// group regardless of iteration order, including the first occurrence.
	activityCounts := map[string]int{}
	sourceCounts := map[string]int{}
	for _, a := range snapshot.Activities {
		activityCounts[a.ActivityID]++
		sourceCounts[a.SourceEventID]++
	}
	for _, a := range snapshot.Activities {
		id := a.ActivityID
		a = a.Canonical()
		key := activityKey(a)
		if activityCounts[id] > 1 || sourceCounts[a.SourceEventID] > 1 {
			bad(id, "activityIdentity", "DUPLICATE", "unique", "duplicate")
			poison(key)
			continue
		}
		if ambiguousRules[a.RuleVersion] || ambiguousTools[profileKey{a.RuleVersion, a.ToolReference}] || ambiguousMaps[profileKey{a.RuleVersion, a.MapReference}] || ambiguousSessions[a.ActivitySessionID] || ambiguousSources[a.SourceEventID] || ambiguousBlocks[a.BlockInstanceID] {
			bad(id, "historicalReference", "UNVERIFIABLE", "", "ambiguous immutable identity")
			poison(key)
			continue
		}
		if ValidateIntent(ActionIntent{a.ActivityID, a.SourceEventID, a.ActivitySessionID, a.BlockID, a.BlockInstanceID}) != nil || !ValidID(a.PlayerID, 128) || a.Status != StatusValid || a.Decision != DecisionAccepted {
			bad(id, "activityIdentity", "INVALID_EVIDENCE", "canonical accepted identity", "")
			poison(key)
			continue
		}
		rule, ok := rules[a.RuleVersion]
		if !ok || rule != DevelopmentManifest() {
			bad(id, "rule", "UNKNOWN_RULE", "", a.RuleVersion)
			poison(key)
			continue
		}
		tool, ok := tools[profileKey{a.RuleVersion, a.ToolReference}]
		if !ok || tool.Kind != SyntheticKind || !ValidID(a.ToolReference, 128) {
			bad(id, "tool", "INVALID_REFERENCE", "", a.ToolReference)
			poison(key)
			continue
		}
		m, ok := maps[profileKey{a.RuleVersion, a.MapReference}]
		if !ok || m.Kind != SyntheticKind || !ValidID(a.MapReference, 128) {
			bad(id, "map", "INVALID_REFERENCE", "", a.MapReference)
			poison(key)
			continue
		}
		session, ok := sessions[a.ActivitySessionID]
		if !ok || session.PlayerID != a.PlayerID || session.BlockID != a.BlockID || session.BlockInstanceID != a.BlockInstanceID || session.RuleVersion != a.RuleVersion || session.ToolReference != a.ToolReference || session.MapReference != a.MapReference {
			bad(id, "session", "INVALID_REFERENCE", "", a.ActivitySessionID)
			poison(key)
			continue
		}
		source, ok := sources[a.SourceEventID]
		if !ok || source.ActivityID != id || source.PlayerID != a.PlayerID || source.BlockID != a.BlockID || source.BlockInstanceID != a.BlockInstanceID || source.ActivitySessionID != a.ActivitySessionID || source.RuleVersion != a.RuleVersion || source.ToolReference != a.ToolReference || source.MapReference != a.MapReference {
			bad(id, "source", "INVALID_REFERENCE", "", a.SourceEventID)
			poison(key)
			continue
		}
		block, ok := blocks[a.BlockInstanceID]
		if !ok {
			bad(id, "block", "INVALID_REFERENCE", "", a.BlockID)
			poison(key)
			continue
		}
		v := a.ValidationSnapshot
		if v.BlockInstanceID != a.BlockInstanceID || v.BlockID != a.BlockID || v.BlockHeight < 1 || v.Status != "OPEN" || v.G20RuleVersion != a.RuleVersion || v.G18RuleVersion == "" || v.SourceEvidenceVersion != "G18_SCHEMA_0012" || v.CreateCommandID == "" || v.ValidatedAt.Before(a.ObservedAt) || v.ValidatedAt.After(a.AcceptedAt) || !v.StartedAt.Equal(block.StartedAt) || !v.ScheduledEndAt.Equal(block.ScheduledEndAt) {
			bad(id, "validationSnapshot", "INVALID_REFERENCE", "historical instance validation", "")
			poison(key)
			continue
		}
		if source.ServerEligibility != StatusValid || source.EvidenceKind != SyntheticEvidence || !a.ObservedAt.Equal(source.ObservedAt) || !a.ExpiresAt.Equal(source.ExpiresAt) || a.ObservedAt.Before(session.OpenedAt) || a.ObservedAt.Before(block.StartedAt) || !a.ObservedAt.Before(block.ScheduledEndAt) || a.AcceptedAt.Before(a.ObservedAt) || !a.AcceptedAt.Before(session.ExpiresAt) || !a.AcceptedAt.Before(source.ExpiresAt) || !a.AcceptedAt.Before(block.ScheduledEndAt) || !source.ExpiresAt.After(source.ObservedAt) || session.ExpiresAt.After(block.ScheduledEndAt) || a.AcceptedAt.Nanosecond()%1000 != 0 {
			bad(id, "evidence", "INVALID_EVIDENCE", "valid server window", "")
			poison(key)
			continue
		}
		expectedInputs := ValidatedInputs{tool.BasePowerUnits, tool.EfficiencyScaled, source.ActivityWeightScaled, m.ModifierScaled}
		if a.Inputs != expectedInputs {
			bad(id, "inputs", "INVALID_REFERENCE", "catalog/source inputs", "different snapshot")
			poison(key)
			continue
		}
		power, err := CalculatePower(rule, a.Inputs)
		if err != nil {
			status := "INVALID_EVIDENCE"
			if err == ErrOverflow {
				status = "OVERFLOW"
			}
			bad(id, "activityPower", status, "", decimal(a.ValidatedPower))
			poison(key)
			continue
		}
		if a.ValidatedPower != power {
			bad(id, "activityPower", "MISMATCH", decimal(power), decimal(a.ValidatedPower))
		}
		if unverifiable[key] {
			continue
		}
		p, exists := aggregates[key]
		if !exists {
			p = MiningParticipant{PlayerID: a.PlayerID, BlockID: a.BlockID, BlockInstanceID: a.BlockInstanceID, ActivitySessionID: a.ActivitySessionID, RuleVersion: a.RuleVersion, ToolReference: a.ToolReference, MapReference: a.MapReference, CreatedAt: a.AcceptedAt, UpdatedAt: a.AcceptedAt}
		}
		if p.ValidatedPower > MaxPower-power || p.ActivityCount == MaxPower {
			bad(key.label(), "participantPower", "OVERFLOW", "", "")
			poison(key)
			continue
		}
		p.ValidatedPower += power
		p.ActivityCount++
		if a.AcceptedAt.Before(p.CreatedAt) {
			p.CreatedAt = a.AcceptedAt
		}
		if a.AcceptedAt.After(p.UpdatedAt) {
			p.UpdatedAt = a.AcceptedAt
		}
		aggregates[key] = p
	}
	for _, p := range aggregates {
		r.Participants = append(r.Participants, p)
	}
	sort.Slice(r.Participants, func(i, j int) bool {
		return aggregateKey(r.Participants[i]).label() < aggregateKey(r.Participants[j]).label()
	})
	sortFindings(r.Findings)
	r.Status = reportStatus(r.Findings)
	return r, unverifiable, nil
}

// ReconcileSnapshot compares both directions without mutating the snapshot.
func ReconcileSnapshot(snapshot Snapshot) (ReconciliationReport, error) {
	rebuilt, unverifiable, err := rebuildParticipants(snapshot)
	if err != nil {
		return ReconciliationReport{}, err
	}
	r := ReconciliationReport{Checked: len(snapshot.Activities) + len(snapshot.Participants), Findings: append([]ReconciliationFinding{}, rebuilt.Findings...)}
	expected := map[participantKey]MiningParticipant{}
	actual := map[participantKey]MiningParticipant{}
	ambiguousActual := map[participantKey]bool{}
	for _, p := range rebuilt.Participants {
		expected[aggregateKey(p)] = p
	}
	for _, p := range snapshot.Participants {
		p = p.Canonical()
		key := aggregateKey(p)
		if _, ok := actual[key]; ok {
			ambiguousActual[key] = true
			r.Findings = append(r.Findings, finding(key.label(), "participant", "DUPLICATE", "unique", "duplicate"))
		}
		actual[key] = p
	}
	for key := range unverifiable {
		value := ""
		if got, ok := actual[key]; ok && !ambiguousActual[key] {
			value = decimal(got.ValidatedPower)
		}
		r.Findings = append(r.Findings, finding(key.label(), "participantPower", "UNVERIFIABLE", "", value))
	}
	compare := func(id, field, want, got string) {
		status := "MATCH"
		if want != got {
			status = "MISMATCH"
		}
		r.Comparisons = append(r.Comparisons, finding(id, field, status, want, got))
		if want != got {
			r.Findings = append(r.Findings, finding(id, field, "MISMATCH", want, got))
		}
	}
	for key, want := range expected {
		if ambiguousActual[key] {
			r.Findings = append(r.Findings, finding(key.label(), "participantPower", "UNVERIFIABLE", decimal(want.ValidatedPower), ""))
			continue
		}
		got, ok := actual[key]
		if !ok {
			r.Findings = append(r.Findings, finding(key.label(), "participant", "MISSING", decimal(want.ValidatedPower), ""))
			continue
		}
		compare(key.label(), "participantPower", decimal(want.ValidatedPower), decimal(got.ValidatedPower))
		compare(key.label(), "blockID", want.BlockID, got.BlockID)
		compare(key.label(), "activityCount", decimal(want.ActivityCount), decimal(got.ActivityCount))
		compare(key.label(), "tool", want.ToolReference, got.ToolReference)
		compare(key.label(), "map", want.MapReference, got.MapReference)
		compare(key.label(), "createdAt", want.CreatedAt.UTC().Format(time.RFC3339Nano), got.CreatedAt.UTC().Format(time.RFC3339Nano))
		compare(key.label(), "updatedAt", want.UpdatedAt.UTC().Format(time.RFC3339Nano), got.UpdatedAt.UTC().Format(time.RFC3339Nano))
	}
	for key, got := range actual {
		if _, ok := expected[key]; !ok && !unverifiable[key] {
			if ambiguousActual[key] {
				r.Findings = append(r.Findings, finding(key.label(), "participantPower", "UNVERIFIABLE", "", ""))
				continue
			}
			r.Findings = append(r.Findings, finding(key.label(), "participant", "EXTRA", "0", decimal(got.ValidatedPower)))
		}
	}
	sortFindings(r.Findings)
	sortFindings(r.Comparisons)
	r.Status = reportStatus(r.Findings)
	return r, nil
}
