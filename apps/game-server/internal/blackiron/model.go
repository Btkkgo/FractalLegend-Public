// Package blackiron defines server-authoritative Black Iron Ore material
// compatibility. It has no mint, currency, mining, reward or client API.
package blackiron

import (
	"errors"
	"math"
	"strings"
	"time"
)

const Version = "G19_BLACK_IRON_V1"
const Kind = "BLACK_IRON_ORE"
const Name = "黑铁矿石"
const EnglishName = "Black Iron Ore"

var ErrInvalid = errors.New("invalid black iron compatibility state")
var ErrNotConfigured = errors.New("black iron identity catalog is not configured")
var ErrMismatch = errors.New("black iron migration reconciliation mismatch")

// Alias is a reviewed material identity, never inferred from display text.
// DefinitionID and LegacyID remain stable; Kind is its canonical semantics.
type Alias struct {
	DefinitionID string `json:"definitionId"`
	LegacyID     int    `json:"legacyId"`
	LegacyName   string `json:"legacyName"`
	Evidence     string `json:"evidence"`
}

type Asset struct {
	InstanceID   string `json:"instanceId"`
	DefinitionID string `json:"definitionId"`
	LegacyID     int    `json:"legacyId"`
	Name         string `json:"name"`
	ItemType     string `json:"itemType"`
	Quantity     int64  `json:"quantity"`
	SlotIndex    int    `json:"slotIndex"`
	Location     string `json:"location"`
}

type Change struct {
	Source Asset `json:"source"`
	Target Asset `json:"target"`
}

type Receipt struct {
	Version        string    `json:"version"`
	CharacterID    string    `json:"characterId"`
	TargetKind     string    `json:"targetKind"`
	Status         string    `json:"status"`
	PreTotal       int64     `json:"preTotal"`
	PostTotal      int64     `json:"postTotal"`
	RevisionBefore int64     `json:"revisionBefore"`
	RevisionAfter  int64     `json:"revisionAfter"`
	Changes        []Change  `json:"changes"`
	CreatedAt      time.Time `json:"createdAt"`
	CompletedAt    time.Time `json:"completedAt"`
}

type Report struct {
	Balanced   bool     `json:"balanced"`
	Checked    int      `json:"checked"`
	Mismatches []string `json:"mismatches"`
}

func Normalize(a Asset, m Alias) (Asset, error) {
	if strings.TrimSpace(m.DefinitionID) == "" || m.LegacyID < 0 || strings.TrimSpace(m.LegacyName) == "" || m.LegacyName == Name || strings.TrimSpace(m.Evidence) == "" ||
		a.DefinitionID != m.DefinitionID || a.LegacyID != m.LegacyID || a.InstanceID == "" || a.ItemType != "MATERIAL" ||
		(a.Name != m.LegacyName && a.Name != Name) || a.Quantity <= 0 || a.Quantity > math.MaxInt32 || a.SlotIndex < 0 || a.Location != "INVENTORY" {
		return Asset{}, ErrInvalid
	}
	a.Name = Name
	return a, nil
}

func AddQuantity(total, quantity int64) (int64, error) {
	if total < 0 || quantity < 0 || total > math.MaxInt64-quantity {
		return 0, ErrInvalid
	}
	return total + quantity, nil
}
