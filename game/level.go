package game

import (
	"fmt"
	"github.com/runningwild/linear"
)

type Door struct {
	Region linear.Poly
	Dest   int
}

type Room struct {
	Walls  map[string]linear.Poly
	Lava   map[string]linear.Poly
	Start  linear.Vec2
	End    linear.Vec2
	Doors  map[string]Door
	Dx, Dy int
	NextId int
}

func (r *Room) AddWall(wall linear.Poly) {
	r.Walls[fmt.Sprintf("%d", r.NextId)] = wall
	r.NextId++
}
