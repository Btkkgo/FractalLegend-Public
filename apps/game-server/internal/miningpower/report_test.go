package miningpower

import (
	"encoding/json"
	"testing"
)

func TestG20ComparisonEvidence(t *testing.T) {
	r, e := ReconcileSnapshot(g20Snapshot())
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	var body map[string]json.RawMessage
	if e = json.Unmarshal(raw, &body); e != nil {
		t.Fatal(e)
	}
	var comparisons []ReconciliationFinding
	if e = json.Unmarshal(body["Comparisons"], &comparisons); e != nil {
		t.Fatalf("expected/actual/delta evidence missing: %v", e)
	}
	found := false
	for _, row := range comparisons {
		if row.Field == "participantPower" && row.Expected != nil && row.Actual != nil && row.Delta != nil && *row.Expected == "100" && *row.Actual == "100" && *row.Delta == "0" && row.Status == "MATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("zero-delta comparison absent: %+v", comparisons)
	}
}
