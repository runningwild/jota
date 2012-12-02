package ability

import (
  "runningwild/tron/base"
  "runningwild/tron/game"
  "runningwild/tron/stats"
)

type noRendering struct{}

func (noRendering) Draw(game *game.Game) {}

type basicPhases struct {
  The_phase game.Phase
}

func (bp *basicPhases) Kill(g *game.Game) {
  bp.The_phase = game.PhaseComplete
}

func (bp *basicPhases) Terminated() bool {
  return bp.The_phase == game.PhaseComplete
}

func (bp *basicPhases) Phase() game.Phase {
  base.Log().Printf("Returning %d", bp.The_phase)
  return bp.The_phase
}

type nullCondition struct{}

func (nullCondition) ModifyBase(base stats.Base) stats.Base {
  return base
}
func (nullCondition) ModifyDamage(damage stats.Damage) stats.Damage {
  return damage
}
func (nullCondition) CauseDamage() stats.Damage {
  return stats.Damage{}
}
