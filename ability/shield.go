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

func (p *shieldProc) Draw(src, obs game.Gid, game *game.Game) {
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	gl.Color4ub(0, 0, 255, 255)
	frac := float32(p.Shield / p.MaxShield)
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "frac", frac)
	base.SetUniformF("status_bar", "inner", 0.2)
	base.SetUniformF("status_bar", "outer", 0.23)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	base.EnableShader("")
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
