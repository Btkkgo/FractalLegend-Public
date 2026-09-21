package rules

import (
	"math/rand"
	"testing"
	"time"
)

func TestManhattanRange(t *testing.T) {
	if !InRange(10, 10, 10, 12, 2) || InRange(10, 10, 12, 12, 2) {
		t.Fatal("Manhattan range boundary is incorrect")
	}
}

func TestDamageFormula(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	damage := Damage(8, 12, 3, 1, rng)
	if damage < 5 || damage > 9 {
		t.Fatalf("damage=%d", damage)
	}
}

func TestMinimumDamage(t *testing.T) {
	if got := Damage(2, 2, 99, 1, rand.New(rand.NewSource(1))); got != 1 {
		t.Fatalf("damage=%d", got)
	}
}

func TestDamageRNGDeterminism(t *testing.T) {
	a := rand.New(rand.NewSource(42))
	b := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		if Damage(8, 12, 2, 1, a) != Damage(8, 12, 2, 1, b) {
			t.Fatal("same server seed produced different damage")
		}
	}
}

func TestCooldown(t *testing.T) {
	last := time.Unix(100, 0)
	if CooldownReady(last, last.Add(499*time.Millisecond), 500*time.Millisecond) {
		t.Fatal("cooldown accepted early attack")
	}
	if !CooldownReady(last, last.Add(500*time.Millisecond), 500*time.Millisecond) {
		t.Fatal("cooldown rejected boundary attack")
	}
}

func TestApplyDamage(t *testing.T) {
	if got := ApplyDamage(5, 9); got != 0 {
		t.Fatalf("hp=%d", got)
	}
}
