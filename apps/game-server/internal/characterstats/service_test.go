package characterstats

import (
	"fractallegend/game-server/internal/equipment"
	"testing"
)

func TestServiceCalculatesBaseModifiersAndRuntimeStats(t *testing.T) {
	s := NewService()
	base := Stats{MaxHP: 120, MaxMP: 40, AttackMin: 8, AttackMax: 12, Defense: 4, MagicDefense: 2, MoveSpeed: 4}
	mods := []equipment.Modifiers{{HPBonus: 10, MPBonus: 5, AttackMinBonus: 3, AttackMaxBonus: 5, DefenseBonus: 2, MagicDefenseBonus: 1}}
	aggregated, runtime, err := s.Calculate(base, mods)
	if err != nil {
		t.Fatal(err)
	}
	if aggregated.HPBonus != 10 || runtime.MaxHP != 130 || runtime.MaxMP != 45 || runtime.AttackMin != 11 || runtime.AttackMax != 17 || runtime.Defense != 6 || runtime.MagicDefense != 3 || runtime.MoveSpeed != 4 {
		t.Fatalf("aggregated=%+v runtime=%+v", aggregated, runtime)
	}
}

func TestServiceClampsCurrentHPAndMPOnlyWhenMaximumFalls(t *testing.T) {
	s := NewService()
	hp, mp := s.ClampVitals(130, 45, Stats{MaxHP: 120, MaxMP: 40})
	if hp != 120 || mp != 40 {
		t.Fatalf("hp=%v mp=%v", hp, mp)
	}
	hp, mp = s.ClampVitals(100, 30, Stats{MaxHP: 130, MaxMP: 45})
	if hp != 100 || mp != 30 {
		t.Fatalf("equip refilled vitals hp=%v mp=%v", hp, mp)
	}
}

func TestServiceRejectsInvalidBounds(t *testing.T) {
	s := NewService()
	_, _, err := s.Calculate(Stats{MaxHP: 1, MaxMP: 1, AttackMin: 10, AttackMax: 9}, nil)
	if err == nil {
		t.Fatal("invalid attack bounds accepted")
	}
}
