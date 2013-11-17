package game

import (
	"encoding/gob"
	"github.com/runningwild/jota/base"
)

// An Ability represents something a player can do that does not directly affect
// the game state.
type Ability interface {
	Input(ent Ent, game *Game, pressAmt float64, trigger bool)
	Think(ent Ent, game *Game)
	Draw(ent Ent, game *Game)

	// Returns true if this is an ability that is in use that uses a trigger
	// buttons.  Abilities, like Cloaking, that don't require a trigger can be
	// used simultaneously with other abilities.
	IsActive() bool
}

type AbilityMaker func(params map[string]float64) Ability

var ability_makers map[string]AbilityMaker

func RegisterAbility(name string, maker AbilityMaker) {
	if ability_makers == nil {
		ability_makers = make(map[string]AbilityMaker)
	}
	ability_makers[name] = maker
}

type UseAbility struct {
	Gid     Gid
	Index   int
	Button  float64
	Trigger bool
}

func init() {
	gob.Register(UseAbility{})
}

func (m UseAbility) Apply(_g interface{}) {
	g := _g.(*Game)
	ent, ok := g.Ents[m.Gid]
	if !ok || ent == nil {
		base.Error().Printf("Got a use ability that made no sense: %v", m)
		return
	}
	abilities := ent.Abilities()
	if m.Index < 0 || m.Index >= len(abilities) {
		base.Error().Printf("Got a UseAbility on index %d with only %d abilities.", m.Index, len(abilities))
		return
	}

	// Don't use the ability if any other abilities are active
	anyActive := false
	for i, ability := range abilities {
		if i == m.Index {
			// It's ok to send input to the active ability, so if the ability this
			// command is trying to use is active that's ok.
			continue
		}
		if ability.IsActive() {
			anyActive = true
			break
		}
	}
	if anyActive {
		return
	}
	abilities[m.Index].Input(ent, g, m.Button, m.Trigger)
}
