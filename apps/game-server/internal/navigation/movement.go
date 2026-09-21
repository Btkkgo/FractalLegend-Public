package navigation

type Status string

const (
	Moved       Status = "MOVED"
	Arrived     Status = "ARRIVED"
	PathBlocked Status = "PATH_BLOCKED"
)

// GreedyStep makes one deterministic orthogonal move. The supplied validator is
// authoritative for bounds, blocked cells, and unknown collision data.
func GreedyStep(x, y, targetX, targetY int, walkable func(int, int) bool) (int, int, Status) {
	if x == targetX && y == targetY {
		return x, y, Arrived
	}
	dx, dy := targetX-x, targetY-y
	steps := make([][2]int, 0, 2)
	if abs(dx) >= abs(dy) {
		if dx != 0 {
			steps = append(steps, [2]int{sign(dx), 0})
		}
		if dy != 0 {
			steps = append(steps, [2]int{0, sign(dy)})
		}
	} else {
		if dy != 0 {
			steps = append(steps, [2]int{0, sign(dy)})
		}
		if dx != 0 {
			steps = append(steps, [2]int{sign(dx), 0})
		}
	}
	for _, step := range steps {
		nx, ny := x+step[0], y+step[1]
		if walkable(nx, ny) {
			return nx, ny, Moved
		}
	}
	return x, y, PathBlocked
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
