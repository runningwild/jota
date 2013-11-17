package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"math"
)

type ControlPoint struct {
	BaseEnt

	// 0.0 - 1.0, measured controlledness of the point
	Control float64

	// If Controlled, this is the side that controls this point.
	// If not Controlled, this is the side that is currently capping it.
	Controller int

	// Whether or not the point is currently controlled.  This is always true when
	// Control is 1.0, but may be true or false otherwise.
	Controlled bool

	// If an enemy comes into LOS of the CP when the AttackTimer is at zero then
	// an attack process will begin and the AttackTimer will be set.  It will
	// count down on every think until it reaches zero again.
	AttackTimer int

	// Radius of region within which players/creeps must be to capture.
	Radius float64

	// Other control points that this one can send creeps to attack.
	Targets []Gid
}

func (g *Game) MakeControlPoints() {
	var cps []*ControlPoint
	for _, towerData := range g.Level.Room.Towers {
		cp := ControlPoint{
			BaseEnt: BaseEnt{
				Abilities_: []Ability{ability_makers["spawnCreeps"](map[string]float64{})},
				Side_:      towerData.Side,
				Position:   towerData.Pos,
				Processes:  make(map[int]Process),
				StatsInst: stats.Make(stats.Base{
					Health: 100000,
					Mass:   1000000,
					Rate:   1,
					Size:   0,
					Vision: 900,
				}),
			},
			Radius: 50,
		}
		cps = append(cps, &cp)
		g.AddEnt(&cp)

		if towerData.Side != -1 {
			cp.Controlled = true
			cp.Control = 1.0
			cp.Controller = towerData.Side
			// Must do this after the call to AddEnt() because BindAi requires that
			// the ent's Gid has been set
			cp.BindAi("tower", g.local.Engine)
		}
	}

	// Now set up the target arrays
	for i := range cps {
		for _, index := range g.Level.Room.Towers[i].Targets {
			cps[i].Targets = append(cps[i].Targets, cps[index].Id())
		}
	}
}
func (cp *ControlPoint) Type() EntType {
	return EntTypeControlPoint
}
func (cp *ControlPoint) Side() int {
	if cp.Controlled {
		return cp.Controller
	}
	return cp.BaseEnt.Side()
}

func (cp *ControlPoint) Think(g *Game) {
	cp.BaseEnt.Think(g)

	// All of this is basic logic for capturing control points

	// Find the first side that isn't -1
	side := -1
	count := 0
	var ents []Ent
	g.local.temp.EntGrid.EntsInRange(cp.Position, cp.Radius, &ents)
	controlRangeSquared := cp.Radius * cp.Radius
	for _, ent := range ents {
		if ent.Side() == -1 {
			continue
		}
		if _, ok := ent.(*ControlPoint); ok {
			continue
		}
		if ent.Pos().Sub(cp.Position).Mag2() > controlRangeSquared {
			continue
		}
		if side == -1 {
			side = ent.Side()
			count++
		} else {
			if ent.Side() != side {
				side = -1
				break
			}
			count++
		}
	}

	// If there is a single side contesting this control point, check that this
	// point can actually be captured by this side.  It can be captured if one of
	// the following is true:
	// 1. This is a base control point (i.e. side == cp.Side_)  *OR*
	// 2. At least one of this control point's targets in controlled by side.
	capturable := false
	if side == cp.Side_ {
		capturable = true
	}
	for _, targetGid := range cp.Targets {
		tcp := g.Ents[targetGid].(*ControlPoint)
		if tcp.Controlled && tcp.Controller == side {
			capturable = true
		}
	}
	if !capturable {
		side = -1
	}

	if side != -1 {
		amt := 0.001 * math.Sqrt(float64(count))
		switch {
		case cp.Controlled && side == cp.Controller:
			// Can't recap something you already control.

		case cp.Controlled && side != cp.Controller:
			cp.Control -= amt

		case !cp.Controlled && side == cp.Controller:
			cp.Control += amt

		case !cp.Controlled && side != cp.Controller:
			cp.Control -= amt
		}
		if cp.Control <= 0.0001 {
			cp.Control = 0
			cp.Controlled = false
			cp.Controller = side
			if cp.ai != nil {
				cp.ai.Terminate()
				cp.ai = nil
			}
		}
		if cp.Control >= 0.999 && cp.Controller == side {
			cp.Control = 1.0
			cp.Controlled = true
			if cp.ai == nil {
				cp.BindAi("tower", g.local.Engine)
			}
		}
	}
}

func (cp *ControlPoint) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.0)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(50, 50, 50, 50)
	texture.Render(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius)

	enemyColor := []gl.Ubyte{255, 0, 0, 100}
	allyColor := []gl.Ubyte{0, 255, 0, 100}
	neutralColor := []gl.Ubyte{100, 100, 100, 100}
	var rgba []gl.Ubyte
	if cp.Controlled {
		if g.local.Side == cp.Controller {
			rgba = allyColor
		} else {
			rgba = enemyColor
		}
	} else {
		rgba = neutralColor
	}

	// The texture is flipped if this is being drawn for the controlling side.
	// This makes it look a little nicer when someone neutralizes a control point
	// because it makes the angle of the pie slice thingy continue going in the
	// same direction as it passes the neutralization point.
	gl.Color4ub(rgba[0], rgba[1], rgba[2], rgba[3])
	base.SetUniformF("status_bar", "frac", float32(cp.Control))
	texture.RenderAdvanced(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius,
		0,
		g.local.Side == cp.Controller)

	base.SetUniformF("status_bar", "inner", 0.45)
	base.SetUniformF("status_bar", "outer", 0.5)
	base.SetUniformF("status_bar", "frac", 1)
	gl.Color4ub(rgba[0], rgba[1], rgba[2], 255)
	texture.RenderAdvanced(
		cp.Position.X-cp.Radius,
		cp.Position.Y-cp.Radius,
		2*cp.Radius,
		2*cp.Radius,
		0,
		g.local.Side == cp.Controller)

	if !cp.Controlled {
		base.SetUniformF("status_bar", "frac", float32(cp.Control))
		if g.local.Side == cp.Controller {
			gl.Color4ub(allyColor[0], allyColor[1], allyColor[2], 255)
		} else {
			gl.Color4ub(enemyColor[0], enemyColor[1], enemyColor[2], 255)
		}
		texture.RenderAdvanced(
			cp.Position.X-cp.Radius,
			cp.Position.Y-cp.Radius,
			2*cp.Radius,
			2*cp.Radius,
			0,
			g.local.Side == cp.Controller)
	}

	base.EnableShader("")
}
func (cp *ControlPoint) Supply(mana Mana) Mana {
	base.DoOrdered(cp.Processes, func(a, b int) bool { return a < b }, func(id int, proc Process) {
		mana = proc.Supply(mana)
	})
	return mana
}
