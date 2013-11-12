package game

import (
	"github.com/runningwild/linear"
)

type Room struct {
	Walls    map[string]linear.Poly
	Dx, Dy   int
	SideData []roomSideData
	Towers   []towerData
}

func (r *Room) ExistsLos(a, b linear.Vec2) bool {
	los := linear.Seg2{a, b}
	for _, wall := range r.Walls {
		for i := range wall {
			seg := wall.Seg(i)
			if seg.DoesIsect(los) {
				return false
			}
		}
	}
	return true
}

// ExistsClearLos checks that there is a clear LoS bettween two points and
// expands all polys in the room so that anything that might normally only clip
// on a vertex will be guaranteed to block LoS.
func (r *Room) ExistsClearLos(a, b linear.Vec2, epsilon float64) bool {
	los := linear.Seg2{a, b}
	var expandedPoly linear.Poly
	for _, wall := range r.Walls {
		expandPoly(wall, &expandedPoly)
		for i := range expandedPoly {
			seg := expandedPoly.Seg(i)
			if seg.DoesIsect(los) {
				return false
			}
		}
	}
	return true
}

type roomSideData struct {
	Base linear.Vec2 // Position of the base for this side
}

type towerData struct {
	Pos linear.Vec2

	// Side that controls the tower at the beginning of the game.
	Side int

	// Indexes into the list of Towers representing towers that this tower can
	// spawn ents to go capture.
	Targets []int
}
