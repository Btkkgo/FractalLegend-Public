package navigation

import "testing"

func TestGreedyStepAndCollisionFallback(t *testing.T) {
	walkable := func(x, y int) bool { return x >= 0 && y >= 0 && x < 8 && y < 8 && !(x == 3 && y == 2) }
	x, y, status := GreedyStep(2, 2, 5, 4, walkable)
	if x != 2 || y != 3 || status != Moved {
		t.Fatalf("step=(%d,%d) status=%s", x, y, status)
	}
	x, y, status = GreedyStep(2, 2, 5, 2, walkable)
	if x != 2 || y != 2 || status != PathBlocked {
		t.Fatalf("blocked=(%d,%d) status=%s", x, y, status)
	}
}

func TestGreedyStepRejectsUnknownAndOutOfBounds(t *testing.T) {
	deny := func(int, int) bool { return false }
	x, y, status := GreedyStep(0, 0, -1, 0, deny)
	if x != 0 || y != 0 || status != PathBlocked {
		t.Fatalf("fail closed=(%d,%d) status=%s", x, y, status)
	}
}
