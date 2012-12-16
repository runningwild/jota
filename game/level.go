package game

import (
	"github.com/runningwild/linear"
)

type Door struct {
	Region linear.Poly
	Dest   int
}

type Room struct {
	Walls []linear.Poly
	Lava  []linear.Poly
	Doors []Door
}
