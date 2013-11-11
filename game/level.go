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
