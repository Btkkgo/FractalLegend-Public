package miningpower

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

type BeneficiaryEvidence struct {
	Version, ActivityID, SourceEventID, AccountID, PlayerID, CharacterID, BlockInstanceID, AuthoritySource string
	BoundAt                                                                                                time.Time
}

func BeneficiaryEvidenceDigest(v BeneficiaryEvidence) (string, error) {
	v.BoundAt = CanonicalTime(v.BoundAt)
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum), nil
}

type IssuanceEvidence struct {
	RuleVersion, SourceType, SourceKey, BlockInstanceID, CharacterID, MaterialDefinitionID string
	Quantity                                                                               int32
	CreatedAt                                                                              time.Time
}

const ReservationBindingVersion = "G21_P0_RESERVATION_V1"

type ReservationBindingEvidence struct {
	Version, InstanceID, SourceID, ReceiptID, BlockID, RuleVersion string
	Amount                                                         int64
}

func ReservationBindingEvidenceDigest(v ReservationBindingEvidence) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum), nil
}

func IssuanceEvidenceDigest(v IssuanceEvidence) (string, error) {
	v.CreatedAt = CanonicalTime(v.CreatedAt)
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum), nil
}
