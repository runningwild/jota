package ability

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/texture"
	"math"
)

// Typical process for draining mana for an ability that can be triggered
// multiple times in discrete units.
type multiDrain struct {
	NullCondition

	// Gid of the Player with this Process
	Gid game.Gid

	// This is the amount of mana for a single trigger's worth of the associated
	// ability.
	Unit game.Mana

	// The number of multiples of Unit currently stored
	Stored float64

	Killed bool
}

func (p *multiDrain) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	base.EnableShader("status_bar")
	frac := p.Stored
	ready := math.Floor(frac)
	if ready == 0 {
		gl.Color4ub(255, 0, 0, 255)
	} else {
		gl.Color4ub(0, 255, 0, 255)
	}
	outer := 0.2
	increase := 0.01
	base.SetUniformF("status_bar", "frac", float32(frac-ready))
	base.SetUniformF("status_bar", "inner", float32(outer-increase*(ready+1)))
	base.SetUniformF("status_bar", "outer", float32(outer))
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	if ready > 0 {
		base.SetUniformF("status_bar", "frac", 1.0)
		base.SetUniformF("status_bar", "inner", float32(outer-ready*increase))
		base.SetUniformF("status_bar", "outer", float32(outer))
		texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	}
	base.EnableShader("")
}
func (p *multiDrain) Supply(mana game.Mana) game.Mana {
	frac := -1.0
	for color, amt := range p.Unit {
		if amt > 0 {
			thisFrac := mana[color] / amt
			if thisFrac < frac || frac == -1.0 {
				frac = thisFrac
			}
		}
	}
	for color := range mana {
		drainAmt := p.Unit[color] * frac
		mana[color] -= drainAmt
	}
	p.Stored += frac
	return mana
}
func (p *multiDrain) Think(g *game.Game) {
	if _, ok := g.Ents[p.Gid].(*game.PlayerEnt); ok {
		// TODO: Get this from the player stats once it's a stat.
		retention := 0.98
		p.Stored *= retention
	} else {
		p.Killed = true
	}
}
func (p *multiDrain) Kill(g *game.Game) {
	p.Killed = true
}
func (p *multiDrain) Dead() bool {
	return p.Killed
}
