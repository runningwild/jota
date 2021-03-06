package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
)

func makeShield(params map[string]float64) game.Ability {
	var s shield
	s.id = NextAbilityId()
	s.maxShield = params["maxShield"]
	s.manaPerShield = params["manaPerShield"]
	return &s
}

func init() {
	game.RegisterAbility("shield", makeShield)
	gob.Register(&shield{})
}

type shield struct {
	id int

	// Params
	maxShield     float64
	manaPerShield float64

	on       bool
	previous struct {
		pressAmt float64
		trigger  bool
	}
}

func (s *shield) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	player := ent.(*game.PlayerEnt)
	if pressAmt == s.previous.pressAmt {
		return
	}
	if !s.on {
		s.on = pressAmt > 0
	} else {
		s.on = pressAmt == 0
	}
	if s.on {
		player.Processes[s.id] = &shieldProc{Gid: player.Gid, MaxShield: s.maxShield, ManaPerShield: s.manaPerShield}
	} else {
		delete(player.Processes, s.id)
		return
	}
}

func (s *shield) Think(ent game.Ent, g *game.Game) {
}
func (f *shield) Draw(ent game.Ent, g *game.Game) {
}
func (f *shield) IsActive() bool {
	return false
}

// Typical process for draining mana for an ability that can be triggered
// multiple times in discrete units.
type shieldProc struct {
	NullCondition

	// Gid of the Player with this Process
	Gid game.Gid

	// The number of multiples of Unit currently stored
	Shield    float64
	MaxShield float64

	// Conversion rate from mana shield
	ManaPerShield float64

	Killed bool
}

func (p *shieldProc) ModifyBase(base stats.Base) stats.Base {
	return base
}
func (p *shieldProc) ModifyDamage(damage stats.Damage) stats.Damage {
	if damage.Amt > p.Shield {
		damage.Amt -= p.Shield
		p.Shield = 0
	} else {
		p.Shield -= damage.Amt
		damage.Amt = 0
	}
	return damage
}
func (p *shieldProc) CauseDamage() stats.Damage {
	return stats.Damage{}
}

func (p *shieldProc) Supply(mana game.Mana) game.Mana {
	remaining := (p.MaxShield - p.Shield) * p.ManaPerShield
	if mana[game.ColorGreen] > remaining {
		mana[game.ColorGreen] -= remaining
		p.Shield = p.MaxShield
	} else {
		p.Shield += mana[game.ColorGreen] / p.ManaPerShield
		mana[game.ColorGreen] = 0
	}
	return mana
}
func (p *shieldProc) Think(g *game.Game) {
	p.Shield *= 0.98
}
func (p *shieldProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *shieldProc) Dead() bool {
	return p.Killed
}
