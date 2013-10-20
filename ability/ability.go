package ability

import (
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	"sync"
)

var abilityId int
var abilityIdMutex sync.Mutex

func NextAbilityId() int {
	abilityIdMutex.Lock()
	defer abilityIdMutex.Unlock()
	abilityId++
	return abilityId
}

type NonResponder struct{}

func (NonResponder) Respond(gid game.Gid, group gin.EventGroup) bool { return false }

type NeverActive struct {
	NonResponder
}

func (NeverActive) Deactivate(gid game.Gid) []cgf.Event { return nil }

type NonThinker struct{}

func (NonThinker) Think(game.Gid, *game.Game) ([]cgf.Event, bool) { return nil, false }

type NonRendering struct{}

func (NonRendering) Draw(gid game.Gid, game *game.Game, side int) {}

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
