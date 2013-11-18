// +build nographics

package game

import (
	"github.com/runningwild/cgf"
)

type cameraInfo struct{}

func (p *PlayerEnt) Draw(game *Game)  {}
func (cp *ControlPoint) Draw(g *Game) {}
func (m *HeatSeeker) Draw(g *Game)    {}
func (m *Mine) Draw(g *Game)          {}
func (c *CreepEnt) Draw(g *Game)      {}

type manaSourceLocalData struct{}

func (ms *ManaSource) Draw(zoom float64, dx float64, dy float64) {}

func (g *Game) SetEngine(engine *cgf.Engine) {
	g.local.Engine = engine
}
