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

	// 0.0 - 1.0, measured controlledness of the point
	Control float64

	// The side that controls the point
	Controller int

	// Whether or not the point is currently controlled.  This is always true when
	// Control is 1.0, but may be true or false otherwise.
	Controlled bool
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
				StatsInst:    stats.Make(100000, 1000000, 0, 0, 1, 50),
			},
		}
		g.AddEnt(&cp)
	}
}

func (cp *ControlPoint) Think(g *Game) {
	cp.BaseEnt.Think(g)

	// All of this is basic logic for capturing control points
	sides := make(map[int]int)
	var theSide int
	for _, ent := range g.temp.AllEnts {
		if _, ok := ent.(*Player); !ok {
			continue
		}
		if ent.Pos().Sub(cp.Position).Mag() > 2*cp.Stats().Size() {
			continue
		}
		sides[ent.Side()]++
		theSide = ent.Side()
	}
	if len(sides) != 1 {
		return
	}
	amt := 0.001 * float64(sides[theSide])
	if !cp.Controlled || theSide == cp.Controller {
		if cp.Control < 1.0 {
			cp.Control += amt
			if cp.Control >= 0.999 {
				cp.Control = 1.0
				cp.Controlled = true
				cp.Controller = theSide
			}
		}
	} else {
		if cp.Control > 0.0 {
			cp.Control -= amt
			if cp.Control <= 0.0001 {
				cp.Control = 0
				cp.Controlled = false
				cp.Controller = -1
			}
		}
	}
}

func (cp *ControlPoint) Draw(g *Game, side int) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.95)
	gl.Color4ub(50, 50, 100, 150)
	texture.Render(
		cp.Position.X-cp.Stats().Size()*2,
		cp.Position.Y-cp.Stats().Size()*2,
		2*cp.Stats().Size()*2,
		2*cp.Stats().Size()*2)

	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.0)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(50, 50, 50, 200)
	texture.Render(
		cp.Position.X-cp.Stats().Size(),
		cp.Position.Y-cp.Stats().Size(),
		2*cp.Stats().Size(),
		2*cp.Stats().Size())

	base.SetUniformF("status_bar", "frac", float32(cp.Control))
	if cp.Controlled {
		if side == cp.Controller {
			gl.Color4ub(0, 255, 0, 255)
		} else {
			gl.Color4ub(255, 0, 0, 255)
		}
	} else {
		gl.Color4ub(100, 100, 100, 255)
	}
	texture.Render(
		cp.Position.X-cp.Stats().Size(),
		cp.Position.Y-cp.Stats().Size(),
		2*cp.Stats().Size(),
		2*cp.Stats().Size())
	base.EnableShader("")
}
func (cp *ControlPoint) Supply(mana Mana) Mana { return Mana{} }
func (cp *ControlPoint) Walls() [][]linear.Vec2 {
	return nil
}
