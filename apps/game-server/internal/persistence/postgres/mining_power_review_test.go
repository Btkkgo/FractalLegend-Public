package postgres

import (
	"context"
	"encoding/json"
	"fractallegend/game-server/internal/miningpower"
	"testing"
)

func TestG20ClosedPersistedJSON(t *testing.T) {
	s, _, i := g20Fixture(t)
	ctx := context.Background()
	source := g20Source(t, s, i)
	for _, tc := range []struct {
		name, table, alias string
		value              any
	}{
		{"source-eligibility", "mining_power_source_events", "servereligibility", source},
		{"tool-base", "mining_power_tool_profiles", "basepowerunits", miningpower.ToolProfile{Reference: "alias-tool", RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, BasePowerUnits: 100, EfficiencyScaled: miningpower.Scale}},
		{"map-coefficient", "mining_power_map_profiles", "modifierscaled", miningpower.MapProfile{Reference: "alias-map", RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, ModifierScaled: miningpower.Scale}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, e := json.Marshal(tc.value)
			if e != nil {
				t.Fatal(e)
			}
			var data map[string]any
			if e = json.Unmarshal(raw, &data); e != nil {
				t.Fatal(e)
			}
			data[tc.alias] = 250
			if tc.table == "mining_power_source_events" {
				data["ID"] = "alias-event"
				data["ActivityID"] = "mpa:alias-event"
				data["ServerEligibility"] = "INVALID"
				data[tc.alias] = "VALID"
			}
			raw, e = json.Marshal(data)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO `+tc.table+`(data) VALUES($1)`, string(raw)); e == nil {
				t.Fatal("case alias overrode canonical SQL field")
			}
		})
	}
	a := g20Accept(t, s, miningpower.Principal{AccountID: "g20-player-account", PlayerID: "g20-player"}, i)
	next := g20Event(t, s, i, "nested-alias")
	a.ActivityID = next.ActivityID
	a.SourceEventID = next.SourceEventID
	raw, e := json.Marshal(a)
	if e != nil {
		t.Fatal(e)
	}
	var data map[string]any
	json.Unmarshal(raw, &data)
	for _, name := range []string{"Inputs", "ValidationSnapshot"} {
		t.Run(name, func(t *testing.T) {
			raw, e := json.Marshal(data)
			if e != nil {
				t.Fatal(e)
			}
			var v map[string]any
			json.Unmarshal(raw, &v)
			next := g20Event(t, s, i, "nested-"+name)
			v["ActivityID"] = next.ActivityID
			v["SourceEventID"] = next.SourceEventID
			v[name].(map[string]any)["unknownAlias"] = 1
			raw, e = json.Marshal(v)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO mining_power_activities(data) VALUES($1)`, string(raw)); e == nil {
				t.Fatal("nested persisted alias accepted")
			}
		})
	}
}
