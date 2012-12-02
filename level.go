package main

import (
  "runningwild/linear"
)

type Door struct {
  Region linear.Poly
  Dest   int
}

type Room struct {
  Polys []linear.Poly
  Doors []Door
}