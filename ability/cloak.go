package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	// "github.com/runningwild/linear"
	// "math"
	// "math/rand"
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
		return
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

func (p *cloakProc) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	gl.Color4ub(255, 0, 0, 255)
	frac := float32(p.Cloak / p.MaxCloak)
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "frac", frac)
	base.SetUniformF("status_bar", "inner", 0.2)
	base.SetUniformF("status_bar", "outer", 0.23)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	base.EnableShader("")
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
