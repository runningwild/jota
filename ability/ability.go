package ability

import (
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/pnf"
)

type neverActive struct{}

func (neverActive) Deactivate(player_id int) []pnf.Event { return nil }

type nonResponder struct{}

func (nonResponder) Respond(player_id int, group gin.EventGroup) bool { return false }

type nonThinker struct{}

func (nonThinker) Think(player_id int) ([]pnf.Event, bool) { return nil, false }

type nonRendering struct{}

func (nonRendering) Draw(player_id int, game *game.Game) {}

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
