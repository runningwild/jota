package main

import (
  "github.com/runningwild/linear"
)

type Door struct {
  Region linear.Poly
  Dest   int
}

type Room struct {
  Walls []linear.Poly
  Doors []Door
}
