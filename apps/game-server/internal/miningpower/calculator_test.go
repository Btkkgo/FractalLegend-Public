package miningpower

import (
	"encoding/json"
	"errors"
	"math"
	"runtime"
	"strconv"
	"testing"
)

func TestG20DeterministicVectors(t *testing.T) {
	logG20NativeRuntime(t)
	for _, tc := range []struct {
		name             string
		b, e, a, m, want int64
		overflow         bool
	}{
		{"T29/basic", 100, 1000000, 1000000, 1000000, 100, false},
		{"T33/advanced", 250, 1000000, 1000000, 1000000, 250, false},
		{"T34/combined", 100, 1250000, 500000, 1500000, 93, false},
		{"T37/fractional-zero", 1, 500000, 500000, 1000000, 0, false},
		{"T34/no-intermediate-floor", 3, 1500000, 1500000, 1000000, 6, false},
		{"T43-T44/wide-max", math.MaxInt64, 1000000, 1000000, 1000000, math.MaxInt64, false},
		{"T42/overflow", math.MaxInt64, 1000001, 1000000, 1000000, 0, true},
		{"T30/efficiency", 100, 1250000, 1000000, 1000000, 125, false},
		{"T31/activity", 100, 1000000, 500000, 1000000, 50, false},
		{"T32/map", 100, 1000000, 1000000, 1500000, 150, false},
		{"T36/zero-base", 0, 1000000, 1000000, 1000000, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inputs := ValidatedInputs{BasePowerUnits: tc.b, EfficiencyScaled: tc.e, ActivityWeightScaled: tc.a, MapModifierScaled: tc.m}
			for repeat := 0; repeat < 10; repeat++ {
				got, err := CalculatePower(DevelopmentManifest(), inputs)
				if repeat == 0 {
					wantError := error(nil)
					if tc.overflow {
						wantError = ErrOverflow
					}
					logG20NativeVector(t, tc.name, DevelopmentManifest(), inputs, tc.want, wantError, got, err)
				}
				if tc.overflow {
					if !errors.Is(err, ErrOverflow) {
						t.Fatalf("output overflow accepted: %d %v", got, err)
					}
					continue
				}
				if err != nil || got != tc.want {
					t.Fatalf("exact vector got=%d want=%d err=%v", got, tc.want, err)
				}
			}
		})
	}
}

func TestG20CalculatorBounds(t *testing.T) {
	logG20NativeRuntime(t)
	base := ValidatedInputs{BasePowerUnits: 100, EfficiencyScaled: 1000000, ActivityWeightScaled: 1000000, MapModifierScaled: 1000000}
	for _, tc := range []struct {
		name   string
		inputs ValidatedInputs
	}{
		{"T38/negative-base", ValidatedInputs{-1, 1000000, 1000000, 1000000}},
		{"T38/negative-efficiency", ValidatedInputs{1, -1, 1000000, 1000000}},
		{"T38/negative-activity", ValidatedInputs{1, 1000000, -1, 1000000}},
		{"T38/negative-map", ValidatedInputs{1, 1000000, 1000000, -1}},
		{"T39/zero-efficiency", ValidatedInputs{1, 0, 1000000, 1000000}},
		{"T35/zero-activity", ValidatedInputs{1, 1000000, 0, 1000000}},
		{"T39/zero-map", ValidatedInputs{1, 1000000, 1000000, 0}},
		{"T41/high-efficiency", ValidatedInputs{1, 1000000001, 1000000, 1000000}},
		{"T41/high-activity", ValidatedInputs{1, 1000000, 1000000001, 1000000}},
		{"T41/high-map", ValidatedInputs{1, 1000000, 1000000, 1000000001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CalculatePower(DevelopmentManifest(), tc.inputs)
			logG20NativeVector(t, tc.name, DevelopmentManifest(), tc.inputs, 0, ErrInvalidInput, got, err)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("invalid trusted input accepted: %v", err)
			}
		})
	}
	manifest := DevelopmentManifest()
	manifest.Version = "unknown"
	gotRule, errRule := CalculatePower(manifest, base)
	logG20NativeVector(t, "T40/unknown-rule", manifest, base, 0, ErrUnknownRule, gotRule, errRule)
	if err := errRule; !errors.Is(err, ErrUnknownRule) {
		t.Fatalf("T40 unknown rule accepted: %v", err)
	}
	manifest = DevelopmentManifest()
	manifest.Scale = 10
	gotScale, errScale := CalculatePower(manifest, base)
	logG20NativeVector(t, "T40/changed-scale", manifest, base, 0, ErrUnknownRule, gotScale, errScale)
	if err := errScale; !errors.Is(err, ErrUnknownRule) {
		t.Fatal("changed historical manifest accepted")
	}
	maxInputs := ValidatedInputs{0, 1000000000, 1000000000, 1000000000}
	got, err := CalculatePower(DevelopmentManifest(), maxInputs)
	logG20NativeVector(t, "T41/max-multipliers", DevelopmentManifest(), maxInputs, 0, nil, got, err)
	if err != nil || got != 0 {
		t.Fatalf("T41 valid maximum multipliers: %d %v", got, err)
	}
}

// Evidence logging uses the existing fixtures and calculator results only.
// It changes no algorithm, golden expectation, power assertion or test case.
func logG20NativeRuntime(t *testing.T) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "go_version": runtime.Version(), "scale": Scale, "rounding": "final division floor"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("T50_RUNTIME_JSON %s", raw)
}

func logG20NativeVector(t *testing.T, name string, rule RuleManifest, inputs ValidatedInputs, want int64, wantError error, got int64, gotError error) {
	t.Helper()
	result := func(power int64, err error) string {
		if err != nil {
			return "ERROR:" + err.Error()
		}
		return "POWER:" + strconv.FormatInt(power, 10)
	}
	raw, err := json.Marshal(map[string]any{"name": name, "rule": rule, "inputs": inputs, "expected": result(want, wantError), "actual": result(got, gotError)})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("T50_VECTOR_JSON %s", raw)
}
