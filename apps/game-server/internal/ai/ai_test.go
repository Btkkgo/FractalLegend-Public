package ai

import "testing"

func TestStatesAndNearestValidPlayer(t *testing.T) {
	states := []State{Idle, Aggro, Chase, Attack, Return, Dead}
	for _, state := range states {
		if !state.Valid() {
			t.Fatalf("invalid G5 state %q", state)
		}
	}
	got, ok := Nearest(10, 10, 6, []Candidate{
		{ID: "far", X: 14, Y: 10, Valid: true},
		{ID: "dead", X: 10, Y: 11, Valid: false},
		{ID: "near-b", X: 12, Y: 10, Valid: true},
		{ID: "near-a", X: 10, Y: 12, Valid: true},
	})
	if !ok || got.ID != "near-a" || got.Distance != 2 {
		t.Fatalf("nearest=%+v ok=%v", got, ok)
	}
	if _, ok = Nearest(10, 10, 1, []Candidate{{ID: "outside", X: 12, Y: 10, Valid: true}}); ok {
		t.Fatal("outside aggro range was selected")
	}
}

func TestManhattan(t *testing.T) {
	if got := Manhattan(2, 3, 7, 1); got != 7 {
		t.Fatalf("distance=%d", got)
	}
}
