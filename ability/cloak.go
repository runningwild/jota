package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
)

func makeCloak(params map[string]float64) game.Ability {
	var c cloak
	c.id = NextAbilityId()
	c.maxCloak = params["maxCloak"]
	c.manaPerCloak = params["manaPerCloak"]
	c.cloakPerTick = params["cloakPerTick"]
	return &c
}

func init() {
	game.RegisterAbility("cloak", makeCloak)
	gob.Register(&cloak{})
}

type cloak struct {
	id int

	// Params
	maxCloak     float64
	manaPerCloak float64
	cloakPerTick float64

	on       bool
	previous struct {
		pressAmt float64
		trigger  bool
	}
}

func (c *cloak) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	player := ent.(*game.PlayerEnt)
	if pressAmt == c.previous.pressAmt {
		return
	}
	if !c.on {
		c.on = pressAmt > 0
	} else {
		c.on = pressAmt == 0
	}
	if c.on {
		player.Processes[c.id] = &cloakProc{Gid: player.Gid, MaxCloak: c.maxCloak, ManaPerCloak: c.manaPerCloak, CloakPerTick: c.cloakPerTick}
	} else {
		delete(player.Processes, c.id)
	}
}

func (c *cloak) Think(ent game.Ent, g *game.Game) {
}
func (f *cloak) Draw(ent game.Ent, g *game.Game) {
}
func (f *cloak) IsActive() bool {
	return false
}

// Typical process for draining mana for an ability that can be triggered
// multiple times in discrete unitc.
type cloakProc struct {
	NullCondition

	// Gid of the Player with this Process
	Gid game.Gid

	// The number of multiples of Unit currently stored
	Cloak        float64
	MaxCloak     float64
	CloakPerTick float64

	// Conversion rate from mana cloak
	ManaPerCloak float64

	Killed bool
}

func (p *cloakProc) ModifyBase(baseStats stats.Base) stats.Base {
	if p.Cloak > p.CloakPerTick {
		baseStats.Cloaking = 1.0
	} else {
		baseStats.Cloaking = 1.0 - (1.0-baseStats.Cloaking)*(1.0-p.Cloak/p.CloakPerTick)
	}
	return baseStats
}
func (p *cloakProc) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (p *cloakProc) CauseDamage() stats.Damage {
	return stats.Damage{}
}

func (p *cloakProc) Supply(mana game.Mana) game.Mana {
	remaining := (p.MaxCloak - p.Cloak) * p.ManaPerCloak
	if mana[game.ColorBlue] > remaining {
		mana[game.ColorBlue] -= remaining
		p.Cloak = p.MaxCloak
	} else {
		p.Cloak += mana[game.ColorBlue] / p.ManaPerCloak
		mana[game.ColorBlue] = 0
	}
	return mana
}
func (p *cloakProc) Think(g *game.Game) {
	if p.Cloak > p.CloakPerTick {
		p.Cloak -= p.CloakPerTick
	} else {
		p.Cloak = 0
	}
	p.Cloak *= 0.98
}
func (p *cloakProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *cloakProc) Dead() bool {
	return p.Killed
}
