package rules

func ManhattanDistance(ax, ay, bx, by int) int {
	dx := ax - bx
	if dx < 0 {
		dx = -dx
	}
	dy := ay - by
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func InRange(ax, ay, bx, by, attackRange int) bool {
	return attackRange >= 0 && ManhattanDistance(ax, ay, bx, by) <= attackRange
}
