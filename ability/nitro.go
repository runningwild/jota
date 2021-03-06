package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
)

func makeNitro(params map[string]float64) game.Ability {
	var n nitro
	n.id = NextAbilityId()
	n.maxNitro = params["maxNitro"]
	n.manaPerNitro = params["manaPerNitro"]
	n.nitroPerTick = params["nitroPerTick"]
	return &n
}

func init() {
	game.RegisterAbility("nitro", makeNitro)
	gob.Register(&nitro{})
}

type nitro struct {
	id int

	// Params
	maxNitro     float64
	manaPerNitro float64
	nitroPerTick float64

	on       bool
	previous struct {
		pressAmt float64
		trigger  bool
	}
}

func (n *nitro) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	player := ent.(*game.PlayerEnt)
	if pressAmt == n.previous.pressAmt {
		return
	}
	if !n.on {
		n.on = pressAmt > 0
	} else {
		n.on = pressAmt == 0
	}
	if n.on {
		player.Processes[n.id] = &nitroProc{Gid: player.Gid, MaxNitro: n.maxNitro, ManaPerNitro: n.manaPerNitro, NitroPerTick: n.nitroPerTick}
	} else {
		delete(player.Processes, n.id)
		return
	}
}

func (n *nitro) Think(ent game.Ent, g *game.Game) {
}
func (f *nitro) Draw(ent game.Ent, g *game.Game) {
}
func (f *nitro) IsActive() bool {
	return false
}

// Typical process for draining mana for an ability that can be triggered
// multiple times in discrete unitn.
type nitroProc struct {
	NullCondition

	// Gid of the Player with this Process
	Gid game.Gid

	// The number of multiples of Unit currently stored
	Nitro        float64
	MaxNitro     float64
	NitroPerTick float64

	// Conversion rate from mana nitro
	ManaPerNitro float64

	Killed bool
}

func (p *nitroProc) ModifyBase(baseStats stats.Base) stats.Base {
	if p.Nitro > p.NitroPerTick {
		baseStats.Acc += p.NitroPerTick
	} else {
		baseStats.Acc += p.Nitro
	}
	return baseStats
}
func (p *nitroProc) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (p *nitroProc) CauseDamage() stats.Damage {
	return stats.Damage{}
}

func (p *nitroProc) Supply(mana game.Mana) game.Mana {
	remaining := (p.MaxNitro - p.Nitro) * p.ManaPerNitro
	if mana[game.ColorRed] > remaining {
		mana[game.ColorRed] -= remaining
		p.Nitro = p.MaxNitro
	} else {
		p.Nitro += mana[game.ColorRed] / p.ManaPerNitro
		mana[game.ColorRed] = 0
	}
	return mana
}
func (p *nitroProc) Think(g *game.Game) {
	if p.Nitro > p.NitroPerTick {
		p.Nitro -= p.NitroPerTick
	} else {
		p.Nitro = 0
	}
	p.Nitro *= 0.98
}
func (p *nitroProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *nitroProc) Dead() bool {
	return p.Killed
}
