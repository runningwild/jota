package ability

import (
  "github.com/runningwild/magnus/base"
  "github.com/runningwild/magnus/game"
  "github.com/runningwild/magnus/stats"
)

type NoRendering struct{}

func (NoRendering) Draw(game *game.Game) {}

type BasicPhases struct {
  The_phase game.Phase
}

func (bp *BasicPhases) Kill(g *game.Game) {
  bp.The_phase = game.PhaseComplete
}

func (bp *BasicPhases) Terminated() bool {
  return bp.The_phase == game.PhaseComplete
}

func (bp *BasicPhases) Phase() game.Phase {
  base.Log().Printf("Returning %d", bp.The_phase)
  return bp.The_phase
}

type NullCondition struct{}

func (NullCondition) ModifyBase(base stats.Base) stats.Base {
  return base
}
func (NullCondition) ModifyDamage(damage stats.Damage) stats.Damage {
  return damage
}
func (NullCondition) CauseDamage() stats.Damage {
  return stats.Damage{}
}
