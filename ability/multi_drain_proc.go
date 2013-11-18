package ability

import (
	"github.com/runningwild/jota/game"
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
