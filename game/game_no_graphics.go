// +build nographics

package game

import (
	"github.com/runningwild/glop/gui"
	g2 "github.com/runningwild/jota/gui"
)

type cameraInfo struct {
}

func (g *Game) RenderLocal(region g2.Region, local *LocalData) {
}

func (p *PlayerEnt) Draw(game *Game) {
}

func (gw *GameWindow) Draw(region gui.Region, style gui.StyleStack) {
}
