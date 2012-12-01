package main

import (
  "runningwild/tron/game"
)

type noRendering struct{}

func (noRendering) Draw(game *game.Game) {}
