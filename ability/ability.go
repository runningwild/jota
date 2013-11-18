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
