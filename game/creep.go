package game

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
)

type CreepEnt struct {
	BaseEnt
}

func init() {
	gob.Register(&CreepEnt{})
}
func (c *CreepEnt) Think(g *Game) {
	c.BaseEnt.Think(g)
}

func (c *CreepEnt) Supply(mana Mana) Mana {
	return mana
}
func (c *CreepEnt) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (g *Game) AddCreeps(pos linear.Vec2, count, side int, params map[string]interface{}) {
	if side < 0 || side >= len(g.Level.Room.SideData) {
		base.Error().Fatalf("Got side %d, but this level only supports sides from 0 to %d.", len(g.Level.Room.SideData)-1)
		return
	}
	for i := 0; i < count; i++ {
		var c CreepEnt
		c.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    100.0,
			Rate:   0.0,
			Size:   8,
			Vision: 600,
		})

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{50, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(count))
		c.Position = pos.Add(rot)

		c.Side_ = side
		c.Gid = g.NextGid()

		// for _, ability := range g.Champs[playerData.champ].Abilities {
		//  c.Abilities_ = append(
		//    c.Abilities_,
		//    ability_makers[ability.Name](ability.Params))
		// }

		// if playerData.gid[0:2] == "Ai" {
		//  c.BindAi("simple", g.local.Engine)
		// }

		g.AddEnt(&c)
		c.BindAi("creep", g.local.Engine)
		for name, value := range params {
			c.ai.SetParam(name, value)
		}
	}
}
