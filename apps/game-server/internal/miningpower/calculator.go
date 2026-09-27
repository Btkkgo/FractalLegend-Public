package miningpower

import "math/big"

func CalculatePower(rule RuleManifest, inputs ValidatedInputs) (int64, error) {
	if rule != DevelopmentManifest() {
		return 0, ErrUnknownRule
	}
	if inputs.BasePowerUnits < 0 || inputs.EfficiencyScaled < 1 || inputs.EfficiencyScaled > MaxMultiplier || inputs.ActivityWeightScaled < 1 || inputs.ActivityWeightScaled > MaxMultiplier || inputs.MapModifierScaled < 1 || inputs.MapModifierScaled > MaxMultiplier {
		return 0, ErrInvalidInput
	}
	n := big.NewInt(inputs.BasePowerUnits)
	n.Mul(n, big.NewInt(inputs.EfficiencyScaled))
	n.Mul(n, big.NewInt(inputs.ActivityWeightScaled))
	n.Mul(n, big.NewInt(inputs.MapModifierScaled))
	d := big.NewInt(Scale)
	d.Mul(d, big.NewInt(Scale))
	d.Mul(d, big.NewInt(Scale))
	n.Quo(n, d)
	if !n.IsInt64() {
		return 0, ErrOverflow
	}
	return n.Int64(), nil
}
