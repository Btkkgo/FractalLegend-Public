package ai

import "sort"

type State string

const (
	Idle   State = "IDLE"
	Aggro  State = "AGGRO"
	Chase  State = "CHASE"
	Attack State = "ATTACK"
	Return State = "RETURN"
	Dead   State = "DEAD"
)

func (s State) Valid() bool {
	switch s {
	case Idle, Aggro, Chase, Attack, Return, Dead:
		return true
	}
	return false
}

func Manhattan(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

type Candidate struct {
	ID       string
	X, Y     int
	Valid    bool
	Distance int
}

// Nearest is deterministic: distance first, stable entity ID second.
func Nearest(x, y, maxDistance int, values []Candidate) (Candidate, bool) {
	valid := make([]Candidate, 0, len(values))
	for _, value := range values {
		if !value.Valid {
			continue
		}
		value.Distance = Manhattan(x, y, value.X, value.Y)
		if value.Distance <= maxDistance {
			valid = append(valid, value)
		}
	}
	if len(valid) == 0 {
		return Candidate{}, false
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Distance == valid[j].Distance {
			return valid[i].ID < valid[j].ID
		}
		return valid[i].Distance < valid[j].Distance
	})
	return valid[0], true
}
