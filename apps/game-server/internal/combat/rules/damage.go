package rules

import "math/rand"

// Damage implements Fractal Combat Formula v1 using inclusive integer bounds.
func Damage(attackMin, attackMax, targetDefense, minimumDamage int, rng *rand.Rand) int {
	if attackMax < attackMin || minimumDamage < 1 || rng == nil {
		return 0
	}
	base := attackMin
	if attackMax > attackMin {
		base += rng.Intn(attackMax - attackMin + 1)
	}
	damage := base - targetDefense
	if damage < minimumDamage {
		return minimumDamage
	}
	return damage
}

func ApplyDamage(hp, damage int) int {
	if damage >= hp {
		return 0
	}
	return hp - damage
}
