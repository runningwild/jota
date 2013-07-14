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
	Doors  map[string]Door
	Dx, Dy int
	NextId int
}

func (r *Room) AddWall(wall linear.Poly) {
	r.Walls[fmt.Sprintf("%d", r.NextId)] = wall
	r.NextId++
}
