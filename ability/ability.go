package ability

import (
	// 	"github.com/runningwild/cgf"
	// 	"github.com/runningwild/glop/gin"
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

// type NonResponder struct{}

// func (NonResponder) Respond(gid game.Gid, group gin.EventGroup) bool { return false }

// type NeverActive struct {
// 	NonResponder
// }

// func (NeverActive) Deactivate(gid game.Gid) []cgf.Event { return nil }

// type NonThinker struct{}

// func (NonThinker) Think(game.Gid, *game.Game) ([]cgf.Event, bool) { return nil, false }

// type NonRendering struct{}

// func (NonRendering) Draw(gid game.Gid, game *game.Game, side int) {}

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
