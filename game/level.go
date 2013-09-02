package game

import (
	"fmt"
	"github.com/runningwild/linear"
)

type Portal struct {
	Region linear.Poly
	Dest   int
}

type Room struct {
	Walls   map[string]linear.Poly
	Starts  []linear.Vec2
	End     linear.Vec2
	Portals map[string]Portal
	Dx, Dy  int
	NextId  int

	// Only filled for moba rooms
	Moba struct {
		SideData []mobaRoomSideData
	}
}

type mobaRoomSideData struct {
	Base   linear.Vec2   // Position of the base for this side
	Towers []linear.Vec2 // Positions of the towers for this side
	// Will also need waypoints for units, production and whatnot.
}

func (r *Room) AddWall(wall linear.Poly) {
	r.Walls[fmt.Sprintf("%d", r.NextId)] = wall
	r.NextId++
}
