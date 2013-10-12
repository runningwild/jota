package effects

import (
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
)

func makeSilence(params map[string]int) game.Process {
	var s silence
	s.Duration = params["duration"]
	s.Ticker = params["duration"]
	return &s
}

func init() {
	game.RegisterEffect("silence", makeSilence)
}

type silence struct {
	Duration int
	Ticker   int
}

func (s *silence) Supply(mana game.Mana) game.Mana {
	return mana
}
func (s *silence) ModifyBase(b stats.Base) stats.Base {
	b.Rate *= 1.0 - float64(s.Ticker)/float64(s.Duration)
	return b
}
func (s *silence) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (s *silence) CauseDamage() stats.Damage {
	return stats.Damage{}
}
func (s *silence) Think(g *game.Game) {
	s.Ticker--
}
func (s *silence) Kill(g *game.Game) {
	s.Ticker = 0
}
func (s *silence) Phase() game.Phase {
	if s.Ticker <= 0 {
		return game.PhaseComplete
	}
	return game.PhaseRunning
}
func (s *silence) Draw(id game.Gid, g *game.Game, side int) {
}
