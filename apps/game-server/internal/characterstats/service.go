package characterstats

import (
	"errors"
	"fractallegend/game-server/internal/equipment"
)

type Stats struct {
	MaxHP        float64 `json:"maxHp"`
	MaxMP        float64 `json:"maxMp"`
	AttackMin    float64 `json:"attackMin"`
	AttackMax    float64 `json:"attackMax"`
	Defense      float64 `json:"defense"`
	MagicDefense float64 `json:"magicDefense"`
	MoveSpeed    float64 `json:"moveSpeed"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Calculate(base Stats, values []equipment.Modifiers) (equipment.Modifiers, Stats, error) {
	var total equipment.Modifiers
	for _, value := range values {
		total.HPBonus += value.HPBonus
		total.MPBonus += value.MPBonus
		total.AttackMinBonus += value.AttackMinBonus
		total.AttackMaxBonus += value.AttackMaxBonus
		total.DefenseBonus += value.DefenseBonus
		total.MagicDefenseBonus += value.MagicDefenseBonus
	}
	runtime := Stats{
		MaxHP: base.MaxHP + total.HPBonus, MaxMP: base.MaxMP + total.MPBonus,
		AttackMin: base.AttackMin + total.AttackMinBonus, AttackMax: base.AttackMax + total.AttackMaxBonus,
		Defense: base.Defense + total.DefenseBonus, MagicDefense: base.MagicDefense + total.MagicDefenseBonus,
		MoveSpeed: base.MoveSpeed,
	}
	if err := s.ValidateStatBounds(runtime); err != nil {
		return equipment.Modifiers{}, Stats{}, err
	}
	return total, runtime, nil
}

func (s *Service) ValidateStatBounds(value Stats) error {
	if value.MaxHP <= 0 || value.MaxMP < 0 || value.AttackMin < 0 || value.AttackMax < value.AttackMin || value.Defense < 0 || value.MagicDefense < 0 || value.MoveSpeed < 0 {
		return errors.New("invalid character stat bounds")
	}
	return nil
}

func (s *Service) ClampVitals(hp, mp float64, runtime Stats) (float64, float64) {
	if hp > runtime.MaxHP {
		hp = runtime.MaxHP
	}
	if mp > runtime.MaxMP {
		mp = runtime.MaxMP
	}
	if hp < 0 {
		hp = 0
	}
	if mp < 0 {
		mp = 0
	}
	return hp, mp
}
