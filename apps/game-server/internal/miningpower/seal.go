package miningpower

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"time"
)

const SealRuleVersion = "G21_P0_SEAL_V1"
const SealSchemaVersion = "G21_P0_SCHEMA_V1"
const TestParticipantCap = 500

// SealedWeight is a historical beneficiary total, never a live participant
// aggregate or a reward. Multiple sessions for one CharacterID merge here.
type SealedWeight struct {
	PlayerID, AccountID, CharacterID string
	Power                            int64
	ActivityCount                    int64
}

type SealedActivityIdentity struct {
	ActivityID, SourceEventID string
}

type SettlementInputSeal struct {
	SchemaVersion, SealRuleVersion, SealID, BlockInstanceID, DisplayBlockID              string
	G18ReservationSourceID, G18ReservationEvidenceDigest, G18RuleVersion, G20RuleVersion string
	WindowIdentity                                                                       string
	WindowStartedAt, WindowEndedAt, SealedAt                                             time.Time
	ActivityCount, ParticipantCount                                                      int
	TotalValidMiningPower                                                                int64
	ParticipantWeights                                                                   []SealedWeight
	AcceptedActivityIdentities                                                           []SealedActivityIdentity
	AcceptedActivityIdentityDigest, BeneficiaryBindingDigest, Eligibility                string
	CanonicalDigest                                                                      string `json:"-"`
}

// CanonicalSealBytes has no maps and excludes the digest itself. The caller
// constructs sorted slices and UTC, monotonic-free, microsecond times.
func CanonicalSealBytes(s SettlementInputSeal) ([]byte, error) {
	if s.SchemaVersion != SealSchemaVersion || s.SealRuleVersion != SealRuleVersion ||
		s.ParticipantCount > TestParticipantCap || s.ParticipantCount != len(s.ParticipantWeights) ||
		s.ActivityCount < 0 || s.ActivityCount != len(s.AcceptedActivityIdentities) {
		return nil, ErrInvariant
	}
	weightTotal := big.NewInt(0)
	for i, w := range s.ParticipantWeights {
		if w.CharacterID == "" || w.Power < 0 || w.ActivityCount <= 0 ||
			(i > 0 && s.ParticipantWeights[i-1].CharacterID >= w.CharacterID) {
			return nil, ErrInvariant
		}
		weightTotal.Add(weightTotal, big.NewInt(w.Power))
	}
	if weightTotal.Cmp(big.NewInt(s.TotalValidMiningPower)) != 0 {
		return nil, ErrInvariant
	}
	for i, a := range s.AcceptedActivityIdentities {
		if a.ActivityID == "" || a.SourceEventID == "" || (i > 0 && s.AcceptedActivityIdentities[i-1].ActivityID >= a.ActivityID) {
			return nil, ErrInvariant
		}
	}
	if s.WindowStartedAt != CanonicalTime(s.WindowStartedAt) || s.WindowEndedAt != CanonicalTime(s.WindowEndedAt) ||
		s.SealedAt != CanonicalTime(s.SealedAt) {
		return nil, ErrInvariant
	}
	return json.Marshal(s)
}

func DigestSealBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum)
}

// SortSealedWeights uses the same CharacterID order for the persisted seal and
// native deterministic evidence. The caller passes a slice it owns.
func SortSealedWeights(weights []SealedWeight) {
	sort.Slice(weights, func(i, j int) bool { return weights[i].CharacterID < weights[j].CharacterID })
}

func AcceptedActivityIdentityDigest(identities []SealedActivityIdentity) (string, error) {
	raw, err := json.Marshal(identities)
	if err != nil {
		return "", err
	}
	return DigestSealBytes(raw), nil
}

func BeneficiaryBindingOrderDigest(digests []string) (string, error) {
	raw, err := json.Marshal(digests)
	if err != nil {
		return "", err
	}
	return DigestSealBytes(raw), nil
}
