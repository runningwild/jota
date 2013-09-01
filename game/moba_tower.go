package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

type ControlPoint struct {
	BaseEnt
}

func (g *Game) MakeControlPoints() {
	data := g.Levels[GidInvadersStart].Room.Moba.SideData
	neutralData := data[len(data)-1]
	for _, towerPos := range neutralData.Towers {
		cp := ControlPoint{
			BaseEnt: BaseEnt{
				Side_:        -1,
				CurrentLevel: GidInvadersStart,
				Position:     towerPos,
				StatsInst:    stats.Make(100000, 1000000, 0, 0, 1, 100),
			},
		}
		g.AddEnt(&cp)
	}
}

func (cp *ControlPoint) Draw(g *Game, ally bool) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.95)
	gl.Color4ub(0, 200, 0, 250)
	texture.Render(
		cp.Position.X-100,
		cp.Position.Y-100,
		2*100,
		2*100)
	base.EnableShader("")
}
func (cp *ControlPoint) Supply(mana Mana) Mana { return Mana{} }
func (cp *ControlPoint) Walls() [][]linear.Vec2 {
	return nil
}
